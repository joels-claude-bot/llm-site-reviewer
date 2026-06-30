package corpus_test

import (
	"fmt"

	"github.com/joels-claude-bot/llm-site-reviewer/internal/corpus"
)

// ExampleParse shows what Parse produces and pins it. While debugging you can
// drop the `// Output:` line and run `go test -run Example -v` to just watch the
// result; once it looks right, paste it back and the example becomes a checked
// test. It also appears in the package docs, so it documents Parse for free.
func ExampleParse() {
	content := []byte(`---
title: Booking guide
expect:
  - category: BROKEN_INTERNAL_LINK
    where: "the 'setup guide' link"
    result: blocking
---
Read the [setup guide](/setup).`)

	fixture, err := corpus.Parse(content)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	want := fixture.Expect[0]
	fmt.Printf("%s: %s (%s)\n", fixture.Title, want.Category, want.Result)
	// Output: Booking guide: BROKEN_INTERNAL_LINK (blocking)
}
