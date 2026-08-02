# llm-site-reviewer

I asked claude to generate some docs or research for me. Half an hour later I'm realising that the content doesn't match the spec, the images don't match their captions and many others.

This repo is my half-baked (learning golang / vibe coded) attempt to create an AI tool which analyses content and decides if it's consistent with itself, checks for contradictions and mistakes


## Installaion
```bash
go install github.com/joels-claude-bot/llm-site-reviewer@latest
```

## Usage
```bash
GEMINI_API_KEY=<gemini_key_here> llm-site-reviewer <target-path>
```


## TODO
- [ ] Add other providers to gemini


