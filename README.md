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


## TODO
- [x] integrate the Playwright screenshot pass into the Go review command
