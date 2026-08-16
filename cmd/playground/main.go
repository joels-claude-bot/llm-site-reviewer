package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"log/slog"
	// Go 1.21+

	"github.com/invopop/jsonschema"
	"github.com/voocel/litellm"
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

var opts = slog.HandlerOptions{}
var logger = slog.New(slog.NewTextHandler(os.Stdout, &opts))

// func testDocSummarise(client *litellm.Client) error {
// 	docPath := "/home/claude/llm-site-reviewer/testie/research.md"
// 	content, err := os.ReadFile(docPath)
// 	if err != nil {
// 		return fmt.Errorf("failed to read file %v: %w", docPath, err)
// 	}
// 	liteLlmMessages := []litellm.Message{litellm.System("Summarise the content of this text file in no less than 200 words. The purpose is provide a condensed version of the document that takes the key details, and anything that might be needed to know if this document is needed for context in another"), litellm.UserText(string(content))}
//
// 	resp, err := client.Chat(context.Background(), litellm.Request{
// 		Model:    "qwen2.5vl:7b",
// 		Messages: liteLlmMessages,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to run litellm request %w", err)
// 	}
// 	fmt.Println(resp.Text())
// 	fmt.Printf("%#v\n", resp.Usage)
// 	// 1. Initialize reader and parser
// 	reader := text.NewReader(content)
// 	parser := goldmark.DefaultParser()
//
// 	// 2. Parse Markdown into an AST root node
// 	doc := parser.Parse(reader)
//
// 	var imagePaths []string
//
// 	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
// 		img, ok := n.(*ast.Image)
// 		if !entering && ok {
// 			dest := string(img.Destination)
// 			imagePaths = append(imagePaths, dest)
// 		}
// 		return ast.WalkContinue, nil
// 	})
//
// 	return nil
// }

// proxyBaseURL is our LiteLLM proxy (defined in dev-setup's Nix config). The
// proxy holds the real provider keys (OpenRouter, Gemini) via systemd; clients
// only ever send the master key. That is the whole point of routing through it.
const proxyBaseURL = "http://127.0.0.1:9177/v1"

func SummariseImage(client *litellm.Client, imagePath string, content []byte) (string, error) {

	ext := filepath.Ext(imagePath)
	mime := mime.TypeByExtension(ext)

	var messages = []litellm.Message{litellm.System("Summarise this image in 200 words or less, covering as much as detail as you can about labels and edges")}

	if ext == ".svg" {
		messages = append(messages, litellm.UserText(string(content)))
	} else {
		messages = append(messages, litellm.User(litellm.ImageBlock{Data: content, MIME: mime}))
	}

	resp, err := client.Chat(context.Background(), litellm.Request{
		Model:    "qwen3-vl", // the model_name from the proxy's model_list
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("image summarisation failed for %v (%v): %w", imagePath, ext, err)
	}
	return resp.Text(), nil
}

// printModelCapabilities lists the client's models and their advertised capabilities
// NOTE: doesnt work as we don't get vision back
// func printModelCapabilities(client *litellm.Client) error {
// 	models, err := client.ListModels(context.Background())
// 	if err != nil {
// 		return fmt.Errorf("failed to list models %w", err)
// 	}
// 	for _, model := range models {
// 		fmt.Printf("%-30s tools=%t vision=%t thinking=%t\n",
// 			model.Name, model.SupportsTools, model.SupportsVision, model.SupportsThinking)
// 	}
// 	return nil
// }

// func runWithLiteLLM() error {
// 	client, err := ollama.NewClient(ollama.Config{BaseURL: "http://desktop-work:11434/v1"})
// 	if err != nil {
// 		return fmt.Errorf("failed to load ollama client %w", err)
// 	}
// 	ctx := context.Background()
// 	models, err := client.ListModels(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to get ollama models %w", err)
// 	}
// 	fmt.Printf("got %v models from ollama\n", len(models))
// 	for i, model := range models {
// 		fmt.Println("model ", i, ": ", model.Name)
// 	}
// 	err = testDocSummarise(client)
// 	if err != nil {
// 		return fmt.Errorf("failed to run litellm request %w", err)
// 	}
// 	return nil
//
// }

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

type MdImage struct {
	ImagePath           string `json:"imagePath"`
	ImamgeSummarisation string `json:"imageSummarisation"`
}

type MdEntry struct {
	ItemPath      string    `json:"itemPath"`
	Content       string    `json:"content"`
	Images        []MdImage `json:"images"`
	Summarisation string    `json:"summarisation"`
}

type MDWithDeps struct {
	itemPath      string
	content       string
	images        []MdImage
	summarisation string
	dependencies  []*MDWithDeps
}

func getMarkdownImages(content []byte, rootPath string) ([]string, error) {
	// 1. Initialize reader and parser
	reader := text.NewReader(content)
	parser := goldmark.DefaultParser()

	// 2. Parse Markdown into an AST root node
	doc := parser.Parse(reader)

	var relPaths []string

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		img, ok := n.(*ast.Image)
		if !entering && ok {
			dest := string(img.Destination)
			relPaths = append(relPaths, dest)
		}
		return ast.WalkContinue, nil
	})

	var fullPaths []string
	for _, relPath := range relPaths {
		fullPath := filepath.Join(rootPath, relPath)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to get abs path for %v: %w", fullPath, err)
		// }
		fullPaths = append(fullPaths, fullPath)
	}

	return fullPaths, nil
}

