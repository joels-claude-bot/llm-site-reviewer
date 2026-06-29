---
title: Approach
---

# Approach

The [spec](/spec/reference) says *what* to catch and, for each failure, *how* it can be proven.
Design is the next step down: *how* we actually catch it. It stops short of code.

```mermaid
flowchart TD
    L1["L1 · Spec — what to catch<br/>each failure tagged with how it is checkable"]
    L2["L2 · The rule — one engine per question"]
    L3["L3 · By category — the approach for each failure type"]
    L4["L4 · Code — exit codes, output, CI wiring"]
    L1 --> L2 --> L3 -.->|deferred| L4
    classDef now fill:#52be80,color:#0f172a,stroke:#196f3d
    classDef later fill:#d5d8dc,color:#566573,stroke:#909497,stroke-dasharray:5 4
    class L1,L2,L3 now
    class L4 later
```

Design covers the green levels. **L4 — the code, the exit-code contract, output formats — is
deliberately out of scope here.** Those are decisions for when the tool is built; they are parked in
the [appendix](/appendix/exit-codes).

## L2 · One engine per question

The whole approach is one move: **send each question to the cheapest engine that can answer it
reliably.** A model is never asked something a tool can prove for free.

```mermaid
flowchart LR
    Q[A failure from the spec] --> D{Can a tool<br/>prove it exactly?}
    D -->|yes| LINT["Linter — free, exact, repeatable<br/>(lychee, AST checks)"]:::det
    D -->|no| J{Is the evidence<br/>visual?}
    J -->|no| TEXT["Text model<br/>(acronyms, soft-404, claims)"]:::llm
    J -->|yes| VIS["Vision model<br/>(layout, raw diagrams, mismatches)"]:::llm
    classDef det fill:#52be80,color:#0f172a,stroke:#196f3d
    classDef llm fill:#85c1e9,color:#0f172a,stroke:#2471a3
```

This is just the [checkability rule](/spec/reference) the spec records per category, read as an
instruction. Two rules govern how it is applied.

**Text or pixels is not a free choice.** When a question does need a model, the modality is forced
by where the evidence lives. Picking the wrong one is the most expensive mistake available here.

| Question | Right modality | Why the other is wrong |
|---|---|---|
| Is this external link dead? | **Text** | The "gone" signal is in the page's title/heading. Rendering a 404 to pixels for a vision model is slower and dearer for no gain. |
| Does this page render well? | **Pixels** | Layout, contrast, overlap, a broken diagram — none survive a Markdown round-trip. Scraping the page back to text throws away the exact layer under review. |
| Does the evidence match the claim? | **Either** | Text compares page claims and code references; vision is needed when the evidence is a screenshot, chart, or rendered state. |

:::info Why this matters
Tools like Crawl4AI and Firecrawl scrape a rendered page back into clean Markdown for a text model.
Excellent for extracting content — wrong for a *visual* review, because they discard the visuals
that are the whole point of looking.
:::

**Escalate cheapest-first.** Inside a single question, run the cheap deterministic tier first and
only spend the expensive engine on what survives. The external-link check below is the clearest case.

## L3 · The approach, category by category

The spec sorts every failure into four categories. Design meets each one the same way — deterministic
where it can, a model only on the remainder — but the details differ. Subcategory codes (`LIKE_THIS`)
are called out only where they change the approach; the full list is in the
[category reference](/spec/reference).

### Links — does every link resolve?

**Internal** (`BROKEN_INTERNAL_LINK`, `BROKEN_ANCHOR`, `MISSING_IMAGE`, `ORPHANED_PAGE`): mechanical,
**blocking**. [lychee](https://github.com/lycheeverse/lychee) runs against the built site — a route,
anchor, or image resolves or it does not. Zero ambiguity, no model.

**External** (`BROKEN_EXTERNAL_LINK`, `SOFT_404`): the only genuinely uncertain bucket, since we do
not control those servers. Escalate cheapest-first and only reach for a model on the ambiguous
survivors. This is a **text** question — the "gone" signal lives in the title and heading.

```mermaid
graph LR
    A[External URL] --> B{lychee: HTTP status}
    B -->|non-200| DEAD[dead]:::flag
    B -->|200 OK| C{title/heading regex}
    C -->|matches a 'gone' phrase| SOFT[soft-404]:::flag
    C -->|ambiguous| D{text model}
    D -->|gone| MODEL[soft-404]:::flag
    D -->|alive| OK[pass]:::ok
    classDef flag fill:#e74c3c,color:#fff,stroke:#922b21
    classDef ok fill:#52be80,color:#145a32,stroke:#196f3d
```

### Rendering — does it look right on screen?

All screenshot review, all **reported** rather than blocking — judgment, not proof. This is the pass
the project exists for: a page can compile and still render wrong.

- **Headful Playwright drives the real page** — scroll, click, expand — and screenshots each state. A
  real visible browser is the only reliable way to confirm a component rendered rather than merely
  compiled; lazy-loaded and interactive elements do not exist until something drives them.
- **Segmented capture** beats one giant full-page image: scroll by viewport height and capture each
  frame, so every screenshot is a normal aspect ratio the vision model reads well.

:::warning Open decision — absolute vs. regression
Score a page's layout cold (**absolute**) or flag only what changed against a known-good render
(**regression**)? Absolute is simpler to start — no baselines — but noisier. Regression needs
baseline management but gives a far cleaner signal. **Leaning absolute first**, since no baselines
exist mid-migration, then adding a snapshot gate once the site stabilises.
:::

### Clarity — can a newcomer follow it?

Text review, **reported**. The deterministic tier does the *finding* and the model does the
*judging*: AST parsing extracts the candidates — acronyms, first-use positions, code references —
and the text model decides which are actually unexplained or assume too much
(`ACRONYM_UNEXPANDED`, `ASSUMED_JARGON`, `MISSING_WHAT_WHY`, `SIMPLER`).

### Mismatch — does it contradict itself or the evidence?

This category splits by where the evidence lives:

- `STALE_REF` — **mechanical, blocking.** Grep the repo for the named symbol; if `process_frame()`
  is documented but absent from the code, that is provable.
- `WRONG_CLAIM`, `CONTRADICTS_PAGE`, `INCONSISTENT_TERM`, `BREAKS_STANDARD` — **text model,
  reported.** Comparing claims across pages and against stated standards needs judgment.
- `IMAGE_MISMATCH` — **vision, reported.** The booking card showing 12 July 2026 for a June request
  only exists in pixels.

:::tip How these run
The four categories collapse into a few **passes** at runtime — one lychee run, one browser session
shared by the visual checks, one text-model pass over the remainder — because that is the cheapest
way to execute them. But the unit of *design* is the category, not the pass.
:::
