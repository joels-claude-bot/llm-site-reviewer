package corpus_test

import (
	"testing"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
)

// corpusRoot is relative to this package directory.
const corpusRoot = "../../corpus"

// TestFixturesAreValid loads the real corpus and runs every fixture through
// Validate. It does not run the reviewer; it only checks that the files on disk
// are well-formed. The rules are unit-tested in parse_test.go; this confirms the
// real fixtures obey them.
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
