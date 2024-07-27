---
title: "Go Template + Tailwindcss = ❤️"
date: 2024-02-24T10:47:37+08:00
tags:
  - TailwindCSS
  - Template
hide: false
hideToc: true
---

> 写很多小项目的时候，我都是使用 `Go` `Template` 实现的，也就是渲染 `HTML` 文件模版，代码比较简单，但是样式的部分却很难满意，因为 `CSS` 实在是难以理解（^^），而 [`Tailwind CSS`](https://tailwindcss.com/) 就是另外一个 `CSS` 框架，它可以用简单的方式实现样式。
> 所以这里介绍下如何将 `Go Template` 和 `Tailwind CSS` 结合使用。

<!-- more -->
## Before Start

> **_NOTE:_** 确保 `Go Template` 的部分已经正常实现

假设项目结构
```
.
├── main.go
├── package.json
├── static
│   └── css
│       ├── main.css
│       └── tailwind.css
├── tailwind.config.js
└── tmpl
    └── index.html
```

## Config
创建 `Tailwind CSS` 必须的文件

1. 初始化 `package.json` 文件
```shell
npm init
```

2. 下载 `Tailwind CSS`
```shell
npm install -D tailwindcss
```

3. 创建 `tailwind.config.js` 文件

```js
module.exports = {
  content: ["./tmpl/*.html"],
  theme: {
    extend: {},
  },
  plugins: [],
};
```

`content` 是代表需要「扫描」并「生成」`CSS` 的 templates，官方 [doc](https://www.tailwindcss.cn/docs/content-configuration)

4. 添加初始化样式 `tailwind.css`
```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

5. 生成 `Tailwind CSS`

这里可以选择通过 `npm script` 来实现，也可以手动运行 `npx tailwindcss -i <source.css> -o <output.css>`
```json
{
  "scripts": {
    "build": "npx tailwindcss -i static/css/tailwind.css -o static/css/main.css",
    "watch": "npx tailwindcss -i static/css/tailwind.css -o static/css/main.css --watch"
  }
}
```

6. `HTML` 添加「生成」的 `CSS` 文件

```html
<link rel="stylesheet" href="static/css/main.css">
```

> **_NOTE:_** 确保文件「相对路径」正确

## Conclusion

这篇写得特别简单，本文内容基本上都可以在 [Xe - How to use Tailwind CSS in your Go programs](https://xeiaso.net/blog/using-tailwind-go/) 找到（知识的剽窃🐛）。

更加复杂的例子，可以看 [memos](https://github.com/abcdlsj/memos)，一个使用 `Tailwind CSS` + `Go Template` 实现的类似 `Google Keep` 功能的项目。