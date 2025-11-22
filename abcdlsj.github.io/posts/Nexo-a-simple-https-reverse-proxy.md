---
title: "Nexo - A Simple HTTPS Reverse Proxy"
date: 2025-05-03T19:00:00+08:00
tags:
  - Reverse Proxy
  - Cert
  - Cloudflare
hide: false
description: "告别 Caddy 和 NPM 的臃肿：分享如何用 Go 语言打造轻量级 HTTPS 反向代理，集成 Cloudflare DNS 和自动证书管理，内存占用不到 10M 的优雅方案。"
languages:
    - cn
changelog: |
  - 2025-05-03: first 1.0 version
---

## Background
> 项目地址：[abcdlsj/nexo](https://github.com/abcdlsj/nexo)

我之前一直在用 `Caddy` 作为证书管理和反向代理服务器。虽然 `Caddy` 很强大，但是它的 `Caddyfile` 配置语法说实话挺反人类的，而且功能太多太复杂了。最让我头疼的是它的 `Cloudflare DNS provider` 是以插件形式提供的，每次都要用 `xcaddy build --with github.com/caddy-dns/cloudflare` 去构建，有点过于麻烦。

不过因为我之前在腾讯云有台 2 核 4G 的机器（买了 3 年），`Caddy` 在上面跑得挺好的，几年前配置过一次就一直能用，也就懒得去折腾了。

但是上个月腾讯云的机器到期了，我换到了一台 0.5 欧元一个月的机器上。这台机器只有 1 核 1G 性能超级差。迁移服务的时候，我本来想着图省事，就不编译 `Caddy` 了，换成了 `Nginx Proxy Manager`。它有个 Web UI，配置很方便，基本上开箱即用。

结果用了才发现，这玩意儿内存占用居然超过 100M！对于一台 1G 内存的小机器来说，这也太夸张了。于是我就想，干脆自己写一个证书管理加反向代理的服务算了。

最开始简单写了反向代理的部分，但是发现最麻烦的是证书管理以及处理各种 `DNS Provider`。

有一天无意间刷到 [go-acme/lego](https://github.com/go-acme/lego) 这个库，于是在 `Cursor` 的帮助下，很快就把基本功能搭建起来了。最近这两周一直在用，不断发现问题，不断优化，现在终于趋于完善了。

## Features

Nexo 的主要特性：
1. 自动管理 HTTPS 证书（申请和续期）
2. 支持通配符证书
3. 用 Cloudflare DNS 验证（不用开 80 端口）
4. 配置简单，一个 YAML 搞定
5. 支持动态更新配置
6. 资源占用极低（对比 Nginx Proxy Manager 的 100M+，Nexo 占用低于 10M）

## Usage

最简单的方式是用 Docker 启动：

```bash
docker run --restart=unless-stopped --name nexo \
  -p 443:443 \
  -v ~/.nexo:/etc/nexo \
  ghcr.io/abcdlsj/nexo:latest
```

当然也可以直接运行二进制：

```bash
sudo nexo server
```

配置文件放在 `~/.nexo/config.yaml`，示例：

```yaml
base_dir: /etc/nexo
cert_dir: /etc/nexo/certs
email: your-email@example.com
cloudflare:
  api_token: your-cloudflare-api-token

# 域名级别配置
wildcards:
  - "*.example.com"    # 表示这个域名使用 wildcard 证书

proxies:
  "api.example.com":
    upstream: http://172.17.0.1:8080
  "blog.example.com":
    upstream: http://172.17.0.1:3000
  "example.com":
    redirect: github.com/yourusername # 会自动带上 https
```

就这么简单，Nexo 会自动：
1. 申请/续期证书
2. 处理 DNS 验证
3. 设置反向代理
4. 监听配置变化并自动重载

## Implementation

整个项目的实现其实很简单，主要是这么几块：

### Certificate Management

证书管理这块用了 `go-acme/lego` 这个库，它对 `Let's Encrypt` 的 ACME 协议支持得很好，而且有现成的 `Cloudflare DNS` 验证实现。

证书的续期和失败重试逻辑很简单：
- 每 24 小时检查一次所有证书
- 如果证书快过期了（30 天内），就自动续期
- 如果申请失败了，会加入重试队列，每小时重试一次

### Performance Optimization

因为是在 1 核 1G 的小机器上跑，所以做了一些针对性优化：

1. 内存控制
- 限制了请求体大小（默认 10MB）
- 限制了请求头大小（默认 1MB）
- 实现了请求体的流式处理，避免大请求占用太多内存

2. 连接优化
- 设置了合理的超时时间（读写 30s，空闲 120s）
- 启用了 `TCP KeepAlive`，保持连接复用
- 对空闲连接做了限制，避免资源浪费
- 针对不同类型的响应设置了合适的 `Cache-Control` 头，减少不必要的请求

3. 证书缓存
- 证书加载后会缓存在内存中
- 只有在证书更新或配置变化时才会重新加载
- 通配符证书优先，可以覆盖多个子域名，减少证书数量

最终在我这台 1 核 1G 的小机器上，`Nexo` 的内存占用稳定低于 10MB（目前没有太多请求的情况下保持在 5MB 左右），比 `Nginx Proxy Manager` 省了 `90%+` 的内存。

<img alt="nexo docker memory" src="/static/img/nexo-docker-memory.png" width="100%" style="border: 1px solid gray;">

PS. 1 核 1G 的机器非常垃圾，如果有高 IO / 高 CPU 的需求，千万别用

<img alt="1h1g memmory usage" src="/static/img/1h1g-memory-usage.png" width="100%" style="border: 1px solid gray;">

上图看到间断的部分，是机器爆炸后我进行的重启...我连 `Memos` 服务存储使用 `Sqlite3 Blob` 都被迫迁移到 `S3` 了，这个之前在 2 核 4G 机器上不存在问题。

## Future Plans

无（对我来说完全够用了

> 没有想到五一还在写代码...