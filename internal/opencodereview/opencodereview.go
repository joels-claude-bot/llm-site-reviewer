// Package opencodereview runs a bounded, read-only review through an OpenCode server.
package opencodereview

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

type Config struct {
	Target, Model, BaseURL string
	Evidence               []string
	Verbose                bool
	Output, TraceOutput    io.Writer
}

type prompt struct {
	Model  model           `json:"model"`
	Format format          `json:"format"`
	Parts  []part          `json:"parts"`
	Tools  map[string]bool `json:"tools"`
}
type model struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}
type format struct {
	Type       string `json:"type"`
	Schema     any    `json:"schema"`
	RetryCount int    `json:"retryCount"`
}
type part struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type response struct {
	Info struct {
		Structured, Error json.RawMessage
		Finish            string
	} `json:"info"`
	Parts json.RawMessage `json:"parts"`
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.Output == nil {
		cfg.Output = io.Discard
	}
	if cfg.TraceOutput == nil {
		cfg.TraceOutput = io.Discard
	}
	target, err := filepath.Abs(cfg.Target)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.IsDir() {
		return fmt.Errorf("opencode target must be a file: %s", target)
	}
	providerID, modelID, ok := strings.Cut(cfg.Model, "/")
	if !ok || providerID == "" || modelID == "" {
		return fmt.Errorf("model must be provider/model, got %q", cfg.Model)
	}
	workspace := FindProjectRoot(filepath.Dir(target))
	for i, p := range cfg.Evidence {
		absolute := p
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(workspace, filepath.FromSlash(p))
		}
		absolute, err = filepath.Abs(absolute)
		if err != nil || !strings.HasPrefix(absolute, workspace+string(filepath.Separator)) {
			return fmt.Errorf("evidence path must be inside workspace: %q", p)
		}
		if info, statErr := os.Stat(absolute); statErr != nil || info.IsDir() {
			return fmt.Errorf("evidence path must be a readable file: %q", p)
		}
		cfg.Evidence[i], err = filepath.Rel(workspace, absolute)
		if err != nil {
			return fmt.Errorf("resolve evidence path %q: %w", p, err)
		}
	}
	client := opencode.NewClient(option.WithBaseURL(cfg.BaseURL))
	session, err := client.Session.New(ctx, opencode.SessionNewParams{Directory: opencode.F(workspace), Title: opencode.F("documentation review")})
	if err != nil {
		return fmt.Errorf("create opencode session at %s: %w", cfg.BaseURL, err)
	}
	exploration := `You may use only glob, grep and read. Read the target once. Use at most five focused grep calls and one glob call for its most distinctive entities, dates and quantities. Then stop searching and use at most five additional read calls on the strongest candidate evidence before returning the result. Combine related terms in each grep, do not re-read files, and do not survey the whole repository.`
	tools := readOnlyTools(true)
	if len(cfg.Evidence) > 0 {
		exploration = fmt.Sprintf("Use the read tool only. Read the target once, then read only these preselected evidence files; do not inspect any other file:\n- %s", strings.Join(cfg.Evidence, "\n- "))
		tools = readOnlyTools(false)
	}
	text := fmt.Sprintf(`Review %s for genuine reader-facing content defects.

This is a bounded, read-only review. %s

Use other Markdown files in the repository as evidence. Find pages discussing the same entities, dates, quantities, performance figures or factual claims, and compare their relevant text with the target. Report clear contradictions, incorrect claims, stale references and malformed content. Do not assume only explicitly linked pages are relevant.

Base every finding only on repository content. Do not browse the web, run shell commands, edit files, create plans, invoke skills or subagents, or assess rendering. Ignore stylistic preferences, harmless rounding differences and claims covering different periods or scopes.

Return exactly one result for the target: id 0, name %q, success false when errors are present and true otherwise. Put at most five concrete findings, ordered by importance, in that result's errors array. Use uppercase category codes such as CONTRADICTS_PAGE, WRONG_CLAIM, STALE_REF or OTHER.`, target, exploration, filepath.Base(target))
	req := prompt{Model: model{providerID, modelID}, Format: format{"json_schema", (&jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}).Reflect(&review.MdResponse{}), 1}, Parts: []part{{"text", text}}, Tools: tools}
	if cfg.Verbose {
		schema, e := json.Marshal(req.Format.Schema)
		if e != nil {
			return fmt.Errorf("marshal opencode output schema: %w", e)
		}
		fmt.Fprintf(cfg.TraceOutput, "opencode trace: output schema: %s\n", schema)
	}
	stop := func() {}
	if cfg.Verbose {
		stop = startTrace(ctx, client, session.ID, workspace, cfg.TraceOutput)
	}
	defer stop()
	var resp response
	path := fmt.Sprintf("session/%s/message?directory=%s", session.ID, url.QueryEscape(workspace))
	if err := client.Execute(ctx, http.MethodPost, path, req, &resp); err != nil {
		return fmt.Errorf("prompt opencode session: %w", err)
	}
	stop()
	if len(resp.Info.Structured) == 0 || string(resp.Info.Structured) == "null" {
		if cfg.Verbose {
			fmt.Fprintf(cfg.TraceOutput, "opencode trace: response finish=%q error=%s parts=%s\n", resp.Info.Finish, printable(resp.Info.Error), printable(resp.Parts))
		}
		if m := errorMessage(resp.Info.Error); m != "" {
			return fmt.Errorf("opencode assistant error: %s", m)
		}
		return fmt.Errorf("opencode returned no structured output")
	}
	var result review.MdResponse
	if err := json.Unmarshal(resp.Info.Structured, &result); err != nil {
		return fmt.Errorf("decode structured output: %w", err)
	}
	if err := validate(result, filepath.Base(target)); err != nil {
		return fmt.Errorf("validate structured output: %w", err)
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cfg.Output, string(encoded))
	return nil
}

