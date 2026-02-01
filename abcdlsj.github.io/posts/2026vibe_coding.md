---
title: "最近 Vibe Coding 项目"
date: 2026-02-01T22:00:49+08:00
tags:
  - Summary
hide: false
toc: true
description: "最近的 Vibe Coding 项目"
languages:
    - cn
---

## Emby CLI 客户端

因为现在懒得自己去找各种资源然后下载，所以都是直接订阅的 Emby 资源库，苦于 MAC 端不少 Emby 客户端都收费，于是 Vibe Coding 了一个。

播放使用 MPV，支持整季连播和字幕。

截图如下：
<img alt="ember-2026-02-01-home" src="/static/img/ember-2026-02-01-home.png" width="100%" style="border: 1px solid gray;">

<img alt="ember-2026-02-01-ping-server" src="/static/img/ember-2026-02-01-ping-server.png" width="100%" style="border: 1px solid gray;">

各种 Features：
1. 支持海报展示，用的 Go Chafa 库（没用原生 Kitty 协议，我不喜欢 Kitty 终端）。
2. 支持多服务器管理以及服务器组测速，以及同前缀服务器用相同配置用以负载均衡。

主要使用 `Claude Opus4.5` 做初始化项目，用 `GLM 4.7` 以及 `MiniMax M2.1` 免费模型进行各种修正和调优。

## Rssy 前端重构

Rssy 是我一个之前写的 RSS 订阅阅读站点，域名 https://rssy.songjian.li。

之前前端页面一直比较朴素，用的 `Water CSS`，我最近使用 `Kimi K2.5` 模型修改了下前端页面。

现在长这样：

首页
<img alt="rssy-2026-02-01-home" src="/static/img/rssy-2026-02-01-home.png" width="100%" style="border: 1px solid gray;">

Stream 各种信息流页面

<img alt="rssy-2026-02-01-stream" src="/static/img/rssy-2026-02-01-stream.png" width="100%" style="border: 1px solid gray;">


使用 `Kimi K2.5` 模型进行前端重构。

## 个人博客前端重构

优化了下页面展示，各种 CSS 以及字体，完全使用 `Kimi K2.5` 模型。

## 总结

`Kimi K2.5` 目前前端审美在第一梯队，Opencode 配合 Agent-Browser 可以做到很好的进行前端开发。

项目初始化搭建最好使用 `Claude Opus4.5`，后续调优可以换其它第一梯队的模型，例如 `Kimi K2.5`、`MiniMax M2.1`、`GLM 4.7` 等。