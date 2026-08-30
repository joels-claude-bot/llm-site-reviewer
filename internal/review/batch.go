// Review documents content via sending request to LLM
package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
	"github.com/voocel/litellm"

	_ "embed"
)

// input schema for parsing files
type MdEntry struct {
	Id           int
	TextContent  string
	ImageContent []byte
	FileName     string
}

// output schema to LLM
type DocPayload struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content,omitempty"`
}

func (entry MdEntry) payload() DocPayload {
	return DocPayload{Id: entry.Id, Name: entry.FileName, Content: entry.TextContent}
}

// categorisation of content error
type MdError struct {
	Category finding.Category `json:"category"`
	Notes    string           `json:"notes"`
}

// a markdown item with its content reviewed
type MdItem struct {
	Id      int       `json:"id"`
	Name    string    `json:"name"`
	Success bool      `json:"success"`
	Errors  []MdError `json:"errors"`
}

// response from LLM about reviewing an item
type MdResponse struct {
	Results []MdItem `json:"results"`
}

// LiteLLMRequest reviews the docs under paths in one batch: collect the source
// files, wrap them into a prompt, send them to the model under a strict schema,
// and print the response.
func LiteLLMRequest(paths []string, recursive bool) error {
	client, err := NewLLMClient(ClientConfigFromEnv(ProviderGemini))
	if err != nil {
		return fmt.Errorf("failed to load gemini client %w", err)
	}

	systemPrompt, err := BuildSystemPrompt()
	if err != nil {
		return fmt.Errorf("failed to build system prompt %w", err)
	}

	entries, err := buildMessages(paths, recursive)
	if err != nil {
		return fmt.Errorf("failed to build messages %w", err)
	}

	liteLlmMessages, err := buildBatchReviewMessages(systemPrompt, entries)
	if err != nil {
		return fmt.Errorf("failed to build litellm messages %w", err)
	}

	schemaJSON, err := buildJsonSchema()
	if err != nil {
		return fmt.Errorf("failed to buils json Schema %w", err)
	}
	format, err := litellm.NewResponseFormatJSONSchema("response_format", "", string(schemaJSON), litellm.StrictEnabled)
	if err != nil {
		return fmt.Errorf("failed to get litellm response format schema %w", err)
	}

	resp, err := client.Chat(context.Background(), litellm.Request{
		Model:          modelReview,
		Messages:       liteLlmMessages,
		ResponseFormat: format,
	})
	if err != nil {
		return fmt.Errorf("failed to run litellm request %w", err)
	}

	slog.Debug("batch review response received", "usage", fmt.Sprintf("%#v", resp.Usage))
	fmt.Println(resp.Text())
	return nil
}

func BuildSystemPrompt() (string, error) {

	jsonCatalog, err := json.Marshal(finding.Catalog)
	if err != nil {
		return "", fmt.Errorf("failed to marshal finding catalog: %v", err)
	}

	// The shape of each source file as it arrives, so the model knows what the
	// input JSON objects look like (same reflector settings as the response schema).
	reflector := &jsonschema.Reflector{
		DoNotReference: true,
		ExpandedStruct: true,
	}
	inputSchema, err := json.Marshal(reflector.Reflect(&DocPayload{}))
	if err != nil {
		return "", fmt.Errorf("failed to marshal input schema: %w", err)
	}

	result := fmt.Sprintf(`
		The goal is to catch documentation that passes the build but still fails the reader.

		A build can prove that files exist, Markdown compiled, and the site emitted HTML. It cannot prove that the rendered page is useful, truthful, or readable. This reviewer fills that gap by checking the page a visitor actually sees.

		below - you will be fed a list of JSONs of the category codes and examples to report errors and respective examples

		if you spot an error that doesnt fall into a category but should be reported use "OTHER"


		below the list of JSON categories you will get a list of markdown sources files that build the docs to review.
		each source file is provided as a JSON object matching this input schema:
		%v

		in your response you should use the "id" attribute to refer to them, "name" as per the filename you receive, and success true if no errors, otherwise success flase and the list of errors specified

		%v`, string(inputSchema), string(jsonCatalog))
	return result, nil

}

