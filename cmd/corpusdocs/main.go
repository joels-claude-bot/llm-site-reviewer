// Command corpusdocs turns the corpus fixtures into the generated fixtures
// gallery (corpus-fixtures.md): every fixture in full, grouped by category, so
// nobody keeps that page in sync by hand. The fixtures are the source of truth.
// The hand-written overview lives separately in corpus.md and is not touched here.
//
// Usage (from the repo root):
//
//	go run ./cmd/corpusdocs
//
//arch:tool
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
	"github.com/nao1215/markdown"
)

const (
	corpusRoot = "corpus"
	outPath    = "docs-site/docs/spec/corpus-fixtures.md"
)

func main() {
	if err := run(os.Stdout, corpusRoot, outPath); err != nil {
		fmt.Fprintln(os.Stderr, "corpusdocs:", err)
		os.Exit(1)
	}
}

// run loads the corpus, renders the gallery, writes it, logs a summary. The IO
// half; renderFixtures does the pure part.
//
//arch:io
func run(log io.Writer, root, out string) error {
	fixtures, err := corpus.Load(root)
	if err != nil {
		return err
	}
	page, err := renderFixtures(fixtures)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(page), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(log, "wrote %s (%d fixtures)\n", out, len(fixtures))
	return nil
}

// The parts markdown/nao1215 does not build: rspress frontmatter and the loud
// auto-generated admonition. Written to the buffer before the library body.
const frontMatter = "---\ntitle: Corpus fixtures\n---\n\n"

const banner = ":::danger Auto-generated: do not edit this page\n" +
	"Every fixture below is written by `cmd/corpusdocs` from the `corpus/` fixtures. Hand edits are\n" +
	"overwritten on the next build. To change one, edit the fixture and run `go run ./cmd/corpusdocs`.\n" +
	":::\n\n"

// renderFixtures builds the gallery with github.com/nao1215/markdown: every
// fixture in full, grouped by category, so a reader can see the actual markdown
// behind each category code. Pure, so tested in memory.
//
//arch:pure
func renderFixtures(fixtures []corpus.Fixture) (string, error) {
	type group struct {
		Display  string
		Fixtures []corpus.Fixture
	}

	byCat := map[string]*group{}
	var order []string
	for _, fixture := range fixtures {
		g, ok := byCat[fixture.Category]
		if !ok {
			g = &group{Display: title(fixture.Category)}
			byCat[fixture.Category] = g
			order = append(order, fixture.Category)
		}
		g.Fixtures = append(g.Fixtures, fixture)
	}
	sort.Strings(order)

	var buf strings.Builder
	buf.WriteString(frontMatter)
	buf.WriteString(banner)

	md := markdown.NewMarkdown(&buf)
	md.H1("Corpus fixtures")
	md.PlainText("")
	md.PlainText("The full source of every fixture in `corpus/`, grouped by category, so you can read the actual markdown behind each category code. For how the corpus is used, see the [corpus overview](/spec/corpus); for the codes, the [reference](/spec/reference).")
	md.PlainText("")

	for _, name := range order {
		g := byCat[name]
		md.H2(g.Display)
		md.PlainText("")
		for _, fixture := range g.Fixtures {
			fixtureSection(md, fixture)
		}
	}

	if err := md.Build(); err != nil {
		return "", err
	}
	buf.WriteString("\n")
	return buf.String(), nil
}

// fixtureSection writes one fixture: path heading, title, what it plants (or the
// look-alike note), the human note, then the body verbatim in a code fence. The
// library joins blocks with a single line feed, so blank lines are explicit.
func fixtureSection(md *markdown.Markdown, fixture corpus.Fixture) {
	md.H3("`" + fixture.Path + "`")
	md.PlainText("")
	md.PlainText("**" + fixture.Title + "**")
	md.PlainText("")

	if fixture.LookAlike() {
		md.PlainText("Look-alike — the reviewer must report nothing.")
	} else {
		md.PlainText("Plants:")
		md.PlainText("")
		items := make([]string, 0, len(fixture.Expect))
		for _, want := range fixture.Expect {
			items = append(items, fmt.Sprintf("`%s` (%s) — %s", want.Category, want.Result, want.Where))
		}
		md.BulletList(items...)
	}
	md.PlainText("")
	md.PlainText(fixture.Note)
	md.PlainText("")

	// The library's CodeBlocks uses a fixed ``` fence, which a fixture carrying
	// its own ``` block would close early, so the fence is built by hand.
	f := fence(fixture.Body)
	md.PlainText(f + "md")
	md.PlainText(fixture.Body)
	md.PlainText(f)
	md.PlainText("")
}

// title upper-cases the first letter of a category name for display.
func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// fence returns a code-fence delimiter long enough to wrap body safely: one more
// backtick than the longest run already in it (some fixtures contain their own
// ``` blocks), and never fewer than three.
func fence(body string) string {
	longest, current := 0, 0
	for _, r := range body {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	return strings.Repeat("`", max(longest+1, 3))
}
