package main

import (
	"io"
	"testing"
)

// TestInspectTreeRuns is the smoke test for the inspector itself: the dump is
// not asserted, but it must run without error against the real corpus, so it
// cannot quietly break. It writes to io.Discard to stay quiet during the suite.
func TestInspectTreeRuns(t *testing.T) {
	if err := inspectTree(io.Discard, "../../corpus"); err != nil {
		t.Fatal(err)
	}
}
