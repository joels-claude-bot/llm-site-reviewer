package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/logging"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/visual"
	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

func main() {
	logging.InitFromEnv()

	mode := flag.String("mode", "batch", "review mode: batch, dependency, opencode, or visual")
	recursive := flag.Bool("recursive", false, "batch mode: descend into nested subdirectories (default: only immediate files of each target directory)")
	model := flag.String("model", "litellm/gemini-flash", "opencode mode: provider/model to use")
	opencodeURL := flag.String("opencode-url", "http://127.0.0.1:4096", "opencode mode: server URL")
	opencodeVerbose := flag.Bool("opencode-verbose", false, "opencode mode: log file-discovery and read tool calls")
	opencodeTimeout := flag.Duration("opencode-timeout", 15*time.Minute, "opencode mode: overall session creation and prompt timeout")
	opencodeEvidence := flag.String("opencode-evidence", "", "opencode mode: comma-separated workspace-relative evidence files; disables search and permits only reads")
	dependencyTargets := flag.String("dependency-targets", "", "dependency mode: comma-separated page paths to resolve and review (all pages are still summarised)")
	dependencySkipImages := flag.Bool("dependency-skip-images", false, "dependency mode: skip image transcription for a content-only run")
	dependencyRecursive := flag.Bool("dependency-recursive", true, "dependency mode: descend into nested corpus directories")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: inspect [FLAGS] target [target...]\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()

	}
	flag.Parse()

	positional := flag.Args()

	if len(positional) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	switch *mode {
	case "batch":
		err = review.LiteLLMRequest(positional, *recursive)
	case "dependency":
		var targets []string
		if *dependencyTargets != "" {
			for _, target := range strings.Split(*dependencyTargets, ",") {
				absolute, absErr := filepath.Abs(strings.TrimSpace(target))
				if absErr != nil {
					err = absErr
					break
				}
				targets = append(targets, absolute)
			}
		}
		if err == nil {
			err = runDependency(positional, targets, *dependencySkipImages, *dependencyRecursive)
		}
	case "opencode":
		if len(positional) != 1 {
			err = fmt.Errorf("opencode mode requires exactly one target file")
			break
		}
		if *opencodeTimeout <= 0 {
			err = fmt.Errorf("opencode-timeout must be positive")
			break
		}
		ctx, cancel := context.WithTimeout(context.Background(), *opencodeTimeout)
		defer cancel()
		var evidence []string
		for _, path := range strings.Split(*opencodeEvidence, ",") {
			if path = strings.TrimSpace(path); path != "" {
				evidence = append(evidence, path)
			}
		}
		err = runOpenCode(ctx, positional[0], *model, *opencodeURL, evidence, *opencodeVerbose)
	case "visual":
		err = runVisual(positional)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q (want batch or dependency)\n\n", *mode)
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("failed to call request: %v", err)
	}

}

type openCodePrompt struct {
	Model  openCodeModel   `json:"model"`
	Format openCodeFormat  `json:"format"`
	Parts  []openCodePart  `json:"parts"`
	Tools  map[string]bool `json:"tools"`
}

type openCodeModel struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type openCodeFormat struct {
	Type       string `json:"type"`
	Schema     any    `json:"schema"`
	RetryCount int    `json:"retryCount"`
}

type openCodePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openCodeResponse struct {
	Info struct {
		Structured json.RawMessage `json:"structured"`
		Error      json.RawMessage `json:"error"`
		Finish     string          `json:"finish"`
	} `json:"info"`
	Parts json.RawMessage `json:"parts"`
}

