---
title: "ADR 0005: Local Vision Models"
---

# ADR 0005 — Vision model for image→text summarisation

**Status:** Proposed (updated 2026-08-09; original 2026-07-24)

**In one line:** to go local without burning cloud tokens, split the pipeline into a *reasoning text model* for content and a *separate vision step* that turns each image into text. This ADR picks that vision model, sorted into the three tiers I'm actually choosing between.

## The three tiers I'm deciding across

Not "which model" — *"which model on which machine"*. There are three buckets, and they map straight onto the hardware question in **dev-setup ADR-010** (`mdx-docs/docs/appendix/adr/010-local-model-hardware.md` — the used-3090-vs-Strix-Halo-vs-Gemini analysis):

1. **Run now** — my **RTX 3060 (12GB)**. Free, already sat here. Ceiling is roughly an 8B vision model at Q4.
2. **Run with a beefier card** — a **used RTX 3090 (24GB, ~£650)** or RTX 5090 (32GB). ~£650–3,400. Opens up the 32B vision class. *This is the tier I want to prove out before spending the money.*
3. **Cloud** — Gemini / Claude via **LiteLLM → OpenRouter**. No hardware, best quality, per-token cost.

:::tip The trial-run trick — de-risk tier 2 before buying
OpenRouter hosts the *exact* model I'd run on a tier-2 card (**Qwen3-VL 32B**). So the LiteLLM proxy isn't just for using cloud frontier models — it lets me call the tier-2 candidate **hosted**, feed it my real corpus, and decide whether it's good enough **before** dropping ~£650+ on a GPU. If Qwen3-VL 32B via OpenRouter isn't good enough, no card will fix that.
:::

## What changed since the first draft

The original table only had `qwen2.5vl` (7b/3b). Since then **Qwen3-VL** shipped (2B / 8B / 32B) and replaced 2.5-VL at the top — better OCR, charts, screenshots and math at every size. That shuffles all three tiers, and notably makes tier 1 less hopeless than I'd written off.

## The experiment so far (RTX 3060, tier 1)

On `desktop-work`, RTX 3060 12GB:

| Model | Result | Fits VRAM | TLDR |
| --- | --- | --- | --- |
| qwen2.5vl:7b | good ✅ | no ❌ | spilled VRAM→RAM, ~5 min per image |
| qwen2.5vl:3b | meh ⚠️ | yes ✅ | <1 min, tracked most text but missed words; OCR not as good as 7b |

Conclusion at the time: the models that *fit* weren't good enough, and the good one didn't fit. **But that was pre-Qwen3-VL.**

## Candidate models, by tier

VRAM figures are Q4 (~½ byte/param + overhead), consistent with ADR-010's "does it fit?" maths.

### Tier 1 — Run now (RTX 3060, 12GB)

| Model | Params | VRAM @ Q4 | Notes |
| --- | --- | --- | --- |
| **Qwen3-VL 8B** ✅ | 8B | ~6 GB | Direct successor to the 2.5vl I tested — and it *fits* (2.5vl:7b didn't). Best tier-1 bet. |
| MiniCPM-V 4.5 | 8B | ~6 GB | OCR specialist; strongest on dense docs / invoices / tables. |
| Qwen3-VL 2B | 2B | ~2 GB | Fast fallback; replaces the "meh" 3b at a smaller footprint. |

**Verdict:** try **Qwen3-VL 8B** before concluding tier 1 is dead. It's a real jump on what I benchmarked. If it still misses on my images, *that's* the signal to move up a tier.

### Tier 2 — Beefier card (RTX 3090 24GB / 5090 32GB)

| Model | Params | VRAM @ Q4 | Notes |
| --- | --- | --- | --- |
| **Qwen3-VL 32B** ⭐ | 32B | ~20 GB | Fits a single 24GB card. The reason to buy a 3090. **Trial it via OpenRouter first.** |
| MiniCPM-V 4.5 (higher quant) | 8B | ~8–12 GB | Run at Q8/full precision with long context + many images per call. |

**Verdict:** the whole tier-2 case rests on **Qwen3-VL 32B**. Prove it hosted on OpenRouter against the corpus; only buy the 3090 if it clears the bar.

### Tier 3 — Cloud (LiteLLM → OpenRouter)

| Model | Notes |
| --- | --- |
| **Gemini (Flash / Pro)** | Current gold standard for multi-image + docs in one shot; the baseline every local option is measured against. |
| Claude (vision) | Strong alternative; also reachable through the same proxy. |
| Qwen3-VL 32B (hosted) | Same weights as tier 2 — the bridge that lets me test tier 2 without hardware. |

**Verdict:** the quality ceiling and the "just works at low volume" option. Per ADR-010, cloud stays cheaper than any box until volume is high — so tier 3 is the default until a local tier proves *good enough* and I'm running it daily.

## Decision

**Proposed.** Ordered plan, cheapest-first:

1. **Now:** benchmark **Qwen3-VL 8B** (and MiniCPM-V 4.5 for OCR-heavy pages) on the 3060 against the real corpus. This is free and directly tests whether tier 1 is viable.
2. **Before buying:** call **Qwen3-VL 32B via OpenRouter** through the existing LiteLLM proxy. If it's good enough → a **used RTX 3090** is justified (breaks even vs Gemini Flash per ADR-010). If not → no GPU purchase, stay on cloud.
3. **Baseline throughout:** keep **Gemini** (tier 3) as the quality yardstick and the low-volume default.

**Open questions**

- Does Qwen3-VL 8B clear my quality bar on the 3060, or is 8B fundamentally short for this task?
- Ollama caveat: newer Qwen 3.5/3.6 vision weren't yet working in Ollama as of ~July 2026 — 3-VL 8B/32B are the Ollama-ready picks; bleeding-edge would need llama.cpp / LM Studio.
- Does hosted Qwen3-VL 32B actually match what a local 24GB card would give (quantisation differences)?
