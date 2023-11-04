---
title: "[WIP] Pipe - Build a tunnel tool like `frp` or `ngrok`"
date: 2023-11-02T18:47:37+08:00
tags:
  - Network
  - Weekly
  - Tunnel
hide: false
---

This article has `English` version，please read [Pipe - Build a tunnel tool like `frp` or `ngrok`](/posts/pipe-build-a-proxy-en.html)

## Background

**简单的转发工具**
公司的项目启动需要使用我们内部 `RPC` 框架的 `Agent` 实现，但是作为本地环境，没有 `Agent` 的条件，所以本地启动服务都是使用 `socat` 将本地 `unix:/xxx.sock` 流量转发到 `tcp://agent.xxx.io:5443` 来启动服务的。
类似于：`socat unix:/xxx.sock,fork tcp:agent.xxx.io:5443`

> `socat` 是一个非常瑞士军刀类的工具，非常强大

借助 `io.Copy()` 以及 `net` 包，动手实现了符合需求的转发工具，代码量很少，加上 `flag` 代码不超过 50 行，但是却非常实用，启动速度很快（实测因为 `socat` 会对流量做加解密，所以直接 `copy` 会快一点点）

**远程端口转发**
`frp` 很适用于内网转发，不过功能很多用不上，`ngrok` 则适用于做一些临时暴露本地服务之类的。
因为去年的尝试，所以准备实现一个类似工具，实现过程中没有借鉴太多其它项目的代码，很多地方都是遇到问题再去查，所以写一篇博客记录下是很值得的，这里写一下思路以及核心代码。

> 开始的版本只实现 `TCP` 转发，含有 `Caddy` 来做 `Auto Subdomain https`，代码不到 `1000` 行。后边优化了下，现在支持 `TCP/UDP` 协议，所以本文只涉及 `TCP/UDP` 实现（不过其它协议也大都类似
> 另外顺带一提，`GitHub` 有非常多类似的实现，比如 [ekzhang/bore](https://github.com/ekzhang/bore) 和 [rapiz1/rathole](https://github.com/rapiz1/rathole/)。特别是 `bore`，代码非常优美（差点忍不住再次**入门** Rust

## How it works

实现**远程端口转发**

假设有一个服务器（Server）和一个客户端（Client）。其中，服务器的 IP 可以直接从公网访问，而客户端的 IP 则不行，并且客户端可以访问服务器。

我们希望有一种方法来建立服务器端口和客户端端口之间的关联，将对服务器端口的访问转发到客户端的对应端口，通过公网访问服务器的端口就相当于访问客户端的端口。

**如何实现这个转发？**
首先 Server 端应该和 Client 端进行通信，对于 Server 端的入站请求，将请求和 Client 端进行**绑定**，Client 则对目标端口和 Server 端通信连接进行**绑定**。**绑定** 的意思是对两个连接进行 `io.Copy()`。

假设我们 Server 通信端口是 8910，要将 Client 的 3000 端口穿透到 Server 的 9000 端口。
最后结构差不多就是这样：
```d2
Flow: {
  server: {
    remoteport 9000
  }
  client: {
    localport 3000
  }
  client <-> server: 1. Prepare(handshark, auth, request forward...)
  user -> server: 2. View remote port 9000
  server -> server: 3. io.Copy() control connection 8910 and user connection
  server -> client: 4. Send start proxy request
  client -> client: 5. io.Copy() control connection 8910 and local connection
}
```

这里省略了一些值得思考的东西，比如：
1. Client 端和 Server 端建立连接 `Auth/Handshake` 的部分，如何「鉴权」？
2. Client 端接收到 `Proxy` 请求后，进行 `io.Copy()` 的是哪个连接，Server 端又怎样处理呢？

> 对于 2，如果没有实现过类似的工具可能不太清楚为什么会有这个问题，看了下面详细的流程大概就清楚了（忽略「鉴权」部分

1. 首先启动 Server 端，「监听」 8910 端口
2. 启动 Client, Client 端和 Server 端建立 `Control` 连接，然后发送一条 `Forward` 接口告诉 Server 端将要转发到 9000 端口
3. Server 端从 `Control` 连接接收到 `Forward` 消息，开始对 9000 端口进行「监听」，准备接收来自用户端的请求
> 此时 Server 是否要 `Copy` 用户连接和 `Control` 连接呢？
> 答案是不应该也不能，因为 `Control` 连接还会有来自 Server 或者 Client 的「其它」的流量，例如 `Close`、`Heartbeat` 消息等，这些流量如果直接 `Copy` 到用户连接上，那就会产生问题。
4. 当有新的用户请求到来时，Server 端通过 `Control` 连接发送 `Exchange` 消息，告诉 Client 端：有新的用户连接，准备开始对流量进行 `Copy`
5. Client 端接收到 `Exchange` 消息，建立连接到 `Local 3000` 端口，准备 `Copy` 流量
> Client 端也不能直接 `Copy` `Control` 连接和 `Local 3000` 连接，和 3 是一样的情况

那么也就是我们遇到了「连接复用」的问题，这个问题在对多端流量进行处理的时候很常见，而且因为这里是直接 `io.Copy` 没办法区分流量的不同。
解决这个问题有很多方法：
1. 可以使用连接复用库，例如 [hashicorp/yamux](https://github.com/hashicorp/yamux)，`frp` 默认使用 `yamux`
2. 对报文在应用层自行区分，于此同时 `Copy` 的部分也要做处理（ps. `yamux` 就是对报文做了处理） 
3. 最简单的方法，也是大多数内网转发工具用的方法，就是如果需要 `Copy` 就新建一个连接，简单有效
> 方法 3 可能存在的问题是，某个端口的连接总数是有限的，但是绝大多数情况，都足够用了（只要实现上没有问题，连接有正常 `Close`

方法 1 和 方法 3 是最适合的，而且 `yamux` 接入并不复杂，我选择方法 3 来实现（后续会抽空加上 `yamux` 支持

选择方法 3，因为 Server 端并不能新建通信连接，所以需要告诉 Client 新建连接，因为 Client 会 `Copy` `Local 3000` 流量到这个新建的连接上，所以对于「主分支」的 Server 来说，它需要判断是 `Forward` 还是 `Exchange` 消息，然后如果是 `Exchange`，需要**拿出**用户连接 `Copy` 到此 `Exchange` 消息的连接上。

所以步骤 3，Server 需要保存用户请求，创建对应的 `Connection UUID`，然后带上发送 `Exchange` 消息到 Client
步骤 5，Client 需要接收到 `Exchange` 消息，新建 Server 连接，然后首先发送带上同样 `UUID` 的 `Exchange` 消息到 Server，然后 `Copy` `Local 3000` 流量到此新建的 Server 连接上，于此同时
之后步骤 6. Server 接收到 `Exchange` 消息，通过 `UUID` 取出对应的用户连接，然后 `Copy` 用户连接和此连接上

至此，用户访问 Server 端 9000 端口的「一个连接」的流量已经完成了
流程很简单，那么接下来，我会写一下每个流程的代码实现

## Implementation

### Structure
### handshark & auth
### forward
### exchange

## Feat
### Auto subdomain https
### Deploy at `fly.io`
### `UDP`
## Done
写了下如何实现一个内网转发的小工具，代码本身还有很多可以优化的地方，比如「错误处理」部分，还有支持更多协议。

感谢阅读！
## refs
<https://pandaychen.github.io/2020/01/01/MAGIC-GO-IO-PACKAGE/>
<https://github.com/ekzhang/bore>
<https://github.com/rapiz1/rathole>