// Package review is the spine: it runs each pass over a site and collects their
// findings into one report. It is not built yet; this file marks the package
// and holds the map, so the shape shows up in the tree before the code does.
//
// How the pieces fit:
//
//	corpus/*.md ──> corpus.Load ──> []corpus.Fixture     (expected answers, for tests)
//
//	a built site ──> review.Review ──> []finding.Finding ──> FormatReport ──> output
//	                     │
//	                     ├─ links.Check      deterministic, no model
//	                     ├─ render.Check      model: looks at screenshots
//	                     ├─ clarity.Check     model: reads the prose
//	                     └─ mismatch.Check    mixed
//
// Every pass returns finding.Finding values, so review does not care how a pass
// reached its answer. The test side feeds the same passes a known corpus and
// compares their findings against the Fixture answers corpus.Load produced.
//
//arch:spine
package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
	"github.com/voocel/litellm"
	"github.com/voocel/litellm/provider/gemini"

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

type MdError struct {
	Category finding.Category `json:"category"`
	Notes    string           `json:"notes"`
}
type MdItem struct {
	Id      int       `json:"id"`
	Name    string    `json:"name"`
	Success bool      `json:"success"`
	Errors  []MdError `json:"errors"`
}

type MdResponse struct {
	Items []MdItem `json:"items"`
}

func buildMessages(dirPath string) (error, []MdEntry) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read dir path %v %w", dirPath, err), nil
	}
	var entries []MdEntry
	for i, e := range files {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		fullPath := filepath.Join(dirPath, e.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read file %v: %w", fullPath, err), nil
		}
		if ext == ".jpeg" || ext == ".png" {
			log.Println("got image", i, e.Name())
			entries = append(entries, MdEntry{Id: i, ImageContent: content, FileName: e.Name()})
			continue
		}
		if ext != ".md" && ext != ".markdown" {
			log.Println("skipping", i, e.Name(), "is not a markdown file")
			continue
		}
		fmt.Println("got entry", i, e.Name())
		entries = append(entries, MdEntry{Id: i, TextContent: string(content), FileName: e.Name()})

	}
	return nil, entries
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
	fmt.Println("schema JSON for parsing:\n", string(schemaJSON))
	return schemaJSON, nil
}

func buildSystemPrmopt() (string, error) {

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

func LiteLLMRequest(dirPath string) error {
	client, err := gemini.NewClient(gemini.Config{APIKey: os.Getenv("GEMINI_API_KEY")})
	if err != nil {
		return fmt.Errorf("failed to load gemini client %w", err)
	}

	systemPrompt, err := buildSystemPrmopt()
	if err != nil {
		return fmt.Errorf("failed to build system prompt %w", err)
	}

	err, entries := buildMessages(dirPath)
	if err != nil {
		return fmt.Errorf("failed to build messages %w", err)
	}

	liteLlmMessages := []litellm.Message{litellm.System(systemPrompt)}
	for _, entry := range entries {
		if entry.ImageContent != nil {
			// litellm can see the image: send the payload (id + name, no bytes) as text
			// plus the raw bytes as an image block.
			metaJSON, err := json.Marshal(entry.payload())
			if err != nil {
				return fmt.Errorf("failed to marshal image meta for %v: %w", entry.FileName, err)
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
			return fmt.Errorf("failed to marshal doc %v: %w", entry.FileName, err)
		}
		liteLlmMessages = append(liteLlmMessages, litellm.UserText(string(docJSON)))
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
		Model:          "gemini-3.6-flash",
		Messages:       liteLlmMessages,
		ResponseFormat: format,
	})
	if err != nil {
		return fmt.Errorf("failed to run litellm request %w", err)
	}

	fmt.Println(resp.Text())
	fmt.Printf("%#v\n", resp.Usage)
	return nil

}
