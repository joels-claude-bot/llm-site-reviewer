package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMessagesCollectsMarkdownAndImages(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.md", "# A\n")
	writeFile(t, dir, "b.markdown", "# B\n")
	writeFile(t, dir, "c.png", "png bytes")
	writeFile(t, dir, "d.txt", "ignored")
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	entries, err := buildMessages([]string{dir}, false)
	if err != nil {
		t.Fatalf("buildMessages returned error: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("entry count = %d, want 3", len(entries))
	}
	if entries[0].FileName != "a.md" || entries[0].TextContent != "# A\n" {
		t.Fatalf("first entry = %+v, want markdown a.md", entries[0])
	}
	if entries[1].FileName != "b.markdown" || entries[1].TextContent != "# B\n" {
		t.Fatalf("second entry = %+v, want markdown b.markdown", entries[1])
	}
	if entries[2].FileName != "c.png" || string(entries[2].ImageContent) != "png bytes" {
		t.Fatalf("third entry = %+v, want image c.png", entries[2])
	}
}

func TestBuildMessagesRecursiveDescendsIntoSubdirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.md", "# A\n")
	writeFile(t, dir, "b.md", "# B\n")
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	writeFile(t, nested, "deep.md", "# Deep\n")

	// Non-recursive: the nested file is skipped, only the immediate files are read.
	entries, err := buildMessages([]string{dir}, false)
	if err != nil {
		t.Fatalf("buildMessages(recursive=false) returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("non-recursive entry count = %d, want 2 (nested dir ignored)", len(entries))
	}
	for _, entry := range entries {
		if entry.FileName == "deep.md" {
			t.Fatalf("non-recursive mode collected nested file deep.md: %+v", entries)
		}
	}

	// Recursive: the nested file is collected too.
	entries, err = buildMessages([]string{dir}, true)
	if err != nil {
		t.Fatalf("buildMessages(recursive=true) returned error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("recursive entry count = %d, want 3 (nested file collected)", len(entries))
	}
	var foundDeep bool
	for _, entry := range entries {
		if entry.FileName == "deep.md" && entry.TextContent == "# Deep\n" {
			foundDeep = true
		}
	}
	if !foundDeep {
		t.Fatalf("recursive mode did not collect nested file deep.md: %+v", entries)
	}

	// IDs must stay a contiguous 0..n-1 sequence in traversal order.
	for index, entry := range entries {
		if entry.Id != index {
			t.Fatalf("entry %d has Id = %d, want contiguous %d", index, entry.Id, index)
		}
	}
}

func TestBuildJsonSchemaProducesValidObjectSchema(t *testing.T) {
	schemaJSON, err := buildJsonSchema()
	if err != nil {
		t.Fatalf("buildJsonSchema returned error: %v", err)
	}
	if !json.Valid(schemaJSON) {
		t.Fatalf("schema is not valid JSON: %s", schemaJSON)
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("schema type = %v, want object", schema["type"])
	}
	if !strings.Contains(string(schemaJSON), `"results"`) {
		t.Fatalf("schema does not contain results field: %s", schemaJSON)
	}
}

func TestBuildSystemPromptIncludesInputShapeAndCatalog(t *testing.T) {
	prompt, err := BuildSystemPrompt()
	if err != nil {
		t.Fatalf("BuildSystemPrompt returned error: %v", err)
	}

	for _, want := range []string{
		"The goal is to catch documentation",
		`"content"`,
		"BROKEN_EXTERNAL_LINK",
		"OTHER",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt does not contain %q:\n%s", want, prompt)
		}
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}
