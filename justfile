# llm-site-reviewer — task runner
# Ports (see .registered-ports.toml): docs = 9811, 9810 reserved for the tool's static server.

set shell := ["bash", "-uc"]

docs_dir := "docs-site"
example_dir := "examples/offsite"

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

# serve the offsite example site
example-site port="9700":
    go run ./cmd/testsite -dir {{ example_dir }} -port {{ port }}

# review the offsite example as one batch request
example-batch:
    go run . -mode batch {{ example_dir }}

# review the offsite example with dependency resolution
example-dependency:
    go run . -mode dependency {{ example_dir }}

# capture one rendered page as overlapping frames
visual-capture url output="generated/visual":
    cd visual && npm run capture -- {{ url }} ../{{ output }}

# deterministic browser smoke test for visual segmentation
visual-test:
    cd visual && npm test

# opt-in visual model smoke test (requires the configured vision endpoint)
visual-model-test:
    cd visual && npm run test:model

# review one or more rendered page URLs with the configured vision model
visual-review *urls:
    cd visual && npm run review -- {{ urls }}
