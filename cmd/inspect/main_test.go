package main

import (
	"io"
	"testing"
)

// run dispatches to inspectTree or inspectFile depending on the target. These
// two smoke tests cover both branches (and printFixture underneath), against the
// real corpus, writing to io.Discard to stay quiet.

func TestRunOnDirectory(t *testing.T) {
	if err := run(io.Discard, "../../corpus"); err != nil {
		t.Fatal(err)
	}
}

func TestRunOnSingleFile(t *testing.T) {
	if err := run(io.Discard, "../../corpus/links/broken-internal.md"); err != nil {
		t.Fatal(err)
	}
}
