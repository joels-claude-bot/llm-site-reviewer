package opencodereview

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/controlledreview"
	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

func TestControlledPhaseModelMakesThreeStructuredCallsWithToolsDisabled(t *testing.T) {
	outputs := []string{
		`{"patterns":["NANC|70\\.9"]}`,
		`{"paths":["docs/evidence.md"]}`,
		`{"results":[{"id":0,"name":"target.md","success":true,"errors":[]}]}`,
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "/session/session-1/message") {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		var body struct {
			Format struct {
				Type string `json:"type"`
			} `json:"format"`
			Tools map[string]bool `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.Format.Type != "json_schema" {
			t.Errorf("format type = %q", body.Format.Type)
		}
		for name, enabled := range body.Tools {
			if enabled {
				t.Errorf("tool %q was enabled", name)
			}
		}
		if calls >= len(outputs) {
			t.Errorf("unexpected extra prompt call")
			http.Error(w, "extra", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"info":{"structured":` + outputs[calls] + `,"finish":"stop"},"parts":[]}`))
		calls++
	}))
	defer server.Close()

	client := opencode.NewClient(option.WithBaseURL(server.URL))
	m := &ControlledPhaseModel{
		client: client, sessionID: "session-1", workspace: "/repo",
		model: model{ProviderID: "ollama", ModelID: "test"},
	}
	ctx := context.Background()
	patterns, err := m.ChooseSearchPatterns(ctx, controlledreview.SearchInput{TargetPath: "/repo/target.md", TargetText: "target", MaxPatterns: 5})
	if err != nil || len(patterns) != 1 || patterns[0] != `NANC|70\.9` {
		t.Fatalf("patterns = %#v, err = %v", patterns, err)
	}
	paths, err := m.SelectEvidence(ctx, controlledreview.EvidenceInput{TargetPath: "/repo/target.md", MaxFiles: 5})
	if err != nil || len(paths) != 1 || paths[0] != "docs/evidence.md" {
		t.Fatalf("paths = %#v, err = %v", paths, err)
	}
	result, err := m.Review(ctx, controlledreview.ReviewInput{TargetPath: "/repo/target.md", TargetText: "target"})
	if err != nil || len(result.Results) != 1 || !result.Results[0].Success {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	if calls != 3 {
		t.Fatalf("prompt calls = %d, want 3", calls)
	}
}

func TestNoToolsDisablesReadAndSearch(t *testing.T) {
	tools := noTools()
	for _, name := range []string{"read", "grep", "glob", "bash", "task", "webfetch"} {
		if enabled, exists := tools[name]; !exists || enabled {
			t.Fatalf("tool %q: exists=%t enabled=%t", name, exists, enabled)
		}
	}
}

func TestControlledPhaseModelTracesRawResponseWhenStructuredOutputIsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"info":{"structured":null,"finish":"stop"},"parts":[{"type":"text","text":"plain JSON-ish output"}]}`))
	}))
	defer server.Close()

	var trace bytes.Buffer
	m := &ControlledPhaseModel{
		client: opencode.NewClient(option.WithBaseURL(server.URL)), sessionID: "session-1", workspace: "/repo",
		model: model{ProviderID: "ollama", ModelID: "test"}, traceOutput: &trace,
	}
	_, err := m.ChooseSearchPatterns(context.Background(), controlledreview.SearchInput{MaxPatterns: 5})
	if err == nil {
		t.Fatal("expected missing structured output error")
	}
	got := trace.String()
	for _, want := range []string{`search patterns raw response`, `finish="stop"`, `structured=null`, `plain JSON-ish output`} {
		if !strings.Contains(got, want) {
			t.Fatalf("trace %q does not contain %q", got, want)
		}
	}
}
