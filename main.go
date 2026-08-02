package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: inspect [FLAGS] target\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()

	}
	flag.Parse()

	positional := flag.Args()
	if len(positional) > 1 {
		fmt.Fprintf(os.Stderr, "too many positional args!\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if len(positional) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	reviewPath := flag.Arg(0)
	err := review.LiteLLMRequest(reviewPath)

	if err != nil {
		log.Fatalf("failed to call request: %v", err)
	}

}
