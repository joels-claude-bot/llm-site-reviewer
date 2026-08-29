package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/logging"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/visual"
)

func main() {
	logging.InitFromEnv()

	mode := flag.String("mode", "batch", "review mode: batch, dependency, or visual (rendered page URLs)")
	recursive := flag.Bool("recursive", false, "batch mode: descend into nested subdirectories (default: only immediate files of each target directory)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: inspect [FLAGS] target [target...]\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()

	}
	flag.Parse()

	positional := flag.Args()

	if len(positional) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	switch *mode {
	case "batch":
		err = review.LiteLLMRequest(positional, *recursive)
	case "dependency":
		err = runDependency(positional)
	case "visual":
		err = runVisual(positional)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q (want batch or dependency)\n\n", *mode)
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("failed to call request: %v", err)
	}

}

func runVisual(urls []string) error {
	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return err
	}
	results, err := review.VisualReview(context.Background(), client, urls, "", visual.DefaultConfig())
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

// runDependency wires up the dependency review path. It takes one or more root
// directories (DependencyReview summarises the pages under all of them and
// resolves cross-page dependencies across the combined set) and prints the
// results as a single JSON array, one object per reviewed page.
func runDependency(paths []string) error {
	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return fmt.Errorf("failed to build litellm proxy client: %w", err)
	}

	results, err := review.DependencyReview(context.Background(), client, paths, review.DefaultDependencyConfig())
	if err != nil {
		return err
	}

	// Emit a single JSON result, mirroring batch mode: one object per reviewed
	// page, each carrying the same MdItem shape (id, name, success, errors).
	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dependency results: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}