func runOpenCode(ctx context.Context, target, model, baseURL string, evidence []string, verbose bool) error {
	target, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}
	if info, err := os.Stat(target); err != nil || info.IsDir() {
		return fmt.Errorf("opencode target must be a file: %s", target)
	}
	providerID, modelID, ok := strings.Cut(model, "/")
	if !ok || providerID == "" || modelID == "" {
		return fmt.Errorf("model must be provider/model, got %q", model)
	}

	workspace := findProjectRoot(filepath.Dir(target))
	for index, path := range evidence {
		absolute := path
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(workspace, filepath.FromSlash(path))
		}
		absolute, err = filepath.Abs(absolute)
		if err != nil || !strings.HasPrefix(absolute, workspace+string(filepath.Separator)) {
			return fmt.Errorf("evidence path must be inside workspace: %q", path)
		}
		if info, statErr := os.Stat(absolute); statErr != nil || info.IsDir() {
			return fmt.Errorf("evidence path must be a readable file: %q", path)
		}
		evidence[index], err = filepath.Rel(workspace, absolute)
		if err != nil {
			return fmt.Errorf("resolve evidence path %q: %w", path, err)
		}
	}
	client := opencode.NewClient(option.WithBaseURL(baseURL))
	session, err := client.Session.New(ctx, opencode.SessionNewParams{
		Directory: opencode.F(workspace),
		Title:     opencode.F("documentation review"),
	})
	if err != nil {
		return fmt.Errorf("create opencode session at %s: %w", baseURL, err)
	}

	reflector := &jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}
	exploration := `You may use only glob, grep and read. Read the target once. Use at most five focused grep calls and one glob call for its most distinctive entities, dates and quantities. Then stop searching and use at most five additional read calls on the strongest candidate evidence before returning the result. Combine related terms in each grep, do not re-read files, and do not survey the whole repository.`
	tools := readOnlyOpenCodeTools(true)
	if len(evidence) > 0 {
		exploration = fmt.Sprintf("Use the read tool only. Read the target once, then read only these preselected evidence files; do not inspect any other file:\n- %s", strings.Join(evidence, "\n- "))
		tools = readOnlyOpenCodeTools(false)
	}
	prompt := fmt.Sprintf(`Review %s for genuine reader-facing content defects.

This is a bounded, read-only review. %s

Use other Markdown files in the repository as evidence. Find pages discussing the same entities, dates, quantities, performance figures or factual claims, and compare their relevant text with the target. Report clear contradictions, incorrect claims, stale references and malformed content. Do not assume only explicitly linked pages are relevant.

Base every finding only on repository content. Do not browse the web, run shell commands, edit files, create plans, invoke skills or subagents, or assess rendering. Ignore stylistic preferences, harmless rounding differences and claims covering different periods or scopes.

Return exactly one result for the target: id 0, name %q, success false when errors are present and true otherwise. Put at most five concrete findings, ordered by importance, in that result's errors array. Use uppercase category codes such as CONTRADICTS_PAGE, WRONG_CLAIM, STALE_REF or OTHER.`, target, exploration, filepath.Base(target))
	request := openCodePrompt{
		Model:  openCodeModel{ProviderID: providerID, ModelID: modelID},
		Format: openCodeFormat{Type: "json_schema", Schema: reflector.Reflect(&review.MdResponse{}), RetryCount: 1},
		Parts:  []openCodePart{{Type: "text", Text: prompt}},
		Tools:  tools,
	}
	if verbose {
		encodedSchema, marshalErr := json.Marshal(request.Format.Schema)
		if marshalErr != nil {
			return fmt.Errorf("marshal opencode output schema: %w", marshalErr)
		}
		fmt.Fprintf(os.Stderr, "opencode trace: output schema: %s\n", encodedSchema)
	}
	stopTrace := func() {}
	if verbose {
		stopTrace = startOpenCodeTrace(ctx, client, session.ID, workspace)
	}
	defer stopTrace()
	var response openCodeResponse
	path := fmt.Sprintf("session/%s/message?directory=%s", session.ID, url.QueryEscape(workspace))
	if err := client.Execute(ctx, http.MethodPost, path, request, &response); err != nil {
		return fmt.Errorf("prompt opencode session: %w", err)
	}
	stopTrace()
	if len(response.Info.Structured) == 0 || string(response.Info.Structured) == "null" {
		if verbose {
			fmt.Fprintf(os.Stderr, "opencode trace: response finish=%q error=%s parts=%s\n", response.Info.Finish, printableJSON(response.Info.Error), printableJSON(response.Parts))
		}
		if message := openCodeErrorMessage(response.Info.Error); message != "" {
			return fmt.Errorf("opencode assistant error: %s", message)
		}
		return fmt.Errorf("opencode returned no structured output")
	}
	var result review.MdResponse
	if err := json.Unmarshal(response.Info.Structured, &result); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "opencode trace: raw structured output: %s\n", response.Info.Structured)
		}
		return fmt.Errorf("decode structured output: %w", err)
	}
	if err := validateOpenCodeResult(result, filepath.Base(target)); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "opencode trace: invalid structured output: %s\n", response.Info.Structured)
		}
		return fmt.Errorf("validate structured output: %w", err)
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal structured output: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}

func readOnlyOpenCodeTools(search bool) map[string]bool {
	return map[string]bool{
		"read": true, "grep": search, "glob": search,
		"bash": false, "edit": false, "write": false, "apply_patch": false,
		"webfetch": false, "websearch": false, "lsp": false, "task": false,
		"skill": false, "todowrite": false, "question": false,
	}
}

func printableJSON(value json.RawMessage) string {
	if len(value) == 0 {
		return "<missing>"
	}
	return string(value)
}

