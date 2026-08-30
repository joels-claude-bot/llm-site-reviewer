package opencodereview

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestOllamaIntegration is deliberately opt-in. It exercises a real OpenCode
// server but refuses non-Ollama models, preventing accidental paid-provider use.
// Example:
//
//	OPENCODE_OLLAMA_INTEGRATION=1 OPENCODE_OLLAMA_MODEL=ollama/qwen3:4b \
//	OPENCODE_URL=http://127.0.0.1:4096 go test ./internal/opencodereview -run TestOllamaIntegration -v
func TestOllamaIntegration(t *testing.T) {
	if os.Getenv("OPENCODE_OLLAMA_INTEGRATION") != "1" {
		t.Skip("set OPENCODE_OLLAMA_INTEGRATION=1 to run against local Ollama")
	}
	model := os.Getenv("OPENCODE_OLLAMA_MODEL")
	if !strings.HasPrefix(model, "ollama/") {
		t.Fatalf("OPENCODE_OLLAMA_MODEL must begin with ollama/; refusing model %q", model)
	}
	baseURL := os.Getenv("OPENCODE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:4096"
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	if err := os.WriteFile(target, []byte("# Local test\n\nThis page contains no claims to compare.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var output bytes.Buffer
	if err := Run(ctx, Config{Target: target, Model: model, BaseURL: baseURL, Evidence: []string{}, Output: &output, TraceOutput: &output}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"results"`) {
		t.Fatalf("missing structured results: %s", output.String())
	}
}
