package review

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// A relative reference resolves against the markdown file's directory and the
// bytes come off disk.
func TestResolveMarkdownImagesRelative(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "route.png", "png-bytes")
	mdPath := filepath.Join(dir, "page.md")
	writeFile(t, dir, "page.md", "![Route](route.png)\n")

	images, err := ResolveMarkdownImages([]byte("![Route](route.png)\n"), mdPath, "")
	if err != nil {
		t.Fatalf("ResolveMarkdownImages returned error: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("image count = %d, want 1", len(images))
	}
	image := images[0]
	if image.Ref != "route.png" {
		t.Errorf("Ref = %q, want route.png", image.Ref)
	}
	if image.Resolved != filepath.Join(dir, "route.png") {
		t.Errorf("Resolved = %q, want %q", image.Resolved, filepath.Join(dir, "route.png"))
	}
	if image.IsRemote {
		t.Errorf("IsRemote = true, want false")
	}
	if string(image.Content) != "png-bytes" {
		t.Errorf("Content = %q, want png-bytes", image.Content)
	}
}

// A "../" reference climbs out of the markdown file's directory.
func TestResolveMarkdownImagesRelativeParent(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	if err := os.Mkdir(assets, 0755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	pages := filepath.Join(root, "pages")
	if err := os.Mkdir(pages, 0755); err != nil {
		t.Fatalf("mkdir pages: %v", err)
	}
	writeFile(t, assets, "diagram.png", "diagram-bytes")
	mdPath := filepath.Join(pages, "page.md")

	images, err := ResolveMarkdownImages([]byte("![D](../assets/diagram.png)\n"), mdPath, "")
	if err != nil {
		t.Fatalf("ResolveMarkdownImages returned error: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("image count = %d, want 1", len(images))
	}
	want := filepath.Join(assets, "diagram.png")
	if images[0].Resolved != want {
		t.Errorf("Resolved = %q, want %q", images[0].Resolved, want)
	}
	if string(images[0].Content) != "diagram-bytes" {
		t.Errorf("Content = %q, want diagram-bytes", images[0].Content)
	}
}

// A leading-slash reference is docs-root-relative (mkdocs semantics): it resolves
// under docsRoot, not the filesystem root, and not the page's own directory.
func TestResolveMarkdownImagesDocsRootRelative(t *testing.T) {
	docsRoot := t.TempDir()
	diagrams := filepath.Join(docsRoot, "diagrams")
	if err := os.Mkdir(diagrams, 0755); err != nil {
		t.Fatalf("mkdir diagrams: %v", err)
	}
	writeFile(t, diagrams, "map.svg", "svg-bytes")
	// The markdown file lives in a nested subdirectory of the docs root.
	pages := filepath.Join(docsRoot, "pages")
	if err := os.Mkdir(pages, 0755); err != nil {
		t.Fatalf("mkdir pages: %v", err)
	}
	mdPath := filepath.Join(pages, "page.md")

	images, err := ResolveMarkdownImages([]byte("![Map](/diagrams/map.svg)\n"), mdPath, docsRoot)
	if err != nil {
		t.Fatalf("ResolveMarkdownImages returned error: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("image count = %d, want 1", len(images))
	}
	want := filepath.Join(diagrams, "map.svg")
	if images[0].Resolved != want {
		t.Errorf("Resolved = %q, want %q", images[0].Resolved, want)
	}
	if string(images[0].Content) != "svg-bytes" {
		t.Errorf("Content = %q, want svg-bytes", images[0].Content)
	}
}

// Without a docsRoot, an absolute filesystem reference is taken as-is.
func TestResolveMarkdownImagesAbsoluteNoDocsRoot(t *testing.T) {
	dir := t.TempDir()
	absImage := filepath.Join(dir, "logo.png")
	if err := os.WriteFile(absImage, []byte("abs-bytes"), 0644); err != nil {
		t.Fatalf("write abs image: %v", err)
	}
	// The markdown file lives somewhere completely different.
	mdPath := filepath.Join(t.TempDir(), "page.md")

	images, err := ResolveMarkdownImages([]byte("![Logo]("+absImage+")\n"), mdPath, "")
	if err != nil {
		t.Fatalf("ResolveMarkdownImages returned error: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("image count = %d, want 1", len(images))
	}
	if images[0].Resolved != absImage {
		t.Errorf("Resolved = %q, want %q (as-is)", images[0].Resolved, absImage)
	}
	if string(images[0].Content) != "abs-bytes" {
		t.Errorf("Content = %q, want abs-bytes", images[0].Content)
	}
}

// A remote reference is downloaded into memory. Served locally via httptest so
// the test never touches the real network.
func TestResolveMarkdownImagesRemote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte("remote-bytes"))
	}))
	defer server.Close()

	remoteURL := server.URL + "/diagram.png"
	mdPath := filepath.Join(t.TempDir(), "page.md")

	images, err := ResolveMarkdownImages([]byte("![Remote]("+remoteURL+")\n"), mdPath, "")
	if err != nil {
		t.Fatalf("ResolveMarkdownImages returned error: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("image count = %d, want 1", len(images))
	}
	image := images[0]
	if !image.IsRemote {
		t.Errorf("IsRemote = false, want true")
	}
	if image.Resolved != remoteURL {
		t.Errorf("Resolved = %q, want %q", image.Resolved, remoteURL)
	}
	if string(image.Content) != "remote-bytes" {
		t.Errorf("Content = %q, want remote-bytes", image.Content)
	}
}

// A non-2xx remote response is non-fatal: the image is skipped so one bad
// reference cannot kill the run.
func TestResolveMarkdownImagesRemoteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "gone", http.StatusNotFound)
	}))
	defer server.Close()

	remoteURL := server.URL + "/missing.png"
	mdPath := filepath.Join(t.TempDir(), "page.md")

	images, err := ResolveMarkdownImages([]byte("![Missing]("+remoteURL+")\n"), mdPath, "")
	if err != nil {
		t.Fatalf("expected 404 remote image to be skipped, got error: %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("image count = %d, want 0 (bad remote image skipped)", len(images))
	}
}

// A missing local file is non-fatal: the image is skipped rather than aborting.
func TestResolveMarkdownImagesMissingLocal(t *testing.T) {
	mdPath := filepath.Join(t.TempDir(), "page.md")
	images, err := ResolveMarkdownImages([]byte("![Gone](nope.png)\n"), mdPath, "")
	if err != nil {
		t.Fatalf("expected missing local image to be skipped, got error: %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("image count = %d, want 0 (missing local image skipped)", len(images))
	}
}
