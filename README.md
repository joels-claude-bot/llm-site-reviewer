# llm-site-reviewer

I asked claude to generate some docs or research for me. Half an hour later I'm realising that the content doesn't match the spec, the images don't match their captions and many others.

This repo is my half-baked (learning golang / vibe coded) attempt to create an AI tool which analyses content and decides if it's consistent with itself, checks for contradictions and mistakes


## Installaion
```bash
go install github.com/joels-claude-bot/llm-site-reviewer@latest
go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6201.1 install chromium
```

## Usage
```bash
GEMINI_API_KEY=<gemini_key_here> llm-site-reviewer <target-path>
```

Rendered-page review uses the existing LiteLLM proxy configuration:

```bash
LITELLM_MASTER_KEY=<key> llm-site-reviewer -mode visual https://docs.example.com/
```

Controlled review keeps filesystem work outside the model: the model chooses a
bounded set of searches, Go executes them concurrently, the model selects up to
five evidence pages, and a final structured call reviews the collected material.
Local image transcriptions are cached by absolute path and SHA-256 content hash.
Changed images use the existing LiteLLM vision path, which sends SVG source as
text and raster formats as image bytes.

```bash
go run . \
  -mode controlled \
  -model opencode-go/deepseek-v4-flash \
  -opencode-url http://127.0.0.1:4097 \
  -opencode-timeout 5m \
  -opencode-verbose \
  docs/page.md
```

The OpenCode integration test is skipped by default and refuses paid models.
Run it explicitly against a local Ollama model:

```bash
OPENCODE_OLLAMA_INTEGRATION=1 \
OPENCODE_OLLAMA_MODEL=ollama/qwen3:4b \
OPENCODE_URL=http://127.0.0.1:4096 \
go test ./internal/opencodereview -run TestOllamaIntegration -v
```


## TODO
- [x] integrate the Playwright screenshot pass into the Go review command
- [ ] define bounded fetching and cache identity for remote HTTP(S) Markdown images
