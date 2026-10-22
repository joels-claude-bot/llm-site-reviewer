---
title: Future work
---

# Future work

A running list of known gaps and planned improvements, kept so intent survives between working sessions. Short by design: each item is a pointer, not a spec.

- **Expand the docs with a flow chart.** A visual of the review pipeline, from input to findings, so the flow reads at a glance rather than only in prose.
- **Add more worked examples.** Broaden coverage beyond the current set.
- **Validate category codes.** Constrain the model's response to the supported category codes (a schema enum, or validation on parse) so a hallucinated code is rejected instead of stored. See the `TODO` in `internal/review/finding.go`.
- **Build the code-tag documentation tool ([ADR 0002](/appendix/decisions/0002-code-tag-marker)).** Generate docs from `//arch:`-style markers in the source. Parked for now as part of a wider project.
