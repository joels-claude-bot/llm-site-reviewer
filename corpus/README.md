# Corpus

The executable spec for the reviewer. Each file is a tiny page that plants exactly one defect, with
the result we expect written into its frontmatter. The reviewer runs against this corpus; the test
checks that what it found matches what each page says it should.

This lives outside `docs-site/` on purpose. These pages genuinely break (dead links, missing images,
pages that say "not found"), so they must never enter the strict docs build.

## Conventions

One defect per page. The page is clean apart from that one planted problem, so a miss points at a
single cause.

The expectation lives in frontmatter, not a separate file, so the page and its answer cannot drift:

```md
---
title: Booking guide
expect:
  - category: BROKEN_INTERNAL_LINK
    where: "the 'setup guide' link"
    result: blocking
note: "Links to /setup, which has no page in this corpus."
---
```

Two sides. A page named `ok-*` is a look-alike: something that looks suspicious but is fine. Its
`expect` is empty, and the reviewer must report nothing. These guard against noise.

`result` is `blocking` (a provable defect that can fail the build) or `report` (a judgment call that
is surfaced but does not gate).

## Notes per category

Most link checks are provable and need no model. Two need explaining:

- External links: the HTTP check is stubbed in tests, so the suite never depends on the live network.
- Orphaned pages: detected from the link graph. `index.md` links every page except the orphan, which
  is how "nothing points to it" is defined.
