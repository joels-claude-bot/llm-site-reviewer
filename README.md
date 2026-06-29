# llm-site-reviewer

Automated review of a **rendered** documentation/MDX site. It catches the bugs the build misses:
broken links, raw diagrams, unreadable pages, unexplained terms, stale references, and screenshots
that contradict the surrounding text. Built for the failure mode where **compile success ≠ usable
docs**.

> Extracted from the `review-docs` Claude Code skill so the deterministic passes can run in CI
> without an LLM in the loop, and the AI passes can version independently.

## Core principle

**Verifiable things are never sent to an LLM.** Internal links, anchors, and HTTP status are
deterministic — an LLM there is pure cost and false-negative risk. The LLM is reserved for the
uncertain questions: *is this external page actually dead?* (text), *does my rendered page look
good?* (vision), and *does this explanation or screenshot match the surrounding evidence?*

The hard line is modality, not scope. Turning a rendered page into markdown for a text model
(Crawl4AI / Firecrawl) throws away the exact visual layer we care about, so visual checks must use
pixels. Clarity and mismatch checks can use text, plus screenshots when the claim depends on
visible evidence.

## Passes

| Pass | Engine | Modality | LLM? | Notes |
|---|---|---|---|---|
| Internal links + anchors | [lychee](https://github.com/lycheeverse/lychee) on built `dist/` | route/anchor | **never** | Hard CI gate. Route resolves in built output or it doesn't — zero ambiguity. |
| External validity | lychee → title/`h1` regex → local text model | **text** | last resort | Tiered, cheapest-first. See below. |
| Rendered formatting/layout | headful [Playwright](https://github.com/microsoft/playwright) + viewport-segmented screenshots | **vision** | yes | The reason this repo exists. |
| Clarity and mismatch review | text extraction + AST checks + screenshots where needed | text / vision | yes, after deterministic extraction | Acronyms, assumed jargon, stale refs, factual claims, and screenshot/text mismatches. |

### External link validity — tiered, LLM last

External links are the only genuinely uncertain bucket (we don't control those servers). Run
cheapest-first and only reach for a model on the ambiguous survivors. This is a **text** question
— the "gone" signal lives in the page title/heading — so a vision model is the wrong, expensive
modality here.

1. **lychee** — HTTP status + redirects. Kills the obvious dead links. Free, fast.
2. **Title / `h1` regex** on survivors — `/404|not found|no longer|moved|deprecated/i` catches
   most [soft-404s](https://www.drlinkcheck.com/) (server returns `200 OK` but the body says the
   page is gone) with zero model.
3. **Local text model** — only the links that survive 1 and 2, fed rendered `title` + `h1` +
   first paragraph. Reuse the headful browser session already open for the visual pass.

### Rendered formatting — pixels → vision

- **Headful Playwright directly** drives the stateful walkthrough (scroll, click, expand
   components) and screenshots each state. Headful because it's the only reliable way to know an
   MDX component actually rendered rather than merely compiled.
- **Viewport-segmented capture** solves the wide/`fullPage` mega-screenshot problem: scroll by
   viewport height, capture each frame, feed frames in sequence. Each image is a normal aspect
   ratio the vision model reads well — one giant image is what breaks.
- **Optional deterministic gate underneath**: Playwright `toHaveScreenshot` snapshot/visual
   regression catches *rendering changed vs. last known-good* for free. Then the vision LLM only
   judges "is this changed state good?" — a cheaper relative question on a baseline instead of an
   absolute one.

## Open decision

The formatting pass can run **absolute** ("score this page's layout cold") or **regression-based**
("flag what changed vs. last good render"). Absolute is simpler to start (no baselines) but
noisier; regression needs baseline management but gives a clean CI signal with far fewer false
alarms. Plan: **start absolute** (no baselines exist mid-migration), add the snapshot gate once
the site stabilises.

## What's off-the-shelf vs. the actual IP

Off-the-shelf (wrap, don't rebuild): lychee (link/anchor walking), Playwright (browser driving),
a local model (the dumb judge). **The IP is the orchestration**: the cheapest-first external tier,
the viewport-segmentation, and the formatting-judgment prompts.

## Status

Scaffold only — design captured, nothing built yet.

### Roadmap

- [ ] CLI skeleton with proper exit codes (so CI can gate on it)
- [ ] `check-links` — lychee wrapper over built `dist/`, internal vs. external split
- [ ] External validity tier 2 (title/`h1` regex) + tier 3 (local model) on survivors
- [ ] Headful Playwright walkthrough driver
- [ ] Viewport-segmented screenshot capture
- [ ] Vision pass (absolute formatting judgment) + prompt set
- [ ] Optional `toHaveScreenshot` regression gate
- [ ] Thin `review-docs` skill wrapper that shells out to this CLI

## References

- [lychee](https://github.com/lycheeverse/lychee) · [docs](https://lychee.cli.rs/) · [GitHub Action](https://github.com/lycheeverse/lychee-action)
- [Dr. Link Check — soft-404 content analysis](https://www.drlinkcheck.com/) · [monitoring soft-404s](https://dev.to/jsmanifest/how-to-monitor-any-url-for-dead-links-including-tricky-soft-404s-58m2)
- [Playwright](https://github.com/microsoft/playwright) · [Playwright MCP for AI UI audits](https://dev.to/debs_obrien/automate-your-screenshot-documentation-with-playwright-mcp-3gk4)
