---
title: "Add content search to the blog"
date: 2024-08-15T23:42:49+08:00
tags:
  - SSG
  - Search
  - Word Segmentation
  - Inverted Index
hide: false
toc: true
description: "为静态网站添加搜索功能的设计记录：从倒排索引设计到中文分词处理，分享如何使用 Go 和 JavaScript 实现一个轻量级的全文搜索系统。"
languages:
    - cn
---

## Background

我目前使用的是自己实现的 SSG（Static Site Generator）程序来生成这个站点，所以很多功能都没有，需要自己添加。

比如 `Feed` 功能！

这个比较简单，特别是目前还有非常强大的 AI 工具，比如 `GitHub Copilot`，`Cursor` 等（强烈推荐 `Cursor` 和 `Claude 3.5 Sonnet`）。

很快就能实现，只需要代码里将 `Feed` 结构体定义好，将 `Post` 转换到结构体里，`xml.Marshal` 一下，然后就能生成 `rss.xml` 文件。将文件放到目录下，定义一个 `Menu` 访问文件链接，就 `ok` 了。

也有稍微麻烦的功能，例如 `Search` 功能。

## Example

之前看到过一个有意思的博客文章，是给 SSG 添加 `Search` 功能，并且是基于布隆过滤器（`Bloom Filter`）和 `Rust` 以及 `WebAssembly` 实现的。

链接在这里：[A Tiny, Static, Full-Text Search Engine using Rust and WebAssembly](https://endler.dev/2019/tinysearch/)

原理就是给每个 `Post` 生成一个 `Bloom Filter`，每个词都设置到 `Bloom Filter` 中，搜索时通过判断搜索词是否在 `Bloom Filter` 中，来判断 `Post` 是否包含该词。

博主没有直接使用 `JavaScript` 实现过滤器的代码，而是使用 `Rust` 编译成 `WebAssembly` 模块，通过 `JavaScript` 调用。

最后效果非常好，生成的模块大小只有几百 KB，并且搜索速度非常快。

所以这是一个思路，但是我没有这样实现，因为我并不懂 `WebAssembly`。

还有的方法就是使用一些服务，例如 `Algolia`，我也不想使用。

## My Implementation

我的实现是基于倒排索引（`Inverted Index`）和 `JavaScript` 脚本实现的。

### Generate Index

首先就是读取所有 `Post` 的内容，然后进行分词，分词后得到一个词典，然后根据词典生成倒排索引。

中文分词可以使用 [sego](https://github.com/huichen/sego) 实现。

> PS: 很早之前使用 `sego` 和 `tfidf` 实现过一个简单的搜索 `wikipedia` 的小程序，在这里 [seeker](https://github.com/abcdlsj/seeker)。


sego 分词器加载词典文件 `dictionary.txt`，这个文件在 repo 里有。
```go
func loadDict() {
	sm.LoadDictionary("dictionary.txt")
}
```

分词部分的代码如下：
```go
func analyze(text string) []string {
	segments := sm.Segment([]byte(text))
	words := sego.SegmentsToSlice(segments, false)

	var filteredWords []string
	for _, word := range words {
		if isNumeric(word) {
			continue
		}

		if isImageFile(word) {
			continue
		}

		if utf8.RuneCountInString(word) < 2 ||
			utf8.RuneCountInString(word) > 10 ||
			len(word) < 2 || len(word) > 10 {
			continue
		}

		if isStopWord(word) {
			continue
		}

		if isGibberish(word) {
			continue
		}

		filteredWords = append(filteredWords, word)
	}

	return filteredWords
}
```
注意过滤一些不需要的「词」，例如纯数字、图片、停用词、单个字符、过长或过短的词。

> 需要注意，汉字需要使用 `utf8.RuneCountInString` 来计算长度。

生成 `index.json` 部分：
```go
func generateSearchIndex() error {
	index := SearchIndex{Words: make(map[string][]string)}

	for _, post := range Posts {
		fullText := post.Meta.Title + " " + stripHTML(post.MDData)
		words := analyze(fullText)

		for _, word := range words {
			word = strings.ToLower(word)
			if !contains(index.Words[word], post.Uname) {
				index.Words[word] = append(index.Words[word], post.Uname)
			}
		}
	}

    // save to file
}
```

### Search
有了 `index.json` 文件后，就可以写一个简单的 `JavaScript` 来实现搜索功能。

> 这是完全通过 `Cursor` 生成的代码，我就加了两行 `console.log` 来调试。

```javascript
let searchIndex = null;

async function loadSearchIndex() {
    if (searchIndex === null) {
        const response = await fetch('/search-index.json');
        searchIndex = await response.json();
    }
}

async function search(query) {
    await loadSearchIndex();
    const words = query.toLowerCase().split(/\s+/);
    const results = new Set();
    
    for (const word of words) {
        if (word in searchIndex.words) {
            for (const postUrl of searchIndex.words[word]) {
                results.add(postUrl);
            }
        }
    }

    return Array.from(results);
}

let searchTimeout = null;
let isComposing = false;

document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');
    
    searchInput.addEventListener('input', () => {
        if (isComposing) return;
        scheduleSearch();
    });

    searchInput.addEventListener('compositionstart', () => {
        isComposing = true;
    });

    searchInput.addEventListener('compositionend', () => {
        isComposing = false;
        scheduleSearch();
    });

    function scheduleSearch() {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => performSearch(), 300);
    }

    async function performSearch() {
        const query = searchInput.value.trim();
        if (query.length < 2) {
            searchResults.innerHTML = '';
            return;
        }

        const results = await search(query);
        searchResults.innerHTML = results.map(url => `<li><a href="/posts/${url}.html">${url}</a></li>`).join('');
    }
});
```

这部分，最开始在搜索中文的时候我发现，中文输入被转换成了拼音，因为 `input` 事件触发时机太快，输入法还没有生成汉字。

`Cursor` 帮我优化了这个小问题，通过 `compositionstart` 和 `compositionend` 来跟踪输入法的状态，从而避免这个问题，并且添加了 `setTimeout` 来优化搜索频率。

## Result

个人觉得搜索效果非常好，可以在首页试试！