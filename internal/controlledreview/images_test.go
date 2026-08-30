package controlledreview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type fakeTranscriber struct {
	mu    sync.Mutex
	calls int
}

func (f *fakeTranscriber) TranscribeImage(_ context.Context, _ string, b []byte) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return "pixels:" + string(b), nil
}

func TestImageCacheReusesHashAndRetranscribesChanges(t *testing.T) {
	d := t.TempDir()
	md := write(t, d, "page.md", "![x](x.png)\n![s](s.svg)\n")
	image := write(t, d, "x.png", "one")
	write(t, d, "s.svg", "<svg>label</svg>")
	cachePath := filepath.Join(d, "cache", "images.json")
	tr := &fakeTranscriber{}
	c, err := OpenImageCache(cachePath, tr, 2)
	if err != nil {
		t.Fatal(err)
	}
	first, err := c.ForMarkdown(context.Background(), []byte("![x](x.png)\n![s](s.svg)"), md, "")
	if err != nil {
		t.Fatal(err)
	}
	if tr.calls != 2 || first[1].Transcription != "pixels:<svg>label</svg>" {
		t.Fatalf("first: calls=%d %#v", tr.calls, first)
	}
	c2, err := OpenImageCache(cachePath, tr, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c2.ForMarkdown(context.Background(), []byte("![x](x.png)\n![s](s.svg)"), md, ""); err != nil {
		t.Fatal(err)
	}
	if tr.calls != 2 {
		t.Fatalf("cache miss: calls=%d", tr.calls)
	}
	if err = os.WriteFile(image, []byte("two"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = c2.ForMarkdown(context.Background(), []byte("![x](x.png)"), md, ""); err != nil {
		t.Fatal(err)
	}
	if tr.calls != 3 {
		t.Fatalf("changed image not transcribed: calls=%d", tr.calls)
	}
	if b, err := os.ReadFile(cachePath); err != nil || len(b) == 0 {
		t.Fatalf("cache not persisted: %v", err)
	}
}

func TestRemoteImageReturnsTypedUnsupportedError(t *testing.T) {
	d := t.TempDir()
	write(t, d, "local.svg", "<svg/>")
	c, _ := OpenImageCache("", &fakeTranscriber{}, 0) // zero workers must default, not deadlock
	images, skips, err := c.ForMarkdownDetailed(context.Background(), []byte("![x](https://example.test/x.png)\n![local](local.svg)"), filepath.Join(d, "x.md"), "")
	if err != nil || len(images) != 1 || len(skips) != 1 || !errors.Is(skips[0].Reason, ErrRemoteImageUnsupported) {
		t.Fatalf("images=%v skips=%v err=%v", images, skips, err)
	}
}

func TestMissingImageTranscriber(t *testing.T) {
	d := t.TempDir()
	md := write(t, d, "x.md", "![x](x.png)")
	write(t, d, "x.png", "x")
	c, _ := OpenImageCache("", nil, 1)
	if _, err := c.ForMarkdown(context.Background(), []byte("![x](x.png)"), md, ""); err == nil {
		t.Fatal("expected error")
	}
}
