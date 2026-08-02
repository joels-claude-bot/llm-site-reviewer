package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestGetCache(t *testing.T) {
	tempDir := t.TempDir()

	// no cache file yet: expect empty result, no error
	caches, err := getCache(tempDir)
	if err != nil {
		t.Fatalf("getCache on empty dir: unexpected error: %v", err)
	}
	if len(caches) > 0 {
		t.Errorf("expected 0 caches for non-existent cache, got %d", len(caches))
	}

	// malformed JSON: expect an error
	cachePath := getCachePath(tempDir)
	if err := os.WriteFile(cachePath, []byte("oh dear! this is not json..."), 0644); err != nil {
		t.Fatalf("failed to write malformed cache file: %v", err)
	}
	if _, err := getCache(tempDir); err == nil {
		t.Error("expected error from malformed JSON in cache, got nil")
	}

	// valid JSON: expect it parsed back into the same slice
	exampleCaches := []DocumentCache{{
		Path:          "/hello",
		Format:        "pls",
		ContentHash:   "abcd",
		Summary:       "a test file",
		Depdendencies: []string{"another-file"},
	}}
	jsonData, err := json.Marshal(exampleCaches)
	if err != nil {
		t.Fatalf("failed to marshal fixture: %v", err)
	}
	if err := os.WriteFile(cachePath, jsonData, 0644); err != nil {
		t.Fatalf("failed to write valid cache file: %v", err)
	}
	caches, err = getCache(tempDir)
	if err != nil {
		t.Fatalf("getCache on valid JSON: unexpected error: %v", err)
	}
	if !reflect.DeepEqual(exampleCaches, caches) {
		t.Errorf("parsed caches do not match fixture:\nwant %v\ngot  %v", exampleCaches, caches)
	}
}
