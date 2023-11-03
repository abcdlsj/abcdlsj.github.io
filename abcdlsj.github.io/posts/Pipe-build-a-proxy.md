---
title: "Pipe - Build a tool like `frp` or `ngrok`"
date: 2023-11-2T18:47:37+08:00
tags:
  - Network
  - Weekly
hide: true
---

## Background
`frp` or `ngrok` is very useful when you want to expose your local service to the internet. And I want to build a proxy tool like them. (as the word I said, for learning and fun)
It's spend me some of time to write, and I had learned some things about `Proxy`.  I didn't refer to too much other code in the process of writing, so I made a lot of mistakes.

> Btw, at these days, I saw [ekzhang/bore](https://github.com/ekzhang/bore). the code is amazing, so I decided to try rust again (the `nth` time)
> The `Rust` version is in progress, after I finish it, I will write a blog about it.(Or maybe never)

If you had `Go` experience, I appreciate you to read the `Go` version [abcdlsj/pipe](https://github.com/abcdlsj/pipe) first. It's easy enough to understand.

Let's start.

## How network forwarder works

```d2 theme=104
start: {
  server {
    svrport 8910
    forwardport 9000
  }
  client {
    localport 3000
    svrport 8910
  }
  user {}

  client -> server: request new forward, client local port 3000 -> server port 9000
  server -> client: accept forward, listening on port 9000
  user -> server: view forward port 9000
  server -> client: start proxy network
}
```
