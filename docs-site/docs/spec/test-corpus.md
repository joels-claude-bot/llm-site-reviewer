---
title: Example tests
---

# Example tests

The reviewer is defined by examples.

We keep a tiny docs site with known mistakes in it. The reviewer runs against that site. The test
checks whether it reported the right things. The [worked example](/spec/worked-example) shows one
such page in full, with every defect it plants.

## One Tiny Case

| Tiny docs page | Expected result |
|---|---|
| `broken-links.mdx` links to `/setup` | Flag a broken internal link |
| No `/setup` page exists | The finding can block the run |

:::tip Key point
To add a new rule, add a tiny page that demonstrates it and write down the expected result.
:::

## Every Rule Needs Two Examples

For each problem type, we need both sides:

### 1. Flag the real problem

Examples:

- a normal prose link points to a missing page
- a Mermaid block renders as raw text
- `GDS` appears with no explanation
- a screenshot shows July 2026 when the text asks for June 2026

If the reviewer misses one of these, the test fails.

### 2. Ignore the look-alike

Examples:

- `https://example.com` appears inside a shell snippet
- an acronym is already explained in a glossary
- a 404 page is shown intentionally to document error handling

If the reviewer flags one of these, the test also fails.

:::danger Noise is a bug
A reviewer that flags harmless examples trains authors to ignore it. Testing what to ignore is as
important as testing what to catch.
:::

## Blocking vs Review Findings

Not every finding should stop the run.

| Result | Use for | Example |
|---|---|---|
| **Blocking** | Mechanical facts with one clear answer | `/setup` is linked, but no `/setup` page exists |
| **Review finding** | Judgment calls that need language or screenshot interpretation | the booking card shows July when the text asks for June |

:::tip Key point
Mechanical checks can be strict. Judgment checks should be reported first, not treated like hard
failures before the prompts and examples are stable.
:::

## Seed Examples

These real cases seed the first example set:

| Example | Expected result | Why |
|---|---|---|
| `ntfy.sh/your-topic` inside a code block | Ignore | It is an example command, not a docs link |
| `example.com` inside a shell snippet | Ignore | Reserved example domains are allowed in code |
| `# TODO` stub section | Ignore | Known-incomplete notes should not be re-reported as defects |
| ASCII diagram too small to read | Flag | The page rendered, but the reader cannot use the diagram |
| Six-column table full of long commands | Flag | The page is technically valid but hard to read |
| Unexpanded `GDS` / `PNR` style acronyms | Flag | A newcomer has to guess domain terms |
| Acronyms explained in a glossary | Ignore | The page already gave the reader the needed context |

## Next Step

Build the first tiny example site from the seed examples above. For each example, record:

- the page
- the thing to flag or ignore
- whether the result is **blocking** or **review-only**
