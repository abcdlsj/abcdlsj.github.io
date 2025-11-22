---
title: "Write a simple cron expression library in Golang"
date: 2023-10-29T23:51:37+08:00
tags:
  - Cron Expression
  - Package Design
hide: false
description: "深入解析 Cron 表达式：从零实现定时任务调度器，掌握时间表达式的解析算法，构建轻量级 Cron 库并理解其底层实现原理。"
languages:
    - cn
---
## Background

> 本文所有的代码都在 [abcdlsj/crone](https://github.com/abcdlsj/crone)

从零实现一个初级 `Cron` 表达式解析器，只有以下几个字段规则：
```
# ┌───────────── minute (0–59)
# │ ┌───────────── hour (0–23)
# │ │ ┌───────────── day of the month (1–31)
# │ │ │ ┌───────────── month (1–12)
# │ │ │ │ ┌───────────── day of the week (0–6) (Sunday to Saturday;
# │ │ │ │ │                                   7 is also Sunday on some systems)
# │ │ │ │ │
# │ │ │ │ │
# * * * * * <command to execute>
```

没有 `yearly` `monthly` `weekly` `daily` `hourly` `reboot` 等特殊规则。
|Entry|Description|Equivalent to|
|---|---|---|
|`@yearly` (or `@annually`)|Run once a year at midnight of 1 January|`0 0 1 1 *`|
|`@monthly`|Run once a month at midnight of the first day of the month|`0 0 1 * *`|
|`@weekly`|Run once a week at midnight on Sunday|`0 0 * * 0`|
|`@daily` (or `@midnight`)|Run once a day at midnight|`0 0 * * *`|
|`@hourly`|Run once an hour at the beginning of the hour|`0 * * * *`|
|`@reboot`|Run at startup|—|

其中字段规则

|Field|Required|Allowed values|Allowed special characters|Remarks|
|---|---|---|---|---|
|Minutes|Yes|0–59|`*` `,` `-`||
|Hours|Yes|0–23|`*` `,` `-`||
|Day of month|Yes|1–31|`*` `,` `-` `?` `L` `W`|`?` `L` `W` only in some implementations|
|Month|Yes|1–12 or JAN–DEC|`*` `,` `-`||
|Day of week|Yes|0–6 or SUN–SAT|`*` `,` `-` `?` `L` `#`|`?` `L` `#` only in some implementations|
|Year|No|1970–2099|`*` `,` `-`|This field is not supported in standard/default implementations.|

可以直观看到 `Cron` 表达式字段都有明确的上下限，以及支持的 `Special characters` 都是 `"*"` `","` `"-"`（这里只考虑一般规则）。

所以其实可以复用同一套规则，然后分别 `Parse` 就可以了。

## Field definition
首先定义 `field`，以及实现一个 `limit` 方法，返回其上下限。
```go
type field int

const (
	minute field = iota
	hour
	day
	month
	weekday
)

func (f field) limit() (int, int) {
	switch f {
	case minute:
		return 0, 59
	case hour:
		return 0, 23
	case day:
		return 1, 31
	case month:
		return 1, 12
	case weekday:
		return 0, 6
	}
	return 0, 0
}
```

## Expr structure

`Expr` 的结构体该怎么定义呢，它本身应该有一个 `expr` 代表表达式字符串。
```go
type Cronexpr struct {
	expr     string
}
```
这里以我的视角（一个 `Cron` 库使用者）的角度出发，我期望 `Cron` 库有下面的功能
1. `New(string)`： 创建新的 `Cronexpr`
2. `Next()`： 获取接下来 `Cron` 触发的时间
3. `NextN(n)`： 获取接下来 `n` 个 `Cron` 触发的时间

这里分别是「创建」和「使用」，因为 `Go` 提倡使用 `Channel` 在 `Goroutine` 之间传递信息，这里还可以加一个方法。

4. `Notify(ctx, outchan)` 类似于这样的函数定义，`ctx` 用来控制函数的退出，会在 `Cron` 触发时发送到 `outchan`

## Match `time`

到目前为止，我都没有写到具体是怎样的思路去实现「**获取 Cron 触发的时间**」。

「获取」Cron 触发时间关键在于「判断」时间是否符合某个 Cron 表达式。

并且，不管是 `weekday` or `day` or `month` or `hour` or `minute` 都是根据规则「枚举」出符合要求的值，解析出枚举过程放在 `parse` 函数中。

（_这里假设已经枚举出符合要求的一系列值_）

实现 `matches` 结构体：
```go
type Matches struct {
	minute  []int
	hour    []int
	day     []int
	month   []int
	weekday []int
}
```
实现 `Match` 方法，判断输入 `Time` 是否符合（触发）
```go
func (m Matches) Match(t time.Time) bool {
	contains := func(arr []int, val int) bool {
		for _, v := range arr {
			if v == val {
				return true
			}
		}
		return false
	}

	return contains(m.minute, t.Minute()) &&
		contains(m.hour, t.Hour()) &&
		contains(m.day, t.Day()) &&
		contains(m.month, int(t.Month())) &&
		contains(m.weekday, int(t.Weekday()))
}
```
很简单的实现，只要 `Time` 都分别在各种类型枚举值内就代表这个时间符合要求。

## Parse `rule`

`parse` 函数用于返回枚举值，然后保存在 `matches` 里，返回 `[]int`（简化了上下限的检查）。

```go
func parse(rule string, f field) ([]int, error) {
	if len(rule) == 0 {
		return nil, errors.New("empty spec")
	}

	specs := strings.Split(rule, ",")
	matches := make([]int, 0)
	low, high := f.limit()

	for _, spec := range specs {
		if spec == "*" {
			for i := low; i < high; i++ {
				matches = append(matches, i)
			}
		} else if strings.Contains(spec, "/") {
			...get step...
			...check...
			for i := low; i < high; i += step {
				matches = append(matches, i)
			}
		} else if strings.Contains(spec, "-") {
			...get start & end...
			...check...
			for i := start; i <= end; i++ {
				matches = append(matches, i)
			}
		} else {
			val, err := strconv.Atoi(spec)
			...check...
			matches = append(matches, val)
		}
	}

	return matches, nil
}
```

## Wrap `matches` 

因为使用表达式解析器的入口是 `Cronexpr`，所以 `matches` 应该是 `Cronexpr` 的字段。
```go
func NewExpr(expr string) *Cronexpr {
	return &Cronexpr{
		expr:     expr,
		matches:  newMatches(expr),
		accurate: time.Minute,
	}
}

func newMatches(expr string) Matches {
	splits := strings.Split(expr, " ")
	if len(splits) != 5 {
		return Matches{}
	}

	mustParse := func(s string, f field) []int {
		matches, err := parse(s, f)
		if err != nil {
			panic(err)
		}
		return matches
	}

	return Matches{
		minute:  mustParse(splits[0], minute),
		hour:    mustParse(splits[1], hour),
		day:     mustParse(splits[2], day),
		month:   mustParse(splits[3], month),
		weekday: mustParse(splits[4], weekday),
	}
}
```
这里利用辅助函数简化了代码（ps. `MustXxx` 在开源项目里很普遍）

`Cronexpr` 的 `accurate` 含义等下会解释

## Functions
到了使用的入口函数了，`Next()` or `NextN(n)` 需要实现这样一个方法 `nextN(n)`

通过 `matches` 我们知道了，我们可以判断某个 `Time` 是否符合 `Cronexpr` 的触发时间。

那 `nexnN(n)` 里，我们就需要对「未来」的时间进行枚举，然后通过 `matches` 判断是否符合。

这里还有一个问题，「未来时间」的间隔应该是多少呢，因为我们表达式里面的最小单位是 `minute`。

所以枚举时间的间隔应该是 `1 * minute`，其中 `Cronexpr.accurate` 就是代表这个。

> 为什么不写成常量呢？
> 
> 因为之后还可以实现支持 `second` 字段的 `Cronexpr`

这里还因为枚举未来时间因为需要一个基准(`zero time`)值，而这个时间对于 `nextN()` 这样的「最小」函数最好是可以外部传入的，所以加上了 `z time.Time`。

当然了，对于 `Next()` 和 `NextN()` 这样的函数 `zero time` 也是可以外部传入的，因为 `Next()` 这个语义并没有明显包括代表当前时间之后的含义。

所以 `nextN()` 应该是这样：
```go
func (e *Cronexpr) nextN(z time.Time, n int) []time.Time {
	ts := make([]time.Time, 0, n)
	lt := z

	for i := 0; i < n; i++ {
		n1 := e.next1(lt)
		ts = append(ts, n1)
		lt = n1
	}

	return ts
}
```

`nextN` 又去调用 `next1`，这是 `next1` 实现（这里 `next1` 的逻辑做成 `nextN` 的内部函数我感觉甚至更好）。
```go
func (e *Cronexpr) next1(z time.Time) time.Time {
	for t := z.Add(e.accurate); t.Before(END); t = t.Add(e.accurate) {
		if e.matches.Match(t) {
			return t
		}
	}

	return END
}
```

这里 `for range` 的时候初始值 `t` 加上 `accurate` 是为了防止如果 `z` 符合要求函数就会一直直接返回 `true`，导致 `nextN` 里的 `for` 循环就会返回同样的值。

`Notify` 函数实现主要是多了 `ctx deadline` 的判断退出，这里可以简单的使用 `time ticker` 来获取需要检查的时间点。
```go
func (e *Cronexpr) Notify(ctx context.Context, out chan<- time.Time) {
	ticker := time.NewTicker(e.accurate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			if e.matches.Match(t) {
				out <- t
			}
		}
	}
}
```

## Conclusion
`Scheduler` 的部分可以看源码，这部分实现比较简单。

实现 `Cronexpr` 解析器还是很有意思的，这是一个简单的小项目。

感谢阅读！

