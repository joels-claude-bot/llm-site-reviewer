---
title: Category reference
---

# Category reference

This is the machine-readable list behind [Reader failures](/problem/reader-failures).

Every category carries two decisions, and together they are the contract the
[design](/concept/approach) has to honour:

- **How checked:** mechanical, text review, or screenshot review.
- **Default result:** blocking finding or review finding.

:::tip The checkability rule
If a tool can prove the answer exactly, a tool proves it — it never goes to a model. A model is
spent only on the categories marked *text review* or *screenshot review*, where the answer needs
judgment. The `How checked` column below is where that decision is recorded for each category.
:::

## Links

| Category | Example | How checked | Default result |
|---|---|---|---|
| `BROKEN_INTERNAL_LINK` | link points to `/setup`, but no `/setup` page exists | Mechanical | Blocking |
| `BROKEN_ANCHOR` | `guide#instal` but the page has `#install` | Mechanical | Blocking |
| `MISSING_IMAGE` | `![](/img/arch.png)` but the image is missing | Mechanical | Blocking |
| `BROKEN_EXTERNAL_LINK` | external page returns 404 | Mechanical | Blocking |
| `SOFT_404` | page returns 200 but title says "Page not found" | Text review | Review |
| `ORPHANED_PAGE` | page exists but no nav/page links to it | Mechanical | Blocking |

## Rendering

| Category | Example | How checked | Default result |
|---|---|---|---|
| `RENDERED_BROKEN` | Mermaid source appears instead of a diagram | Screenshot review | Review |
| `DIAGRAM_SYNTAX_ERROR` | diagram renders an error box | Screenshot review | Review |
| `TRUNCATION` | page cuts off mid-section | Screenshot review | Review |
| `TEXT_LEGIBILITY` | diagram text is too small to read | Screenshot review | Review |
| `FLAT_COLOUR` | page is a grey wall with no visual hierarchy | Screenshot review | Review |
| `WEAK_HIERARCHY` | scan cannot tell what matters most | Screenshot review | Review |
| `HIGH_DENSITY` | huge table or unbroken prose block | Screenshot review | Review |

## Clarity

| Category | Example | How checked | Default result |
|---|---|---|---|
| `ACRONYM_UNEXPANDED` | `GDS` appears with no explanation | Text review | Review |
| `ASSUMED_JARGON` | "send it through the GDS" with no context | Text review | Review |
| `MISSING_WHAT_WHY` | page starts with commands before saying what they are for | Text review | Review |
| `MISSING_DOES_NOT` | page never states what it does not cover | Text review | Review |
| `SIMPLER` | complex prose that should be a table or short example | Text review | Review |

## Mismatch

| Category | Example | How checked | Default result |
|---|---|---|---|
| `STALE_REF` | docs name `process_frame()`, but the repo has no such function | Mechanical | Blocking |
| `WRONG_CLAIM` | docs say timeout is 30 seconds, code sets 60 | Text review | Review |
| `CONTRADICTS_PAGE` | page A says default on, page B says default off | Text review | Review |
| `INCONSISTENT_TERM` | same thing is called `doc_build`, `build dir`, and `output/` | Text review | Review |
| `BREAKS_STANDARD` | style guide says expand acronyms; page does not | Text review | Review |
| `UNVERIFIABLE` | claim cannot be checked from available context | Text review | Review |
| `IMAGE_MISMATCH` | text asks for June 2026; screenshot shows July 2026 | Screenshot review | Review |

## Look-Alikes To Ignore

These are not defects:

| Looks suspicious | Why to ignore it |
|---|---|
| `https://example.com` inside a code block | It is example code, not a docs link |
| `ntfy.sh/your-topic` inside a shell snippet | Placeholder value for a user to replace |
| `# TODO` in an explicit stub section | Known incomplete work, not a new finding |
| acronym defined in a glossary | The reader already has the context |
| intentional 404 page in an error-handling guide | The 404 is the example |

See [Example tests](/spec/test-corpus) for how these become test cases.
