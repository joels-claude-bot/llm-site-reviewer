---
title: Category table
---

:::danger Auto-generated: do not edit this page
Every row below is written by `cmd/refdocs` from `internal/finding.Catalog`. Hand edits are
overwritten on the next build. To change a category, edit the catalog in Go and run `just gen-reference`.
:::

# Category table
The full failure taxonomy. See the [category reference](/spec/reference) for what the columns mean and the checkability rule behind them.
## How checked

| Value | Meaning |
|---------|---------|
| Mechanical | A tool proves the answer exactly; no model is involved. |
| Cognitive Text Review | A text model reads the prose and judges what needs judgment. |
| Cognitive Screenshot Review | A vision model looks at a screenshot of the rendered page. |

## Rendering

| Category | Example | How checked |
|---------|---------|---------|
| `RENDERED_BROKEN` | Mermaid source appears instead of a diagram | Cognitive Screenshot Review |
| `DIAGRAM_SYNTAX_ERROR` | diagram renders an error box | Cognitive Screenshot Review |
| `TRUNCATION` | page cuts off mid-section | Cognitive Screenshot Review |
| `TEXT_LEGIBILITY` | diagram text is too small to read | Cognitive Screenshot Review |
| `FLAT_COLOUR` | page is a grey wall with no visual hierarchy | Cognitive Screenshot Review |
| `WEAK_HIERARCHY` | scan cannot tell what matters most | Cognitive Screenshot Review |
| `HIGH_DENSITY` | huge table or unbroken prose block | Cognitive Screenshot Review |

## Clarity

| Category | Example | How checked |
|---------|---------|---------|
| `ACRONYM_UNEXPANDED` | `GDS` appears with no explanation | Cognitive Text Review |
| `ASSUMED_JARGON` | "send it through the GDS" with no context | Cognitive Text Review |
| `MISSING_WHAT_WHY` | page starts with commands before saying what they are for | Cognitive Text Review |
| `MISSING_DOES_NOT` | page never states what it does not cover | Cognitive Text Review |
| `SIMPLER` | complex prose that should be a table or short example | Cognitive Text Review |

## Mismatch

| Category | Example | How checked |
|---------|---------|---------|
| `STALE_REF` | docs name `process_frame()`, but the repo has no such function | Mechanical |
| `WRONG_CLAIM` | docs say timeout is 30 seconds, code sets 60 | Cognitive Text Review |
| `CONTRADICTS_PAGE` | page A says default on, page B says default off | Cognitive Text Review |
| `INCONSISTENT_TERM` | same thing is called `doc_build`, `build dir`, and `output/` | Cognitive Text Review |
| `BREAKS_STANDARD` | style guide says expand acronyms; page does not | Cognitive Text Review |
| `UNVERIFIABLE` | claim cannot be checked from available context | Cognitive Text Review |
| `IMAGE_MISMATCH` | text asks for June 2026; screenshot shows July 2026 | Cognitive Screenshot Review |

## Look-Alikes To Ignore
These are not defects:

| Looks suspicious | Why to ignore it |
|---------|---------|
| `https://example.com` inside a code block | It is example code, not a docs link |
| `ntfy.sh/your-topic` inside a shell snippet | Placeholder value for a user to replace |
| `# TODO` in an explicit stub section | Known incomplete work, not a new finding |
| acronym defined in a glossary | The reader already has the context |
| intentional 404 page in an error-handling guide | The 404 is the example |


