---
title: Review modes and structured output
---

# Review modes and structured output

The reviewer supports two modes: **batch** and **OpenCode**. They share a result contract but use different model integrations.

## Batch

Batch gathers the selected files and sends them in one request through the configured LiteLLM client. It is simple to operate and works best when the corpus fits comfortably in the model context.

## OpenCode

OpenCode lets an agent inspect repository evidence with read-only tools before producing a structured review. Start a local server with `just opencode-server`, then point the review command at it with `-url`.

The command relies on credentials saved by the OpenCode CLI. Sign in with `opencode auth login`; an API key for the CLI's provider is configured through OpenCode's normal authentication flow. The Go client does not need a separate copy of that key when it uses the local server.

Example:

```bash
opencode auth login
go run . opencode \
  -model opencode-go/kimi-k3 \
  -url http://127.0.0.1:4097 \
  -timeout 8m \
  -verbose \
  examples/offsite/research.md
```

The provider must support OpenCode's JSON Schema response format and expose the completed structured output. The `opencode-go/...` provider uses the OpenCode Go subscription; `opencode/...` selects the separate Zen credit route.

## Result contract

Both modes return the same `MdResponse` structure. OpenCode derives a JSON Schema from those Go types and requests schema-constrained output. Batch uses the corresponding schema with its response-format API. A response is accepted only after it parses and validates against the shared shape.

## Evidence and limits

OpenCode's prompt suggests search and read limits, but the agent chooses its own path, so the limits are guidance rather than hard guarantees. Its runtime and token usage depend on the evidence it follows. Batch has a more predictable, single-request flow, while its entire selected corpus shares one context window.

Earlier experiments compared OpenRouter text output, LiteLLM compatibility routes, and local Ollama models. Those results explain why the current OpenCode path uses provider-native structured output.
