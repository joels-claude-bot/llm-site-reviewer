# llm-site-reviewer — task runner
# Ports (see .registered-ports.toml): docs = 9811, 9810 reserved for the tool's static server.

set shell := ["bash", "-uc"]

docs_dir := "docs-site"
example_dir := "examples/offsite"

# list recipes
default:
    @just --list

# --- docs site (Rspress / MDX) ---
[working-directory: docs_dir]
docs-dev:
    npm install
    npm run dev

# run the Go test suite
test:
    go test ./...

# serve the offsite example site
example-site port="9700":
    go run ./cmd/testsite -dir {{ example_dir }} -port {{ port }}

# run OpenCode's local API server in the foreground
opencode-server port="4097":
    opencode serve --hostname 127.0.0.1 --port {{ port }}

# review the offsite example as one batch request
example-batch:
    go run . batch {{ example_dir }}
