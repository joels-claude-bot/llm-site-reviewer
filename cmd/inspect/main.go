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
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
	"github.com/ollama/ollama/api"
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

// Define a struct matching your front matter YAML
type Expectation struct {
	Category string `yaml:"category"`
	Where    string `yaml:"where"`
	Result   string `yaml:"result"`
}

type MyConfig struct {
	Title  string        `yaml:"title"`
	Expect []Expectation `yaml:"expect"`
	Note   string        `yaml:"note"`
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

	// 	prompts := []string{
	// 		"you are reviewing to following markdown content according to set rules",
	// 		`The goal is to catch documentation that passes the build but still fails the reader.
	//
	// A build can prove that files exist, Markdown compiled, and the site emitted HTML. It cannot prove that the rendered page is useful, truthful, or readable. This reviewer fills that gap by checking the page a visitor actually sees`,
	// 	}

	dirPath := "./testie/"
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		log.Fatalf("failed to read testie files %v", err)
	}

	type MdEntry struct {
		Id       int
		Content  string
		FileName string
	}
	type MdError struct {
		Category finding.Category `json:"category"`
		Notes    string           `json:"notes"`
	}
	type MdItem struct {
		Id      int       `json:"id"`
		Success bool      `json:"success"`
		Errors  []MdError `json:"errors"`
	}
	type MdResponse struct {
		Items []MdItem `json:"items"`
	}

	var messages []api.Message
	messages = append(messages, api.Message{Role: "user", Content: `
		The goal is to catch documentation that passes the build but still fails the reader.

		A build can prove that files exist, Markdown compiled, and the site emitted HTML. It cannot prove that the rendered page is useful, truthful, or readable. This reviewer fills that gap by checking the page a visitor actually sees.

		below - you will be fed a list of JSONs of the category codes and examples to report errors and respective examples

		if you spot an error that doesnt fall into a category but should be reported use "OTHER"

		below the list of JSON categories you will get a list of markdown sources files that build the docs to review.
		in your response you shuold use the "id" attribute to refer to them, and success true if no errors, otherwise success flase and the list of errors specified
	`})
	jsonCatalog, err := json.Marshal(finding.Catalog)
	if err != nil {
		log.Fatalf("could not jsonify catalog %v", err)
	}
	messages = append(messages, api.Message{Role: "user", Content: string(jsonCatalog)})

	mdEntries := []MdEntry{}
	for i, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".md" && ext != ".markdown" {
			log.Println("skipping", i, e.Name(), "is not a markdown file")
			continue
		}
		fullPath := filepath.Join(dirPath, e.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Fatalf("could not read file %v: %v", fullPath, err)
		}
		fmt.Println("got entry", i, e)
		mdEntries = append(mdEntries, MdEntry{Id: i, Content: string(content), FileName: e.Name()})

		fmt.Println("added to entries:\n", e.Name(), "\n", string(content))
	}

	jsonEntries, err := json.Marshal(mdEntries)
	if err != nil {
		log.Fatalf("error jsonifying entries %v", err)
	}
	messages = append(messages, api.Message{Role: "user", Content: string(jsonEntries)})

	ctx := context.Background()
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatalf("error getting client from env %v", err)
	}

	sampleResponse := MdResponse{}
	schema := jsonschema.Reflect(&sampleResponse)
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		log.Fatalf("failed to jsonify schema %v", err)
	}

	stream := false
	req := api.ChatRequest{
		// Model:    "qwen2.5vl:3b",
		Model:    "gemini-3-flash-preview",
		Messages: messages,
		Stream:   &stream,
		Format:   schemaJSON,
	}
	err = client.Chat(ctx, &req, func(resp api.ChatResponse) error {
		log.Printf("got a response! %+v", resp.Message.Content)
		return nil
	})
	if err != nil {
		log.Fatalf("chat request failed %v", err)
	}
	log.Println("dunzo")

}
