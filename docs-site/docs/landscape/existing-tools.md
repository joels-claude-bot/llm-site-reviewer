---
title: Existing tools
---

# Existing tools in the space

The reviewer is mostly **orchestration over off-the-shelf tools**. This page records what exists,
what we wrap, and the small slice we actually build. The rule: **wrap, don't rebuild.**

## What we wrap

| Tool | Language | Job here | Why this one |
|---|---|---|---|
| [lychee](https://github.com/lycheeverse/lychee) | Rust | Internal + external link / anchor walking | Checks the **built** site (not raw Markdown source), concurrent, deterministic HTTP status. Has a ready [GitHub Action](https://github.com/lycheeverse/lychee-action). |
| [Playwright](https://github.com/microsoft/playwright) | Node / TS | Headful browser driving + screenshots | Scrolls, clicks tabs, expands accordions, isolates a single `.mermaid` element, and snapshots — none of which `chromium --headless --screenshot` can do (it blindly crops with `--window-size` and hangs on never-idle pages). |
| [Vale](https://vale.sh/) | Go | Prose / acronym / style linting | Deterministic rulesets; fails on unapproved terminology **without hallucinating**. Good first tier for clarity checks. |
| A model via the [`llm` CLI](https://llm.datasette.io/) | — | The "dumb judge" for the two uncertain questions | Swappable provider/model behind one interface — see the swappable-LLM note below. |

> TS = TypeScript.

## What we build (the actual IP)

Off-the-shelf tools answer their own narrow questions well. The value is in **how they're wired
together**:

1. **The cheapest-first external tier** — lychee → title/`h1` regex → model, escalating only on the
   ambiguous survivors. No single tool does this gradient.
2. **Viewport-segmentation** — turning one un-reviewable mega-screenshot into a sequence of
   model-readable frames.
3. **The formatting-judgment prompts** — the per-state questions that turn a screenshot into a
   verdict.
4. **The exit-code orchestration** — making the whole thing a clean [CI gate](/appendix/exit-codes).

:::tip The swappable LLM layer
The model is a **swappable adapter**, not a hardcoded vendor. The original skill used cloud Gemini;
the design calls for a local model. Both sit behind one interface (default: the `llm` CLI), so
"local vs. Gemini vs. anything else" is a config change, not a rewrite. This keeps the
[checkability rule](/spec/reference) — _verifiable things never hit an LLM_ — independent of *which*
LLM.
:::

## Approaches we explicitly reject

- **`chromium --headless --screenshot`** — hangs on never-idle pages (an MDX site keeps a
  single-page-app observer, a search web-worker, and Mermaid animation-frame loops alive, so
  network-idle never arrives) and crops blindly. Playwright replaces it.
- **Markdown round-trip for visual review** (Crawl4AI / Firecrawl) — discards the visual layer that
  is the entire subject of the visual pass. Fine for content extraction, wrong here.
- **One monolithic LLM pass** — the [original skill's](/) approach. Burns tokens on link-counting
  an LLM does *worse* than a linter. The whole redesign is a reaction to this.

## Prior art in this repo's history

- **ADR-002 — Robust Docs Review** proposed exactly this hybrid (lychee + Vale + AST for
  deterministic checks, Playwright + vision for the rest). This tool is that ADR, built.
- The `review-docs` **skill** is the monolith being replaced; its prompts are harvested for the
  visual, clarity, and mismatch passes.

> ADR = Architecture Decision Record.

## References

- lychee · [docs](https://lychee.cli.rs/) · [GitHub Action](https://github.com/lycheeverse/lychee-action)
- [Dr. Link Check — soft-404 content analysis](https://www.drlinkcheck.com/) · [monitoring soft-404s](https://dev.to/jsmanifest/how-to-monitor-any-url-for-dead-links-including-tricky-soft-404s-58m2)
- Playwright · [Playwright MCP for AI UI audits](https://dev.to/debs_obrien/automate-your-screenshot-documentation-with-playwright-mcp-3gk4)
- [Vale](https://vale.sh/) · [`llm` CLI](https://llm.datasette.io/)
