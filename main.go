package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/controlledreview"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/logging"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/opencodereview"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/visual"
)

func main() {
	logging.InitFromEnv()

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch mode := os.Args[1]; mode {
	case "batch":
		err = runBatch(os.Args[2:])
	case "dependency":
		err = runDependency(os.Args[2:])
	case "opencode":
		err = runOpencode(os.Args[2:])
	case "controlled":
		err = runControlled(os.Args[2:])
	case "visual":
		err = runVisual(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n\n", mode)
		usage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: llm-site-reviewer <mode> [flags] target...

Modes:
  batch       review markdown files under one or more directories
  dependency  review pages with cross-page dependency context
  opencode    review one page via an opencode server session
  controlled  review one page; Go runs the searches the model requests
  visual      screenshot rendered pages and review the images

Run "llm-site-reviewer <mode> -h" for that mode's flags.
`)
}

func runBatch(args []string) error {
	flags := flag.NewFlagSet("batch", flag.ExitOnError)
	recursive := flags.Bool("recursive", false, "descend into nested subdirectories (default: only immediate files of each target directory)")
	flags.Parse(args)
	if flags.NArg() == 0 {
		return fmt.Errorf("batch mode requires at least one target directory")
	}
	return review.LiteLLMRequest(flags.Args(), *recursive)
}

// runDependency reviews the pages under one or more root directories,
// resolving cross-page dependencies across the combined set.
func runDependency(args []string) error {
	flags := flag.NewFlagSet("dependency", flag.ExitOnError)
	targetList := flags.String("targets", "", "comma-separated page paths to resolve and review (all pages are still summarised)")
	skipImages := flags.Bool("skip-images", false, "skip image transcription for a content-only run")
	recursive := flags.Bool("recursive", true, "descend into nested corpus directories")
	flags.Parse(args)
	if flags.NArg() == 0 {
		return fmt.Errorf("dependency mode requires at least one corpus directory")
	}

	var targets []string
	for target := range strings.SplitSeq(*targetList, ",") {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		absolute, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		targets = append(targets, absolute)
	}

	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return fmt.Errorf("failed to build litellm proxy client: %w", err)
	}
	config := review.DefaultDependencyConfig()
	config.TargetPaths = targets
	config.SkipImages = *skipImages
	config.Recursive = *recursive
	results, err := review.DependencyReview(context.Background(), client, flags.Args(), config)
	if err != nil {
		return err
	}
	return printJSON(results)
}

func runOpencode(args []string) error {
	flags := flag.NewFlagSet("opencode", flag.ExitOnError)
	model := flags.String("model", "litellm/gemini-flash", "provider/model to use")
	baseURL := flags.String("url", "http://127.0.0.1:4096", "opencode server URL")
	verbose := flags.Bool("verbose", false, "log file-discovery and read tool calls")
	timeout := flags.Duration("timeout", 15*time.Minute, "overall session creation and prompt timeout")
	evidenceList := flags.String("evidence", "", "comma-separated workspace-relative evidence files; disables search and permits only reads")
	flags.Parse(args)
	if flags.NArg() != 1 {
		return fmt.Errorf("opencode mode requires exactly one target file")
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	var evidence []string
	for path := range strings.SplitSeq(*evidenceList, ",") {
		if path = strings.TrimSpace(path); path != "" {
			evidence = append(evidence, path)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	return opencodereview.Run(ctx, opencodereview.Config{
		Target: flags.Arg(0), Model: *model, BaseURL: *baseURL,
		Evidence: evidence, Verbose: *verbose, Output: os.Stdout, TraceOutput: os.Stderr,
	})
}

func runControlled(args []string) error {
	flags := flag.NewFlagSet("controlled", flag.ExitOnError)
	root := flags.String("root", "", "workspace root the model may search (required)")
	docsRoot := flags.String("docs-root", "", "docs root for leading-slash image references (default: -root)")
	imageCache := flags.String("image-cache", "", "image transcription cache path (default: <root>/.llm-site-reviewer/image-transcriptions.json)")
	model := flags.String("model", "litellm/gemini-flash", "provider/model to use")
	baseURL := flags.String("url", "http://127.0.0.1:4096", "opencode server URL")
	verbose := flags.Bool("verbose", false, "log phase requests and search tool calls")
	timeout := flags.Duration("timeout", 15*time.Minute, "overall session creation and prompt timeout")
	visionModel := flags.String("vision-model", "qwen3-vl", "LiteLLM vision model for changed raster images")
	flags.Parse(args)
	if flags.NArg() != 1 {
		return fmt.Errorf("controlled mode requires exactly one target file")
	}
	if *root == "" {
		return fmt.Errorf("controlled mode requires -root")
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	rootAbs, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return err
	}
	cfg := controlledreview.Config{Root: rootAbs, DocsRoot: rootAbs}
	if *docsRoot != "" {
		if cfg.DocsRoot, err = filepath.Abs(*docsRoot); err != nil {
			return err
		}
	}
	cfg.ImageCachePath = *imageCache
	if cfg.ImageCachePath == "" {
		cfg.ImageCachePath = filepath.Join(rootAbs, ".llm-site-reviewer", "image-transcriptions.json")
	}
	var traceOutput io.Writer
	if *verbose {
		traceOutput = os.Stderr
		cfg.TraceOutput = traceOutput
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	phaseModel, err := opencodereview.NewControlledPhaseModel(ctx, opencodereview.ControlledConfig{
		BaseURL: *baseURL, Model: *model, Workspace: rootAbs, TraceOutput: traceOutput,
	})
	if err != nil {
		return err
	}
	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return fmt.Errorf("failed to build litellm proxy client: %w", err)
	}
	engine, err := controlledreview.New(cfg, phaseModel, transcriber{client: client, model: *visionModel})
	if err != nil {
		return err
	}
	result, err := engine.Run(ctx, targetAbs)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runVisual(args []string) error {
	flags := flag.NewFlagSet("visual", flag.ExitOnError)
	flags.Parse(args)
	if flags.NArg() == 0 {
		return fmt.Errorf("visual mode requires at least one URL")
	}
	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return err
	}
	results, err := review.VisualReview(context.Background(), client, flags.Args(), "", visual.DefaultConfig())
	if err != nil {
		return err
	}
	return printJSON(results)
}

// transcriber adapts the shared review client to controlledreview.ImageTranscriber.
type transcriber struct {
	client review.ChatClient
	model  string
}

func (t transcriber) TranscribeImage(ctx context.Context, path string, content []byte) (string, error) {
	return review.TranscribeImage(ctx, t.client, t.model, path, content)
}

func printJSON(value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}
