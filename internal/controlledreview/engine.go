// Package controlledreview implements a bounded, model-directed review. The
// model chooses what to search for and which evidence to inspect; Go controls
// the amount of filesystem material returned to it.
package controlledreview

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/review"
)

type ImageTranscription struct{ Path, Hash, Transcription string }
type ImageSkip struct {
	Ref    string
	Reason error
}
type SearchInput struct {
	TargetPath, TargetText string
	Images                 []ImageTranscription
	SkippedImages          []ImageSkip
	MaxPatterns            int
}
type SearchResult struct {
	Pattern   string
	Matches   []string
	Truncated bool
}
type EvidenceInput struct {
	TargetPath, TargetText string
	Images                 []ImageTranscription
	Searches               []SearchResult
	MaxFiles               int
}
type EvidenceDocument struct {
	Path, Text    string
	Images        []ImageTranscription
	SkippedImages []ImageSkip
}
type ReviewInput struct {
	TargetPath, TargetText string
	TargetImages           []ImageTranscription
	SkippedTargetImages    []ImageSkip
	Searches               []SearchResult
	Evidence               []EvidenceDocument
}

// PhaseModel is deliberately provider-neutral: an OpenCode, Ollama, or test
// implementation can provide the three structured decisions.
type PhaseModel interface {
	ChooseSearchPatterns(context.Context, SearchInput) ([]string, error)
	SelectEvidence(context.Context, EvidenceInput) ([]string, error)
	Review(context.Context, ReviewInput) (review.MdResponse, error)
}

type Config struct {
	Root, DocsRoot                                                          string
	MaxPatterns, MaxMatchesPerPattern, MaxBytesPerPattern, MaxEvidenceFiles int
	ImageCachePath                                                          string
	ImageWorkers                                                            int
	TraceOutput                                                             io.Writer
}

func (c Config) defaults() Config {
	if c.MaxPatterns <= 0 {
		c.MaxPatterns = 5
	}
	if c.MaxMatchesPerPattern <= 0 {
		c.MaxMatchesPerPattern = 20
	}
	if c.MaxBytesPerPattern <= 0 {
		c.MaxBytesPerPattern = 12 << 10
	}
	if c.MaxEvidenceFiles <= 0 {
		c.MaxEvidenceFiles = 5
	}
	if c.ImageWorkers <= 0 {
		c.ImageWorkers = 4
	}
	return c
}

type Engine struct {
	cfg    Config
	model  PhaseModel
	images *ImageCache
}

func New(cfg Config, model PhaseModel, transcriber ImageTranscriber) (*Engine, error) {
	cfg = cfg.defaults()
	if model == nil {
		return nil, errors.New("controlled review: nil phase model")
	}
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}
	cfg.Root = root
	cache, err := OpenImageCache(cfg.ImageCachePath, transcriber, cfg.ImageWorkers)
	if err != nil {
		return nil, err
	}
	return &Engine{cfg: cfg, model: model, images: cache}, nil
}

func (e *Engine) Run(ctx context.Context, targetPath string) (review.MdResponse, error) {
	phase := time.Now()
	target, err := e.readInsideRoot(targetPath)
	if err != nil {
		return review.MdResponse{}, err
	}
	targetPath, _ = filepath.Abs(targetPath)
	targetImages, targetSkips, err := e.images.ForMarkdownDetailed(ctx, target, targetPath, e.cfg.DocsRoot)
	if err != nil {
		return review.MdResponse{}, err
	}
	e.trace(phase, "target read and image transcription")
	phase = time.Now()
	patterns, err := e.model.ChooseSearchPatterns(ctx, SearchInput{TargetPath: targetPath, TargetText: string(target), Images: targetImages, SkippedImages: targetSkips, MaxPatterns: e.cfg.MaxPatterns})
	if err != nil {
		return review.MdResponse{}, fmt.Errorf("choose search patterns: %w", err)
	}
	patterns = uniqueLimit(patterns, e.cfg.MaxPatterns)
	e.trace(phase, "search planning")
	phase = time.Now()
	searches := e.search(ctx, patterns)
	e.trace(phase, fmt.Sprintf("%d bounded searches", len(patterns)))
	phase = time.Now()
	paths, err := e.model.SelectEvidence(ctx, EvidenceInput{TargetPath: targetPath, TargetText: string(target), Images: targetImages, Searches: searches, MaxFiles: e.cfg.MaxEvidenceFiles})
	if err != nil {
		return review.MdResponse{}, fmt.Errorf("select evidence: %w", err)
	}
	paths = uniqueLimit(paths, e.cfg.MaxEvidenceFiles)
	e.trace(phase, "evidence selection")
	phase = time.Now()
	evidence, err := e.readEvidence(ctx, paths, targetPath)
	if err != nil {
		return review.MdResponse{}, err
	}
	e.trace(phase, fmt.Sprintf("%d evidence reads and image transcriptions", len(paths)))
	phase = time.Now()
	result, err := e.model.Review(ctx, ReviewInput{TargetPath: targetPath, TargetText: string(target), TargetImages: targetImages, SkippedTargetImages: targetSkips, Searches: searches, Evidence: evidence})
	if err != nil {
		return review.MdResponse{}, fmt.Errorf("final review: %w", err)
	}
	if err := ValidateResponse(result, targetPath); err != nil {
		return review.MdResponse{}, err
	}
	e.trace(phase, "final review")
	return result, nil
}