// buildMessages collects reviewable entries from the given paths, then numbers
// them 0..n-1 in traversal order so the model can refer back to each by Id.
func buildMessages(paths []string, recursive bool) ([]MdEntry, error) {
	slog.Debug("building batch review messages", "paths", paths, "recursive", recursive)
	var entries []MdEntry
	for _, path := range paths {
		collected, err := collectPath(path, recursive)
		if err != nil {
			return nil, err
		}
		entries = append(entries, collected...)
	}
	for index := range entries {
		entries[index].Id = index
	}
	slog.Debug("built batch review entries", "entry_count", len(entries))
	return entries, nil
}

// collectPath returns the reviewable entries under path (a file or directory),
// recursing into itself for subdirectories. The path handed in is always read;
// nested subdirectories are only descended into when recursive is true.
func collectPath(path string, recursive bool) ([]MdEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %v: %w", path, err)
	}

	if !info.IsDir() {
		entry, ok, err := entryFromFile(path)
		if err != nil {
			return nil, err
		}
		if ok {
			return []MdEntry{entry}, nil
		}
		return nil, nil
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir path %v %w", path, err)
	}
	var entries []MdEntry
	for _, entry := range files {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			if !recursive {
				slog.Debug("skipping subdirectory", "entry_name", entry.Name())
				continue
			}
			nested, err := collectPath(fullPath, recursive)
			if err != nil {
				return nil, err
			}
			entries = append(entries, nested...)
			continue
		}
		fileEntry, ok, err := entryFromFile(fullPath)
		if err != nil {
			return nil, err
		}
		if ok {
			entries = append(entries, fileEntry)
		}
	}
	return entries, nil
}

// entryFromFile reads one file into an MdEntry - file accepted if ok true. Id is
// left zero and assigned by buildMessages once the full ordered set is known.
func entryFromFile(fullPath string) (parsed MdEntry, ok bool, err error) {
	name := filepath.Base(fullPath)
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".md" && ext != ".markdown" && ext != ".jpeg" && ext != ".png" {
		slog.Debug("skipping non-reviewable file", "file_name", name, "extension", ext)
		return MdEntry{}, false, nil
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return MdEntry{}, false, fmt.Errorf("failed to read file %v: %w", fullPath, err)
	}
	if ext == ".jpeg" || ext == ".png" {
		slog.Debug("adding image to batch review", "file_name", name, "extension", ext)
		return MdEntry{ImageContent: content, FileName: name}, true, nil
	}
	slog.Debug("adding markdown to batch review", "file_name", name, "extension", ext)
	return MdEntry{TextContent: string(content), FileName: name}, true, nil
}

func buildBatchReviewMessages(systemPrompt string, entries []MdEntry) ([]litellm.Message, error) {
	liteLlmMessages := []litellm.Message{litellm.System(systemPrompt)}
	for _, entry := range entries {
		if entry.ImageContent != nil {
			// litellm can see the image: send the payload (id + name, no bytes) as text
			// plus the raw bytes as an image block.
			metaJSON, err := json.Marshal(entry.payload())
			if err != nil {
				return nil, fmt.Errorf("failed to marshal image meta for %v: %w", entry.FileName, err)
			}
			mime := "image/jpeg"
			if strings.HasSuffix(strings.ToLower(entry.FileName), ".png") {
				mime = "image/png"
			}
			liteLlmMessages = append(liteLlmMessages, litellm.User(
				litellm.Text(string(metaJSON)),
				litellm.ImageBlock{Data: entry.ImageContent, MIME: mime},
			))
			continue
		}
		docJSON, err := json.Marshal(entry.payload())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal doc %v: %w", entry.FileName, err)
		}
		liteLlmMessages = append(liteLlmMessages, litellm.UserText(string(docJSON)))
	}
	return liteLlmMessages, nil
}

func buildJsonSchema() ([]byte, error) {
	sampleResponse := MdResponse{}
	// Ollama's grammar converter can't follow $ref/$defs; force a flat, inline
	// schema or it silently stops enforcing (bare strings instead of objects).
	reflector := &jsonschema.Reflector{
		DoNotReference: true, // inline defs, no $defs block
		ExpandedStruct: true, // no top-level $ref wrapper
	}
	schema := reflector.Reflect(&sampleResponse)
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to stringify schema %w", err)
	}
	slog.Debug("built batch review response schema", "schema", string(schemaJSON))
	return schemaJSON, nil
}
