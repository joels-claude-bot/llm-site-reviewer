// and what Load parsed from them. Run it with no arguments for the default dump
// of the whole corpus, or pass a path to inspect one directory or file.
//
//	go run ./cmd/inspect                                    # dump the whole corpus
//	go run ./cmd/inspect corpus/links/broken-internal.md    # one file
//
//arch:tool
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
	"github.com/ollama/ollama/api"
	"github.com/voocel/litellm"
	"github.com/voocel/litellm/provider/gemini"
	"gopkg.in/yaml.v2"

	_ "embed"
)

//go:embed broken-anchor.md
var sampleCorpusBytes []byte

func print_fixture(fixture corpus.Fixture) {
	bytes, err := yaml.Marshal(fixture)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println(string(bytes))

}

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

func liteLllmRequest(dirPath string) error {
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

// HasVisionCapability reports whether the named Ollama model can accept images.
// Ollama populates resp.Capabilities (e.g. ["completion", "vision"]).
func HasVisionCapability(ctx context.Context, client *api.Client, modelName string) (bool, error) {
	resp, err := client.Show(ctx, &api.ShowRequest{Name: modelName})
	if err != nil {
		return false, fmt.Errorf("failed to show model %s: %w", modelName, err)
	}
	return slices.Contains(resp.Capabilities, "vision"), nil
}

func ollamaRequest(dirPath string) error {
	ctx := context.Background()
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return fmt.Errorf("error getting ollama client from env %w", err)
	}

	// Model options (swap the string):
	//   qwen3:30b-a3b        — text-only MoE reasoner (current)
	//   qwen2.5vl:7b         — vision, fits 12GB
	//   gpt-oss:20b / :120b  — text-only
	//   deepseek-v4-flash:cloud / minimax-m3:cloud — cloud (no schema enforcement)
	model := "qwen3:30b-a3b"

	hasVision, err := HasVisionCapability(ctx, client, model)
	if err != nil {
		return fmt.Errorf("failed to check vision capability for %v: %w", model, err)
	}
	log.Printf("model %s vision=%v", model, hasVision)

	systemPrompt, err := buildSystemPrmopt()
	if err != nil {
		return fmt.Errorf("failed to build system prompt %w", err)
	}

	err, entries := buildMessages(dirPath)
	if err != nil {
		return fmt.Errorf("failed to build messages %w", err)
	}

	ollamaMessages := []api.Message{{Role: "system", Content: systemPrompt}}
	for _, entry := range entries {
		if entry.ImageContent != nil && !hasVision {
			// model is text-only: it has no vision encoder, so drop the image.
			log.Println("ollama: model is text-only, dropping image", entry.FileName)
			continue
		}
		docJSON, err := json.Marshal(entry.payload())
		if err != nil {
			return fmt.Errorf("failed to marshal doc %v: %w", entry.FileName, err)
		}
		if entry.ImageContent != nil {
			// one image per message, with the id/name in the text (Ollama images carry no id).
			ollamaMessages = append(ollamaMessages, api.Message{Role: "user", Content: string(docJSON), Images: []api.ImageData{entry.ImageContent}})
			continue
		}
		ollamaMessages = append(ollamaMessages, api.Message{Role: "user", Content: string(docJSON)})
	}

	schemaJSON, err := buildJsonSchema()
	if err != nil {
		return fmt.Errorf("failed to buils json Schema %w", err)
	}

	stream := false
	req := api.ChatRequest{
		Model:    model,
		Messages: ollamaMessages,
		Stream:   &stream,
		Format:   schemaJSON,
		// Options: map[string]any{
		// 	"num_ctx": 16384, // ollama defaults to a tiny 4096; raise so we never hit the buggy context-shift path
		// },
	}
	err = client.Chat(ctx, &req, func(resp api.ChatResponse) error {
		log.Printf("got a response!\n%+v", resp.Message.Content)
		// token counts only arrive on the final (Done) response
		if resp.Done {
			log.Printf("tokens: input=%d output=%d total=%d",
				resp.PromptEvalCount, resp.EvalCount, resp.PromptEvalCount+resp.EvalCount)
			// Ollama's built-in dump: durations + tokens/s, printed to stderr
			resp.Summary()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ollama chat request failed %v", err)
	}
	return nil

}

func main() {
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "set for super-duper logging")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: inspect [FLAGS] target\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()

	}
	flag.Parse()

	if verbose {
		fmt.Println("\n\nYOU ASKED for verbosity")
	}

	positional := flag.Args()
	if len(positional) > 1 {
		fmt.Fprintf(os.Stderr, "too many positional args!\n\n")
		flag.Usage()
		os.Exit(1)
	}
	target := "corpus"
	if len(positional) == 1 {
		target = positional[0]
	}
	fmt.Println("using target:", target)

	fmt.Println("using test corpus for inspect: broken-anchor.md")
	fmt.Println("below is the yamlified version of the parsed corpus:\n\t")
	test_fixture, err := corpus.Parse(sampleCorpusBytes)
	if err != nil {
		log.Fatalf("failed to parse sample corpus %v", err)
	}
	print_fixture(test_fixture)

	fmt.Println("below is a blank fixture with values (not) provided!:", "\n")
	blank_fixture := corpus.Fixture{}
	print_fixture(blank_fixture)

	myReader := strings.NewReader(string(sampleCorpusBytes))
	var myFinding corpus.Fixture
	parseResult, err := frontmatter.Parse(myReader, &myFinding)
	if err != nil {
		log.Fatalf("failed to parse frontmatter %+v", err)
	}
	fmt.Printf("got finding! %+v\n", myFinding)
	fmt.Println("rest is: ", string(parseResult))

	paths, err := corpus.FixturePaths("corpus")
	if err != nil {
		log.Fatalf("error getting fixture paths %v", err)
	}

	log.Println("got ", len(paths), "paths")
	for i, v := range paths {
		log.Println(i, ": ", v)
	}

	parsed, err := corpus.Load("corpus")
	if err != nil {
		log.Fatalf("error loading fixtures")
	}
	for i, v := range parsed {
		log.Println("fixture ", i, ": ", v.Path, ", title: ", v.Title)
	}

	reviewPath := "testie"
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "ollama" {
		err = ollamaRequest(reviewPath)
	} else if provider == "litellm" {
		err = liteLllmRequest(reviewPath)
	} else {
		log.Fatalf("cannot deal with unknown provider %v", provider)
	}

	if err != nil {
		log.Fatalf("failed to call request with provier %v: %v", provider, err)
	}

}
