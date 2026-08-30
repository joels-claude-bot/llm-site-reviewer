package review

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/voocel/litellm"
)

type fakeChatClient struct {
	responses []string
	err       error
	requests  []litellm.Request
}

func (f *fakeChatClient) Chat(_ context.Context, req litellm.Request) (*litellm.Response, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.responses) == 0 {
		return &litellm.Response{}, nil
	}
	text := f.responses[0]
	f.responses = f.responses[1:]
	return &litellm.Response{Blocks: []litellm.Block{litellm.TextBlock{Text: text}}}, nil
}

func TestSummariesJSON(t *testing.T) {
	data, err := summariesJSON([]docPage{
		{Path: "a.md", Summary: "A"},
		{Path: "b.md", Summary: "B"},
	})
	if err != nil {
		t.Fatalf("summariesJSON returned error: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("summary JSON is invalid: %s", data)
	}
	if !strings.Contains(string(data), `"path":"a.md"`) || !strings.Contains(string(data), `"summary":"B"`) {
		t.Fatalf("summary JSON = %s", data)
	}
}

func TestParseResolverResponse(t *testing.T) {
	parsed, err := parseResolverResponse("a.md", `{"status":"success","dependencyPaths":["b.md"]}`)
	if err != nil {
		t.Fatalf("parseResolverResponse returned error: %v", err)
	}
	if len(parsed.DependencyPaths) != 1 || parsed.DependencyPaths[0] != "b.md" {
		t.Fatalf("parsed = %+v", parsed)
	}
}

func TestParseResolverResponseSurfacesErrorStatus(t *testing.T) {
	parsed, err := parseResolverResponse("a.md", `{"status":"error","error":"nope"}`)
	if err != nil {
		t.Fatalf("parseResolverResponse returned error: %v", err)
	}
	if parsed.Status != dependencyStatusError || parsed.Error != "nope" {
		t.Fatalf("parsed = %+v, want error status surfaced", parsed)
	}
}

func TestValidatePaths(t *testing.T) {
	byPath := map[string]docPage{"a.md": {Path: "a.md"}, "b.md": {Path: "b.md"}}
	got := validatePaths("a.md", []string{"b.md"}, byPath)
	if len(got) != 1 || got[0] != "b.md" {
		t.Fatalf("got = %v, want [b.md]", got)
	}
}

// An unknown or self-referential path is dropped (not fatal): one page's bad
// resolver output must not abort the whole review.
func TestValidatePathsDropsUnknownAndSelf(t *testing.T) {
	byPath := map[string]docPage{"a.md": {Path: "a.md"}, "b.md": {Path: "b.md"}}
	got := validatePaths("a.md", []string{"missing.md", "a.md", "b.md"}, byPath)
	if len(got) != 1 || got[0] != "b.md" {
		t.Fatalf("got = %v, want [b.md] (missing + self dropped)", got)
	}
}

func TestValidatePathsTruncatesToMax(t *testing.T) {
	byPath := map[string]docPage{}
	var deps []string
	for _, name := range []string{"1.md", "2.md", "3.md", "4.md", "5.md", "6.md", "7.md"} {
		byPath[name] = docPage{Path: name}
		deps = append(deps, name)
	}
	got := validatePaths("a.md", deps, byPath)
	if len(got) != maxDependencies {
		t.Fatalf("got %d paths, want %d", len(got), maxDependencies)
	}
}

func TestBuildReviewPayload(t *testing.T) {
	page := docPage{
		Path:            "a.md",
		Content:         "# A",
		Images:          []docImage{{Path: "a.svg", Summary: "diagram"}},
		DependencyPaths: []string{"b.md"},
	}
	byPath := map[string]docPage{
		"a.md": page,
		"b.md": {Path: "b.md", Content: "# B"},
	}

	payload, err := buildReviewPayload(page, byPath)
	if err != nil {
		t.Fatalf("buildReviewPayload returned error: %v", err)
	}
	if payload.Target.Path != "a.md" || payload.Target.Content != "# A" {
		t.Fatalf("target = %+v", payload.Target)
	}
	if len(payload.Images) != 1 || payload.Images[0].Content != "diagram" {
		t.Fatalf("images = %+v", payload.Images)
	}
	if len(payload.Dependencies) != 1 || payload.Dependencies[0].Content != "# B" {
		t.Fatalf("dependencies = %+v", payload.Dependencies)
	}
}

func TestResolvePaths(t *testing.T) {
	client := &fakeChatClient{responses: []string{
		`{"status":"success","dependencyPaths":["b.md"]}`,
		`{"status":"success","dependencyPaths":[]}`,
	}}
	pages := []docPage{
		{Path: "a.md", Content: "# A", Summary: "A"},
		{Path: "b.md", Content: "# B", Summary: "B"},
	}

	resolved, err := resolvePaths(context.Background(), client, pages, "dep-model")
	if err != nil {
		t.Fatalf("resolvePaths returned error: %v", err)
	}
	if len(resolved[0].DependencyPaths) != 1 || resolved[0].DependencyPaths[0] != "b.md" {
		t.Fatalf("resolved deps = %+v", resolved[0].DependencyPaths)
	}
	if len(pages[0].DependencyPaths) != 0 {
		t.Fatalf("resolvePaths mutated input pages: %+v", pages[0].DependencyPaths)
	}
}

func TestResolvePathsSkipsErrorStatus(t *testing.T) {
	client := &fakeChatClient{responses: []string{
		`{"status":"error","error":"cannot resolve"}`,
		`{"status":"success","dependencyPaths":["a.md"]}`,
	}}
	pages := []docPage{
		{Path: "a.md", Content: "# A", Summary: "A"},
		{Path: "b.md", Content: "# B", Summary: "B"},
	}

	resolved, err := resolvePaths(context.Background(), client, pages, "dep-model")
	if err != nil {
		t.Fatalf("resolvePaths returned error: %v", err)
	}
	if len(resolved[0].DependencyPaths) != 0 {
		t.Fatalf("errored page should have no deps, got %v", resolved[0].DependencyPaths)
	}
	if len(resolved[1].DependencyPaths) != 1 || resolved[1].DependencyPaths[0] != "a.md" {
		t.Fatalf("second page should still resolve, got %v", resolved[1].DependencyPaths)
	}
}

func TestResolvePathsPropagatesChatError(t *testing.T) {
	client := &fakeChatClient{err: errors.New("network")}
	_, err := resolvePaths(context.Background(), client, []docPage{{Path: "a.md"}}, "dep-model")
	if err == nil || !strings.Contains(err.Error(), "network") {
		t.Fatalf("error = %v, want chat error", err)
	}
}

func TestReviewPage(t *testing.T) {
	client := &fakeChatClient{responses: []string{
		`{"results":[{"id":0,"name":"a.md","success":true,"errors":[]}]}`,
	}}
	page := docPage{Path: "a.md", Content: "# A", DependencyPaths: []string{"b.md"}}
	pages := []docPage{page, {Path: "b.md", Content: "# B"}}
	byPath := pagesByPath(pages)

	systemPrompt, err := BuildSystemPrompt()
	if err != nil {
		t.Fatalf("BuildSystemPrompt returned error: %v", err)
	}
	schemaJSON, err := buildJsonSchema()
	if err != nil {
		t.Fatalf("buildJsonSchema returned error: %v", err)
	}
	format, err := litellm.NewResponseFormatJSONSchema("response_format", "", string(schemaJSON), litellm.StrictEnabled)
	if err != nil {
		t.Fatalf("NewResponseFormatJSONSchema returned error: %v", err)
	}

	result, err := reviewPage(context.Background(), client, page, byPath, "review-model", systemPrompt, format)
	if err != nil {
		t.Fatalf("reviewPage returned error: %v", err)
	}
	if result.PagePath != "a.md" || len(result.Items) != 1 || !result.Items[0].Success {
		t.Fatalf("result = %+v", result)
	}
}

func TestFillDefaults(t *testing.T) {
	config := fillDefaults(DependencyConfig{ReviewModel: "custom-review"})
	if config.VisionModel == "" {
		t.Fatalf("defaults not filled: %+v", config)
	}
	if config.ReviewModel != "custom-review" {
		t.Fatalf("review model = %q, want custom-review", config.ReviewModel)
	}
}
