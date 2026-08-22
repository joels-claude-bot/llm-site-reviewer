package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/logging"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
)

func main() {
	logging.InitFromEnv()

	mode := flag.String("mode", "batch", "review mode: batch (all targets in one request) or dependency (per-page with resolved dependencies)")
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
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q (want batch or dependency)\n\n", *mode)
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("failed to call request: %v", err)
	}

}

// runDependency wires up the dependency review path. It takes one or more root
// directories (DependencyReview summarises the pages under all of them and
// resolves cross-page dependencies across the combined set) and prints one line
// per reviewed page.
func runDependency(paths []string) error {
	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return fmt.Errorf("failed to build litellm proxy client: %w", err)
	}

	results, err := review.DependencyReview(context.Background(), client, paths, review.DefaultDependencyConfig())
	if err != nil {
		return err
	}

	for _, result := range results {
		fmt.Printf("%s: %d reviewed item(s)\n", result.PagePath, len(result.Items))
	}
	return nil
}