func inner(client *litellm.Client, dirPath string, rootPath string) ([]MdEntry, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir path %v %w", dirPath, err)
	}
	var entries []MdEntry
	for _, e := range files {
		if e.IsDir() {

			innerDir := e.Name()
			newEntries, err := inner(client, innerDir, rootPath)
			if err != nil {
				return nil, fmt.Errorf("failed running for dir %v: %w", innerDir, err)
			}
			entries = append(entries, newEntries...)

			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".md" && ext != ".markdown" {
			continue
		}
		fullPath := filepath.Join(dirPath, e.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %v: %w", fullPath, err)
		}
		imagePaths, err := getMarkdownImages(content, rootPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get markdown images from %v: %w", fullPath, err)
		}

		var images []MdImage

		system := `
			Summarise this page in 400 words or less, covering as much as detail as you can about it. 
			It will be used as a reference point for other pages to know if their context is dependent on this page.
			Thus, conveying the core message is vital, but also as many reference points as possible
			`
		var builder strings.Builder

		for i, imagePath := range imagePaths {
			imageContent, err := os.ReadFile(imagePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read image content %v: %w", imagePath, err)
			}
			imageSummarisation, err := SummariseImage(client, imagePath, imageContent)
			if err != nil {
				return nil, fmt.Errorf("failed to summarise image for %v: %w", imagePath, err)
			}
			if i == 0 {
				builder.WriteString("include any relevant details from transcribed images below\n")
			}
			fmt.Fprintf(&builder, "image %v is summarised as follows\n%s\n", imagePath, imageSummarisation)
			images = append(images, MdImage{ImagePath: imagePath, ImamgeSummarisation: imageSummarisation})
		}

		fmt.Fprintf(&builder, "the page content is:\n%v\n", string(content))

		resp, err := client.Chat(context.Background(), litellm.Request{
			Model:    "qwen3-vl",
			Messages: []litellm.Message{litellm.System(system), litellm.UserText(builder.String())},
		})
		if err != nil {
			return nil, fmt.Errorf("summarisation for entry %v failed: %w", fullPath, err)
		}
		summarisation := resp.Text()

		entries = append(entries, MdEntry{ItemPath: fullPath, Content: string(content), Images: images, Summarisation: summarisation})
	}
	return entries, nil
}

