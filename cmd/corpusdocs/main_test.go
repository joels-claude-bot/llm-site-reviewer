package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
)

func TestTitle(t *testing.T) {
	cases := map[string]string{"links": "Links", "": "", "a": "A"}
	for in, want := range cases {
		if got := title(in); got != want {
			t.Errorf("title(%q) = %q, want %q", in, got, want)
		}
	}
}

// renderFixtures is pure, so it is tested with fixtures built in memory. The
// body carries its own ``` fence, so the check also proves the wrapper fence
// grows past it.
func TestRenderFixturesEmbedsBodyAndExpect(t *testing.T) {
	fixtures := []corpus.Fixture{
		{
			Path:     "links/ok-placeholder.md",
			Category: "links",
			Title:    "API example",
			Note:     "example.com in a code fence",
			Body:     "# Rebooking\n\n```bash\ncurl https://example.com\n```",
		},
		{
			Path:     "links/broken-anchor.md",
			Category: "links",
			Title:    "Travel notes",
			Note:     "links to a missing anchor",
			Expect:   []finding.Finding{{Category: finding.BrokenAnchor, Where: "the #packing link", Result: finding.Blocking}},
			Body:     "# Travel notes\n\nJump to the [list](#packing).",
		},
	}

	page, err := renderFixtures(fixtures)
	if err != nil {
		t.Fatalf("renderFixtures: %v", err)
	}
	for _, want := range []string{
		"## Links",
		"### `links/broken-anchor.md`",
		"**Travel notes**",
		"`BROKEN_ANCHOR` (blocking) — the #packing link",
		"Look-alike — the reviewer must report nothing.",
		"````md", // fence grew past the body's own ```
		"curl https://example.com",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("rendered gallery missing %q:\n%s", want, page)
		}
	}
}

func TestFence(t *testing.T) {
	cases := map[string]string{
		"no backticks":     "```",
		"one ` here":       "```",
		"a ``` block":      "````",
		"nested ```` deep": "`````",
	}
	for body, want := range cases {
		if got := fence(body); got != want {
			t.Errorf("fence(%q) = %q, want %q", body, got, want)
		}
	}
}

// run writes a real file, so it is smoke-tested against a temp path.
func TestRunWritesGallery(t *testing.T) {
	out := filepath.Join(t.TempDir(), "corpus-fixtures.md")
	if err := run(os.Stderr, "../../corpus", out); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading gallery: %v", err)
	}
	if !strings.Contains(string(data), "# Corpus fixtures") {
		t.Error("gallery page has no heading")
	}
}
