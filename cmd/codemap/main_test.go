package main

import (
	"io"
	"testing"
)

// run wires Extract to Text; the smoke test confirms it works end to end against
// the real source. Extract and Text are unit-tested in internal/codemap.
func TestRun(t *testing.T) {
	if err := run(io.Discard, "../../internal", "../../cmd"); err != nil {
		t.Fatal(err)
	}
}
