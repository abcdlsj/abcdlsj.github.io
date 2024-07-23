---
title: "Tally - Build a tool like tokei/scc"
date: 2023-08-16T18:47:37+08:00
tags:
  - Counter
hide: false
tocPosition: left-sidebar
---

> Changelog:
> - 2023-08-16: first version
> - 2024-04-03: bechmark refactor

## Background
I want to build a tool like `scc`, `tokei`, just for learning. It was very easy to write a simple version: [tally - first commit](https://github.com/abcdlsj/share/blob/7ac6cbbf36a9d72b09603b160569db5f5a27fa81/go/tally/main.go).
I will to optimize it at the second half of this post.

First, allow me to explain it to you.

## Steps
The counting-line machine worked similar to the `Putting elephants in the freezer`, so the steps are: 
1. Walk directory tree.
2. Read file and Count lines.
1. Output result.

### Walk directory

Use `filepath.Walk` to walk directory tree, it's very easy to use.
```go
filepath.Walk(os.Args[1], func(path string, info os.FileInfo, err error) error {
	if err != nil {
		panic(err)
	}

	if info.IsDir() {
		return nil
	}

	return countLine(path)
})
```

### `Read` and `Count`
Count line we need to know what is a line. A line is a string end with `\n` or `\r\n`. So we can just split the file content with `\n` or `\r\n` to get the lines.

Because we need to count the code lines, so we need to ignore the comment lines. I just use a simple way to ignore the comment lines, just ignore the line start with a rule string(**by the way: the first version I just conside the single line comment.**).

```go
type Counter struct {
	idx     int
	lang    string
	comment string
	exts    []string
}

var (
	Go       = Counter{1, "Go", "//", vec(".go")}
	Rust     = Counter{2, "Rust", "//", vec(".rs")}
	Java     = Counter{3, "Java", "//", vec(".java")}
	Python   = Counter{4, "Python", "#", vec(".py")}
	C        = Counter{5, "C", "//", vec(".c", ".h")}
	Cpp      = Counter{6, "C++", "//", vec(".cpp", ".hpp")}
	Js       = Counter{7, "Javascript", "//", vec(".js")}
	Ts       = Counter{8, "Typescript", "//", vec(".ts")}
	HTML     = Counter{9, "HTML", "//", vec(".html", ".htm")}
	JSON     = Counter{10, "JSON", "//", vec(".json")}
	Protobuf = Counter{11, "Protobuf", "//", vec(".proto")}
	Markdown = Counter{12, "Markdown", "//", vec(".md")}
	Shell    = Counter{13, "Shell", "#", vec(".sh")}
	YAML     = Counter{14, "YAML", "#", vec(".yaml", ".yml")}
)
```

> vec is a function to create a slice. (Nostalgia for `Rust` `vec!` :p)

Count line logic:
```go
func countLine(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	c := guessLang(path)

	if c.lang == "" {
		return nil
	}

	lines := bytes.Split(data, []byte("\n"))

	item := Item{
		lang:  c.lang,
		lines: len(lines),
		files: 1,
	}

	for _, line := range lines {
		line := bytes.TrimSpace(line)
		if len(line) == 0 {
			item.blank++
			continue
		}

		if c.isComment(line) {
			item.comment++
			continue
		}

		item.code++
	}

	result.Add(c, item)
	return nil
}
```

### Output style
Actually, this is the most hard part. you need to output the result intuitively. thanks to `tokei` and `scc`, I just need to do a `copy` the output format from them :smile:.

```go
func (r *Result) String() {
	itemF := "%-10s %10d %10d %10d %10d %10d\n"
	headerF := "%-10s %10s %10s %10s %10s %10s\n"
	fmt.Printf(strings.Repeat("━", 65) + "\n")
	fmt.Printf(headerF, "Language", "Files", "Lines", "Code", "Comments", "Blanks")
	fmt.Printf(strings.Repeat("━", 65) + "\n")

	var total Item

	sort.Slice(r.data, func(i, j int) bool {
		return r.data[i].lines > r.data[j].lines
	})
	for _, item := range r.data {
		if item.files == 0 {
			continue
		}

		total = mergeItem(total, item)
		fmt.Printf(itemF, item.lang, item.files, item.lines, item.code, item.comment, item.blank)
	}

	fmt.Printf(strings.Repeat("━", 65) + "\n")
	fmt.Printf(itemF, "Total", total.files, total.lines, total.code, total.comment, total.blank)
	fmt.Printf(strings.Repeat("━", 65) + "\n")
}
```

Let's Test it.

```shell
tally .
```

```text
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Language        Files      Lines       Code   Comments     Blanks
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Go                  1        242        199          0         43
Markdown            1          3          2          0          1
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total               2        245        201          0         44
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

It's looks very good :)

At next, I will benchmark and optimize it.

## Benchmark
use `time` for benchmarks is not accurate enough.

> [stackoverflow - Is the UNIX `time` command accurate enough for benchmarks? [closed]](https://stackoverflow.com/questions/9006596/is-the-unix-time-command-accurate-enough-for-benchmarks)

> TLDR: Can use `perf stat` to benchmarks
> `perf stat -r 10 -d <CMD>`

There are also many of powerful tools for benchmarks.

I use `hyperfine` for benchmarks at my `MacBook Pro 16 (2019)` machine.
> [hyperfine - Command-line benchmarking tool](https://github.com/sharkdp/hyperfine)

`hyperfine` support warmup file system to reduce noise, and it can compare multiple commands and export to markdown. (very nice! :smile:)

```shell
hyperfine 'tokei .' 'scc .' 'tally .' --warmup 3 --export-markdown bench.md
```

### Small repo

I use the repo <https://github.com/firecracker-microvm/firecracker> to do the benchmarks

Results:

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `tokei .` | 36.2 ± 6.3 | 25.0 | 56.5 | 1.00 |
| `scc .` | 41.0 ± 7.7 | 28.0 | 58.9 | 1.13 ± 0.29 |
| `tally .` | 114.4 ± 4.8 | 108.5 | 123.6 | 3.16 ± 0.57 |

We can see, the origin version is not fast enough. but I thought it's already usable.

Let's see large repo.

### Large repo

Use <https://github.com/moby/moby>

Results:
| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `tokei .` | 447.5 ± 60.5 | 385.4 | 541.4 | 1.30 ± 0.21 |
| `scc .` | 343.7 ± 29.4 | 296.0 | 399.3 | 1.00 |
| `tally .` | 1038.1 ± 77.2 | 947.3 | 1173.8 | 3.02 ± 0.34 |

Okay, seems there had a relatively large gap between `tokei/scc` and `tally`.

## Optimize1 - Improve the code

My first version is very simple and it's just support inline comments, and use `split` to count lines.

There have two optimal point:
1. Use `Bufio.Scanner` to readline
2. Support multiple comments

For these two, can see the commits <https://github.com/abcdlsj/share/commits/master/go/tally>

There's nothing to say here.

### Compare

- Small repo

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `tally .` | 70.2 ± 5.9 | 56.4 | 86.1 | 1.00 |
| `tally1 .` | 119.9 ± 11.3 | 108.5 | 151.2 | 1.71 ± 0.22 |

- Large repo

| Command | Mean [s] | Min [s] | Max [s] | Relative |
|:---|---:|---:|---:|---:|
| `tally .` | 1.073 ± 0.129 | 0.892 | 1.295 | 1.00 |
| `tally1 .` | 1.113 ± 0.139 | 0.928 | 1.277 | 1.04 ± 0.18 |

We can see, for the small repo the gap is smaller. use `bufio.Scanner` is very effective. but for large repo, it seems not very effective.

## Optimize2 - Faster filepath walking

Based on this post [You Don't Need a Library for File Walking in Go](https://engineering.kablamo.com.au/posts/2021/quick-comparison-between-go-file-walk-implementations/), the result shows the `offical filepath walk` are faster enough. And I want to build `tally` without any `third-party dependencies`, so I won't use other faster `walkdir` library.

Currently, I use the `filepath.Walk` to `walking` dir, use `filepth.Walkdir` will be faster a little.

```shell
> hyperfine 'tally .' 'tally_walkdir .' --warmup 3
Benchmark 1: tally .
  Time (mean ± σ):      66.4 ms ±   6.6 ms    [User: 21.8 ms, System: 43.0 ms]
  Range (min … max):    58.0 ms …  82.8 ms    37 runs

Benchmark 2: tally_walkdir .
  Time (mean ± σ):      62.8 ms ±   8.1 ms    [User: 21.5 ms, System: 39.1 ms]
  Range (min … max):    51.8 ms …  89.9 ms    43 runs

Summary
  tally_walkdir . ran
    1.06 ± 0.17 times faster than tally .
```

(Just a small improvement, So I won't commit the change.)

## Optimize3 - Parallelism

### Fanout

`Go` supports concurrency perfectly, we can use the parttern named `Fanout` to `allocate` file to `worker`.

We follow there steps:
1. Passing variables via `channel`
2. `Walk` dir will send files firstly to `channel`
3. Use multiple `worker` to `read` files from `channel`
4. Use `sync.WaitGroup` to `wait` for `worker` done
5. Use `mutex` to `lock` the `result` data

These are changes:
```diff
diff --git a/go/tally/main.go b/go/tally/main.go
index f184b31..23f5fae 100644
--- a/go/tally/main.go
+++ b/go/tally/main.go
@@ -6,8 +6,10 @@ import (
 	"fmt"
 	"os"
 	"path/filepath"
+	"runtime"
 	"sort"
 	"strings"
+	"sync"
 )
 
 type Counter struct {
@@ -48,18 +50,29 @@ func init() {
 		}
 	}
 
-	result = NewResult()
+	result = &Result{
+		data: make([]Item, registedNum),
+	}
 }
 
 var result *Result
 
