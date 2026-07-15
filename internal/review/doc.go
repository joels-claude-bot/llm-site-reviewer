// Package review is the spine: it runs each pass over a site and collects their
// findings into one report. It is not built yet; this file marks the package
// and holds the map, so the shape shows up in the tree before the code does.
//
// How the pieces fit:
//
//	corpus/*.md ──> corpus.Load ──> []corpus.Fixture     (expected answers, for tests)
//
//	a built site ──> review.Review ──> []finding.Finding ──> FormatReport ──> output
//	                     │
//	                     ├─ links.Check      deterministic, no model
//	                     ├─ render.Check      model: looks at screenshots
//	                     ├─ clarity.Check     model: reads the prose
//	                     └─ mismatch.Check    mixed
//
// Every pass returns finding.Finding values, so review does not care how a pass
// reached its answer. The test side feeds the same passes a known corpus and
// compares their findings against the Fixture answers corpus.Load produced.
//
//arch:spine
package review
