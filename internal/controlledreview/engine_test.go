package controlledreview

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
)

type fakeModel struct {
	searches  []string
	selected  []string
	response  review.MdResponse
	gotSearch EvidenceInput
	gotReview ReviewInput
}

func (f *fakeModel) ChooseSearchPatterns(_ context.Context, in SearchInput) ([]string, error) {
	return f.searches, nil
}
func (f *fakeModel) SelectEvidence(_ context.Context, in EvidenceInput) ([]string, error) {
	f.gotSearch = in
	return f.selected, nil
}
func (f *fakeModel) Review(_ context.Context, in ReviewInput) (review.MdResponse, error) {
	f.gotReview = in
	return f.response, nil
}

func TestEngineRunsThreeBoundedPhases(t *testing.T) {
	root := t.TempDir()
	target := write(t, root, "target.md", "Acme says 42.\n")
	write(t, root, "a.md", "Acme says 41.\nAcme again.\n")
	write(t, root, "b.md", "Other.\n")
	f := &fakeModel{searches: []string{"Acme", "ignored"}, selected: []string{"a.md"}, response: review.MdResponse{Results: []review.MdItem{{Id: 0, Name: "target.md", Success: false, Errors: []review.MdError{{Category: finding.Category("CONTRADICTS_PAGE"), Notes: "42 conflicts with 41"}}}}}}
	e, err := New(Config{Root: root, MaxPatterns: 1, MaxMatchesPerPattern: 1, MaxBytesPerPattern: 1000}, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.Run(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 || len(f.gotSearch.Searches) != 1 {
		t.Fatalf("unexpected result %#v/searches %#v", got, f.gotSearch.Searches)
	}
	if len(f.gotSearch.Searches[0].Matches) != 1 || !f.gotSearch.Searches[0].Truncated {
		t.Fatalf("search was not bounded: %#v", f.gotSearch.Searches[0])
	}
	if len(f.gotReview.Evidence) != 1 || f.gotReview.Evidence[0].Text != "Acme says 41.\nAcme again.\n" {
		t.Fatalf("bad evidence: %#v", f.gotReview.Evidence)
	}
}

func TestEngineRejectsEvidenceOutsideRoot(t *testing.T) {
	root := t.TempDir()
	target := write(t, root, "target.md", "x")
	outside := write(t, t.TempDir(), "outside.md", "x")
	f := &fakeModel{selected: []string{outside}}
	e, _ := New(Config{Root: root}, f, nil)
	if _, err := e.Run(context.Background(), target); err == nil {
		t.Fatal("expected outside-root error")
	}
}

func TestValidateResponse(t *testing.T) {
	cases := []review.MdResponse{
		{},
		{Results: []review.MdItem{{Id: 1, Name: "x.md", Success: true}}},
		{Results: []review.MdItem{{Id: 0, Name: "x.md", Success: true, Errors: []review.MdError{{Category: finding.Category("OTHER"), Notes: "bad"}}}}},
	}
	for _, tc := range cases {
		if ValidateResponse(tc, "x.md") == nil {
			t.Fatalf("expected invalid: %#v", tc)
		}
	}
}

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}
