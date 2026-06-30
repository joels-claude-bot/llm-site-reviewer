// Command inspect prints what the corpus pipeline produced, so you can read by
// eye the parts no assertion checks directly: which files FixturePaths found,
// and what Load parsed from them. Run it with no arguments for the default dump
// of the whole corpus, or pass a path to inspect one directory or file.
//
//	go run ./cmd/inspect                                    # dump the whole corpus
//	go run ./cmd/inspect corpus/links/broken-internal.md    # one file
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
)

func main() {
	target := "corpus"
	if len(os.Args) == 2 {
		target = os.Args[1]
	}
	if err := run(os.Stdout, target); err != nil {
		fmt.Fprintln(os.Stderr, "inspect:", err)
		os.Exit(1)
	}
}

func run(out io.Writer, target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return inspectTree(out, target)
	}
	return inspectFile(out, target)
}

// inspectTree dumps the two stages assertions never show: which files
// FixturePaths discovered, and what Load parsed from them.
func inspectTree(out io.Writer, root string) error {
	paths, err := corpus.FixturePaths(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "FixturePaths(%q) found %d files:\n", root, len(paths))
	for _, path := range paths {
		fmt.Fprintln(out, "  ", path)
	}

	fixtures, err := corpus.Load(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nLoad(%q) parsed %d fixtures:\n", root, len(fixtures))
	for _, fixture := range fixtures {
		printFixture(out, fixture)
	}
	return nil
}

func inspectFile(out io.Writer, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fixture, err := corpus.Parse(content)
	if err != nil {
		return err
	}
	fixture.Path = filepath.ToSlash(path)
	printFixture(out, fixture)
	return nil
}

func printFixture(out io.Writer, fixture corpus.Fixture) {
	kind := "defect"
	if fixture.LookAlike() {
		kind = "look-alike (flags nothing)"
	}
	fmt.Fprintf(out, "\n%s\n  title: %s\n  kind:  %s\n", fixture.Path, fixture.Title, kind)
	for _, want := range fixture.Expect {
		fmt.Fprintf(out, "  flag:  %-22s %-9s %s\n", want.Category, want.Result, want.Where)
	}
	if fixture.Note != "" {
		fmt.Fprintf(out, "  note:  %s\n", fixture.Note)
	}
}
