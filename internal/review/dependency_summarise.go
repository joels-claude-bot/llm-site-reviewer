package review

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/voocel/litellm"
)

// docImage is a page image plus the model's text transcription of it, so
// diagrams can travel as words into stages that only read text.
type docImage struct {
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

// docPage is one documentation page: its raw content, transcribed images, a
// short summary used for cheap cross-page reference, and the dependency paths
// the resolve stage fills in later.
type docPage struct {
	Path            string     `json:"path"`
	Content         string     `json:"content"`
	Images          []docImage `json:"images"`
	Summary         string     `json:"summary"`
	DependencyPaths []string   `json:"dependencyPaths"`
}

const dependencySummaryPrompt = `
Summarise this page in 400 words or less, covering as much detail as you can about it.
It will be used as a reference point for other pages to know if their context is dependent on this page.
Thus, conveying the core message is vital, but also as many reference points as possible.
`

const imageSummaryPrompt = "Summarise this image in 200 words or less, covering as much detail as you can about labels and edges."

// summarisePages summarises every markdown page under path (a file or directory)
// into docPages, recursing into itself for subdirectories. The path handed in is
// always read; nested subdirectories are only descended into when recursive is true.
func summarisePages(ctx context.Context, client ChatClient, path, model string, recursive bool) ([]docPage, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %v: %w", path, err)
	}

	if !info.IsDir() {
		page, ok, err := pageFromFile(ctx, client, model, path)
		if err != nil {
			return nil, err
		}
		if ok {
			return []docPage{page}, nil
		}
		return nil, nil
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir path %v %w", path, err)
	}
	var pages []docPage
	for _, entry := range files {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			if !recursive {
				slog.Debug("skipping subdirectory", "entry_name", entry.Name())
				continue
			}
			nested, err := summarisePages(ctx, client, fullPath, model, recursive)
			if err != nil {
				return nil, err
			}
			pages = append(pages, nested...)
			continue
		}
		page, ok, err := pageFromFile(ctx, client, model, fullPath)
		if err != nil {
			return nil, err
		}
		if ok {
			pages = append(pages, page)
		}
	}
	return pages, nil
}

func buildPageSummaryMessages(content string, images []docImage) []litellm.Message {
	var builder strings.Builder
	if len(images) > 0 {
		builder.WriteString("include any relevant details from transcribed images below\n")
	}
	for _, image := range images {
		fmt.Fprintf(&builder, "image %v is summarised as follows\n%s\n", image.Path, image.Summary)
	}
	fmt.Fprintf(&builder, "the page content is:\n%v\n", content)

	return []litellm.Message{
		litellm.System(dependencySummaryPrompt),
		litellm.UserText(builder.String()),
	}
}

// imageKind decides how an image reaches the model: an SVG travels as text (its
// source is readable markup), everything else as raw bytes tagged with a MIME
// type the model can decode.
type imageKind struct {
	asText bool
	mime   string // set only when asText is false
}

func classifyImage(imagePath string) imageKind {
	ext := strings.ToLower(filepath.Ext(imagePath))
	if ext == ".svg" {
		return imageKind{asText: true}
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return imageKind{mime: mimeType}
}

func buildImageSummaryMessages(imagePath string, content []byte) []litellm.Message {
	messages := []litellm.Message{litellm.System(imageSummaryPrompt)}
	kind := classifyImage(imagePath)
	if kind.asText {
		return append(messages, litellm.UserText(string(content)))
	}
	return append(messages, litellm.User(litellm.ImageBlock{Data: content, MIME: kind.mime}))
}

func summariseImage(ctx context.Context, client ChatClient, model string, imagePath string, content []byte) (docImage, error) {
	messages := buildImageSummaryMessages(imagePath, content)
	resp, err := client.Chat(ctx, litellm.Request{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return docImage{}, fmt.Errorf("image summarisation failed for %v: %w", imagePath, err)
	}
	return docImage{Path: imagePath, Summary: resp.Text()}, nil
}

// pageFromFile summarises one file into a docPage - page accepted if ok true.
func pageFromFile(ctx context.Context, client ChatClient, model string, fullPath string) (docPage, bool, error) {
	ext := strings.ToLower(filepath.Ext(fullPath))
	if ext != ".md" && ext != ".markdown" {
		slog.Debug("skipping non-markdown dependency review entry", "entry_name", filepath.Base(fullPath), "extension", ext)
		return docPage{}, false, nil
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return docPage{}, false, fmt.Errorf("failed to read file %v: %w", fullPath, err)
	}

	markdownImages, err := ResolveMarkdownImages(content, fullPath)
	if err != nil {
		return docPage{}, false, fmt.Errorf("failed to get markdown images from %v: %w", fullPath, err)
	}
	slog.Debug("found dependency review images", "path", fullPath, "image_count", len(markdownImages))

	var images []docImage
	for _, markdownImage := range markdownImages {
		imageSummary, err := summariseImage(ctx, client, model, markdownImage.Resolved, markdownImage.Content)
		if err != nil {
			return docPage{}, false, fmt.Errorf("failed to summarise image for %v: %w", markdownImage.Resolved, err)
		}
		images = append(images, imageSummary)
	}

	messages := buildPageSummaryMessages(string(content), images)
	resp, err := client.Chat(ctx, litellm.Request{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return docPage{}, false, fmt.Errorf("summarisation for entry %v failed: %w", fullPath, err)
	}

	return docPage{
		Path:    fullPath,
		Content: string(content),
		Images:  images,
		Summary: resp.Text(),
	}, true, nil
}
