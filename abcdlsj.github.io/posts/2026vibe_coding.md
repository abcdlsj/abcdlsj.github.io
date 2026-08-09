---
title: "近日部分 Vibe Coding 项目"
date: 2026-02-01T22:00:49+08:00
tags:
  - Summary
published: true
toc: true
summary: "Vibe Coding 太好玩辣！"
languages:
    - cn
---

## `Emby TUI` 客户端

> 代码地址 https://github.com/abcdlsj/ember

`Mac` 端目前好用的 `Emby` 客户端都是收费，加上日常使用终端想要试试终端的 `Emby`，于是 `Vibe` 了一个 `TUI` 版本的。播放使用 `MPV`，支持整季连播和字幕。

截图如下：
<img alt="ember-2026-02-01-home" src="/static/img/ember-2026-02-01-home.png" width="100%" style="border: 1px solid gray;">

<img alt="ember-2026-02-01-ping-server" src="/static/img/ember-2026-02-01-ping-server.png" width="100%" style="border: 1px solid gray;">

各种 `Features`：
1. 支持海报展示，用的 `Go Chafa` 库（没用原生 `Kitty` 协议，我不喜欢 `Kitty` 终端）。
2. 支持多服务器管理以及服务器组测速，以及同前缀服务器用相同配置用以负载均衡。

主要使用 `Claude Opus 4.5` 做初始化项目，用 `GLM 4.7` 以及 `MiniMax M2.1` 免费模型进行各种修正和调优。

## `Gnar` 重构

> `Gnar` 是一个内网穿透的 `CLI`，类似 `frp`/`ngrok`，可以看这篇博文 [A tunnel proxy like frp/ngrok](https://blog.songjian.li/posts/gnar-build-a-proxy.html)

<img alt="gnar-2026-08-09" src="/static/img/gnar-2026-08-09.png" width="100%" style="border: 1px solid gray;">

`Gnar` 之前迭代过很多功能，但是从易用性上还是不够「直接」，好的 `UX` 设计应该是将用户路径化为最简。于是最近用 `Rust` 重构了一波，最终使用只需要 `$ gnar` 一个命令就行，会自动检测当前本地的服务。

用的 `GPT-5.6 Sol` 全程开发。

## Gump 一个 `Agent Kanban` 任务管理 `TUI`

> 地址 https://github.com/abcdlsj/gump

<img alt="gump-2026-02-01" src="/static/img/gump-2026-02-01.png" width="100%" style="border: 1px solid gray;">

1. 使用 `Tmux` 进行任务管理，一个 `Agent` 一个 `Tmux Session`。
2. `Git Worktree` 管理分支。

基本使用 `Claude Opus 4.5` 实现，免费模型能力有限改动比较少。

## 个人博客前端重构

优化了下页面展示，各种 `CSS` 以及字体，完全使用 `Kimi K2.5` 模型。

## 总结

`Kimi K2.5` 目前前端审美在第一梯队，`OpenCode` 配合 `Agent-Browser` 可以做到很好的进行前端开发。

项目初始化搭建最好使用 `Claude Opus 4.5`，后续调优可以换其它第一梯队的模型，例如 `Kimi K2.5`、`MiniMax M2.1`、`GLM 4.7` 等。

> 26 年 8 月初更新：`GPT-5.6 Sol` 很强，`DeepSeek-V4-Flash-0731` 更让人惊喜。
