package main

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"
)

func TestGetMarkdownImages(t *testing.T) {
	content := `
# Travel Research

Proposed arrangements for the Lisbon offsite. See the [specification](spec.md) for the
agreed plan these are meant to match.

## Flights

Our booked flights are below.

![Flight itinerary, London to Lisbon](flights.svg)

Booking notes: fares were loaded in the GDS via our TMC, and the PNR includes an ARNK
segment so it will not be purged. Watch the MCT at LIS: our ETA leaves only a tight
window before the OTA cutoff, and any schedule change will trigger an ADM.

## Hotel

We have reserved rooms at the **Serra da Estrela Mountain Lodge** - get there by driving

## Airport transfer

The transfer route from the airport is shown below.

![Airport transfer route](route.svg)
`
	bytes := []byte(content)
	images, err := getMarkdownImages(bytes)
	if err != nil {
		t.Fatalf("faled to get markdown images %v", err)
	}
	expectedImages := []string{"flights.svg", "route.svg"}
	if !slices.Equal(images, expectedImages) {
		t.Fatalf("did not get expected images %v, got %v", expectedImages, images)
	}
}
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
