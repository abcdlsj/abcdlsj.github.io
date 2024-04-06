---
title: "Graceful upgrade - how to implement"
date: 2024-04-06T12:47:37+08:00
tags:
  - Graceful-Upgrade
  - Network
  - WIP
hide: false
wip: true
tocPosition: left-sidebar
---

COMMING SOON...

### Background

之前实现过一个 `proxy` 工具，和 `Ngrok` 和 `Frp` 很类似，用于端口转发（具体如何实现的，可以看这个链接 [Gnar - Build a tunnel tool like frp/ngrok](https://abcdlsj.github.io/posts/gnar-build-a-proxy.html)

工具本身还是「可以用」的，我自己有时候也会使用，但是个人使用上有几个地方其实感觉不是特别好

- 配置项比较多而且麻烦
  - 比如 `server-addr`、`local-port`、`remote-port` 之类的，每个都有 `cmd flag` 标定太臃肿了，可以用更简洁的配置来 `parse`，比如可以用 `gnar -c <server-addr>:tcp(<local-port>:<remote-port):<token>` 这种格式
  - 不变量可以做一些 `global set`，支持 `gnar set --global <key>=<value>` 写入到 `$HOME/.config/gnar` 目录作为默认配置，更进一步，可以类似于 `gitconfig` 配置做到目录级别
- 不支持「平滑升级」，`server` 端如果有版本更新，那么就需要重启 `server` 端才能生效，但是 `client` 端无法连接上 `server` 就会断开，如果实现了优雅升级（graceful upgrade），就不会导致连接断开的情况了

> BTW. 目前 `server` 配置里比如 `token`、`port`、`domain` 等「更新」都会导致 `client` 连接出现问题，所以优雅升级也只是作为 `feature`，不太实用
> 这里主要还是想找实际的例子来实操优雅升级，刚好之前写过这样一个服务，就可以动动手实现下

### What is `graceful upgrade`?

> 代表「升级」是在「不中断」的情况下进行的，放在 `Web Service` 这个语境下就是升级过程中，服务端依旧提供服务，客户端对于服务升级本身无感知

