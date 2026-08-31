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

Later control runs established an important narrower conclusion: OpenRouter can
run the OpenCode agent loop quickly and can return JSON-looking ordinary text.
What failed was OpenCode's API-level structured-output contract, not tool use,
JSON generation in general, the SDK transport, or OpenCode itself.

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

## Whole-site reviews

Do not review an entire documentation site in one growing OpenCode session. That
eventually recreates batch mode: context grows with every page, earlier content
competes with later instructions, and failures become difficult to retry. A
shared session also risks findings from one page leaking into another page's
result.

Instead, use a map-and-reduce shape:

1. Discover and hash the corpus once. Resolve links, extract searchable text and
   transcribe changed images into persistent caches.
2. Build one shared immutable lexical index of pages, headings, named entities,
   dates and quantities.
3. Review each changed page in its own controlled session against a bounded set
   of evidence from that index.
4. Run several page reviews concurrently, subject to provider rate and spend
   limits, and checkpoint each result so interrupted runs can resume.
5. Make one final site-level synthesis call over the compact per-page findings,
   not over all raw Markdown again.

This retains cross-page comparison without creating a single enormous prompt.
Keeping a session per page preserves the useful three-phase context within that
review; keeping one session for all pages does not provide enough cache benefit
to justify its context growth. With four concurrent workers, twenty cached-image
reviews that each take roughly three minutes have a wall-clock floor closer to
fifteen minutes than one hour, before rate limiting.

## Getting it working

Start OpenCode with the OpenCode Go credential available to the server process:

```bash
export OPENCODE_GO_API_KEY="$(jq -r '.opencode.key' ~/.local/share/opencode/auth.json)"
opencode serve --hostname 127.0.0.1 --port 4097
```

Run the full agent with a capable OpenCode Go model:

```bash
go run . opencode \
  -model opencode-go/kimi-k3 \
  -url http://127.0.0.1:4097 \
  -timeout 8m \
  -verbose \
  docs/page.md
```

Use the `controlled` mode with the same flags plus `-root <workspace>` for the
bounded three-call path.
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

## Invocation layers

OpenCode exposes several interfaces that look similar at the terminal but make
different output promises. The experiments used all three below.

| Interface | What it sends | What `json` means | Suitable for automation? |
|---|---|---|---|
| Interactive TUI | An ordinary session message | Prompt wording only | Useful manual control. |
| `opencode run --attach` | An ordinary session message through an existing server | `--format json` serialises lifecycle events; it does **not** constrain the model response | Yes, after extracting the final text event. |
| SDK `session.prompt` without `body.format` | The same kind of ordinary session message | Prompt wording only | Yes, after extracting and validating JSON locally. |
| SDK `session.prompt` with `body.format.type = "json_schema"` | A structured-output request | OpenCode returns `info.structured_output` or a structured error | Preferred when the provider complies. |

`--attach` avoids starting another embedded OpenCode server. The CLI connects to
the already-running server, creates a session there, sends the message, and
streams events until the session stops:

```text
opencode run -> existing server on :4097 -> session -> provider/model
SDK prompt   -> existing server on :4097 -> session -> provider/model
```

The representative CLI control used the full prompt from the
[OpenCode playground page](/spec/opencode-prompt):

```bash
cd /path/to/repository

opencode run \
  --attach http://127.0.0.1:4097 \
  --dir /path/to/repository \
  --model openrouter/qwen/qwen3-coder-next \
  --format json \
  "$PROMPT"
```

The text-output SDK control made the equivalent request, deliberately omitting
`format`:

```ts
await client.session.prompt({
  sessionID,
  directory,
  model: {
    providerID: "openrouter",
    modelID: "qwen/qwen3-coder-next",
  },
  tools: {
    read: true, grep: true, glob: true,
    bash: false, edit: false, write: false,
  },
  parts: [{ type: "text", text: prompt }],
})
```

The production `opencode` mode instead includes a generated schema and reads
`info.structured_output`:

```ts
format: {
  type: "json_schema",
  schema: generatedMdResponseSchema,
  retryCount: 1,
}
```

These are output-protocol differences, not different search engines. The CLI
and SDK both hand the investigation to an OpenCode session. Invoking the CLI
from Go would add process management and event-stream parsing; it does not by
itself make model inference faster.

## Measured control runs

All review runs used `10-decision-tree-cap-clean.md` and the same generated
`MdResponse` schema. The known recall probe was the page's `NANC ~29% in 2024`
claim versus `26.8%` in the backtest page.

