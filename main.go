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
	"sync"
	"time"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/controlledreview"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/logging"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/opencodereview"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/visual"
)

func main() {
	logging.InitFromEnv()

	mode := flag.String("mode", "batch", "review mode: batch, dependency, opencode, controlled, or visual")
	recursive := flag.Bool("recursive", false, "batch mode: descend into nested subdirectories (default: only immediate files of each target directory)")
	model := flag.String("model", "litellm/gemini-flash", "opencode mode: provider/model to use")
	opencodeURL := flag.String("opencode-url", "http://127.0.0.1:4096", "opencode mode: server URL")
	opencodeVerbose := flag.Bool("opencode-verbose", false, "opencode mode: log file-discovery and read tool calls")
	opencodeTimeout := flag.Duration("opencode-timeout", 15*time.Minute, "opencode mode: overall session creation and prompt timeout")
	opencodeEvidence := flag.String("opencode-evidence", "", "opencode mode: comma-separated workspace-relative evidence files; disables search and permits only reads")
	controlledPatterns := flag.Int("controlled-max-patterns", 5, "controlled mode: maximum model-selected search patterns")
	controlledMatches := flag.Int("controlled-max-matches", 20, "controlled mode: maximum snippets per search pattern")
	controlledEvidence := flag.Int("controlled-max-evidence", 5, "controlled mode: maximum evidence pages")
	controlledImageWorkers := flag.Int("controlled-image-workers", 4, "controlled mode: concurrent local image transcriptions")
	controlledImageCache := flag.String("controlled-image-cache", "", "controlled mode: image transcription cache path (default: <repo>/.llm-site-reviewer/image-transcriptions.json)")
	controlledDocsRoot := flag.String("controlled-docs-root", "", "controlled mode: docs root for leading-slash image references (default: repository root)")
	controlledVisionModel := flag.String("controlled-vision-model", "qwen3-vl", "controlled mode: LiteLLM vision model for changed raster images")
	dependencyTargets := flag.String("dependency-targets", "", "dependency mode: comma-separated page paths to resolve and review (all pages are still summarised)")
	dependencySkipImages := flag.Bool("dependency-skip-images", false, "dependency mode: skip image transcription for a content-only run")
	dependencyRecursive := flag.Bool("dependency-recursive", true, "dependency mode: descend into nested corpus directories")

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
		var targets []string
		if *dependencyTargets != "" {
			for _, target := range strings.Split(*dependencyTargets, ",") {
				absolute, absErr := filepath.Abs(strings.TrimSpace(target))
				if absErr != nil {
					err = absErr
					break
				}
				targets = append(targets, absolute)
			}
		}
		if err == nil {
			err = runDependency(positional, targets, *dependencySkipImages, *dependencyRecursive)
		}
	case "opencode":
		if len(positional) != 1 {
			err = fmt.Errorf("opencode mode requires exactly one target file")
			break
		}
		if *opencodeTimeout <= 0 {
			err = fmt.Errorf("opencode-timeout must be positive")
			break
		}
		ctx, cancel := context.WithTimeout(context.Background(), *opencodeTimeout)
		defer cancel()
		var evidence []string
		for _, path := range strings.Split(*opencodeEvidence, ",") {
			if path = strings.TrimSpace(path); path != "" {
				evidence = append(evidence, path)
			}
		}
		err = opencodereview.Run(ctx, opencodereview.Config{
			Target: positional[0], Model: *model, BaseURL: *opencodeURL,
			Evidence: evidence, Verbose: *opencodeVerbose, Output: os.Stdout, TraceOutput: os.Stderr,
		})
	case "controlled":
		if len(positional) != 1 {
			err = fmt.Errorf("controlled mode requires exactly one target file")
			break
		}
		if *opencodeTimeout <= 0 {
			err = fmt.Errorf("opencode-timeout must be positive")
			break
		}
		err = runControlled(positional[0], *model, *opencodeURL, *opencodeTimeout, *opencodeVerbose, controlledreview.Config{
			MaxPatterns: *controlledPatterns, MaxMatchesPerPattern: *controlledMatches,
			MaxEvidenceFiles: *controlledEvidence, ImageWorkers: *controlledImageWorkers,
			ImageCachePath: *controlledImageCache, DocsRoot: *controlledDocsRoot,
		}, *controlledVisionModel)
	case "visual":
		err = runVisual(positional)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q (want batch, dependency, opencode, controlled, or visual)\n\n", *mode)
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("failed to call request: %v", err)
	}

}

type lazyImageTranscriber struct {
	model  string
	once   sync.Once
	client review.ChatClient
	err    error
}

func (t *lazyImageTranscriber) TranscribeImage(ctx context.Context, path string, content []byte) (string, error) {
	t.once.Do(func() {
		t.client, t.err = review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	})
	if t.err != nil {
		return "", t.err
	}
	return review.TranscribeImage(ctx, t.client, t.model, path, content)
}

func runControlled(target, model, baseURL string, timeout time.Duration, verbose bool, cfg controlledreview.Config, visionModel string) error {
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	workspace := opencodereview.FindProjectRoot(filepath.Dir(targetAbs))
	cfg.Root = workspace
	if cfg.DocsRoot == "" {
		cfg.DocsRoot = workspace
		candidate := filepath.Join(workspace, "docs")
		if rel, relErr := filepath.Rel(candidate, targetAbs); relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			cfg.DocsRoot = candidate
		}
	}
	if cfg.ImageCachePath == "" {
		cfg.ImageCachePath = filepath.Join(workspace, ".llm-site-reviewer", "image-transcriptions.json")
	}
	var traceOutput io.Writer
	if verbose {
		traceOutput = os.Stderr
		cfg.TraceOutput = traceOutput
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	phaseModel, err := opencodereview.NewControlledPhaseModel(ctx, opencodereview.ControlledConfig{
		BaseURL: baseURL, Model: model, Workspace: workspace, TraceOutput: traceOutput,
	})
	if err != nil {
		return err
	}
	engine, err := controlledreview.New(cfg, phaseModel, &lazyImageTranscriber{model: visionModel})
	if err != nil {
		return err
	}
	result, err := engine.Run(ctx, targetAbs)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
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
func runDependency(paths, targets []string, skipImages, recursive bool) error {
	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return fmt.Errorf("failed to build litellm proxy client: %w", err)
	}

	config := review.DefaultDependencyConfig()
	config.TargetPaths = targets
	config.SkipImages = skipImages
	config.Recursive = recursive
	results, err := review.DependencyReview(context.Background(), client, paths, config)
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