func readOnlyTools(search bool) map[string]bool {
	return map[string]bool{"read": true, "grep": search, "glob": search, "bash": false, "edit": false, "write": false, "apply_patch": false, "webfetch": false, "websearch": false, "lsp": false, "task": false, "skill": false, "todowrite": false, "question": false}
}
func printable(v json.RawMessage) string {
	if len(v) == 0 {
		return "<missing>"
	}
	return string(v)
}
func errorMessage(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var r struct {
		Name string `json:"name"`
		Data struct {
			Message    string `json:"message"`
			StatusCode int    `json:"statusCode"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &r) != nil || r.Data.Message == "" {
		return string(raw)
	}
	if r.Data.StatusCode != 0 {
		return fmt.Sprintf("%s: %s (HTTP %d)", r.Name, r.Data.Message, r.Data.StatusCode)
	}
	return fmt.Sprintf("%s: %s", r.Name, r.Data.Message)
}
func validate(r review.MdResponse, name string) error {
	if len(r.Results) != 1 {
		return fmt.Errorf("got %d results, want exactly 1", len(r.Results))
	}
	v := r.Results[0]
	if v.Id != 0 || v.Name != name {
		return fmt.Errorf("result identity is id=%d name=%q, want id=0 name=%q", v.Id, v.Name, name)
	}
	if len(v.Errors) > 5 {
		return fmt.Errorf("got %d findings, maximum is 5", len(v.Errors))
	}
	if v.Success != (len(v.Errors) == 0) {
		return fmt.Errorf("success=%t disagrees with %d findings", v.Success, len(v.Errors))
	}
	for i, f := range v.Errors {
		if strings.TrimSpace(string(f.Category)) == "" || strings.TrimSpace(f.Notes) == "" {
			return fmt.Errorf("finding %d has an empty category or notes", i)
		}
	}
	return nil
}

type traceClock func() time.Time
type traceRecorder struct {
	out               io.Writer
	started, last     time.Time
	running, finished map[string]bool
	now               traceClock
}

func newTraceRecorder(out io.Writer, now traceClock) *traceRecorder {
	t := now()
	return &traceRecorder{out: out, started: t, last: t, running: map[string]bool{}, finished: map[string]bool{}, now: now}
}
func (r *traceRecorder) record(p opencode.ToolPart) {
	if p.Tool != "glob" && p.Tool != "grep" && p.Tool != "read" {
		return
	}
	now := r.now()
	switch s := p.State.AsUnion().(type) {
	case opencode.ToolStateRunning:
		if r.running[p.ID] {
			return
		}
		r.running[p.ID] = true
		fmt.Fprintf(r.out, "opencode trace: +%s (gap %s) %s started input=%s\n", now.Sub(r.started).Round(time.Millisecond), now.Sub(r.last).Round(time.Millisecond), p.Tool, compact(s.Input))
		r.last = now
	case opencode.ToolStateCompleted:
		if r.finished[p.ID] {
			return
		}
		r.finished[p.ID] = true
		d := time.Duration(s.Time.End-s.Time.Start) * time.Millisecond
		fmt.Fprintf(r.out, "opencode trace: +%s (gap %s) %s completed in %s input=%s\n", now.Sub(r.started).Round(time.Millisecond), now.Sub(r.last).Round(time.Millisecond), p.Tool, d.Round(time.Millisecond), compact(s.Input))
		r.last = now
	case opencode.ToolStateError:
		if r.finished[p.ID] {
			return
		}
		r.finished[p.ID] = true
		d := time.Duration(s.Time.End-s.Time.Start) * time.Millisecond
		fmt.Fprintf(r.out, "opencode trace: +%s (gap %s) %s failed in %s error=%q input=%s\n", now.Sub(r.started).Round(time.Millisecond), now.Sub(r.last).Round(time.Millisecond), p.Tool, d.Round(time.Millisecond), s.Error, compact(s.Input))
		r.last = now
	}
}
func startTrace(ctx context.Context, c *opencode.Client, session, workspace string, out io.Writer) func() {
	traceCtx, cancel := context.WithCancel(ctx)
	stream := c.Event.ListStreaming(traceCtx, opencode.EventListParams{Directory: opencode.F(workspace)})
	done := make(chan struct{})
	rec := newTraceRecorder(out, time.Now)
	var mu sync.Mutex
	go func() {
		defer close(done)
		for stream.Next() {
			e, ok := stream.Current().AsUnion().(opencode.EventListResponseEventMessagePartUpdated)
			if !ok {
				continue
			}
			p, ok := e.Properties.Part.AsUnion().(opencode.ToolPart)
			if !ok || p.SessionID != session {
				continue
			}
			mu.Lock()
			rec.record(p)
			mu.Unlock()
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			at := time.Now()
			cancel()
			_ = stream.Close()
			<-done
			mu.Lock()
			defer mu.Unlock()
			fmt.Fprintf(out, "opencode trace: response returned +%s (gap since last tool event %s)\n", at.Sub(rec.started).Round(time.Millisecond), at.Sub(rec.last).Round(time.Millisecond))
		})
	}
}
func compact(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		return fmt.Sprintf("%q", fmt.Sprint(v))
	}
	return string(b)
}

// FindProjectRoot returns the closest ancestor containing .git, falling back
// to start when the target is not in a Git working tree.
func FindProjectRoot(start string) string {
	for cur := start; ; cur = filepath.Dir(cur) {
		if _, e := os.Stat(filepath.Join(cur, ".git")); e == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return start
		}
	}
}
