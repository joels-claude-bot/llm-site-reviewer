// Command codemap prints the architecture map built from //arch: tags in the
// source. Run it from the repo root.
//
//	go run ./cmd/codemap
//
//arch:tool
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/codemap"
)

func main() {
	if err := run(os.Stdout, "internal", "cmd"); err != nil {
		fmt.Fprintln(os.Stderr, "codemap:", err)
		os.Exit(1)
	}
}

// run extracts the tags under roots and writes the rendered map to out.
func run(out io.Writer, roots ...string) error {
	entries, err := codemap.Extract(roots...)
	if err != nil {
		return err
	}
	fmt.Fprint(out, codemap.Text(entries))
	return nil
}