| Harness and model | Time | Structured output | NANC probe | Result |
|---|---:|---|---|---|
| Full `opencode`, `opencode-go/kimi-k3` | 4m25s | Valid | Found | Five supported findings. |
| `controlled`, `opencode-go/kimi-k3` | 3m59s | Valid | Found | Five supported findings; included a one-off 56.7s transcription of 21 images, now cached. |
| `controlled`, `opencode-go/deepseek-v4-flash` | 3m18s | Valid | Missed | Three supported findings; bounded orchestration reduced tool churn but lost recall on this run. |
| `controlled`, `ollama/qwen3:30b-a3b` | 3m23s to phase-one failure | Invalid | Not reached | Ran with an undersized 4K runtime context and did not produce the search-pattern schema. |
| `controlled`, `ollama/gpt-oss:20b` | 2m16s to phase-one failure | Invalid | Not reached | The trace preserved a prose answer to text embedded in the target instead of the requested schema. |

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

### OpenRouter text-output controls

These controls used Qwen3 Coder Next through OpenRouter and asked for JSON as
ordinary assistant text. They intentionally omitted `body.format`, so they test
agent-loop performance rather than provider-enforced structured output.

| Invocation | Wall time | Tool calls | Model turns | Last-turn context | Outcome |
|---|---:|---:|---:|---:|---|
| Interactive OpenCode TUI | 22.1s | Not captured | Not captured | About 52K | Returned JSON-looking text; manual baseline. |
| `opencode run --attach`, JSON event stream | 32.2s | 23: 11 reads, 11 greps, 1 glob | 6 | 56,852 | Final text event was directly parseable JSON. |
| TypeScript SDK, no `format` | 42.18s | 30: 23 reads, 6 greps, 1 glob | 7 | 109,936 | Returned a prose preamble plus fenced valid JSON; local fenced-block extraction was required. |

The SDK session was created in 71ms. Its thirty filesystem tool calls consumed
732ms in total. The remaining time was spent in the seven provider/model turns.
The CLI trace showed the same pattern: reads and searches took milliseconds;
model turns dominated wall time.

The SDK run's larger context is not evidence that the SDK duplicates content.
The autonomous agent chose twelve more reads than the CLI run, and every tool
result became part of the continuing session. The reported 109,936 and 56,852
figures are the context for each run's final model turn, not cumulative token
sums across all turns. The difference is explained by the different stochastic
search paths and volume of evidence read.

Both automated runs also exceeded the prompt's requested tool budget. Prompted
limits are advisory in full agent mode. The 32-second CLI run used 23 tools and
the 42-second SDK run used 30; speed did not come from following the requested
five-grep/five-read cap.

The result explains the earlier multi-minute traces. Those runs used different
models and provider routes, where gaps of tens of seconds occurred between tool
events. Qwen3 Coder Next on OpenRouter processed repeated cached tool turns much
faster. The SDK transport was not adding minutes of overhead. Conversely, the
same OpenRouter route still did not populate `structured_output` when
`json_schema` was requested, so this speed result does not reverse the structured
compliance finding.

Ordinary-text JSON is viable as an explicitly weaker fallback: extract a fenced
JSON block or the outermost object, unmarshal it into `MdResponse`, apply the same
identity/category/cardinality validation, and optionally retry once. It must not
be described as provider-enforced structured output. Result quality must also be
evaluated separately: several fast Qwen findings used weak `UNVERIFIABLE`
reasoning or treated supporting evidence as a contradiction.

### Local Ollama diagnosis

The two Ollama failures are not evidence that the model architectures cannot
produce structured JSON. The OpenCode provider advertised a 32K context limit,
but `ollama ps` showed that `gpt-oss:20b` was actually loaded on `desktop-work`
with only 4,096 tokens. The failed response reported exactly 4,096 input tokens.
Once the target, phase instructions and schema exceeded that window, the model
answered a research question embedded inside the target as ordinary Markdown.
OpenCode then correctly returned `StructuredOutputError`.

Ollama's OpenAI-compatible endpoint has no per-request context-size override.
It requires either a larger server default or a model alias created with
`PARAMETER num_ctx 32768`. Even with 4K context, `gpt-oss:20b` took about thirty
seconds to answer a trivial greeting on `desktop-work`; moving to 32K would add
KV-cache memory and prompt-ingestion cost. Treat that host as underpowered for
20B/30B production reviews. It remains useful for small, no-cost transport and
integration canaries.

## Consequences

- Prefer full `opencode` with a capable OpenCode Go model when recall matters
  more than latency.
- Prefer `controlled` when deterministic model-call and filesystem bounds matter.
- Treat image transcription as a cached preprocessing cost, not agent tool work.
- Do not claim that local inference is production-ready from cost alone. The
  current `desktop-work` hardware/runtime is too slow and was configured with a
  context window too small for this review prompt.
- Scale whole-site runs with cached preprocessing, isolated per-page sessions,
  bounded concurrency and resumable results—not one site-wide agent session.
- Keep a known-defect probe such as the NANC discrepancy in repeat evaluations,
  and record both content quality and schema compliance.
- Treat CLI, SDK text output and SDK structured output as separate invocation
  controls. Record tools, model turns, final context and parse behaviour so a
  fast run is not incorrectly attributed to the transport layer.
