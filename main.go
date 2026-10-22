package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/batchreview"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/logging"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/opencodereview"
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
	case "opencode":
		err = runOpencode(os.Args[2:])
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
  opencode    review one page via an opencode server session

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
	return batchreview.Run(flags.Args(), *recursive)
}

func runOpencode(args []string) error {
	flags := flag.NewFlagSet("opencode", flag.ExitOnError)
	model := flags.String("model", "litellm/gemini-flash", "provider/model to use")
	baseURL := flags.String("url", "http://127.0.0.1:4097", "URL of the running OpenCode server")
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
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	return opencodereview.Run(ctx, opencodereview.Config{
		Target: flags.Arg(0), Model: *model, BaseURL: *baseURL,
		Evidence: evidence, Verbose: *verbose, Output: os.Stdout, TraceOutput: os.Stderr,
	})
}
