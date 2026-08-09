---
title: "Tally - Build a tool like tokei/scc"
date: 2023-08-16T18:47:37+08:00
tags:
  - Line Counter
  - Optimization
published: true
summary: "Building a high-performance code statistics tool from scratch: Deep dive into the implementation principles of tokei/scc."
changelog: |
  - 2023-08-16: first version
  - 2024-04-03: benchmark refactor
---

## Background
I want to build a tool like `scc` or `tokei`, just for learning. Writing a simple version was very easy: [tally - first commit](https://github.com/abcdlsj/share/blob/7ac6cbbf36a9d72b09603b160569db5f5a27fa81/go/tally/main.go). I'll optimize it in the second half of this post.

First, allow me to explain it to you.

## Steps
The counting machine works a lot like the classic "putting an elephant in the freezer" analogy:
1. Walk the directory tree.
2. Read files and count lines.
3. Output the results.

### Walk directory

`filepath.Walk` makes walking the directory tree very easy:
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

### Read and count
To count lines, we first need to define what a line is: a string ending with `\n` or `\r\n`. So we can split the file content on `\n` (or `\r\n`) to get the lines.

Since we're counting code, we also need to ignore comment lines. I used a simple approach: ignore any line that starts with a rule string. (By the way, the first version only handled single-line comments.)

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

> `vec` is a helper that creates a slice — a little nostalgia for Rust's `vec!` :p

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
Actually, this was the hardest part — presenting results in a readable way. Thanks to `tokei` and `scc`, I just copied their output format :smile:.

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

Let's test it.

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

It looks very good :)

Next, I'll benchmark and optimize it.

## Benchmark
Using the `time` command for benchmarks isn't accurate enough.

> [stackoverflow - Is the UNIX `time` command accurate enough for benchmarks? [closed]](https://stackoverflow.com/questions/9006596/is-the-unix-time-command-accurate-enough-for-benchmarks)

> TLDR: use `perf stat` for benchmarks:
> `perf stat -r 10 -d <CMD>`

There are plenty of powerful benchmarking tools.

I use `hyperfine` for benchmarks at my `MacBook Pro 16 (2019)` machine.
> [hyperfine - Command-line benchmarking tool](https://github.com/sharkdp/hyperfine)

`hyperfine` supports a warmup phase to reduce noise, compares multiple commands, and can export results as markdown. (Very nice! :smile:)

```shell
hyperfine 'tokei .' 'scc .' 'tally .' --warmup 3 --export-markdown bench.md
```

### Small repo

I benchmarked against the <https://github.com/firecracker-microvm/firecracker> repo.

Results:

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `tokei .` | 36.2 ± 6.3 | 25.0 | 56.5 | 1.00 |
| `scc .` | 41.0 ± 7.7 | 28.0 | 58.9 | 1.13 ± 0.29 |
| `tally .` | 114.4 ± 4.8 | 108.5 | 123.6 | 3.16 ± 0.57 |

The original version isn't fast enough, but it's still usable.

Now let's try a large repo.

### Large repo

Use <https://github.com/moby/moby>

Results:
| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `tokei .` | 447.5 ± 60.5 | 385.4 | 541.4 | 1.30 ± 0.21 |
| `scc .` | 343.7 ± 29.4 | 296.0 | 399.3 | 1.00 |
| `tally .` | 1038.1 ± 77.2 | 947.3 | 1173.8 | 3.02 ± 0.34 |

There's a fairly large gap between `tokei`/`scc` and `tally`.

## Optimize1 - Improve the code

My first version was very simple: it only supported inline comments and used `split` to count lines.

There are two optimization points:
1. Use `bufio.Scanner` to read lines
2. Support multiple comments

See the commits <https://github.com/abcdlsj/share/commits/master/go/tally> for both.

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

The gap is much smaller on the small repo — `bufio.Scanner` is effective there — but on the large repo it barely matters.

## Optimize2 - Faster filepath walking

According to [this post](https://engineering.kablamo.com.au/posts/2021/quick-comparison-between-go-file-walk-implementations/), the official `filepath.Walk` is already fast enough. I also want `tally` to have zero third-party dependencies, so I won't pull in a faster `walkdir` library.

I currently use `filepath.Walk`; switching to `filepath.WalkDir` would be a little faster.

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

(It's only a small improvement, so I won't commit it.)

## Optimize3 - Parallelism

### Fanout

`Go` makes concurrency easy. I used a fan-out pattern to distribute files to workers.

The steps are:
1. Pass files through a `channel`.
2. The directory walk sends files to the `channel` first.
3. Multiple workers read files from the `channel`.
4. `sync.WaitGroup` waits for the workers to finish.
5. A `mutex` protects the shared result data.

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

The `fileChan` buffer is 100 and the worker count is 2 × the CPU core count. I didn't do much tuning here.

The general rule of thumb:
- For CPU-bound tasks, workers shouldn't exceed the number of logical CPU cores — more workers only adds context-switching overhead.
- For I/O-bound tasks, more workers than cores usually helps, because blocked I/O lets other workers proceed.


### Result

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `tally .` | 65.6 ± 6.1 | 56.8 | 85.1 | 1.89 ± 0.33 |
| `tally_fanout .` | 34.8 ± 5.2 | 26.9 | 53.2 | 1.00 |

The fan-out pattern speeds things up a lot.

## Optimize4 - Use Pprof

TODO...

## End

This post is my learning journey — the source is at [github - abcdlsj/tally](https://github.com/abcdlsj/share/tree/master/go/tally).

I'll benchmark and optimize more in the future.

## Ref

https://github.com/boyter/scc/
https://boyter.org/posts/sloc-cloc-code/
https://blog.burntsushi.net/ripgrep/
https://engineering.kablamo.com.au/posts/2021/quick-comparison-between-go-file-walk-implementations/
