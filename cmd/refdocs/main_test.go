package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
)

// render is pure, so it is tested with a catalog built in memory.
func TestRenderGroupsAndLookAlikes(t *testing.T) {
	catalog := []finding.Meta{
		{Code: finding.BrokenInternalLink, Group: "Links", Example: "points to a missing page", How: finding.Mechanical},
		{Code: finding.Soft404, Group: "Links", Example: "200 but says not found", How: finding.CognitiveTextReview},
	}
	lookAlikes := []finding.LookAlike{{Looks: "example.com in a fence", Why: "it is example code"}}

	page, err := render(catalog, lookAlikes)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"## Links", "BROKEN_INTERNAL_LINK", "Cognitive Text Review", "Look-Alikes To Ignore", "example.com in a fence"} {
		if !strings.Contains(page, want) {
			t.Errorf("rendered page missing %q:\n%s", want, page)
		}
	}
}

// TestGeneratedPageIsCommitted is the staleness guard: if catalog.go or the
// template changes and nobody regenerates, the committed page drifts. run writes
// exactly render()'s bytes, so a plain byte compare against the on-disk page is
// enough. Path is relative to cmd/refdocs (the test's working dir).
func TestGeneratedPageIsCommitted(t *testing.T) {
	want, err := render(finding.Catalog, finding.LookAlikes)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	const committed = "../../docs-site/docs/spec/categories.md"
	got, err := os.ReadFile(committed)
	if err != nil {
		t.Fatalf("reading committed page: %v", err)
	}
	if string(got) != want {
		t.Errorf("reference.md is stale -- run `just gen-reference` (or `go run ./cmd/refdocs`) and commit")
	}
}

// run writes a real file, so it is smoke-tested against a temp path. It renders
// the real catalog, so it also proves every category has valid metadata.
func TestRunWritesPage(t *testing.T) {
	out := filepath.Join(t.TempDir(), "reference.md")
	if err := run(os.Stderr, out); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if !strings.Contains(string(data), "# Category table") {
		t.Error("generated page has no heading")
	}
}
