---
title: Corpus
---

# Corpus

This page is generated from the fixtures in `corpus/`. Each fixture is a small page that
plants one defect, with the result we expect written into its own frontmatter. The reviewer runs
against these; the test checks that what it finds matches what each page says it should. See the
[test corpus](/spec/test-corpus) for how that works and the [reference](/spec/reference) for the
category codes.

Do not edit this page by hand. Change a fixture and regenerate with `go run ./cmd/corpusdocs`.

## Links

These pages each plant one defect the reviewer must flag.

| Fixture | Category | Result | What is planted |
|---|---|---|---|
| `links/broken-anchor.md` | `BROKEN_ANCHOR` | blocking | Links to #packing-list, but this page has no heading with that anchor. |
| `links/broken-external.md` | `BROKEN_EXTERNAL_LINK` | blocking | Links to a dead external URL. The HTTP check is stubbed in tests, so the suite never hits the live network. |
| `links/broken-internal.md` | `BROKEN_INTERNAL_LINK` | blocking | Links to /setup, which has no page in this corpus. |
| `links/missing-image.md` | `MISSING_IMAGE` | blocking | References /img/nice-promenade.jpg, which is not in the corpus. |
| `links/orphaned-page.md` | `ORPHANED_PAGE` | blocking | This page exists but nothing links to it. index.md deliberately omits it, which is how the orphan is defined. |
| `links/soft-404.md` | `SOFT_404` | report | Returns a normal 200, but the heading and body say the page is gone. A link to it should be reported as a soft 404, detected by reading the title and heading. |

These look suspicious but are fine. The reviewer must report nothing.

| Fixture | Why it is fine |
|---|---|
| `links/ok-intentional-404.md` | This page intentionally shows a 'Page not found' example to document error handling. The 404 is the subject, not a defect. |
| `links/ok-placeholder-url.md` | example.com appears inside a code fence as a placeholder, not a real docs link. It must not be flagged. |
