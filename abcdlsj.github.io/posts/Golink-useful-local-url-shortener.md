---
title: "Golink - Useful local url shortener"
date: 2023-04-16T22:47:37+08:00
tags:
  - Shortener
  - Template
published: true
toc: false
summary: "Create your personal local URL shortening service."
languages:
    - en
---

> What is `url shortener`?
> Get the Answer: [URL Shortening](https://en.wikipedia.org/wiki/URL_shortening)

<!--more-->

There are many posts about `url shorteners` on the internet, but I wanted to write this post to record my own thoughts.

For a good `url shortener`, it should have a short domain (e.g., `t.ly`, `bit.ly`).

Short domains can be quite expensive.

An alternative is to use a `DNS server` to map a short domain to a long domain.

However, in most cases, this is not necessary for me.

I just want a shortener for my local network, which is exactly what a local `url shortener` does.

## /etc/hosts
> `/etc/hosts` is a file in Linux that can link a domain to an IP address.

For example, I can add the following line to `/etc/hosts`:
```text
127.0.0.1      go
```
This allows me to access `go` in my browser.

When I enter `go` in my browser, it sends a request to `127.0.0.1:80` (you cannot specify a port in `/etc/hosts` because it works like a `DNS server` and can only link a domain to an IP address).

## Code
Writing a url shortener is quite simple: just create an HTTP server that maps a domain to a URL.

You can write one yourself or take a look at my [abcdlsj/golink](https://github.com/abcdlsj/share/tree/master/go/golink) implementation (it uses `SQLite3` and supports `import`/`export` features).

## Run it
Run it as a daemon:
```shell
nohup golink &
```

## View

<img alt="golink screenshot" src="/static/img/golink-screenshot.png" width="100%" style="border: 1px solid gray;">

You can also add a `Chrome` site-search shortcut to make it easier to use.

<img alt="Chrome Site-search setting" src="/static/img/golink-chrome-site-search.png" width="100%" style="border: 1px solid gray;">

## End

This post doesn't include any code samples, as I believe that you can write it yourself for the purpose of learning or practicing a programming language.

The idea is from [tailscale/golink](https://github.com/tailscale/golink/tree/main), which relies on `Tailscale Magic DNS`. You can find more information [here](https://tailscale.com/kb/1081/magicdns/)
