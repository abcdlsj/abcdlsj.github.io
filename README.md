# abcdlsj.github.io

`static` site build by `Go`.

## Build

```bash
go run .
```

## Post Frontmatter (New Recommended Schema)

The generator now supports both old and new keys. Recommended:

```yaml
---
title: "Post title"
date: 2026-04-14T10:00:00+08:00
tags: ["go", "blog"]
cover: "/static/img/example.jpg"
summary: "Short summary for list cards and SEO."
menus: ["posts"] # use ["about"] for about page
published: true  # false => hidden
draft: false     # true => WIP section
toc: true        # false => hide table of contents
languages: ["en"]
changelog: |
  2026-04-14: initial
---
```

Compatibility notes:

- `cover`, `thumbnail`, `hero` are treated as cover candidates (in this order).
- `summary`, `description`, `excerpt` are treated as excerpt candidates.
- `published: false` maps to `hide: true`.
- `draft: true` maps to `wip: true`.
- `toc: false` maps to `hideToc: true`.
- `language: "en"` and `languages: ["en"]` are both accepted.
