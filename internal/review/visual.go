package review

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
	visualcapture "github.com/joels-claude-bot/llm-site-reviewer/internal/visual"
	"github.com/voocel/litellm"
)

type VisualFinding struct {
	Category finding.Category `json:"category"`
	Notes    string           `json:"notes"`
	Frame    int              `json:"frame"`
}
type VisualResult struct {
	URL      string          `json:"url"`
	Success  bool            `json:"success"`
	Findings []VisualFinding `json:"findings"`
}
type visualResponse struct {
	Findings []VisualFinding `json:"findings"`
}

var screenshotCategories = []finding.Category{finding.RenderedBroken, finding.DiagramSyntaxError, finding.Truncation, finding.TextLegibility, finding.FlatColour, finding.WeakHierarchy, finding.HighDensity, finding.ImageMismatch}

func VisualReview(ctx context.Context, client ChatClient, urls []string, model string, cfg visualcapture.Config) ([]VisualResult, error) {
	if model == "" {
		model = modelVision
	}
	schema, err := json.Marshal((&jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}).Reflect(&visualResponse{}))
	if err != nil {
		return nil, err
	}
	format, err := litellm.NewResponseFormatJSONSchema("visual_response", "", string(schema), litellm.StrictEnabled)
	if err != nil {
		return nil, err
	}
	categories, _ := json.Marshal(screenshotCategories)
	results := make([]VisualResult, 0, len(urls))
	for _, url := range urls {
		capture, err := visualcapture.CapturePage(url, cfg)
		if err != nil {
			return nil, err
		}
		prompt := fmt.Sprintf("Review %s from %d overlapping screenshots in top-to-bottom order. Report only defects visible in pixels. Do not review links, prose style, or outside facts. Allowed categories: %s. Overlap repeats content, so report each defect once. Frame is the 1-based screenshot with clearest evidence.", url, len(capture.Frames), categories)
		blocks := []litellm.Block{litellm.Text(prompt)}
		for _, frame := range capture.Frames {
			blocks = append(blocks, litellm.ImageBlock{Data: frame.PNG, MIME: "image/png"})
		}
		resp, err := client.Chat(ctx, litellm.Request{Model: model, Messages: []litellm.Message{litellm.User(blocks...)}, ResponseFormat: format})
		if err != nil {
			return nil, fmt.Errorf("visual review %s: %w", url, err)
		}
		var parsed visualResponse
		if err := json.Unmarshal([]byte(resp.Text()), &parsed); err != nil {
			return nil, fmt.Errorf("parse visual review %s: %w", url, err)
		}
		allowed := map[finding.Category]bool{}
		for _, category := range screenshotCategories {
			allowed[category] = true
		}
		for _, item := range parsed.Findings {
			if !allowed[item.Category] {
				return nil, fmt.Errorf("visual review returned unknown category %q", item.Category)
			}
		}
		results = append(results, VisualResult{URL: url, Success: len(parsed.Findings) == 0, Findings: parsed.Findings})
	}
	return results, nil
}
