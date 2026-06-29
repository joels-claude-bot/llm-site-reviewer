---
title: Page not found
expect:
  - category: SOFT_404
    where: "this page's own heading and body"
    result: report
note: "Returns a normal 200, but the heading and body say the page is gone. A link to it should be reported as a soft 404, detected by reading the title and heading."
---

# Page not found

This guide has moved. The content is no longer available here.
