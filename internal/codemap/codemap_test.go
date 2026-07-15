package codemap_test

import (
	"strings"
	"testing"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/codemap"
)

// Text is pure, so it is tested with literal entries.
func TestTextGroupsAndSortsByRole(t *testing.T) {
	entries := []codemap.Entry{
		{Role: "pure", Pkg: "corpus", Name: "Parse", File: "internal/corpus/corpus.go", Line: 50, Synopsis: "decode bytes"},
		{Role: "io", Pkg: "corpus", Name: "Load", File: "internal/corpus/corpus.go", Line: 100, Synopsis: "read files"},
	}
	out := codemap.Text(entries)

	if !strings.Contains(out, "corpus.Load") || !strings.Contains(out, "internal/corpus/corpus.go:100") {
		t.Errorf("entry or file:line missing:\n%s", out)
	}
	if strings.Index(out, "IO") > strings.Index(out, "PURE") {
		t.Error("roles should be sorted, io before pure")
	}
}

// Extract smoke test: read the real source and find the planted tags. Doesn't
// assert the full map, just that tagging works end to end.
func TestExtractFindsPlantedTags(t *testing.T) {
	entries, err := codemap.Extract("../../internal", "../../cmd")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no //arch: tags found; expected several")
	}

	roles := map[string]bool{}
	for _, entry := range entries {
		roles[entry.Role] = true
	}
	for _, want := range []string{"pure", "io"} {
		if !roles[want] {
			t.Errorf("expected to find a %q-tagged symbol", want)
		}
	}
}
