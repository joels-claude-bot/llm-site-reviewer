package corpus_test

import (
	"strings"
	"testing"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
)

// Parse is pure, so these run on literal strings with no filesystem.

func TestParseReadsOneDefect(t *testing.T) {
	content := []byte(`---
title: Booking guide
expect:
  - category: BROKEN_INTERNAL_LINK
    where: "the 'setup guide' link"
    result: blocking
note: "Links to /setup, which has no page."
---

# Booking guide

Read the [setup guide](/setup).`)

	fixture, err := corpus.Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if fixture.Title != "Booking guide" {
		t.Errorf("Title = %q, want %q", fixture.Title, "Booking guide")
	}
	if len(fixture.Expect) != 1 {
		t.Fatalf("len(Expect) = %d, want 1", len(fixture.Expect))
	}
	got := fixture.Expect[0]
	if got.Category != finding.BrokenInternalLink {
		t.Errorf("Category = %q, want %q", got.Category, finding.BrokenInternalLink)
	}
	if got.Result != finding.Blocking {
		t.Errorf("Result = %q, want %q", got.Result, finding.Blocking)
	}
	if fixture.LookAlike() {
		t.Error("fixture with a defect should not be a look-alike")
	}
	if !strings.Contains(fixture.Body, "[setup guide](/setup)") {
		t.Errorf("Body did not preserve the page content: %q", fixture.Body)
	}
}

func TestParseLookAlikeExpectsNothing(t *testing.T) {
	content := []byte(`---
title: API example
expect: []
note: "example.com in a code fence is a placeholder."
---

curl https://example.com/book`)

	fixture, err := corpus.Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !fixture.LookAlike() {
		t.Error("an empty expect list should be a look-alike")
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		fixture corpus.Fixture
		wantErr bool
	}{
		{
			name: "valid defect",
			fixture: corpus.Fixture{
				Path:  "links/broken-internal.md",
				Title: "Booking guide",
				Expect: []finding.Finding{
					{Category: finding.BrokenInternalLink, Where: "the link", Result: finding.Blocking},
				},
			},
		},
		{
			name:    "valid look-alike",
			fixture: corpus.Fixture{Path: "links/ok-placeholder.md", Title: "API example"},
		},
		{
			name:    "missing title",
			fixture: corpus.Fixture{Path: "links/x.md", Expect: []finding.Finding{{Category: finding.MissingImage, Where: "img", Result: finding.Blocking}}},
			wantErr: true,
		},
		{
			name: "unknown category",
			fixture: corpus.Fixture{
				Path:   "links/x.md",
				Title:  "x",
				Expect: []finding.Finding{{Category: "NOT_A_REAL_CODE", Where: "here", Result: finding.Blocking}},
			},
			wantErr: true,
		},
		{
			name: "invalid result",
			fixture: corpus.Fixture{
				Path:   "links/x.md",
				Title:  "x",
				Expect: []finding.Finding{{Category: finding.Soft404, Where: "here", Result: "maybe"}},
			},
			wantErr: true,
		},
		{
			name:    "defect file with no expectation",
			fixture: corpus.Fixture{Path: "links/broken-internal.md", Title: "x"},
			wantErr: true,
		},
		{
			name: "ok file that flags something",
			fixture: corpus.Fixture{
				Path:   "links/ok-placeholder.md",
				Title:  "x",
				Expect: []finding.Finding{{Category: finding.BrokenExternalLink, Where: "here", Result: finding.Blocking}},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := corpus.Validate(tc.fixture)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