func openCodeErrorMessage(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var response struct {
		Name string `json:"name"`
		Data struct {
			Message    string `json:"message"`
			StatusCode int    `json:"statusCode"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &response) != nil || response.Data.Message == "" {
		return string(raw)
	}
	if response.Data.StatusCode != 0 {
		return fmt.Sprintf("%s: %s (HTTP %d)", response.Name, response.Data.Message, response.Data.StatusCode)
	}
	return fmt.Sprintf("%s: %s", response.Name, response.Data.Message)
}

func validateOpenCodeResult(response review.MdResponse, targetName string) error {
	if len(response.Results) != 1 {
		return fmt.Errorf("got %d results, want exactly 1", len(response.Results))
	}
	result := response.Results[0]
	if result.Id != 0 || result.Name != targetName {
		return fmt.Errorf("result identity is id=%d name=%q, want id=0 name=%q", result.Id, result.Name, targetName)
	}
	if len(result.Errors) > 5 {
		return fmt.Errorf("got %d findings, maximum is 5", len(result.Errors))
	}
	if result.Success != (len(result.Errors) == 0) {
		return fmt.Errorf("success=%t disagrees with %d findings", result.Success, len(result.Errors))
	}
	for index, finding := range result.Errors {
		if strings.TrimSpace(string(finding.Category)) == "" || strings.TrimSpace(finding.Notes) == "" {
			return fmt.Errorf("finding %d has an empty category or notes", index)
		}
	}
	return nil
}

func startOpenCodeTrace(ctx context.Context, client *opencode.Client, sessionID, workspace string) func() {
	traceCtx, cancel := context.WithCancel(ctx)
	stream := client.Event.ListStreaming(traceCtx, opencode.EventListParams{Directory: opencode.F(workspace)})
	done := make(chan struct{})
	type traceTiming struct {
		startedAt   time.Time
		lastEventAt time.Time
	}
	timing := make(chan traceTiming, 1)
	go func() {
		seenRunning := make(map[string]bool)
		seenFinished := make(map[string]bool)
		traceStarted := time.Now()
		lastToolEvent := traceStarted
		defer func() {
			timing <- traceTiming{startedAt: traceStarted, lastEventAt: lastToolEvent}
			close(done)
		}()
		for stream.Next() {
			event, ok := stream.Current().AsUnion().(opencode.EventListResponseEventMessagePartUpdated)
			if !ok {
				continue
			}
			part, ok := event.Properties.Part.AsUnion().(opencode.ToolPart)
			if !ok || part.SessionID != sessionID {
				continue
			}
			if part.Tool != "glob" && part.Tool != "grep" && part.Tool != "read" {
				continue
			}

			now := time.Now()
			switch state := part.State.AsUnion().(type) {
			case opencode.ToolStateRunning:
				if seenRunning[part.ID] {
					continue
				}
				seenRunning[part.ID] = true
				fmt.Fprintf(os.Stderr, "opencode trace: +%s (gap %s) %s started input=%s\n",
					now.Sub(traceStarted).Round(time.Millisecond), now.Sub(lastToolEvent).Round(time.Millisecond), part.Tool, compactJSON(state.Input))
				lastToolEvent = now
			case opencode.ToolStateCompleted:
				if seenFinished[part.ID] {
					continue
				}
				seenFinished[part.ID] = true
				duration := time.Duration(state.Time.End-state.Time.Start) * time.Millisecond
				fmt.Fprintf(os.Stderr, "opencode trace: +%s (gap %s) %s completed in %s input=%s\n",
					now.Sub(traceStarted).Round(time.Millisecond), now.Sub(lastToolEvent).Round(time.Millisecond), part.Tool, duration.Round(time.Millisecond), compactJSON(state.Input))
				lastToolEvent = now
			case opencode.ToolStateError:
				if seenFinished[part.ID] {
					continue
				}
				seenFinished[part.ID] = true
				duration := time.Duration(state.Time.End-state.Time.Start) * time.Millisecond
				fmt.Fprintf(os.Stderr, "opencode trace: +%s (gap %s) %s failed in %s error=%q input=%s\n",
					now.Sub(traceStarted).Round(time.Millisecond), now.Sub(lastToolEvent).Round(time.Millisecond), part.Tool, duration.Round(time.Millisecond), state.Error, compactJSON(state.Input))
				lastToolEvent = now
			}
		}
	}()
	stopped := false
	return func() {
		if stopped {
			return
		}
		stopped = true
		responseAt := time.Now()
		cancel()
		_ = stream.Close()
		<-done
		measured := <-timing
		fmt.Fprintf(os.Stderr, "opencode trace: response returned +%s (gap since last tool event %s)\n",
			responseAt.Sub(measured.startedAt).Round(time.Millisecond), responseAt.Sub(measured.lastEventAt).Round(time.Millisecond))
	}
}

func compactJSON(value interface{}) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%q", fmt.Sprint(value))
	}
	return string(encoded)
}

func findProjectRoot(start string) string {
	for current := start; ; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return start
		}
	}
}

func runVisual(urls []string) error {
	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return err
	}
	results, err := review.VisualReview(context.Background(), client, urls, "", visual.DefaultConfig())
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

// runDependency wires up the dependency review path. It takes one or more root
// directories (DependencyReview summarises the pages under all of them and
// resolves cross-page dependencies across the combined set) and prints the
// results as a single JSON array, one object per reviewed page.
func runDependency(paths, targets []string, skipImages, recursive bool) error {
	client, err := review.NewLLMClient(review.ClientConfigFromEnv(review.ProviderLiteLLMProxy))
	if err != nil {
		return fmt.Errorf("failed to build litellm proxy client: %w", err)
	}

	config := review.DefaultDependencyConfig()
	config.TargetPaths = targets
	config.SkipImages = skipImages
	config.Recursive = recursive
	results, err := review.DependencyReview(context.Background(), client, paths, config)
	if err != nil {
		return err
	}

	// Emit a single JSON result, mirroring batch mode: one object per reviewed
	// page, each carrying the same MdItem shape (id, name, success, errors).
	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dependency results: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}
