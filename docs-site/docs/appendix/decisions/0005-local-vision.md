---
title: "ADR 0005: Comparison Local Vision"
---

# ADR 0004: Comparing local vision

Status: Proposed (2026-07-24)

Frontier models like Gemini can handle all I throw at them, multiple docs pages + images.

But to move local and not burn through cloud tokens, the approach needs to move towards a combination of a powerful reasoning text model for content, with a separate step using vision to convert the images to text.

Experimenting with local vision models on `desktop-work` atempting to make use of the rtx 3060 card

| Model | Result | Fits on VRAM | TLDR |
| --- | --- | --- | --- |
| qwen2.5vl:7b | good ✅ | no ❌ | VRAM spilled into RAM, took ~5 mins |
| qwen2.5vl:3b | meh ⚠️ | yes ✅ | OCR not quite as good as 7b |

`3b` ran in <1m on the image and tracked (most) of the text but missed some words



