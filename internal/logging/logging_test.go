package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestInitSetsDebugLevel(t *testing.T) {
	var out bytes.Buffer
	Init("debug", &out)

	slog.Debug("debug visible", "key", "value")

	if !strings.Contains(out.String(), "debug visible") || !strings.Contains(out.String(), "key=value") {
		t.Fatalf("debug output = %q", out.String())
	}
}

func TestInitDefaultsToInfoLevel(t *testing.T) {
	var out bytes.Buffer
	Init("", &out)

	slog.Debug("hidden")
	slog.Info("visible")

	if strings.Contains(out.String(), "hidden") {
		t.Fatalf("debug log should have been filtered: %q", out.String())
	}
	if !strings.Contains(out.String(), "visible") {
		t.Fatalf("info log missing from output: %q", out.String())
	}
}
