# llm-site-reviewer

Review Markdown documentation for broken links, unclear writing, contradictions, and claims that do not match the supplied source material.

## Requirements

- Go 1.26 or newer
- A `GEMINI_API_KEY` for batch mode
- OpenCode CLI installed and signed in for OpenCode mode

## Run a review

Batch sends the selected files together in one model request:

```bash
GEMINI_API_KEY=<gemini-key> go run . batch examples/offsite
```

Start the local OpenCode API server in one terminal:

```bash
opencode auth login
just opencode-server
```

Then run the review in another terminal:

```bash
go run . opencode \
  -model opencode-go/kimi-k3 \
  -url http://127.0.0.1:4097 \
  -timeout 8m \
  -verbose \
  examples/offsite/research.md
```

OpenCode authentication is provided by the OpenCode CLI's saved credentials. Stop the server with Ctrl+C when you are done.

Install the project with `go install github.com/joels-claude-bot/llm-site-reviewer@latest`. To install the browser used by rendered-page checks, run `go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6201.1 install chromium`.
