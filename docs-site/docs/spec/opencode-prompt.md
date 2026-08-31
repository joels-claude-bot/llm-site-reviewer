---
title: OpenCode playground prompt
---

# Try the review in OpenCode

Paste the prompt below into an OpenCode session opened at the repository root. Replace only
`/absolute/path/to/repository/docs/target.md` with the page you want to review.

This is the human-driven equivalent of `-mode opencode`: the agent may follow evidence across the
repository, but it is told to remain read-only and keep the investigation bounded.

:::warning Keep the first experiment small
Start with one target page. Tool limits in a chat prompt are guidance rather than hard enforcement,
and a large local model may also need a larger context window than its default.
:::

## Copy-paste prompt

```text
Review /absolute/path/to/repository/docs/target.md for genuine reader-facing content defects.

This is a bounded, read-only review.

You may use only glob, grep, and read. Read the target once. Use at most five focused grep calls and
one glob call for its most distinctive entities, dates, quantities, performance figures, and factual
claims. Combine related terms in each grep. Then stop searching and use at most five additional read
calls on the strongest candidate evidence. Do not re-read files or survey the whole repository.

Use other Markdown files in the repository as evidence. Find pages discussing the same entities,
dates, quantities, performance figures, or factual claims and compare their relevant text with the
target. Do not assume that only explicitly linked pages are relevant.

Report only genuine defects that affect a reader. Useful categories include:

- CONTRADICTS_PAGE: the target conflicts with another repository page covering the same fact, period,
  and scope.
- WRONG_CLAIM: the target makes a claim disproved by repository evidence.
- STALE_REF: the target refers to an old name, state, ownership, API, path, or behaviour.
- INCONSISTENT_TERM: the same concept is named inconsistently in a way that could confuse a reader.
- BREAKS_STANDARD: the target violates an explicit repository documentation standard.
- UNVERIFIABLE: the target presents a material claim as fact but the supplied repository context
  cannot support it.
- ACRONYM_UNEXPANDED or ASSUMED_JARGON: unexplained language blocks the intended reader.
- MISSING_WHAT_WHY or MISSING_DOES_NOT: essential scope or purpose is absent.
- SIMPLER: the content is materially harder to understand than a clearly identifiable simpler form.
- OTHER: a concrete reader-facing defect that does not fit a more specific category.

Do not report stylistic preferences, harmless rounding differences, claims covering different periods
or scopes, placeholder values in examples, intentional examples of failure, or issues based only on
outside knowledge.

Base every finding only on repository content. Do not browse the web, run shell commands, edit files,
create plans, invoke skills or subagents, or assess rendering.

Return only JSON matching this shape, with no prose or Markdown around it:

{
  "results": [
    {
      "id": 0,
      "name": "the exact basename of the target file",
      "success": false,
      "errors": [
        {
          "category": "UPPERCASE_CATEGORY_CODE",
          "notes": "A concrete explanation citing the target location and the repository evidence."
        }
      ]
    }
  ]
}

Return exactly one result. Set id to 0. Set name to the exact basename of the target path. Set success
to false when errors are present and true when the errors array is empty. Return at most five findings,
ordered by importance. Every finding must identify both the questionable target claim and the
repository evidence supporting the finding.
```

## What this does not reproduce

Pasting a prompt does not attach OpenCode's API-level JSON Schema. The CLI's OpenCode mode supplies a
schema generated from the Go response structs and validates the returned object. A playground model
can still wrap the JSON in prose or return the wrong shape.

For repeatable or automated runs, use the CLI instead:

```bash
llm-site-reviewer \
  -mode opencode \
  -model opencode-go/glm-5.3-flash \
  -opencode-url http://127.0.0.1:4097 \
  -opencode-timeout 5m \
  -opencode-verbose \
  /absolute/path/to/repository/docs/target.md
```

See the [category table](/spec/categories) for the full taxonomy and [ADR 0007](/appendix/decisions/0007-review-modes-and-defect-coverage) for provider and structured-output findings.
