package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/invopop/jsonschema"
	"github.com/voocel/litellm"
)

// maxDependencies caps how many dependency pages the resolver may attach to a
// single page. The resolver prompt asks for at most this many; we truncate if
// the model returns more rather than trusting it.
const maxDependencies = 5

type DependencyConfig struct {
	VisionModel string // summarise pages + images, resolve dependency paths
	ReviewModel string // final content review
	Recursive   bool   // descend into subdirectories when summarising pages
}

func DefaultDependencyConfig() DependencyConfig {
	return DependencyConfig{
		VisionModel: modelVision,
		ReviewModel: modelReviewProxy,
		Recursive:   true,
	}
}

// DependencyResult is the review outcome for one page: its path and the
// per-item findings the model returned.
type DependencyResult struct {
	PagePath string   `json:"pagePath"`
	Items    []MdItem `json:"items"`
}

func DependencyReview(ctx context.Context, client ChatClient, rootPaths []string, config DependencyConfig) ([]DependencyResult, error) {
	config = fillDefaults(config)
	var pages []docPage
	for _, rootPath := range rootPaths {
		// Each CLI root is its own docs_dir root: leading-slash image references
		// inside its pages resolve against it (mkdocs semantics).
		collected, err := summarisePages(ctx, client, rootPath, rootPath, config.VisionModel, config.Recursive)
		if err != nil {
			return nil, fmt.Errorf("failed to summarise dependency review pages: %w", err)
		}
		pages = append(pages, collected...)
	}
	resolved, err := resolvePaths(ctx, client, pages, config.VisionModel)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependency paths: %w", err)
	}
	return reviewPages(ctx, client, resolved, config.ReviewModel)
}

func resolvePaths(ctx context.Context, client ChatClient, pages []docPage, model string) ([]docPage, error) {
	format, err := resolverResponseFormat()
	if err != nil {
		return nil, err
	}
	summaries, err := summariesJSON(pages)
	if err != nil {
		return nil, err
	}
	byPath := pagesByPath(pages)

	// Build the result by value so we never mutate the caller's slice: page is a
	// copy, and DependencyPaths is replaced (not appended in place), so pages[i]
	// is untouched.
	resolved := make([]docPage, len(pages))
	for index := range pages {
		page := pages[index]
		pageJSON, err := json.Marshal(page)
		if err != nil {
			return nil, fmt.Errorf("failed to jsonify page %v: %w", page.Path, err)
		}

		resp, err := client.Chat(ctx, litellm.Request{
			Model: model,
			Messages: []litellm.Message{
				litellm.System(dependencyResolverPrompt),
				litellm.UserText(string(pageJSON)),
				litellm.UserText(string(summaries)),
			},
			ResponseFormat: format,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve deps for %v: %w", page.Path, err)
		}

		parsed, err := parseResolverResponse(page.Path, resp.Text())
		if err != nil {
			return nil, err
		}
		if parsed.Status != dependencyStatusSuccess {
			slog.Warn("dependency resolver returned error; leaving page with no dependencies", "page_path", page.Path, "error", parsed.Error)
			resolved[index] = page
			continue
		}

		page.DependencyPaths = validatePaths(page.Path, parsed.DependencyPaths, byPath)
		resolved[index] = page
		slog.Debug("resolved dependency paths", "page_path", page.Path, "dependency_count", len(page.DependencyPaths))
	}
	return resolved, nil
}

func reviewPages(ctx context.Context, client ChatClient, pages []docPage, model string) ([]DependencyResult, error) {
	systemPrompt, err := BuildSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to build system prompt: %w", err)
	}
	schemaJSON, err := buildJsonSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to build JSON schema: %w", err)
	}
	format, err := litellm.NewResponseFormatJSONSchema("response_format", "", string(schemaJSON), litellm.StrictEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to get litellm response format schema: %w", err)
	}
	byPath := pagesByPath(pages)

	var results []DependencyResult
	for _, page := range pages {
		result, err := reviewPage(ctx, client, page, byPath, model, systemPrompt, format)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func reviewPage(ctx context.Context, client ChatClient, page docPage, byPath map[string]docPage, model, systemPrompt string, format *litellm.ResponseFormat) (DependencyResult, error) {
	payload, err := buildReviewPayload(page, byPath)
	if err != nil {
		return DependencyResult{}, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return DependencyResult{}, fmt.Errorf("failed to marshal dependency review payload for %v: %w", page.Path, err)
	}

	resp, err := client.Chat(ctx, litellm.Request{
		Model: model,
		Messages: []litellm.Message{
			litellm.System(systemPrompt),
			litellm.UserText(string(payloadJSON)),
		},
		ResponseFormat: format,
	})
	if err != nil {
		return DependencyResult{}, fmt.Errorf("failed to run dependency review for %v: %w", page.Path, err)
	}

	var parsed MdResponse
	if err := json.Unmarshal([]byte(resp.Text()), &parsed); err != nil {
		return DependencyResult{}, fmt.Errorf("failed to unmarshal review response for %v: %w", page.Path, err)
	}
	slog.Debug("dependency review response parsed", "page_path", page.Path, "item_count", len(parsed.Items))
	return DependencyResult{PagePath: page.Path, Items: parsed.Items}, nil
}

// dependencyResponseStatus reports whether the resolver could answer for a page.
type dependencyResponseStatus string

const (
	dependencyStatusSuccess dependencyResponseStatus = "success"
	dependencyStatusError   dependencyResponseStatus = "error"
)

// dependencyResponse is the resolver's reply for one page: the pages it depends
// on, or an error status the caller can skip past.
type dependencyResponse struct {
	Status          dependencyResponseStatus `json:"status" jsonschema:"enum=success,enum=error"`
	Error           string                   `json:"error" jsonschema:"description=error details if status is error"`
	DependencyPaths []string                 `json:"dependencyPaths" jsonschema:"description=list of dependency paths if status is success"`
}

func resolverResponseFormat() (*litellm.ResponseFormat, error) {
	reflector := &jsonschema.Reflector{
		DoNotReference: true,
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&dependencyResponse{})
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to stringify dependency response schema: %w", err)
	}
	return litellm.NewResponseFormatJSONSchema("response_format", "", string(schemaJSON), litellm.StrictEnabled)
}