func (e *Engine) trace(start time.Time, action string) {
	if e.cfg.TraceOutput != nil {
		fmt.Fprintf(e.cfg.TraceOutput, "controlled trace: %s completed in %s\n", action, time.Since(start).Round(time.Millisecond))
	}
}

func uniqueLimit(values []string, max int) []string {
	seen := map[string]bool{}
	out := make([]string, 0, min(len(values), max))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
			if len(out) == max {
				break
			}
		}
	}
	return out
}

func (e *Engine) search(ctx context.Context, patterns []string) []SearchResult {
	results := make([]SearchResult, len(patterns))
	var wg sync.WaitGroup
	for i, pattern := range patterns {
		wg.Add(1)
		go func(i int, pattern string) { defer wg.Done(); results[i] = e.searchOne(ctx, pattern) }(i, pattern)
	}
	wg.Wait()
	return results
}

func (e *Engine) searchOne(ctx context.Context, pattern string) SearchResult {
	r := SearchResult{Pattern: pattern}
	re, err := regexp.Compile(pattern)
	if err != nil {
		r.Matches = []string{"invalid pattern: " + err.Error()}
		return r
	}
	used := 0
	errBoundReached := errors.New("search bound reached")
	_ = filepath.WalkDir(e.cfg.Root, func(path string, d os.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		scan := bufio.NewScanner(f)
		line := 0
		bounded := false
		for scan.Scan() {
			line++
			if !re.MatchString(scan.Text()) {
				continue
			}
			rel, _ := filepath.Rel(e.cfg.Root, path)
			match := fmt.Sprintf("%s:%d:%s", filepath.ToSlash(rel), line, scan.Text())
			if len(r.Matches) >= e.cfg.MaxMatchesPerPattern || used+len(match) > e.cfg.MaxBytesPerPattern {
				r.Truncated = true
				bounded = true
				break
			}
			r.Matches = append(r.Matches, match)
			used += len(match)
		}
		_ = f.Close()
		if bounded {
			return errBoundReached
		}
		return nil
	})
	return r
}

func (e *Engine) readInsideRoot(path string) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(e.cfg.Root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path outside review root: %s", path)
	}
	return os.ReadFile(abs)
}

func (e *Engine) readEvidence(ctx context.Context, paths []string, target string) ([]EvidenceDocument, error) {
	type result struct {
		i   int
		d   EvidenceDocument
		err error
	}
	ch := make(chan result, len(paths))
	targetAbs, _ := filepath.Abs(target)
	for i, p := range paths {
		go func(i int, p string) {
			abs := p
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(e.cfg.Root, p)
			}
			abs, _ = filepath.Abs(abs)
			if abs == targetAbs {
				ch <- result{i: i, err: errors.New("target cannot be selected as evidence")}
				return
			}
			b, err := e.readInsideRoot(abs)
			if err != nil {
				ch <- result{i: i, err: err}
				return
			}
			imgs, skips, err := e.images.ForMarkdownDetailed(ctx, b, abs, e.cfg.DocsRoot)
			ch <- result{i: i, d: EvidenceDocument{Path: abs, Text: string(b), Images: imgs, SkippedImages: skips}, err: err}
		}(i, p)
	}
	out := make([]EvidenceDocument, len(paths))
	for range paths {
		r := <-ch
		if r.err != nil {
			return nil, fmt.Errorf("read evidence: %w", r.err)
		}
		out[r.i] = r.d
	}
	return out, nil
}

func ValidateResponse(resp review.MdResponse, target string) error {
	if len(resp.Results) != 1 {
		return fmt.Errorf("controlled review: expected one result, got %d", len(resp.Results))
	}
	item := resp.Results[0]
	if item.Id != 0 {
		return fmt.Errorf("controlled review: result id must be 0")
	}
	if item.Name != filepath.Base(target) && item.Name != target {
		return fmt.Errorf("controlled review: unexpected result name %q", item.Name)
	}
	if item.Success != (len(item.Errors) == 0) {
		return errors.New("controlled review: success disagrees with errors")
	}
	if len(item.Errors) > 5 {
		return errors.New("controlled review: more than five findings")
	}
	for _, finding := range item.Errors {
		if strings.TrimSpace(string(finding.Category)) == "" || strings.TrimSpace(finding.Notes) == "" {
			return errors.New("controlled review: empty finding category or notes")
		}
	}
	return nil
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
