package review

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyImage(t *testing.T) {
	for _, tt := range []struct {
		path string
		want imageKind
	}{
		{"diagram.svg", imageKind{asText: true}},
		{"route.png", imageKind{mime: "image/png"}},
		{"photo.jpeg", imageKind{mime: "image/jpeg"}},
		{"unknown.zzz", imageKind{mime: "application/octet-stream"}},
	} {
		if got := classifyImage(tt.path); got != tt.want {
			t.Errorf("classifyImage(%q) = %+v, want %+v", tt.path, got, tt.want)
		}
	}
}

func TestSummariseDependencyImage(t *testing.T) {
	client := &fakeChatClient{responses: []string{"image summary"}}
	image, err := summariseImage(context.Background(), client, "vision-model", "route.png", []byte("png"))
	if err != nil {
		t.Fatalf("summariseImage returned error: %v", err)
	}
	if image.Path != "route.png" || image.Summary != "image summary" {
		t.Fatalf("image summary = %+v", image)
	}
}

func TestSummariseDependencyPages(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "page.md", "![Route](route.svg)\n\n# Page\n")
	writeFile(t, dir, "route.svg", "<svg>route</svg>")

	client := &fakeChatClient{responses: []string{"route summary", "page summary"}}
	pages, err := summarisePages(context.Background(), client, dir, "summary-model", true)
	if err != nil {
		t.Fatalf("summarisePages returned error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("page count = %d, want 1", len(pages))
	}
	if pages[0].Path != filepath.Join(dir, "page.md") || pages[0].Summary != "page summary" {
		t.Fatalf("page = %+v", pages[0])
	}
	if len(pages[0].Images) != 1 || pages[0].Images[0].Summary != "route summary" {
		t.Fatalf("images = %+v", pages[0].Images)
	}
}

func TestSummarisePagesRecursiveDescendsIntoSubdirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "top.md", "# Top\n")
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	writeFile(t, nested, "deep.md", "# Deep\n")

	// Non-recursive: only the top-level page is collected, the nested dir is skipped.
	client := &fakeChatClient{responses: []string{"top summary"}}
	pages, err := summarisePages(context.Background(), client, dir, "summary-model", false)
	if err != nil {
		t.Fatalf("summarisePages(recursive=false) returned error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("non-recursive page count = %d, want 1 (nested dir ignored)", len(pages))
	}
	if pages[0].Path != filepath.Join(dir, "top.md") {
		t.Fatalf("non-recursive collected unexpected page: %+v", pages[0])
	}

	// Recursive: the nested page is collected too.
	client = &fakeChatClient{responses: []string{"top summary", "deep summary"}}
	pages, err = summarisePages(context.Background(), client, dir, "summary-model", true)
	if err != nil {
		t.Fatalf("summarisePages(recursive=true) returned error: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("recursive page count = %d, want 2 (nested page collected)", len(pages))
	}
	var foundDeep bool
	for _, page := range pages {
		if page.Path == filepath.Join(nested, "deep.md") {
			foundDeep = true
		}
	}
	if !foundDeep {
		t.Fatalf("recursive mode did not collect nested page deep.md: %+v", pages)
	}
}

// TestSummariseImageReadsLabelsE2E is a capability canary: a deliberately clear
// image with bold, unambiguous labels must come back with those labels intact.
// If it fails, the model, the prompt, or the image wiring has drifted, and it
// warrants a human look rather than an automatic pass.
func TestSummariseImageReadsLabelsE2E(t *testing.T) {
	client := e2eClient(t)

	fixture := "testdata/boarding-pass.svg"
	content, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixture, err)
	}

	image, err := summariseImage(context.Background(), client, modelVision, fixture, content)
	if err != nil {
		t.Fatalf("summariseImage returned error: %v", err)
	}

	// Three big, unambiguous labels from the fixture. Require at least two to
	// survive so one dropped detail doesn't flake the canary.
	containsAtLeast(t, image.Summary, []string{"UA328", "B12", "14C"}, 2)
}