func resolveDeps(client *litellm.Client, entries []MdEntry) ([]MDWithDeps, error) {
	type ResponseStatus string
	const StatusSuccess ResponseStatus = "success"
	const StatusError ResponseStatus = "error"

	type MDSummarisation struct {
		ItemPath      string `json:"itemPath"`
		Summarisation string `json:"summarisation"`
	}
	var summarisations []MDSummarisation

	type MdResponse struct {
		Status          string   `json:"status" jsonschema:"enum=success,enum=error"`
		Error           string   `json:"error" jsonschema:"description=error details if status is error"`
		DependencyPaths []string `json:"dependencyPaths" jsonschema:"description=list of dependency paths is status is success"`
	}
	sampleResponse := MdResponse{}
	reflector := &jsonschema.Reflector{
		DoNotReference: true, // inline defs, no $defs block
		ExpandedStruct: true, // no top-level $ref wrapper
	}
	schema := reflector.Reflect(&sampleResponse)
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to stringify schema %w", err)
	}

	format, err := litellm.NewResponseFormatJSONSchema("response_format", "", string(schemaJSON), litellm.StrictEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to get litellm response format schema %w", err)
	}

	for _, entry := range entries {
		summarisations = append(summarisations, MDSummarisation{ItemPath: entry.ItemPath, Summarisation: entry.Summarisation})
	}
	jsonSummarisations, err := json.Marshal(summarisations)
	if err != nil {
		return nil, fmt.Errorf("failed to jsonify summarisations %w", err)
	}

	system := `your task is to compute the dependencies of the following page content
			you can pick up to a maximum of 5 pages to return as dependencies to this
			a dependency is another page that is required to understand the context of this page and review its content
	
			only include pages required to interpret, validate, or resolve
			an explicit reference in this page; do not include pages that are merely part of the same site or hypothetically useful.

			do NOT include images

			this is to be derived from a serialised form of the page and its content and transcribed images, 
			followed by all pages serialised and summarised into a condensed form for you to review
	`
	for _, entry := range entries {

		jsonEntry, err := json.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to jsonify entry %v: %w", entry.ItemPath, err)
		}

		resp, err := client.Chat(context.Background(), litellm.Request{
			Model: "qwen3-vl",
			Messages: []litellm.Message{
				litellm.System(system),
				litellm.UserText(string(jsonEntry)),
				litellm.UserText(string(jsonSummarisations)),
			},
			ResponseFormat: format,
		})
		fmt.Println("\n\n\n\n-------------------- DEPS for", entry.ItemPath, " --------------------------------------")
		fmt.Println("got summarisations\n", resp.Text())
	}

	return nil, nil

}

// // run each summariser with the dependency tree
// func runWithDeps(client *litellm.Client, entryPath string) error {
//
// 	var entries []MdEntry
// 	err := inner(client, entryPath, entries)
// 	if err != nil {
// 		return err
// 	}
//
// 	fmt.Println(entries)
//
// 	return nil
//
// }

func getClient() (*litellm.Client, error) {
	masterKey := os.Getenv("LITELLM_MASTER_KEY")
	if masterKey == "" {
		return nil, fmt.Errorf("LITELLM_MASTER_KEY not set (the client key for the proxy; see dev-setup litellm-env secret)")
	}
	client, err := openai.NewClient(openai.Config{APIKey: masterKey, BaseURL: proxyBaseURL})
	if err != nil {
		return nil, fmt.Errorf("failed to build proxy client %w", err)
	}
	return client, nil

}

func main() {
	client, err := getClient()
	if err != nil {
		panic(err)
	}
	// err = printModelCapabilities(client)
	// if err != nil {
	// 	panic(err)
	// }
	// imagePath := "/home/claude/llm-site-reviewer/testie/route.jpeg"
	// content, err := os.ReadFile(imagePath)
	// if err != nil {
	// 	panic(err)
	// }
	// summarised, err := SummariseImage(client, imagePath, content)
	// }
	// fmt.Printf("got summarisation of %v\n%v\n", imagePath, summarised)
	rootPath := "/home/claude/llm-site-reviewer/testie"
	entries, err := inner(client, rootPath, rootPath)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		fmt.Println("------------------------------------------ entry  ------------------------------------")
		fmt.Println("entry ", entry.ItemPath)
		for i, img := range entry.Images {
			fmt.Println("image ", i, ": ", img.ImagePath)
			fmt.Println("------- image summarisation --------")
			fmt.Println(img.ImamgeSummarisation)
			fmt.Println("-------")
		}
		fmt.Println("--------------entry summarisation -------------")
		fmt.Println(entry.Summarisation)
		fmt.Println("----------------------- END -----------------------------------------------------------")

	}

	resolveDeps(client, entries)
}
