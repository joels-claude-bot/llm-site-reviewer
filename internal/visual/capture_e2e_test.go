package visual

import (
	"bytes"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestCapturePageE2E(t *testing.T) {
	if os.Getenv("VISUAL_E2E") == "" {
		t.Skip("set VISUAL_E2E=1 to run the browser test")
	}
	abs, err := filepath.Abs("testdata/colours.html")
	if err != nil {
		t.Fatal(err)
	}
	capture, err := CapturePage((&url.URL{Scheme: "file", Path: abs}).String(), DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(capture.Frames) < 2 {
		t.Fatalf("got %d frames", len(capture.Frames))
	}
	if capture.Frames[len(capture.Frames)-1].Y != capture.PageHeight-capture.Height {
		t.Fatal("last frame is not at page bottom")
	}
	for _, frame := range capture.Frames {
		config, err := png.DecodeConfig(bytes.NewReader(frame.PNG))
		if err != nil {
			t.Fatal(err)
		}
		if config.Width != capture.Width || config.Height != capture.Height {
			t.Fatalf("frame is %dx%d", config.Width, config.Height)
		}
	}
}
