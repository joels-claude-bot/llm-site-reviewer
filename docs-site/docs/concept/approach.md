---
title: Approach
---

# Approach

The [spec](/spec/reference) says *what* to catch. This is *how*: one question, cut two ways, each cell
sent to the cheapest engine that can answer it. Code — exit codes, output, CI — is
[deferred](/appendix/exit-codes).

```mermaid
flowchart LR
    Q(["Does it make sense?<br/>consistent · accurate"])
    Q --> C["📄 Content<br/>the source"]
    Q --> R["🖼️ Rendering<br/>the render"]

    C --> C1(["on its own<br/>one page"])
    C --> C2(["in context<br/>whole project"])
    R --> R1(["on its own<br/>one page"])
    R --> R2(["in context<br/>whole project"])

    C1 --> CLARITY["<b>Clarity</b> · model · report<br/>ACRONYM_UNEXPANDED · ASSUMED_JARGON<br/>MISSING_WHAT_WHY · SIMPLER"]:::model
    C2 --> ILINK["<b>Internal links</b> · tool · blocking<br/>BROKEN_INTERNAL_LINK · BROKEN_ANCHOR<br/>MISSING_IMAGE · ORPHANED_PAGE"]:::tool
    C2 --> XLINK["<b>External links</b> · tool→model · report<br/>BROKEN_EXTERNAL_LINK · SOFT_404"]:::mixed
    C2 --> MIS["<b>Mismatch</b> · grep→model<br/>STALE_REF <i>(blocking)</i> · WRONG_CLAIM<br/>CONTRADICTS_PAGE · INCONSISTENT_TERM<br/>BREAKS_STANDARD <i>(report)</i>"]:::mixed
    R1 --> LAYOUT["<b>Layout & legibility</b> · vision · report<br/>RENDERED_BROKEN · DIAGRAM_SYNTAX_ERROR<br/>TRUNCATION · TEXT_LEGIBILITY · FLAT_COLOUR<br/>WEAK_HIERARCHY · HIGH_DENSITY"]:::model
    R2 --> IMG["<b>Image ↔ claim</b> · vision · report<br/>IMAGE_MISMATCH"]:::model

    classDef tool fill:#52be80,color:#0f172a,stroke:#196f3d
    classDef model fill:#85c1e9,color:#0f172a,stroke:#2471a3
    classDef mixed fill:#f7dc6f,color:#0f172a,stroke:#b7950b
```

**🟩 tool** — provable, blocking &nbsp;·&nbsp; **🟦 model** — judgment, reported &nbsp;·&nbsp;
**🟨 tool→model** — tool filters, model judges the survivors.

- **Content vs rendering** — the source an author wrote vs the page a visitor sees. A page can have
  perfect source and still render wrong, so they take different eyes.
- **On its own vs in context** — visible in one page, vs only real against the rest (link graph,
  sibling pages, code). This sets what a pass loads: one page, or the whole project.

Mismatch is the one straddle: mostly content-in-context, but its `IMAGE_MISMATCH` sits in the
rendering cell — the axes follow the *evidence*, and that bucket draws from both.

## When a model is needed, the modality is forced

Not a free choice — picking wrong is the most expensive mistake here.

| Question | Modality | Why the other is wrong |
|---|---|---|
| Is this external link dead? | **Text** | The "gone" signal is in the title/heading. Rendering a 404 to pixels is slower and dearer for no gain. |
| Does this page render well? | **Pixels** | Layout, contrast, a broken diagram — none survive a Markdown round-trip. Scraping back to text discards the layer under review. |
| Does the evidence match the claim? | **Either** | Text for claims and code refs; vision when the evidence is a screenshot or rendered state. |

This is why scrape-to-Markdown tools (Crawl4AI, Firecrawl) are wrong for *visual* review: they discard
the visuals that are the whole point of looking.

## Two design details behind the cells

**External links escalate cheapest-first** — spend the model only on ambiguous survivors:

```mermaid
graph LR
    A[External URL] --> B{lychee: HTTP status}
    B -->|non-200| DEAD[dead]:::flag
    B -->|200 OK| C{title/heading regex}
    C -->|'gone' phrase| SOFT[soft-404]:::flag
    C -->|ambiguous| D{text model}
    D -->|gone| MODEL[soft-404]:::flag
    D -->|alive| OK[pass]:::ok
    classDef flag fill:#e74c3c,color:#fff,stroke:#922b21
    classDef ok fill:#52be80,color:#145a32,stroke:#196f3d
```

**Rendering drives a real browser** — headful Playwright scrolls, clicks and expands so lazy and
interactive elements actually exist, then captures **per-viewport** (not one giant image) so each shot
is an aspect ratio the vision model reads well.

:::warning Open decision — absolute vs. regression
Score layout cold (**absolute**, no baselines, noisier) or flag only what changed against a known-good
render (**regression**, cleaner but needs baseline management)? **Leaning absolute first** while the
site is mid-migration, adding a snapshot gate once it stabilises.
:::

:::tip Design by cell, run by pass
The grid is the unit of *design*, not execution. At runtime the cells collapse into a few passes — one
lychee run, one shared browser session, one text-model pass over the remainder — because that is
cheapest.
:::
