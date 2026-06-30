# llm-site-reviewer — task runner
# Ports (see .registered-ports.toml): docs = 9811, 9810 reserved for the tool's static server.

set shell := ["bash", "-uc"]

docs_dir  := "docs-site"
docs_port := "9811"

# list recipes
default:
    @just --list

# --- docs site (Rspress / MDX) ---

# live dev server: hot reload + working search, on :9811
docs-dev: preflight
    cd {{docs_dir}} && npm run dev

# build docs to docs-site/doc_build (strict dead-link check fails the build)
docs-build:
    cd {{docs_dir}} && npm run build

# serve the built docs on :9811
docs-preview:
    cd {{docs_dir}} && npm run preview

# install docs deps
docs-install:
    cd {{docs_dir}} && npm install

# --- the reviewer (Go) ---

# run the Go test suite (corpus validation, and passes as they land)
test:
    go test ./...

# regenerate the navigable corpus page from the corpus/ fixtures
gen-corpus:
    go run ./cmd/corpusdocs

# print what the corpus pipeline produced (the glue assertions don't show)
inspect:
    go run ./cmd/inspect

# generated map: every package with its one-line purpose (expand one with `go doc <path>`)
map:
    @go list -f '{{.ImportPath}}: {{.Doc}}' ./...

# --- the reviewer's own passes, dogfooded against these docs ---

# deterministic link + anchor check over the built docs (needs lychee from devenv)
links: docs-build
    lychee --no-progress --include-fragments '{{docs_dir}}/doc_build/**/*.html'

# prose / acronym lint (needs vale from devenv)
prose:
    vale {{docs_dir}}/docs || true

# everything a CI gate would run
check: docs-build links

# --- e2e smoke ---

# build + serve + curl-verify the docs actually render
e2e:
    bash scripts/e2e-docs.sh

# --- meta ---

# fail loudly if the docs port is already held (prints the holder)
preflight:
    @if lsof -i :{{docs_port}} >/dev/null 2>&1; then echo "port {{docs_port}} in use:"; lsof -i :{{docs_port}}; exit 1; else echo "port {{docs_port}} free"; fi
