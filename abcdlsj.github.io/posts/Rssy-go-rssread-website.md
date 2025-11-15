---
title: "Rssy - A miniflux rss reader alternative"
date: 2024-07-01T22:47:37+08:00
tags:
  - SQLite3
  - Postgres
  - RSS
  - Template
hide: false
hideToc: true
description: "Rssy 是一个 Miniflux 的替代品，使用 Go 语言实现"
---

For a long time, I used [Miniflux](https://miniflux.app/) to read RSS feeds. However, I encountered several issues with it:

1. Miniflux only supports `PostgreSQL`, whereas I typically prefer using `SQLite3` due to its simplicity for self-hosting. Unfortunately, PostgreSQL can be resource-intensive, especially my server only has 2 cores and 4GB of RAM.
   
2. I found that Miniflux offers many features that I don't necessarily need. I mainly need to pull and display RSS feeds on a regular basis and manage subscriptions.

After a challenging night of migrating my local PostgreSQL data to a free PostgreSQL service like `Supabase`, I encountered numerous data errors. This experience prompted me to develop my own RSS reader in `Go`.

The result is [rssy](https://github.com/abcdlsj/rssy), a project built with support for `GitHub OAuth`, `Water CSS`, `Go Template`, `SQLite3`, and `PostgreSQL`.

Here is a screenshot of the project:
![rss simple screenshot](/static/img/rssy-simple-shot.png)

More pictures you can find [here](https://github.com/abcdlsj/rssy/blob/main/README.md)