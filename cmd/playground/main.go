package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	// Go 1.21+
	"github.com/ollama/ollama/api"
	"github.com/voocel/litellm"
	"github.com/voocel/litellm/provider/ollama"
	"github.com/voocel/litellm/provider/openai"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// so the workflow is
// #0 - iterate through cached documents - if one removed, all depds need re-computing
// #1 - iterate through live documents: content changed? (vs hash) no - all gd
//			 - if yes (or new), recompute summary
//			 - and re-compute deps

type ImageCache struct {
}

type DocumentFormat string

const (
	FormatText  DocumentFormat = "text"
	FormatImage DocumentFormat = "image"
)

type DocumentCache struct {
	Path          string         `json:"path"`
	Format        DocumentFormat `json:"format"`
	ContentHash   string         `json:"contentHash"`
	Summary       string         `json:"summary"`
	Depdendencies []string       `json:"dependencies"`
}

func testDocSummarise(client *litellm.Client) error {
	docPath := "/home/claude/llm-site-reviewer/testie/research.md"
	content, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("failed to read file %v: %w", docPath, err)
	}
	liteLlmMessages := []litellm.Message{litellm.System("Summarise the content of this text file in no less than 200 words. The purpose is provide a condensed version of the document that takes the key details, and anything that might be needed to know if this document is needed for context in another"), litellm.UserText(string(content))}

	resp, err := client.Chat(context.Background(), litellm.Request{
		Model:    "qwen2.5vl:7b",
		Messages: liteLlmMessages,
	})
	if err != nil {
		return fmt.Errorf("failed to run litellm request %w", err)
	}
	fmt.Println(resp.Text())
	fmt.Printf("%#v\n", resp.Usage)
	// 1. Initialize reader and parser
	reader := text.NewReader(content)
	parser := goldmark.DefaultParser()

	// 2. Parse Markdown into an AST root node
	doc := parser.Parse(reader)

	var imagePaths []string

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		img, ok := n.(*ast.Image)
		if !entering && ok {
			dest := string(img.Destination)
			fmt.Println("found an image: ", dest)
			imagePaths = append(imagePaths, dest)
		}
		return ast.WalkContinue, nil
	})

	fmt.Println(imagePaths)

	return nil
}

func testImageParse() error {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return fmt.Errorf("error getting ollama client from env %w", err)
	}
	imagePath := "/home/claude/llm-site-reviewer/testie/route.jpeg"
	content, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("failed to read file %v: %w", imagePath, err)
	}
	msg := api.Message{Role: "user", Content: "List every text label, number, place name, and arrow/connection in this image, verbatim.", Images: []api.ImageData{content}}
	stream := false
	// model := "qwen2.5vl:3b"
	// too big model - kept overflowing from my VRAM on the RTX 3060
	model := "qwen2.5vl:7b"
	fmt.Println("using model", model)
	req := api.ChatRequest{
		Model:    model,
		Messages: []api.Message{msg},
		Stream:   &stream,
		Options: map[string]any{
			"repeat_penalty": 1.5,
			"repeat_last_n":  256,
			"num_predict":    512,
		},
	}
	ctx := context.Background()
	err = client.Chat(ctx, &req, func(resp api.ChatResponse) error {
		log.Printf("got a response!\n%+v", resp.Message.Content)
		if resp.Done {
			log.Printf("tokens: input=%d output=%d total=%d",
				resp.PromptEvalCount, resp.EvalCount, resp.PromptEvalCount+resp.EvalCount)
			resp.Summary()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ollama chat request failed %v", err)
	}

	return nil
}

// proxyBaseURL is our LiteLLM proxy (defined in dev-setup's Nix config). The
// proxy holds the real provider keys (OpenRouter, Gemini) via systemd; clients
// only ever send the master key. That is the whole point of routing through it.
const proxyBaseURL = "http://127.0.0.1:9177/v1"

// testOpenRouter sends a one-line "hello" to Qwen3-Coder-30B-A3B via our LiteLLM
// proxy. We talk to the proxy as an OpenAI-compatible endpoint: the model name
// is the proxy's model_name ("qwen3-coder"), NOT the OpenRouter slug, and auth
// is the LiteLLM master key — the OpenRouter key stays in the service.
func testOpenRouter() error {
	masterKey := os.Getenv("LITELLM_MASTER_KEY")
	if masterKey == "" {
		return fmt.Errorf("LITELLM_MASTER_KEY not set (the client key for the proxy; see dev-setup litellm-env secret)")
	}

	client, err := openai.NewClient(openai.Config{APIKey: masterKey, BaseURL: proxyBaseURL})
	if err != nil {
		return fmt.Errorf("failed to build proxy client %w", err)
	}

	resp, err := client.Chat(context.Background(), litellm.Request{
		Model:    "qwen3-coder", // the model_name from the proxy's model_list
		Messages: []litellm.Message{litellm.UserText("Say hello in one short sentence.")},
	})
	if err != nil {
		return fmt.Errorf("proxy chat request failed %w", err)
	}
	fmt.Println(resp.Text())
	fmt.Printf("%#v\n", resp.Usage)
	return nil
}

func runWithLiteLLM() error {
	client, err := ollama.NewClient(ollama.Config{BaseURL: "http://desktop-work:11434/v1"})
	if err != nil {
		return fmt.Errorf("failed to load ollama client %w", err)
	}
	ctx := context.Background()
	models, err := client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ollama models %w", err)
	}
	fmt.Printf("got %v models from ollama\n", len(models))
	for i, model := range models {
		fmt.Println("model ", i, ": ", model.Name)
	}
	err = testDocSummarise(client)
	if err != nil {
		return fmt.Errorf("failed to run litellm request %w", err)
	}
	return nil

}

func getCachePath(path string) string {
	return filepath.Join(path, ".doc_cache")
}

// get the parsed list of cache documents from the path specified
func getCache(path string) ([]DocumentCache, error) {
	resolvedPath := getCachePath(path)
	println("using path: ", resolvedPath)
	_, err := os.Stat(resolvedPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		fmt.Println("cache exists")
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, err
	}

	var caches []DocumentCache

	err = json.Unmarshal(data, &caches)
	if err != nil {
		return nil, fmt.Errorf("could not JSON parse file %v: %v", resolvedPath, err)
	}

	return caches, nil
}

func main() {
	myPath := "."
	caches, err := getCache(myPath)
	if err != nil {
		panic(err)
	}

	fmt.Println("caches: ", caches)

	err = runWithLiteLLM()
	fmt.Println("done")
	if err != nil {
		panic(err)
	}

}
