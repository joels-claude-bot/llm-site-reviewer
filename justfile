# llm-site-reviewer — task runner
# Ports (see .registered-ports.toml): docs = 9811, 9810 reserved for the tool's static server.

set shell := ["bash", "-uc"]

docs_dir := "docs-site"

# list recipes
default:
    @just --list

# --- docs site (Rspress / MDX) ---
docs-dev:
    cd {{ docs_dir }} && npm run dev

# run the Go test suite
test:
    go test ./...

# regenerate the category table page (markdown) from internal/finding.Catalog
gen-categories:
    go run ./cmd/gencategories -out docs-site/docs/spec/categories.md

# run the dev server for the docs from the testie dir
testie:
    go run ./cmd/testsite -dir testie -port 9699
