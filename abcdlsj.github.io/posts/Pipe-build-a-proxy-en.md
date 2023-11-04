---
title: "Pipe - Build a tunnel tool like `frp` or `ngrok`"
date: 2023-11-02T18:47:37+08:00
tags:
  - Network
  - Weekly
  - Tunnel
hide: true
---

## Background

`frp` or `ngrok` is very useful when you want to expose your local service to the internet. And I want to build a tunnel tool like them. (as the word I said, for learning and fun)
It's spend me some of time to write the tool, and I had learned some things about `tunnel`. I didn't refer to too much other code in the process of writing, so I made a lot of mistakes.
If you had `Go` experience, I appreciate you to read the source code [abcdlsj/pipe](https://github.com/abcdlsj/pipe) first. It's easy enough to understand.

> Btw, There had many of implementation of `this`, [ekzhang/bore](https://github.com/ekzhang/bore) and [rapiz1/rathole](https://github.com/rapiz1/rathole/) are great. Especially `bore`, `Rust` is amazing!.

> These days, I've changed a lot of the code, so it doesn't `look` as simple as it could be. But It's still easy to understand.

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