-func main() {
-	if len(os.Args) < 2 {
-		fmt.Println("Usage: tally <path>")
-		os.Exit(1)
+var fileChan = make(chan string, 100)
+
+func process(dir string) {
+	var wg sync.WaitGroup
+	wg.Add(runtime.NumCPU() * 2)
+
+	for i := 0; i < runtime.NumCPU()*2; i++ {
+		go func() {
+			defer wg.Done()
+			for file := range fileChan {
+				countLine(file)
+			}
+		}()
 	}
 
-	filepath.Walk(os.Args[1], func(path string, info os.FileInfo, err error) error {
+	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
 		if err != nil {
 			panic(err)
 		}
@@ -68,9 +81,24 @@ func main() {
 			return nil
 		}
 
-		return countLine(path)
+		fileChan <- path
+
+		return nil
 	})
 
+	close(fileChan)
+
+	wg.Wait()
+}
+
+func main() {
+	if len(os.Args) < 2 {
+		fmt.Println("Usage: tally <path>")
+		os.Exit(1)
+	}
+
+	process(os.Args[1])
+
 	result.String()
 }
 
@@ -95,16 +123,13 @@ func mergeItem(a, b Item) Item {
 }
 
 type Result struct {
+	mu   sync.Mutex
 	data []Item
 }
 
-func NewResult() *Result {
-	return &Result{
-		data: make([]Item, registedNum),
-	}
-}
-
 func (r *Result) Add(c Counter, item Item) {
+	r.mu.Lock()
+	defer r.mu.Unlock()
 	r.data[c.idx-1] = mergeItem(r.data[c.idx-1], item)
 }
 
@@ -174,6 +199,7 @@ func guessLang(file string) Counter {
 func countLine(path string) error {
 	f, err := os.Open(path)
 	scanner := bufio.NewScanner(f)
+
 	if err != nil {
 		return err
 	}
```

The `fileChan` size is 100, the worker number is 2 * `CPU number`. I don't do much test here.

According common knowledge:
- CPU-bound task, the number of workers should not exceed the number of logical CPU cores available, as having more workers than cores will not improve performance and may even degrade it due to context switching
- I/O-bound tasks, more workers than CPU cores is probably a good idea, because I/O operations often block, allowing other workers to proceed with their tasks


### Result

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `tally .` | 65.6 ± 6.1 | 56.8 | 85.1 | 1.89 ± 0.33 |
| `tally_fanout .` | 34.8 ± 5.2 | 26.9 | 53.2 | 1.00 |

Use `Fanout` pattern will speed up a lot.

## Optimize4 - Use Pprof

TODO...

## End

The post is just a learning progress, you can find source code at [github - abcdlsj/tally](https://github.com/abcdlsj/share/tree/master/go/tally)

I'll do more test and optimize at the future.

## Ref

https://github.com/boyter/scc/
https://boyter.org/posts/sloc-cloc-code/
https://blog.burntsushi.net/ripgrep/
https://engineering.kablamo.com.au/posts/2021/quick-comparison-between-go-file-walk-implementations/

