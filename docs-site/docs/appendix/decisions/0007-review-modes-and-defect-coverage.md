---
title: "ADR 0007: Review modes and defect coverage"
---

# ADR 0007: Review modes and defect coverage

Status: Draft (2026-08-29)

This record compares three ways of reviewing the 26 Markdown files in `~/politician-trading-research/docs`.

| Mode | What it does | Model in this run |
|---|---|---|
| Batch | Reviews the corpus in one request. | Gemini 3.6 Flash |
| Dependency | Summarises each page, finds related pages, then reviews the page with those dependencies. | Gemini 2.5 Flash for review; qwen3-vl for the earlier stages |
| opencode | Gives an agent one file at a time and lets it search the rest of the corpus. | Gemini 2.5 Flash |

These runs compare whole systems, not just models: the prompts, context, tools and output limits also differ, and each configuration was run only once.

## Defects used for grading

The required set contains content defects that can be confirmed from the files alone.

| # | File | Defect |
|---|---|---|
| D1 | `raw_backtest_data.md` | Boilerplate is cut off mid-word at lines 148 and 578. |
| D2 | `raw_niche_edge.md` | Boilerplate is cut off mid-word at lines 146, 964, 1096 and 1164. |
| D3 | `raw_europe_clean.md` | Several passages are cut off mid-word, including lines 80, 100 and 122. |
| D4 | `raw_broader_clean.md` | Many mid-file passages are cut off mid-word or mid-sentence, and the file ends halfway through a sentence at line 671. |
| D5 | `raw_europe.md` | Two pasted HTML blocks are cut off mid-tag, and prose is cut off at line 812. |
| D6 | `02-backtest-pelosi-copy.md` | KRUZ's 2024 return is 14.5% at line 20 and about 18% at line 70. |
| D8 | `02-backtest-pelosi-copy.md`, `10-decision-tree-and-uk-landscape.md`, `RESEARCH.md` | NANC's 2024 return is 26.8% in the first file and about 29% in the other two. |
| D12 | `06-data-sources.md` | The Tussell row puts “Paid (commercial)” in the API column and “Free” in the cost column. |
| D13 | `06-data-sources.md` | Open Access UK appears twice in the same table. |
| D14 | `01-us-vs-uk-comparison.md` | PDMR is used several times without being expanded on the page. |
| D15 | `index.md` | The project tree says the numbered documents end at 15, but `16-next-step.md` exists. |
| D18 | `raw_uk_deep_dive.md` | Content is cut off mid-word at lines 92, 225 and 276. |
| D19 | `index.md` | The line 13 link includes an extra `docs/` segment and resolves to a missing page. |
| D20 | `10-decision-tree-and-uk-landscape.md` | The five-column table has a two-cell `Verdict` row at line 94. |

The earlier version of D15 also called “33 diagrams” stale; that was wrong because the directory contains 33 diagrams rendered once as PNG and once as SVG.

## Candidates excluded from grading

These passages deserve review, but the corpus alone does not prove that either statement is false.

| # | Candidate | Why it is not a confirmed defect |
|---|---|---|
| D7 | Pelosi's return is 744.8% at line 31 and about 838% at line 79 of `02-backtest-pelosi-copy.md`. | The first figure is a named Quiver strategy from May 2014 to the present, while the second is a separate estimate for “the past decade.” |
| D9 | The infrastructure pipeline is £530bn in `06-data-sources.md` and £718bn in `05-niche-opportunities-ranked.md`. | The £718bn passage calls itself the latest pipeline, so the figures may come from different editions; the dates or sources should be made explicit. |
| D11 | `13-incumbent-deep-dives.md` says Dragonfly Eye was divested to Dow Jones, while other pages say FiscalNote acquired it in 2023. | An acquisition followed by a divestment is possible, so the missing date or source is the problem rather than a proven contradiction. |

D10, a 24.9% versus 25.0% rounding difference, has been removed; rendering-only D16 and D17 are outside this content review.

## Coverage

✅ means the saved response contains the finding, ❌ means it does not, — means the file was outside that run, and ⏳ means the new fixture has not yet been graded against the saved responses.

| # | Defect | Batch | Dependency | opencode per file |
|---|---|---|---|---|
| D1 | `raw_backtest_data.md` truncation | ✅ | ✅ | ✅ |
| D2 | `raw_niche_edge.md` truncation | ✅ | ✅ | ✅ |
| D3 | `raw_europe_clean.md` truncation | ✅ | ✅ | ✅ |
| D4 | `raw_broader_clean.md` truncation | ✅ | ✅ | ✅ |
| D5 | `raw_europe.md` pasted and truncated HTML | ✅ | — | ✅ |
| D6 | KRUZ 14.5% versus about 18% | ✅ | ✅ | ❌ |
| D8 | NANC 26.8% versus about 29% | ❌ | ✅ | ✅ |
| D12 | Tussell row has conflicting columns | ❌ | ✅ | ❌ |
| D13 | Open Access UK row is duplicated | ❌ | ✅ | ✅ |
| D14 | PDMR is not expanded | ✅ | ✅ | ✅ |
| D15 | Index says documents end at 15 | ❌ | ❌ | ❌ |
| D18 | `raw_uk_deep_dive.md` truncation | ⏳ | ⏳ | ⏳ |
| D19 | Broken relative link in `index.md` | ⏳ | ⏳ | ⏳ |
| D20 | Malformed table row in `10-decision-tree-and-uk-landscape.md` | ⏳ | ⏳ | ⏳ |

The dependency results were previously transcribed incorrectly: `/tmp/dep-output.txt` explicitly reports both D13 and the D9 candidate, at lines 595 and 1265 respectively.

For completeness, opencode found D7, dependency found D9, and none of the saved responses found D11; these candidates do not count towards the required score.

The “283 findings” reported for dependency mode are repeated review items across dependency contexts, not 283 distinct defects.

## Findings

- Among the graded fixtures, batch found every confirmed single-page prose defect except the two table faults and the stale document range.
- Dependency had the best coverage here, but its output included many repeated acronym and unverifiable-link findings.
- Per-file opencode completed the corpus and found D8 and D13, but missed D6 and D12.
- The single all-corpus opencode call ran out of output space after its first review, so one bounded call per file is the viable shape.
- A checkmark records one saved response, not a model's general ability to find that defect.

## Follow-ups

- Run every harness on Gemini 2.5 Flash with the same confirmed defect set.
- Repeat each configuration enough times to measure how stable its recall is.
- Require short, evidence-backed findings with file names and quoted values.
- Give opencode an explicit checklist for truncation, internal consistency, tables, acronyms, stale references and sibling-file comparisons.
- Deduplicate dependency findings before presenting them.

## Performance note

Dependency mode is slower because it makes several calls per document and transcribes images before review; its independent per-file stages can be parallelised and image work can be skipped for a content-only run.
