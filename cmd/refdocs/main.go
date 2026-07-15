// Command refdocs turns the category catalog into the generated category-table
// page, so nobody has to keep those tables in sync by hand. internal/finding.Catalog
// is the source of truth; this page is generated from it. The hand-written intro
// lives separately in reference.md and is not touched here.
//
// Usage (from the repo root):
//
//	go run ./cmd/refdocs
//
//arch:tool
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
	"github.com/nao1215/markdown"
)

const outPath = "docs-site/docs/spec/categories.md"

func main() {
	if err := run(os.Stdout, outPath); err != nil {
		fmt.Fprintln(os.Stderr, "refdocs:", err)
		os.Exit(1)
	}
}

// run renders the page and writes it. The IO half; render does the pure part.
//
//arch:io
func run(log io.Writer, out string) error {
	page, err := render(finding.Catalog, finding.LookAlikes)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(page), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(log, "wrote %s (%d categories)\n", out, len(finding.Catalog))
	return nil
}

// The parts markdown/nao1215 does not build: rspress frontmatter and the loud
// auto-generated admonition. Written to the buffer before the library body.
const frontMatter = "---\ntitle: Category table\n---\n\n"

const banner = ":::danger Auto-generated: do not edit this page\n" +
	"Every row below is written by `cmd/refdocs` from `internal/finding.Catalog`. Hand edits are\n" +
	"overwritten on the next build. To change a category, edit the catalog in Go and run `just gen-reference`.\n" +
	":::\n\n"

// render builds the markdown page with github.com/nao1215/markdown, grouped by
// category group in catalog order, with the How glossary generated from
// finding.HowKinds. Pure, so tested in memory.
//
//arch:pure
func render(catalog []finding.Meta, lookAlikes []finding.LookAlike) (string, error) {
	type group struct {
		Name string
		Rows [][]string
	}
	byName := map[string]*group{}
	var groups []*group
	for _, meta := range catalog {
		g, ok := byName[meta.Group]
		if !ok {
			g = &group{Name: meta.Group}
			byName[meta.Group] = g
			groups = append(groups, g) // first-seen order, so Links stays first
		}
		g.Rows = append(g.Rows, []string{
			"`" + string(meta.Code) + "`",
			meta.Example,
			string(meta.How),
		})
	}

	var buf strings.Builder
	buf.WriteString(frontMatter)
	buf.WriteString(banner)

	md := markdown.NewMarkdown(&buf)
	md.H1("Category table")
	md.PlainText("The full failure taxonomy. See the [category reference](/spec/reference) for what the columns mean and the checkability rule behind them.")

	// section writes a heading, a blank line, then a table. The blank line is
	// what remark needs to parse the table rather than read it as a paragraph.
	section := func(title string, set markdown.TableSet) {
		md.H2(title)
		md.PlainText("")
		md.Table(set)
	}

	howRows := make([][]string, 0, len(finding.HowKinds))
	for _, kind := range finding.HowKinds {
		howRows = append(howRows, []string{string(kind.How), kind.Desc})
	}
	section("How checked", markdown.TableSet{Header: []string{"Value", "Meaning"}, Rows: howRows})

	for _, g := range groups {
		section(g.Name, markdown.TableSet{
			Header: []string{"Category", "Example", "How checked"},
			Rows:   g.Rows,
		})
	}

	md.H2("Look-Alikes To Ignore")
	md.PlainText("These are not defects:")
	md.PlainText("")
	laRows := make([][]string, 0, len(lookAlikes))
	for _, la := range lookAlikes {
		laRows = append(laRows, []string{la.Looks, la.Why})
	}
	md.Table(markdown.TableSet{Header: []string{"Looks suspicious", "Why to ignore it"}, Rows: laRows})

	if err := md.Build(); err != nil {
		return "", err
	}
	buf.WriteString("\n")
	return buf.String(), nil
}
