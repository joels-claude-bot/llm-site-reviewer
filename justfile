# llm-site-reviewer — task runner
# Ports (see .registered-ports.toml): docs = 9811, 9810 reserved for the tool's static server.

set shell := ["bash", "-uc"]

docs_dir := "docs-site"

# list recipes
default:
    @just --list

# --- docs site (Rspress / MDX) ---

# build docs to docs-site/doc_build (strict dead-link check fails the build)
docs-build:
    cd {{ docs_dir }} && npm run build

# serve the built docs on :9811
docs-preview:
    cd {{ docs_dir }} && npm run preview

# install docs deps
docs-install:
    cd {{ docs_dir }} && npm install

# --- the reviewer (Go) ---

# run the Go test suite (corpus validation, and passes as they land)
test:
    go test ./...

# per-function coverage: the "every function is tested or demonstrated" check.
# Only main() should read 0% (a trivial wrapper around a covered run).
cover:
    @go test ./internal/... ./cmd/... -coverpkg=./internal/...,./cmd/... -coverprofile=/tmp/lsr-cover.out >/dev/null
    @go tool cover -func=/tmp/lsr-cover.out

# regenerate the navigable corpus page from the corpus/ fixtures
gen-corpus:
    go run ./cmd/corpusdocs

# regenerate the category reference page from internal/finding.Catalog
gen-reference:
    go run ./cmd/refdocs

# regenerate every source-derived docs page
gen: gen-corpus gen-reference

# print what the corpus pipeline produced (the glue assertions don't show)
inspect *flags:
    go run ./cmd/inspect {{ flags }}

# architecture map by role, from //arch: tags in the source (real file:line links)
codemap:
    @go run ./cmd/codemap

# --- the reviewer's own passes, dogfooded against these docs ---

# deterministic link + anchor check over the built docs (needs lychee from devenv)
links: docs-build
    lychee --no-progress --include-fragments '{{ docs_dir }}/doc_build/**/*.html'

# prose / acronym lint (needs vale from devenv)
prose:
    vale {{ docs_dir }}/docs || true

# everything a CI gate would run
check: docs-build links

# --- e2e smoke ---

# build + serve + curl-verify the docs actually render
e2e:
    bash scripts/e2e-docs.sh

testie:
    go run ./cmd/testsite -dir testie -port 9699
