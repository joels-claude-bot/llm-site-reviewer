package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const workspace = "/home/claude/politician-trading-research"

type finding struct {
	Category string `json:"category"`
	Notes    string `json:"notes"`
}

type result struct {
	ID      int       `json:"id"`
	Name    string    `json:"name"`
	Success bool      `json:"success"`
	Errors  []finding `json:"errors"`
}

type structuredResponse struct {
	Results []result `json:"results"`
}

type session struct {
	ID string `json:"id"`
}

type promptResponse struct {
	Info struct {
		ProviderID string          `json:"providerID"`
		ModelID    string          `json:"modelID"`
		Structured json.RawMessage `json:"structured"`
		Cost       float64         `json:"cost"`
		Tokens     struct {
			Total     int `json:"total"`
			Input     int `json:"input"`
			Output    int `json:"output"`
			Reasoning int `json:"reasoning"`
			Cache     struct {
				Read  int `json:"read"`
				Write int `json:"write"`
			} `json:"cache"`
		} `json:"tokens"`
	} `json:"info"`
}

func main() {
	targetFlag := flag.String("target", filepath.Join(workspace, "docs/10-decision-tree-cap-clean.md"), "Markdown file to review")
	serverFlag := flag.String("server", "http://127.0.0.1:4097", "OpenCode server URL")
	timeoutFlag := flag.Duration("timeout", 15*time.Minute, "complete HTTP request timeout")
	flag.Parse()

	if err := run(*targetFlag, *serverFlag, *timeoutFlag); err != nil {
		fmt.Fprintln(os.Stderr, "control failed:", err)
		os.Exit(1)
	}
}

func run(target, serverURL string, timeout time.Duration) error {
	target, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve target: %w", err)
	}
	if !strings.HasPrefix(target, workspace+string(filepath.Separator)) {
		return fmt.Errorf("target must be inside %s", workspace)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.IsDir() {
		return fmt.Errorf("target must be a readable file: %s", target)
	}

	client := &http.Client{Timeout: timeout}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var created session
	if err := requestJSON(ctx, client, http.MethodPost, serverURL+"/session?directory="+url.QueryEscape(workspace), map[string]string{
		"title": "curated Go structured-output control",
	}, &created); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	if created.ID == "" {
		return errors.New("create session returned no id")
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{"results": map[string]any{
			"type": "array", "minItems": 1, "maxItems": 1,
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":      map[string]any{"type": "integer", "const": 0},
					"name":    map[string]any{"type": "string", "const": filepath.Base(target)},
					"success": map[string]any{"type": "boolean"},
					"errors": map[string]any{
						"type": "array", "maxItems": 5,
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"category": map[string]any{"type": "string", "enum": []string{"CONTRADICTS_PAGE", "WRONG_CLAIM", "STALE_REF", "OTHER"}},
								"notes":    map[string]any{"type": "string", "minLength": 1},
							},
							"required": []string{"category", "notes"}, "additionalProperties": false,
						},
					},
				},
				"required": []string{"id", "name", "success", "errors"}, "additionalProperties": false,
			},
		}},
		"required": []string{"results"}, "additionalProperties": false,
	}

	prompt := fmt.Sprintf(`Review %s for genuine reader-facing factual contradictions.

This is a deliberately bounded review. Use the read tool only, at most five times. Read the target, then consult only these evidence locations:
- docs/02-backtest-pelosi-cap-clean.md
- docs/13-incumbent-deep-dives.md
- docs/research/competitors/datagardener/profile.md
- docs/raw_niche_edge.md, only lines 315-330

Do not inspect any other file. Do not re-read a file. Do not search, browse, run commands, edit files, create tasks, or use subagents. The shortlist was selected because it contains direct evidence for quantitative, ownership, acquisition, and founding-date claims in the target.

Report only clear contradictions supported by the listed repository evidence. Ignore style, missing detail, harmless rounding, and claims with different scopes or periods. Return one result: id 0, name %q, success false iff errors is non-empty, and no more than five findings. Each note must name both conflicting claims and the evidence file.`, target, filepath.Base(target))

	tools := map[string]bool{
		"read": true, "grep": false, "glob": false, "bash": false,
		"edit": false, "write": false, "apply_patch": false,
		"webfetch": false, "websearch": false, "task": false,
		"skill": false, "todowrite": false, "question": false,
	}
	body := map[string]any{
		"model":  map[string]string{"providerID": "opencode-go", "modelID": "glm-5.3-flash"},
		"tools":  tools,
		"format": map[string]any{"type": "json_schema", "schema": schema, "retryCount": 1},
		"parts":  []map[string]string{{"type": "text", "text": prompt}},
	}

	started := time.Now()
	fmt.Fprintf(os.Stderr, "prompting session %s with read-only curated evidence (timeout %s)\n", created.ID, timeout)
	var response promptResponse
	endpoint := fmt.Sprintf("%s/session/%s/message?directory=%s", strings.TrimRight(serverURL, "/"), created.ID, url.QueryEscape(workspace))
	if err := requestJSON(ctx, client, http.MethodPost, endpoint, body, &response); err != nil {
		return fmt.Errorf("prompt session after %s: %w", time.Since(started).Round(time.Second), err)
	}

	var structured structuredResponse
	if err := json.Unmarshal(response.Info.Structured, &structured); err != nil {
		return fmt.Errorf("locally decode structured output %s: %w", response.Info.Structured, err)
	}
	if err := validate(structured, filepath.Base(target)); err != nil {
		return fmt.Errorf("locally validate structured output: %w", err)
	}

	encoded, err := json.MarshalIndent(structured, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	fmt.Fprintf(os.Stderr, "completed in %s; provider=%s model=%s tokens=%d (input=%d output=%d reasoning=%d cache-read=%d) cost=$%.6f\n",
		time.Since(started).Round(time.Second), response.Info.ProviderID, response.Info.ModelID,
		response.Info.Tokens.Total, response.Info.Tokens.Input, response.Info.Tokens.Output,
		response.Info.Tokens.Reasoning, response.Info.Tokens.Cache.Read, response.Info.Cost)
	return nil
}

func validate(response structuredResponse, targetName string) error {
	if len(response.Results) != 1 {
		return fmt.Errorf("got %d results, want exactly 1", len(response.Results))
	}
	result := response.Results[0]
	if result.ID != 0 || result.Name != targetName {
		return fmt.Errorf("result identity is id=%d name=%q", result.ID, result.Name)
	}
	if len(result.Errors) > 5 {
		return fmt.Errorf("got %d findings, maximum is 5", len(result.Errors))
	}
	if result.Success != (len(result.Errors) == 0) {
		return fmt.Errorf("success=%t disagrees with %d findings", result.Success, len(result.Errors))
	}
	allowed := map[string]bool{"CONTRADICTS_PAGE": true, "WRONG_CLAIM": true, "STALE_REF": true, "OTHER": true}
	for index, finding := range result.Errors {
		if !allowed[finding.Category] || strings.TrimSpace(finding.Notes) == "" {
			return fmt.Errorf("finding %d has invalid category or empty notes", index)
		}
	}
	return nil
}

func requestJSON(ctx context.Context, client *http.Client, method, endpoint string, body, output any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %s: %s", response.Status, strings.TrimSpace(string(payload)))
	}
	if err := json.Unmarshal(payload, output); err != nil {
		return fmt.Errorf("decode HTTP %s response: %w", response.Status, err)
	}
	return nil
}
