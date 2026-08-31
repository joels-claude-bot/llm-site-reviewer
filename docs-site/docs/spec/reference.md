---
title: Categorisation
---

# Category reference

The failure taxonomy behind [Reader failures](/problem/reader-failures): every kind of defect the
reviewer can report

> Failure taxonomy is very 🤖 - ELI5 is more like "how do we classify" this. E.g. alarm broken taxonomy, just classifying it into "failure mode": "loss of function", "detection method": "user observed" etc

Categories are simply a fixed name for a type of error flagged by the review of a document.

We also *use deterministic checks* (`Mechanical`) where we can - as its more reliable that using an LLM model - e.g. for internal broken links with markdown

This leads to a fundamental categorisation of problems:
- `Mechanical`, from deterministic scripts (e.g. "does this file exist,  yes or no => pass/fail")
- `Cognitive`, from a human/model (e.g. "does this chart look rendered properly => pass/fail")


## The full list

Every category, its example, and how it is checked lives on one page — the
**[category table](/spec/categories)**.

That page is generated straight from `internal/finding.Catalog` in the Go source, so it can never
drift from the code. This page is the human explanation; that page is the machine-readable list.



