package review

// Shared utility for both the batch and dependency review paths: it extracts
// image references embedded in a markdown file and resolves each one to usable
// bytes, whether the reference is a local file or a remote URL.

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownImage is one image reference found in a markdown file, together with
// the bytes it resolves to. Callers decide how to send those bytes to a model
// (e.g. SVG-as-text vs raster-as-bytes); this type stays MIME-agnostic.
type MarkdownImage struct {
	Ref      string // original destination string from the markdown
	Resolved string // absolute local path, or the URL for remote references
	IsRemote bool
	Content  []byte // the image bytes, read from disk or downloaded
}

// imageDownloadTimeout bounds a single remote image fetch. We use a dedicated
// client rather than http.DefaultClient, which has no timeout and would let a
// hung server block the whole review indefinitely.
const imageDownloadTimeout = 30 * time.Second

// markdownImageDestinations walks the goldmark AST and returns the raw
// destination string of every image reference, in document order.
func markdownImageDestinations(content []byte) ([]string, error) {
	reader := text.NewReader(content)
	parser := goldmark.DefaultParser()
	doc := parser.Parse(reader)

	var destinations []string
	err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		image, ok := node.(*ast.Image)
		if entering && ok {
			destinations = append(destinations, string(image.Destination))
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk markdown AST: %w", err)
	}
	return destinations, nil
}

// isRemoteRef reports whether a destination string is an http(s) URL. Anything
// else (bare names, ./relative, /absolute) is treated as a filesystem path.
func isRemoteRef(dest string) bool {
	parsed, err := url.Parse(dest)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// resolveLocalRef turns a filesystem destination into an absolute path.
//
// A leading-slash reference is docs-root-relative in mkdocs terms (the slash
// means the docs_dir root, not the filesystem root), so it is joined onto
// docsRoot when one is known. Without a docsRoot such a reference falls back to
// as-is. A relative reference is joined with the directory of the markdown file
// it was found in.
func resolveLocalRef(dest, mdFileDir, docsRoot string) string {
	if filepath.IsAbs(dest) {
		if docsRoot != "" {
			return filepath.Join(docsRoot, strings.TrimPrefix(dest, "/"))
		}
		return dest
	}
	return filepath.Join(mdFileDir, dest)
}

// downloadImage fetches a remote image into memory, wrapping failures with the
// URL and, on a non-2xx response, the status code.
func downloadImage(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: imageDownloadTimeout}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote image %v: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to fetch remote image %v: status %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote image body %v: %w", rawURL, err)
	}
	return body, nil
}

// ResolveMarkdownImages extracts every image reference from a markdown file and
// resolves each to its bytes. mdFilePath is the absolute path of the markdown
// file itself; relative references are resolved against its directory, and
// leading-slash references against docsRoot (the mkdocs docs_dir root). Pass an
// empty docsRoot when there is no such root and absolute refs are literal.
//
// A single image that cannot be resolved (missing file, failed download) is
// logged and skipped rather than aborting the whole run: one bad reference in a
// large docs set must not kill the review.
func ResolveMarkdownImages(content []byte, mdFilePath, docsRoot string) ([]MarkdownImage, error) {
	destinations, err := markdownImageDestinations(content)
	if err != nil {
		return nil, err
	}

	mdFileDir := filepath.Dir(mdFilePath)
	var images []MarkdownImage
	for _, dest := range destinations {
		if isRemoteRef(dest) {
			slog.Debug("resolving remote markdown image", "ref", dest)
			body, err := downloadImage(dest)
			if err != nil {
				slog.Warn("skipping unresolvable remote markdown image", "ref", dest, "md_file", mdFilePath, "error", err)
				continue
			}
			images = append(images, MarkdownImage{
				Ref:      dest,
				Resolved: dest,
				IsRemote: true,
				Content:  body,
			})
			continue
		}

		resolved := resolveLocalRef(dest, mdFileDir, docsRoot)
		slog.Debug("resolving local markdown image", "ref", dest, "resolved", resolved)
		body, err := os.ReadFile(resolved)
		if err != nil {
			slog.Warn("skipping unresolvable local markdown image", "ref", dest, "resolved", resolved, "md_file", mdFilePath, "error", err)
			continue
		}
		images = append(images, MarkdownImage{
			Ref:      dest,
			Resolved: resolved,
			Content:  body,
		})
	}
	return images, nil
}
