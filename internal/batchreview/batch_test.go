package batchreview

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMessagesCollectsSupportedFilesAndIgnoresOthers(t *testing.T) {
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
	byName := make(map[string]MdEntry, len(entries))
	for _, entry := range entries {
		byName[entry.FileName] = entry
	}
	if got := byName["a.md"].TextContent; got != "# A\n" {
		t.Errorf("a.md content = %q, want %q", got, "# A\n")
	}
	if got := byName["b.markdown"].TextContent; got != "# B\n" {
		t.Errorf("b.markdown content = %q, want %q", got, "# B\n")
	}
	if got := string(byName["c.png"].ImageContent); got != "png bytes" {
		t.Errorf("c.png content = %q, want %q", got, "png bytes")
	}
	for _, excluded := range []string{"d.txt", "deep.md"} {
		if _, ok := byName[excluded]; ok {
			t.Errorf("unexpected file collected: %s", excluded)
		}
	}
	assertUniqueEntryIDs(t, entries)
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

	// The default only reads files directly inside the selected directory.
	entries, err := buildMessages([]string{dir}, false)
	if err != nil {
		t.Fatalf("buildMessages(recursive=false) returned error: %v", err)
	}
	if got := entryNames(entries); len(got) != 2 || !got["a.md"] || !got["b.md"] {
		t.Fatalf("non-recursive files = %v, want a.md and b.md", got)
	}

	// Recursive: the nested file is collected too.
	entries, err = buildMessages([]string{dir}, true)
	if err != nil {
		t.Fatalf("buildMessages(recursive=true) returned error: %v", err)
	}
	if got := entryNames(entries); len(got) != 3 || !got["a.md"] || !got["b.md"] || !got["deep.md"] {
		t.Fatalf("recursive files = %v, want a.md, b.md, and deep.md", got)
	}
	assertUniqueEntryIDs(t, entries)
}

func TestBuildJsonSchemaProducesValidObjectSchema(t *testing.T) {
	schemaJSON, err := buildJsonSchema()
	if err != nil {
		t.Fatalf("buildJsonSchema returned error: %v", err)
	}
	if !json.Valid(schemaJSON) {
		t.Fatalf("schema is not valid JSON: %s", schemaJSON)
	}

	var schema struct {
		Type       string `json:"type"`
		Properties struct {
			Results struct {
				Type  string `json:"type"`
				Items struct {
					Required []string `json:"required"`
				} `json:"items"`
			} `json:"results"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}
	if schema.Type != "object" || schema.Properties.Results.Type != "array" {
		t.Fatalf("response schema must describe an object containing a results array: %s", schemaJSON)
	}
	for _, field := range []string{"id", "name", "success", "errors"} {
		if !contains(schema.Properties.Results.Items.Required, field) {
			t.Errorf("result schema does not require %q: %s", field, schemaJSON)
		}
	}
}

func entryNames(entries []MdEntry) map[string]bool {
	names := make(map[string]bool, len(entries))
	for _, entry := range entries {
		names[entry.FileName] = true
	}
	return names
}

func assertUniqueEntryIDs(t *testing.T, entries []MdEntry) {
	t.Helper()
	ids := make(map[int]bool, len(entries))
	for _, entry := range entries {
		if ids[entry.Id] {
			t.Errorf("duplicate entry id %d", entry.Id)
		}
		ids[entry.Id] = true
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}
