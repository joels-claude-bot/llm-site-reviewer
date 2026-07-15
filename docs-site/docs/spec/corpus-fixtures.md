---
title: Corpus fixtures
---

:::danger Auto-generated: do not edit this page
Every fixture below is written by `cmd/corpusdocs` from the `corpus/` fixtures. Hand edits are
overwritten on the next build. To change one, edit the fixture and run `go run ./cmd/corpusdocs`.
:::

# Corpus fixtures

The full source of every fixture in `corpus/`, grouped by category, so you can read the actual markdown behind each category code. For how the corpus is used, see the [corpus overview](/spec/corpus); for the codes, the [reference](/spec/reference).

## Links

### `links/broken-anchor.md`

**Travel notes**

Plants:

- `BROKEN_ANCHOR` (blocking) — the link to #packing-list

Links to #packing-list, but this page has no heading with that anchor.

```md
# Travel notes

Jump to the [packing list](#packing-list) for what to bring.

## What to bring

A light jacket and sunscreen.
```

### `links/broken-external.md`

**Partner booking**

Plants:

- `BROKEN_EXTERNAL_LINK` (blocking) — the partner site link

Links to a dead external URL. The HTTP check is stubbed in tests, so the suite never hits the live network.

```md
# Partner booking

Complete your booking on [our partner site](https://partner-bookings.example/holiday/nce-2026).
```

### `links/broken-internal.md`

**Booking guide**

Plants:

- `BROKEN_INTERNAL_LINK` (blocking) — the 'setup guide' link

Links to /setup, which has no page in this corpus.

```md
# Booking guide

Before you start, read the [setup guide](/setup) to configure your account.
```

### `links/missing-image.md`

**Nice**

Plants:

- `MISSING_IMAGE` (blocking) — the nice-promenade.jpg image

References /img/nice-promenade.jpg, which is not in the corpus.

```md
# Nice

![The promenade at Nice](/img/nice-promenade.jpg)

A sunny spot on the south coast.
```

### `links/ok-intentional-404.md`

**Handling errors**

Look-alike — the reviewer must report nothing.

This page intentionally shows a 'Page not found' example to document error handling. The 404 is the subject, not a defect.

```md
# Handling a 404

When a route is missing, the site shows:

> 404: Page not found

That is expected behaviour, documented here on purpose.
```

### `links/ok-placeholder-url.md`

**API example**

Look-alike — the reviewer must report nothing.

example.com appears inside a code fence as a placeholder, not a real docs link. It must not be flagged.

````md
# Rebooking

To rebook from the command line:

```bash
curl https://example.com/book?dest=NCE
```
````

### `links/orphaned-page.md`

**Old itinerary**

Plants:

- `ORPHANED_PAGE` (blocking) — this page

This page exists but nothing links to it. index.md deliberately omits it, which is how the orphan is defined.

```md
# Old itinerary

A leftover page that no current page links to.
```

### `links/soft-404.md`

**Page not found**

Plants:

- `SOFT_404` (report) — this page's own heading and body

Returns a normal 200, but the heading and body say the page is gone. A link to it should be reported as a soft 404, detected by reading the title and heading.

```md
# Page not found

This guide has moved. The content is no longer available here.
```

