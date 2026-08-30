---
title: "ADR 0007: OpenCode review modes and structured output"
---

# ADR 0007: OpenCode review modes and structured output

Status: Accepted (2026-08-30)

## Context

The reviewer needs to compare one Markdown page with related repository content
and return `MdResponse` as JSON Schema-constrained output. A single context dump
has poor recall, while an unrestricted agent can spend minutes repeatedly using
cheap filesystem tools.

Provider behaviour matters as much as model quality. Sending the same schema to
an OpenAI-compatible endpoint does not guarantee that the provider enforces it.
In particular, OpenRouter-backed Qwen runs returned arrays encoded as strings or
no structured result even though the captured request contained the correct
schema. Prompt wording did not repair that transport/provider mismatch.

OpenCode's direct providers do support the `body.format` structured-output path.
The Go implementation generates the schema from `review.MdResponse`; it does not
maintain a second handwritten response schema.

## Decision

Support two OpenCode review shapes:

- `opencode` is the high-recall mode. The model receives read-only `read`, `grep`
  and `glob` tools and decides how to investigate. Prompted call limits reduce
  excess, but OpenCode does not enforce per-tool quotas.
- `controlled` is the bounded mode. It makes exactly three tool-free structured
  model calls: choose search patterns, select evidence pages, and review the
  assembled evidence. Go enforces search, snippet and evidence limits and runs
  filesystem and image work concurrently.

Use an `opencode-go/...` model for the hosted path. OpenCode Go is the subscription
provider; `opencode/...` is the separate pay-as-you-go Zen provider. Do not route
production structured reviews through OpenRouter or the LiteLLM compatibility
proxy until an end-to-end compliance test proves the exact backend works.

Keep Ollama as the recommended integration-test backend because it cannot spend
API credit. A local model is not automatically suitable for production: it must
first demonstrate structured-output compliance and adequate context capacity.

## Getting it working

Start OpenCode with the OpenCode Go credential available to the server process:

```bash
export OPENCODE_GO_API_KEY="$(jq -r '.opencode.key' ~/.local/share/opencode/auth.json)"
opencode serve --hostname 127.0.0.1 --port 4097
```

Run the full agent with a capable OpenCode Go model:

```bash
go run . \
  -mode opencode \
  -model opencode-go/kimi-k3 \
  -opencode-url http://127.0.0.1:4097 \
  -opencode-timeout 8m \
  -opencode-verbose \
  docs/page.md
```

Use `-mode controlled` with the same flags for the bounded three-call path.
Verbose mode records model gaps separately from tool execution. This matters:
the measured searches and reads take milliseconds; nearly all elapsed time is
model inference.

The controlled path resolves images with `ResolveMarkdownImages` and uses the
existing image transcription code. SVG source is sent to the vision model as
text; PNG, JPEG and WebP are sent as image bytes. Transcriptions are cached by
absolute path and SHA-256 content hash, so unchanged images are not recomputed.
Remote HTTP(S) images remain deferred.

The OpenCode integration test is opt-in and refuses non-Ollama model IDs:

```bash
OPENCODE_OLLAMA_INTEGRATION=1 \
OPENCODE_OLLAMA_MODEL=ollama/gpt-oss:20b \
OPENCODE_URL=http://127.0.0.1:4097 \
go test ./internal/opencodereview -run TestOllamaIntegration -v
```

## Measured control runs

All review runs used `10-decision-tree-cap-clean.md` and the same generated
`MdResponse` schema. The known recall probe was the page's `NANC ~29% in 2024`
claim versus `26.8%` in the backtest page.

| Harness and model | Time | Structured output | NANC probe | Result |
|---|---:|---|---|---|
| Full `opencode`, `opencode-go/kimi-k3` | 4m25s | Valid | Found | Five supported findings. |
| `controlled`, `opencode-go/kimi-k3` | 3m59s | Valid | Found | Five supported findings; included a one-off 56.7s transcription of 21 images, now cached. |
| `controlled`, `opencode-go/deepseek-v4-flash` | 3m18s | Valid | Missed | Three supported findings; bounded orchestration reduced tool churn but lost recall on this run. |
| `controlled`, `ollama/qwen3:30b-a3b` | 3m23s to phase-one failure | Invalid | Not reached | Model did not produce the search-pattern schema. |
| `controlled`, `ollama/gpt-oss:20b` | 2m25s to phase-one failure | Invalid | Not reached | Same structured-output failure. |

Kimi's full run searched for NANC but did not read the dedicated backtest file;
the bounded grep snippets still contained enough evidence for the final finding.
This demonstrates why a correct search does not always lead to the same selected
evidence, and why repeated evaluation is needed before treating one run as a
general model score.

An attempted `opencode/gemini-3.6-flash` run did not reach the model because the
Zen workspace had insufficient pay-as-you-go balance. It is not counted as a
quality or compliance result. The distinction is intentional: `opencode/...`
and `opencode-go/...` are different billing providers despite both appearing in
OpenCode's model list.

## Consequences

- Prefer full `opencode` with a capable OpenCode Go model when recall matters
  more than latency.
- Prefer `controlled` when deterministic model-call and filesystem bounds matter.
- Treat image transcription as a cached preprocessing cost, not agent tool work.
- Do not claim that local inference is production-ready from cost alone; the two
  configured local models failed before phase one completed successfully.
- Keep a known-defect probe such as the NANC discrepancy in repeat evaluations,
  and record both content quality and schema compliance.
