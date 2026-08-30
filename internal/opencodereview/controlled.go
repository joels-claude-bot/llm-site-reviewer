package opencodereview

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/controlledreview"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

// ControlledConfig configures the OpenCode implementation of the three-phase
// controlled review model. Each PhaseModel method makes one structured prompt
// call and exposes no tools to the model.
type ControlledConfig struct {
	BaseURL, Model, Workspace string
	TraceOutput               io.Writer
}

// ControlledPhaseModel uses one OpenCode session for the three bounded model
// decisions. Filesystem access remains entirely in controlledreview.Engine.
type ControlledPhaseModel struct {
	client      *opencode.Client
	sessionID   string
	workspace   string
	model       model
	traceOutput io.Writer
	mu          sync.Mutex
}

type searchPatternsOutput struct {
	Patterns []string `json:"patterns"`
}

type evidencePathsOutput struct {
	Paths []string `json:"paths"`
}

// NewControlledPhaseModel creates the sole OpenCode session used by a
// controlled review. The caller controls the lifetime and timeout via ctx.
func NewControlledPhaseModel(ctx context.Context, cfg ControlledConfig) (*ControlledPhaseModel, error) {
	providerID, modelID, ok := strings.Cut(cfg.Model, "/")
	if !ok || providerID == "" || modelID == "" {
		return nil, fmt.Errorf("model must be provider/model, got %q", cfg.Model)
	}
	workspace, err := filepath.Abs(cfg.Workspace)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}
	client := opencode.NewClient(option.WithBaseURL(cfg.BaseURL))
	session, err := client.Session.New(ctx, opencode.SessionNewParams{
		Directory: opencode.F(workspace),
		Title:     opencode.F("controlled documentation review"),
	})
	if err != nil {
		return nil, fmt.Errorf("create opencode session at %s: %w", cfg.BaseURL, err)
	}
	return &ControlledPhaseModel{
		client: client, sessionID: session.ID, workspace: workspace,
		model: model{ProviderID: providerID, ModelID: modelID}, traceOutput: cfg.TraceOutput,
	}, nil
}

func (m *ControlledPhaseModel) ChooseSearchPatterns(ctx context.Context, in controlledreview.SearchInput) ([]string, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("encode search input: %w", err)
	}
	prompt := fmt.Sprintf(`Choose at most %d focused Go/RE2 regular expressions that will find repository Markdown evidence for factual claims in the target.

Combine closely related distinctive entities, dates, and quantities into each expression. Prefer patterns likely to reveal contradictions or stale facts. Do not merely list broad generic words. You have no tools and must decide only from the supplied target and image transcriptions.

Return only the structured patterns result.

INPUT JSON:
%s`, in.MaxPatterns, payload)
	var out searchPatternsOutput
	if err := m.structured(ctx, "search patterns", prompt, &out); err != nil {
		return nil, err
	}
	return out.Patterns, nil
}

func (m *ControlledPhaseModel) SelectEvidence(ctx context.Context, in controlledreview.EvidenceInput) ([]string, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("encode evidence input: %w", err)
	}
	prompt := fmt.Sprintf(`Select at most %d Markdown evidence files from the supplied bounded search results.

Choose only paths actually present in the search matches, prioritising pages most likely to confirm or contradict concrete target claims. Do not select the target itself. Use repository-relative paths. You have no tools.

Return only the structured paths result.

INPUT JSON:
%s`, in.MaxFiles, payload)
	var out evidencePathsOutput
	if err := m.structured(ctx, "evidence selection", prompt, &out); err != nil {
		return nil, err
	}
	return out.Paths, nil
}

func (m *ControlledPhaseModel) Review(ctx context.Context, in controlledreview.ReviewInput) (review.MdResponse, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return review.MdResponse{}, fmt.Errorf("encode review input: %w", err)
	}
	prompt := fmt.Sprintf(`Review the target for genuine reader-facing content defects using only the supplied target, image transcriptions, bounded search snippets, and selected evidence documents.

Report clear contradictions, incorrect claims, stale references, and malformed content. Ignore stylistic preferences, harmless rounding, and claims covering different periods or scopes. Base every finding only on supplied repository content. Return exactly one result with id 0 and name %q; success is false when errors exist and true otherwise. Include at most five concrete findings ordered by importance, using uppercase categories such as CONTRADICTS_PAGE, WRONG_CLAIM, STALE_REF, or OTHER.

Return only the structured review result.

INPUT JSON:
%s`, filepath.Base(in.TargetPath), payload)
	var out review.MdResponse
	if err := m.structured(ctx, "final review", prompt, &out); err != nil {
		return review.MdResponse{}, err
	}
	return out, nil
}

func (m *ControlledPhaseModel) structured(ctx context.Context, phase, text string, out any) error {
	// Serialisation makes the exact one-session, three-call behaviour safe even
	// if a caller accidentally invokes phases concurrently.
	m.mu.Lock()
	defer m.mu.Unlock()

	req := prompt{
		Model: m.model,
		Format: format{
			Type: "json_schema", Schema: (&jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}).Reflect(out), RetryCount: 1,
		},
		Parts: []part{{Type: "text", Text: text}},
		Tools: noTools(),
	}
	started := time.Now()
	var resp response
	path := fmt.Sprintf("session/%s/message?directory=%s", m.sessionID, url.QueryEscape(m.workspace))
	if err := m.client.Execute(ctx, http.MethodPost, path, req, &resp); err != nil {
		return fmt.Errorf("%s prompt: %w", phase, err)
	}
	if m.traceOutput != nil {
		fmt.Fprintf(m.traceOutput, "opencode controlled trace: %s completed in %s\n", phase, time.Since(started).Round(time.Millisecond))
	}
	if len(resp.Info.Structured) == 0 || string(resp.Info.Structured) == "null" {
		if message := errorMessage(resp.Info.Error); message != "" {
			return fmt.Errorf("%s assistant error: %s", phase, message)
		}
		return fmt.Errorf("%s returned no structured output", phase)
	}
	if err := json.Unmarshal(resp.Info.Structured, out); err != nil {
		return fmt.Errorf("decode %s structured output: %w", phase, err)
	}
	return nil
}

func noTools() map[string]bool {
	return map[string]bool{
		"read": false, "grep": false, "glob": false, "bash": false,
		"edit": false, "write": false, "apply_patch": false,
		"webfetch": false, "websearch": false, "lsp": false,
		"task": false, "skill": false, "todowrite": false, "question": false,
	}
}

var _ controlledreview.PhaseModel = (*ControlledPhaseModel)(nil)
