---
title: "Pipe - Build a proxy"
date: 2023-09-04T18:47:37+08:00
tags:
  - proxy
hide: true
---

## Background
`frp` or `ngrok` is very useful when you want to expose your local service to the internet. And I want to build a proxy tool like them. (as the word I said, for learning and fun)
It's spend me some of time to write, and I had learned some things about `Proxy`.  I didn't refer to too much other code in the process of writing, so I made a lot of mistakes.

> Btw, at these days, I saw [ekzhang/bore](https://github.com/ekzhang/bore). the code is amazing, so I decided to try rust again (the `n`th time)
> The `Rust` version is in progress, after I finish it, I will write a blog about it.

If you had `Go` experience, I appreciate you to read the `Go` version [abcdlsj/pipe](https//github.com/abcdlsj/pipe) first. It's easy enough to understand.

Let's start.

## What is a proxy

```d2 theme=104
start: {
  server {
    server start pipe (listen on port 8910)
  }
  client {
    client start pipe (connect to server on port 8910)
  }

  client -> server: request new forward, client local port 3000 -> server port 9000
  server -> client: accept forward, listening on port 9000
}

user-conn: {
  user -> server: view port 9000
  server -> client: send user-conn request to client (with the first start connection), with connection UUID
  client -> client: receive user-conn request with connection UUID, start a local connection to forward local port 3000
  client -> server: create a new connection to server, send connection request with connection UUID
  client -> client: stream IO.copy between local connection and new remote connection
  server -> server: receive connection request with connection UUID, start streaming IO.copy between user-conn and this connection.
}
```
