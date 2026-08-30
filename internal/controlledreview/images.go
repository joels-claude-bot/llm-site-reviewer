package controlledreview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

const transcriptionVersion = "existing-vision-v1"

var ErrRemoteImageUnsupported = errors.New("controlled review: remote markdown images are not supported yet")

type ImageTranscriber interface {
	TranscribeImage(context.Context, string, []byte) (string, error)
}
type CacheEntry struct {
	ImagePath     string `json:"imagePath"`
	Hash          string `json:"hash"`
	Transcription string `json:"transcription"`
	Version       string `json:"version"`
}
type cacheFile struct {
	Entries []CacheEntry `json:"entries"`
}
type ImageCache struct {
	path        string
	transcriber ImageTranscriber
	workers     int
	mu          sync.Mutex
	entries     map[string]CacheEntry
}

func OpenImageCache(path string, tr ImageTranscriber, workers int) (*ImageCache, error) {
	if workers <= 0 {
		workers = 1
	}
	c := &ImageCache{path: path, transcriber: tr, workers: workers, entries: map[string]CacheEntry{}}
	if path == "" {
		return c, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	var data cacheFile
	if err = json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("decode image cache: %w", err)
	}
	for _, v := range data.Entries {
		c.entries[v.ImagePath] = v
	}
	return c, nil
}

func imageRefs(content []byte) ([]string, error) {
	doc := goldmark.DefaultParser().Parse(text.NewReader(content))
	var out []string
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if im, ok := n.(*ast.Image); ok && entering {
			out = append(out, string(im.Destination))
		}
		return ast.WalkContinue, nil
	})
	return out, err
}

func (c *ImageCache) ForMarkdown(ctx context.Context, content []byte, mdPath, docsRoot string) ([]ImageTranscription, error) {
	images, _, err := c.ForMarkdownDetailed(ctx, content, mdPath, docsRoot)
	return images, err
}

// ForMarkdownDetailed skips remote images and reports each skip explicitly.
// TODO(remote-images): add a bounded downloader and cache identity policy before
// enabling HTTP(S) image transcription.
func (c *ImageCache) ForMarkdownDetailed(ctx context.Context, content []byte, mdPath, docsRoot string) ([]ImageTranscription, []ImageSkip, error) {
	refs, err := imageRefs(content)
	if err != nil {
		return nil, nil, err
	}
	localMarkdown := strings.Builder{}
	skips := make([]ImageSkip, 0)
	for _, ref := range refs {
		u, _ := url.Parse(ref)
		if u != nil && (u.Scheme == "http" || u.Scheme == "https") {
			skips = append(skips, ImageSkip{Ref: ref, Reason: ErrRemoteImageUnsupported})
			continue
		}
		fmt.Fprintf(&localMarkdown, "![](<%s>)\n", strings.ReplaceAll(ref, ">", "%3E"))
	}
	resolved, err := review.ResolveMarkdownImages([]byte(localMarkdown.String()), mdPath, docsRoot)
	if err != nil {
		return nil, skips, err
	}
	paths := make([]string, 0, len(resolved))
	seen := make(map[string]bool, len(refs))
	for _, image := range resolved {
		p, err := filepath.Abs(image.Resolved)
		if err != nil {
			return nil, skips, err
		}
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	type res struct {
		i   int
		t   ImageTranscription
		err error
	}
	ch := make(chan res, len(paths))
	sem := make(chan struct{}, c.workers)
	for i, p := range paths {
		go func(i int, p string) {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				ch <- res{i: i, err: ctx.Err()}
				return
			}
			t, err := c.transcribe(ctx, p)
			ch <- res{i, t, err}
		}(i, p)
	}
	out := make([]ImageTranscription, len(paths))
	changed := false
	for range paths {
		r := <-ch
		if r.err != nil {
			return nil, skips, r.err
		}
		out[r.i] = r.t
		c.mu.Lock()
		old, ok := c.entries[r.t.Path]
		if !ok || old.Hash != r.t.Hash || old.Transcription != r.t.Transcription {
			c.entries[r.t.Path] = CacheEntry{r.t.Path, r.t.Hash, r.t.Transcription, transcriptionVersion}
			changed = true
		}
		c.mu.Unlock()
	}
	if changed {
		if err := c.save(); err != nil {
			return nil, skips, err
		}
	}
	return out, skips, nil
}

func (c *ImageCache) transcribe(ctx context.Context, path string) (ImageTranscription, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ImageTranscription{}, err
	}
	sum := sha256.Sum256(b)
	hash := hex.EncodeToString(sum[:])
	c.mu.Lock()
	entry, ok := c.entries[path]
	c.mu.Unlock()
	if ok && entry.Hash == hash && entry.Version == transcriptionVersion {
		return ImageTranscription{path, hash, entry.Transcription}, nil
	}
	if c.transcriber == nil {
		return ImageTranscription{}, fmt.Errorf("no image transcriber configured for %s", path)
	}
	transcription, err := c.transcriber.TranscribeImage(ctx, path, b)
	if err != nil {
		return ImageTranscription{}, err
	}
	return ImageTranscription{path, hash, transcription}, nil
}

func (c *ImageCache) save() error {
	if c.path == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entries := make([]CacheEntry, 0, len(c.entries))
	for _, k := range sortedKeys(c.entries) {
		entries = append(entries, c.entries[k])
	}
	b, err := json.MarshalIndent(cacheFile{entries}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(c.path), ".image-cache-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, c.path); err != nil {
		return err
	}
	return nil
}
