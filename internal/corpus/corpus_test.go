package corpus_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
)

// corpusRoot is relative to this package directory.
const corpusRoot = "../../corpus"

// TestFixturePaths checks discovery directly: finds fixtures, sorted, skips index.md.
func TestFixturePaths(t *testing.T) {
	paths, err := corpus.FixturePaths(corpusRoot)
	if err != nil {
		t.Fatalf("FixturePaths: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("found no fixtures")
	}
	if !sort.StringsAreSorted(paths) {
		t.Error("paths should be sorted")
	}
	for _, path := range paths {
		if filepath.Base(path) == "index.md" {
			t.Errorf("index.md should be skipped, got %s", path)
		}
	}
}

// TestFixturesAreValid runs every real fixture through Validate: not the
// reviewer, just that the files on disk are well-formed. The rules themselves are
// unit-tested in parse_test.go.
func TestFixturesAreValid(t *testing.T) {
	fixtures, err := corpus.Load(corpusRoot)
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found; expected at least the links category")
	}

	for _, fixture := range fixtures {
		t.Run(fixture.Path, func(t *testing.T) {
			if err := corpus.Validate(fixture); err != nil {
				t.Error(err)
			}
		})
	}
}
