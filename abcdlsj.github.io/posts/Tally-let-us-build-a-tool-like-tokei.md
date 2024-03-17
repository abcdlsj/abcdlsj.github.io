---
title: "Tally - Build a tool like tokei/scc"
date: 2023-08-16T18:47:37+08:00
tags:
  - Counter
hide: false
---

## First
I want to build a tool like `scc`, `tokei`, just for learning. It was very easy to write a simple version: [tally - first commit](https://github.com/abcdlsj/share/blob/7ac6cbbf36a9d72b09603b160569db5f5a27fa81/go/tally/main.go).
I will to optimize it at the second half of this post.

First, allow me to explain it to you.

## Explain
The counting-line machine worked similar to the `Putting elephants in the freezer`, so the steps are: 
1. Walk directory tree.
2. Read file and Count lines.
1. Output result.

### Walk directory tree
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

### Read file and Count lines
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
**vec is a function to create a slice. (Nostalgia for `Rust vec!`  :smile:)**

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

### Output result
Actually, this is the most hard part. you need to output the result intuitively. thanks to `tokei` and `scc`, I just copy the output format from them :smile:.
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

Test it.

```shell
go install
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

All done! let's benchmark it.

## Benchmark
use `time`  for benchmarks is not accurate enough. [stackoverflow - Is the UNIX `time` command accurate enough for benchmarks? [closed]](https://stackoverflow.com/questions/9006596/is-the-unix-time-command-accurate-enough-for-benchmarks)
so use `perf stat` for benchmarks at my `Intel(R) N100` machine.
```shell
perf stat -r 10 -d {xxx} .
```

Test repo: https://github.com/firecracker-microvm/firecracker
### Tokei
```shell
 Performance counter stats for 'tokei .' (10 runs):

             69.13 msec task-clock                #    2.481 CPUs utilized            ( +- 11.54% )
                51      context-switches          #  553.733 /sec                     ( +-  6.59% )
                 3      cpu-migrations            #   32.573 /sec                     ( +- 12.62% )
               684      page-faults               #    7.427 K/sec                    ( +-  1.02% )
       188,619,115      cycles                    #    2.048 GHz                      ( +-  0.12% )
       436,763,869      instructions              #    2.31  insn per cycle           ( +-  0.02% )
       107,324,341      branches                  #    1.165 G/sec                    ( +-  0.02% )
           827,429      branch-misses             #    0.77% of all branches          ( +-  0.25% )
       117,783,424      L1-dcache-loads           #    1.279 G/sec                    ( +-  0.02% )
   <not supported>      L1-dcache-load-misses
            69,479      LLC-loads                 #  754.369 K/sec                    ( +-  0.72% )
            16,783      LLC-load-misses           #   23.79% of all LL-cache accesses  ( +-  3.05% )

           0.02786 +- 0.00227 seconds time elapsed  ( +-  8.13% )
```

### Tally
```shell
 Performance counter stats for '/home/ubtu/.go/bin/tally .' (10 runs):

             41.15 msec task-clock                #    1.050 CPUs utilized            ( +-  1.02% )
               199      context-switches          #    4.812 K/sec                    ( +-  3.37% )
                 7      cpu-migrations            #  169.267 /sec                     ( +- 13.47% )
             6,710      page-faults               #  162.254 K/sec                    ( +-  1.52% )
       133,015,106      cycles                    #    3.216 GHz                      ( +-  0.40% )
       147,895,882      instructions              #    1.12  insn per cycle           ( +-  0.26% )
        30,824,314      branches                  #  745.361 M/sec                    ( +-  0.22% )
           268,672      branch-misses             #    0.86% of all branches          ( +-  0.27% )
        38,861,478      L1-dcache-loads           #  939.707 M/sec                    ( +-  0.26% )
   <not supported>      L1-dcache-load-misses
           633,051      LLC-loads                 #   15.308 M/sec                    ( +-  0.41% )
           322,198      LLC-load-misses           #   51.64% of all LL-cache accesses  ( +-  0.65% )

          0.039192 +- 0.000357 seconds time elapsed  ( +-  0.91% )
```

**Result**: we can see, the origin version is faster enough even faster then `tokei`, because `tokei` have more complex features and the count of files is small.

## Optimize1 - Parallel
Walking file paralleling. I searched out a post about this, [stackoverflow - Concurrent filesystem scanning](https://stackoverflow.com/questions/44255814/concurrent-filesystem-scanning)

I write a version with `parallel`, this is the `diff`.

```diff
 func main() {
-	filepath.Walk(os.Args[1], func(path string, info os.FileInfo, err error) error {
+	var wgCalc sync.WaitGroup
+	paths := make(chan string, 100)
+	defer close(paths)
+
+	wgCalc.Add(1)
+	go func() {
+		defer wgCalc.Done()
+		for path := range paths {
+			if path == "__end_flag_tally" {
+				return
+			}
+			wgCalc.Add(1)
+			go func(path string) {
+				defer wgCalc.Done()
+				calculate(path)
+			}(path)
+		}
+	}()
+
+	var wgWalk sync.WaitGroup
+	wgWalk.Add(1)
+	go func() {
+		defer wgWalk.Done()
+		WalkParallel(&wgWalk, os.Args[1], paths)
+	}()
+
+	wgWalk.Wait()
+	paths <- "__end_flag_tally"
+	wgCalc.Wait()
+
+	result.String()
+}
+
+func WalkParallel(wg *sync.WaitGroup, dir string, paths chan<- string) {
+	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
 		if err != nil {
 			panic(err)
 		}
 
-		if info.IsDir() {
-			return nil
+		if info.IsDir() && path != dir {
+			wg.Add(1)
+			go func() {
+				defer wg.Done()
+				WalkParallel(wg, path, paths)
+			}()
+			return filepath.SkipDir
 		}
 
-		return calculate(path)
+		paths <- path
+		return nil
 	})
 }
-
```

### test chan capacity 10

with `paths chan capacity=10` 
```shell
 Performance counter stats for '/home/ubtu/.go/bin/tally_walk10 .' (10 runs):

            105.76 msec task-clock                #    2.573 CPUs utilized            ( +-  5.49% )
               392      context-switches          #    3.432 K/sec                    ( +- 15.47% )
                41      cpu-migrations            #  358.962 /sec                     ( +- 10.96% )
             7,335      page-faults               #   64.219 K/sec                    ( +-  3.52% )
       181,829,636      cycles                    #    1.592 GHz                      ( +-  1.05% )
       188,274,662      instructions              #    0.99  insn per cycle           ( +-  0.65% )
        38,091,153      branches                  #  333.495 M/sec                    ( +-  0.57% )
           392,566      branch-misses             #    1.00% of all branches          ( +-  1.98% )
        48,955,270      L1-dcache-loads           #  428.612 M/sec                    ( +-  0.63% )
   <not supported>      L1-dcache-load-misses
           713,386      LLC-loads                 #    6.246 M/sec                    ( +-  1.48% )
           186,519      LLC-load-misses           #   25.09% of all LL-cache accesses  ( +-  6.00% )

           0.04110 +- 0.00219 seconds time elapsed  ( +-  5.33% )
```

But the result shows it's not `expected`.
Why?

### test chan capacity 100/1000/10000
```shell
 Performance counter stats for '/home/ubtu/.go/bin/tally_walk100 .' (10 runs):

            149.63 msec task-clock                #    2.970 CPUs utilized            ( +-  3.94% )
             1,250      context-switches          #    9.855 K/sec                    ( +- 10.07% )
               229      cpu-migrations            #    1.805 K/sec                    ( +- 11.05% )
             8,715      page-faults               #   68.707 K/sec                    ( +-  3.08% )
       246,295,457      cycles                    #    1.942 GHz                      ( +-  2.79% )
       239,901,377      instructions              #    1.18  insn per cycle           ( +-  2.46% )
        47,265,335      branches                  #  372.630 M/sec                    ( +-  2.35% )
           498,222      branch-misses             #    1.20% of all branches          ( +-  2.47% )
        61,741,968      L1-dcache-loads           #  486.761 M/sec                    ( +-  2.39% )
   <not supported>      L1-dcache-load-misses
           770,443      LLC-loads                 #    6.074 M/sec                    ( +-  1.42% )
           167,332      LLC-load-misses           #   21.61% of all LL-cache accesses  ( +-  9.58% )

           0.05037 +- 0.00534 seconds time elapsed  ( +- 10.59% )
```

```shell
 Performance counter stats for '/home/ubtu/.go/bin/tally_walk1000 .' (10 runs):

            126.02 msec task-clock                #    2.401 CPUs utilized            ( +-  7.14% )
               534      context-switches          #    4.190 K/sec                    ( +- 31.54% )
                43      cpu-migrations            #  337.390 /sec                     ( +- 82.12% )
             8,730      page-faults               #   68.498 K/sec                    ( +-  3.02% )
       200,521,361      cycles                    #    1.573 GHz                      ( +-  3.37% )
       201,667,306      instructions              #    0.95  insn per cycle           ( +-  3.08% )
        40,664,133      branches                  #  319.063 M/sec                    ( +-  2.75% )
           411,992      branch-misses             #    0.95% of all branches          ( +-  3.61% )
        52,419,468      L1-dcache-loads           #  411.298 M/sec                    ( +-  2.91% )
   <not supported>      L1-dcache-load-misses
           699,381      LLC-loads                 #    5.488 M/sec                    ( +-  2.91% )
           209,549      LLC-load-misses           #   26.89% of all LL-cache accesses  ( +-  6.15% )

           0.05249 +- 0.00584 seconds time elapsed  ( +- 11.12% )
```

```shell
 Performance counter stats for '/home/ubtu/.go/bin/tally_walk10000 .' (10 runs):

            165.04 msec task-clock                #    3.075 CPUs utilized            ( +-  5.71% )
             1,986      context-switches          #   15.979 K/sec                    ( +- 10.74% )
               440      cpu-migrations            #    3.540 K/sec                    ( +- 10.92% )
            11,124      page-faults               #   89.502 K/sec                    ( +-  3.41% )
       254,677,268      cycles                    #    2.049 GHz                      ( +-  4.05% )
       263,349,968      instructions              #    1.25  insn per cycle           ( +-  3.98% )
        51,394,364      branches                  #  413.512 M/sec                    ( +-  3.66% )
           505,212      branch-misses             #    1.16% of all branches          ( +-  3.36% )
        67,546,889      L1-dcache-loads           #  543.473 M/sec                    ( +-  3.77% )
   <not supported>      L1-dcache-load-misses
           837,174      LLC-loads                 #    6.736 M/sec                    ( +-  2.54% )
           211,730      LLC-load-misses           #   27.27% of all LL-cache accesses  ( +-  7.50% )

           0.05367 +- 0.00622 seconds time elapsed  ( +- 11.59% )
```

### test with large number of files
dir with `3800+` files.
```shell
root@beelink100 /h/u/workspace# perf stat -r 10 -d /home/ubtu/.go/bin/tally_walk10000 . >/dev/null

 Performance counter stats for '/home/ubtu/.go/bin/tally_walk10000 .' (10 runs):

          2,121.55 msec task-clock                #    2.028 CPUs utilized            ( +-  5.87% )
            12,183      context-switches          #    5.139 K/sec                    ( +-  3.39% )
             1,167      cpu-migrations            #  492.255 /sec                     ( +-  5.36% )
           203,339      page-faults               #   85.771 K/sec                    ( +- 17.22% )
     5,990,351,339      cycles                    #    2.527 GHz                      ( +-  6.79% )
     2,930,469,142      instructions              #    0.43  insn per cycle           ( +- 24.59% )
       542,443,958      branches                  #  228.809 M/sec                    ( +- 24.28% )
         2,905,949      branch-misses             #    0.41% of all branches          ( +- 13.97% )
       914,997,532      L1-dcache-loads           #  385.957 M/sec                    ( +- 19.22% )
   <not supported>      L1-dcache-load-misses
        31,492,061      LLC-loads                 #   13.284 M/sec                    ( +-  2.38% )
        11,437,469      LLC-load-misses           #   33.55% of all LL-cache accesses  ( +-  3.87% )

             1.046 +- 0.127 seconds time elapsed  ( +- 12.13% )

root@beelink100 /h/u/workspace# perf stat -r 10 -d tokei . >/dev/null

 Performance counter stats for 'tokei .' (10 runs):

            299.67 msec task-clock                #    3.368 CPUs utilized            ( +-  2.28% )
               201      context-switches          #  655.026 /sec                     ( +-  4.36% )
                 1      cpu-migrations            #    3.259 /sec                     ( +- 30.73% )
             2,381      page-faults               #    7.759 K/sec                    ( +-  0.36% )
       872,948,453      cycles                    #    2.845 GHz                      ( +-  0.22% )
     2,063,804,765      instructions              #    2.36  insn per cycle           ( +-  0.06% )
       483,436,637      branches                  #    1.575 G/sec                    ( +-  0.04% )
         3,572,630      branch-misses             #    0.74% of all branches          ( +-  0.35% )
       568,333,584      L1-dcache-loads           #    1.852 G/sec                    ( +-  0.06% )
   <not supported>      L1-dcache-load-misses
           371,106      LLC-loads                 #    1.209 M/sec                    ( +-  0.82% )
            76,251      LLC-load-misses           #   20.26% of all LL-cache accesses  ( +-  2.06% )

           0.08897 +- 0.00211 seconds time elapsed  ( +-  2.38% )

root@beelink100 /h/u/workspace# perf stat -r 10 -d /home/ubtu/.go/bin/tally . >/dev/null

 Performance counter stats for '/home/ubtu/.go/bin/tally .' (10 runs):

          1,207.93 msec task-clock                #    1.029 CPUs utilized            ( +-  0.17% )
             1,793      context-switches          #    1.481 K/sec                    ( +-  3.76% )
                28      cpu-migrations            #   23.127 /sec                     ( +- 13.70% )
           177,609      page-faults               #  146.698 K/sec                    ( +-  0.57% )
     4,008,330,188      cycles                    #    3.311 GHz                      ( +-  0.06% )
     2,280,290,491      instructions              #    0.57  insn per cycle           ( +-  0.23% )
       428,399,527      branches                  #  353.841 M/sec                    ( +-  0.21% )
         1,902,235      branch-misses             #    0.44% of all branches          ( +-  0.22% )
       750,980,580      L1-dcache-loads           #  620.280 M/sec                    ( +-  0.18% )
   <not supported>      L1-dcache-load-misses
        28,800,720      LLC-loads                 #   23.788 M/sec                    ( +-  0.08% )
        16,184,744      LLC-load-misses           #   56.33% of all LL-cache accesses  ( +-  0.13% )

           1.17430 +- 0.00151 seconds time elapsed  ( +-  0.13% )

root@beelink100 /h/u/workspace# perf stat -r 10 -d /home/ubtu/.go/bin/tally_walk10 . >/dev/null

 Performance counter stats for '/home/ubtu/.go/bin/tally_walk10 .' (10 runs):

          2,094.44 msec task-clock                #    2.632 CPUs utilized            ( +-  0.78% )
            13,318      context-switches          #    6.068 K/sec                    ( +-  4.65% )
             1,022      cpu-migrations            #  465.636 /sec                     ( +-  7.66% )
           191,164      page-faults               #   87.097 K/sec                    ( +-  1.66% )
     6,180,096,296      cycles                    #    2.816 GHz                      ( +-  0.58% )
     2,814,759,285      instructions              #    0.44  insn per cycle           ( +-  0.55% )
       522,446,609      branches                  #  238.033 M/sec                    ( +-  0.53% )
         3,255,157      branch-misses             #    0.63% of all branches          ( +-  1.01% )
       886,619,049      L1-dcache-loads           #  403.955 M/sec                    ( +-  0.44% )
   <not supported>      L1-dcache-load-misses
        33,985,463      LLC-loads                 #   15.484 M/sec                    ( +-  1.09% )
         9,946,393      LLC-load-misses           #   29.66% of all LL-cache accesses  ( +-  3.42% )

            0.7958 +- 0.0206 seconds time elapsed  ( +-  2.58% )
```

### conclusion
Based on the result, I think the `parallel` won't make too great improvement.
Seems the most time killer is not the `line counter`, it's the `filepath walker`.
But my result is very abbreviate, It effected by many factors (I'll do more test at the future) 

## Optimize2 - Bufio scanner countline
[tally - main.go#L145-L182](https://github.com/abcdlsj/share/blob/fd5bb216a651246352a25d27ddc189396cdec13a/go/tally/main.go#L145-L182)

```shell
 Performance counter stats for '/home/ubtu/.go/bin/tally .' (10 runs):

             22.23 msec task-clock                #    0.967 CPUs utilized            ( +-  1.89% )
                67      context-switches          #    2.905 K/sec                    ( +-  8.96% )
                 4      cpu-migrations            #  173.427 /sec                     ( +- 11.33% )
               875      page-faults               #   37.937 K/sec                    ( +-  0.33% )
        71,646,295      cycles                    #    3.106 GHz                      ( +-  0.59% )
       165,519,597      instructions              #    2.26  insn per cycle           ( +-  0.04% )
        44,646,031      branches                  #    1.936 G/sec                    ( +-  0.03% )
           364,548      branch-misses             #    0.82% of all branches          ( +-  0.25% )
        30,861,725      L1-dcache-loads           #    1.338 G/sec                    ( +-  0.06% )
   <not supported>      L1-dcache-load-misses
            75,727      LLC-loads                 #    3.283 M/sec                    ( +-  0.37% )
            32,681      LLC-load-misses           #   42.57% of all LL-cache accesses  ( +-  0.81% )

          0.023000 +- 0.000407 seconds time elapsed  ( +-  1.77% )
```

it's so `fast` than [origin `counter` tally](#tally)

## Optimize3 - Faster filepath walking
At the [`Optimize1`](#optimize1---parallel), I found the `filepath.Walk` is the most time killer.
Based on this post [You Don't Need a Library for File Walking in Go](https://engineering.kablamo.com.au/posts/2021/quick-comparison-between-go-file-walk-implementations/), the result shows the `offical filepath walk` are faster enough. And I want to build `tally` without any `third-party dependencies`, so I won't use other faster `walkdir` library...
Currently, I use the `filepath.Walk` to `walking` dir, I'll modify to `filepth.Walkdir`, It's faster a little.

I bench the two version (`filepath.Walk` and `filepath.Walkdir`) at my `Mbp16` using [hyperfine](https://github.com/sharkdp/hyperfine)

```shell
$ hyperfine 'tally_walk .' 'tally_walkdir .' --warmup 3
Benchmark 1: tally_walk .
  Time (mean ± σ):      3.035 s ±  0.519 s    [User: 0.594 s, System: 2.382 s]
  Range (min … max):    2.419 s …  4.233 s    10 runs

Benchmark 2: tally_walkdir .
  Time (mean ± σ):      2.475 s ±  0.500 s    [User: 0.510 s, System: 1.924 s]
  Range (min … max):    2.100 s …  3.780 s    10 runs

Summary
  tally_walkdir . ran
    1.23 ± 0.32 times faster than tally_walk .
```

## End
**Not the end**
The post is just a learning progress, you can find source code at [github - abcdlsj/tally](https://github.com/abcdlsj/share/tree/master/go/tally)
I'll do more test and optimize at the future.

## Ref

https://github.com/boyter/scc/
https://boyter.org/posts/sloc-cloc-code/
https://blog.burntsushi.net/ripgrep/
https://engineering.kablamo.com.au/posts/2021/quick-comparison-between-go-file-walk-implementations/