// dependencySummary is the condensed form of a page (path + summary only) sent
// to the resolver, so it can judge dependencies without every full page body.
type dependencySummary struct {
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

func summariesJSON(pages []docPage) ([]byte, error) {
	summaries := make([]dependencySummary, 0, len(pages))
	for _, page := range pages {
		summaries = append(summaries, dependencySummary{Path: page.Path, Summary: page.Summary})
	}
	data, err := json.Marshal(summaries)
	if err != nil {
		return nil, fmt.Errorf("failed to jsonify summaries: %w", err)
	}
	return data, nil
}

// parseResolverResponse only fails on malformed JSON. A well-formed error status
// is surfaced to the caller, which decides whether to skip the page.
func parseResolverResponse(pagePath, raw string) (dependencyResponse, error) {
	var parsed dependencyResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return dependencyResponse{}, fmt.Errorf("failed to unmarshal dependency response for %v: %w", pagePath, err)
	}
	return parsed, nil
}

// validatePaths keeps only the dependency paths the resolver returned that are
// real, distinct pages. A self-reference or an unknown path is the model
// misbehaving on one page; like the resolver's own error status, it is dropped
// with a warning rather than aborting the whole review.
func validatePaths(pagePath string, depPaths []string, byPath map[string]docPage) []string {
	if len(depPaths) > maxDependencies {
		slog.Warn("resolver returned more dependencies than allowed; truncating", "page_path", pagePath, "returned", len(depPaths), "max", maxDependencies)
		depPaths = depPaths[:maxDependencies]
	}
	valid := make([]string, 0, len(depPaths))
	for _, depPath := range depPaths {
		if depPath == pagePath {
			slog.Warn("dropping self-referential dependency path", "page_path", pagePath)
			continue
		}
		if _, ok := byPath[depPath]; !ok {
			slog.Warn("dropping unknown dependency path", "page_path", pagePath, "dependency_path", depPath)
			continue
		}
		valid = append(valid, depPath)
	}
	return valid
}

// reviewDoc is one document as the reviewer sees it (target, image, or
// dependency): a path and its text content.
type reviewDoc struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// reviewPayload is everything the reviewer needs for one page: the page itself,
// its transcribed images, and its resolved dependency pages.
type reviewPayload struct {
	Target       reviewDoc   `json:"target"`
	Images       []reviewDoc `json:"images"`
	Dependencies []reviewDoc `json:"dependencies"`
}

func buildReviewPayload(page docPage, byPath map[string]docPage) (reviewPayload, error) {
	payload := reviewPayload{
		Target: reviewDoc{Path: page.Path, Content: page.Content},
	}
	for _, image := range page.Images {
		payload.Images = append(payload.Images, reviewDoc{Path: image.Path, Content: image.Summary})
	}
	for _, depPath := range page.DependencyPaths {
		depPage, ok := byPath[depPath]
		if !ok {
			return reviewPayload{}, fmt.Errorf("unknown dependency path %q for %s", depPath, page.Path)
		}
		payload.Dependencies = append(payload.Dependencies, reviewDoc{Path: depPage.Path, Content: depPage.Content})
	}
	return payload, nil
}

// pagesByPath indexes pages by their path for both membership checks and lookup,
// so callers build it once instead of rebuilding per page.
func pagesByPath(pages []docPage) map[string]docPage {
	byPath := make(map[string]docPage, len(pages))
	for _, page := range pages {
		byPath[page.Path] = page
	}
	return byPath
}

func fillDefaults(config DependencyConfig) DependencyConfig {
	defaults := DefaultDependencyConfig()
	if config.VisionModel == "" {
		config.VisionModel = defaults.VisionModel
	}
	if config.ReviewModel == "" {
		config.ReviewModel = defaults.ReviewModel
	}
	return config
}

const dependencyResolverPrompt = `your task is to compute the dependencies of the following page content
you can pick up to a maximum of 5 pages to return as dependencies to this page
a dependency is another page that is required to understand the context of this page and review its content

only include pages required to interpret, validate, or resolve an explicit reference in this page
do not include pages that are merely part of the same site or hypothetically useful
do not include images

you will receive:
1. a serialized page with its content and transcribed images
2. all pages serialized and summarized into a condensed form
`
