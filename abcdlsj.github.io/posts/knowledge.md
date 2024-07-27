---
title: "Kenowledge"
date: 2024-03-31T22:30:59+08:00
tags:
  - Kenowledge
hide: true
wip: false
tocPosition: left-sidebar
---

## Golang

 https://www.yuque.com/aceld/golang 

### Questions

- 简述 slice 的底层原理，slice 和数组的区别是什么？
- 简单介绍 GMP 模型以及该模型的优点
- 简述 Golang 垃圾回收的机制
- 协程与进程，线程的区别是什么？协程有什么优势？ 
- 简述 defer 的执行顺序
- Golang 有哪些优缺点、错误处理有什么优缺点？
- 两次 GC 周期重叠会引发什么问题，GC 触发机制是什么样的？
- Golang 的协程通信方式有哪些？
- 简述 Golang 的伪抢占式调度
- 什么是 goroutine 泄漏
- Groutinue 什么时候会被挂起？
- Golang 标准库中 map 的底层数据结构是什么样子的？
- Map 的查询时间复杂度如何分析？
- 极端情况下有很多哈希冲突，Golang 标准库如何去避免最坏的查询时间复杂度？
- Golang map Rehash 的策略是怎样的？什么时机会发生 Rehash？
- Rehash 具体会影响什么？哈希结果会受到什么影响？
- Rehash 过程中存放在旧桶的元素如何迁移？

### 标准库

#### slice 和数组

https://go.dev/blog/slices-intro

slice 的底层数据是数组，slice 是对数组的封装，它描述一个数组的片段。两者都可以通过下标来访问单个元素。

数组是定长的，长度定义好之后，不能再更改。在 Go 中，数组是不常见的，因为其长度是类型的一部分，限制了它的表达能力，比如 `[3]int` 和 `[4]int` 就是不同的类型。

切片则可以动态地扩容。切片的类型和长度无关。

数组就是一片连续的内存，slice 实际上是一个结构体，包含三个字段：长度、容量、底层数组。

```go
// runtime/slice.go
type slice struct {
	array unsafe.Pointer // 元素指针
	len   int // 长度
	cap   int // 容量
}
```

slice 做切片不会有数据拷贝，只是修改 `array/len/cap`
##### growth

向切片中添加元素，如果 cap 不足，就会产生扩容 `len(slice) + len(data) > cap(slice)`

`necap = (len(slice) + len(data)) * 2`

其中
1. 如果期望 cap 大于当前 cap 的两倍就会使用期望 cap
2. 如果当前切片的长度小于 1024 就会将 cap 翻倍
3. 如果当前切片的长度大于 1024 就会每次增加 25% 的 cap，直到新 cap 大于期望 cap

最后按照内存对齐，向上修正

问题？
1. **不够平滑**：容量小于 1024 时 2 倍扩容，大于 1024 突然降到 1.25 倍扩容
2. **容量增长不单调**：正常应该是较大的初始容量扩容后有较大的最终容量

1.18 版本更新：
使用不同容量的扩容因子，小于 256，使用 2 倍，大于 256，逐步降低，直到 1.25 倍x

##### 二维切片的分配
https://go.dev/doc/effective_go#two_dimensional_slices

```go
// Allocate the top-level slice.
picture := make([][]uint8, YSize) // One row per unit of y.
// Loop over the rows, allocating the slice for each row.
for i := range picture {
    picture[i] = make([]uint8, XSize)
}
```

分配单个切片，然后循环中分配
```go
// Allocate the top-level slice, the same as before.
picture := make([][]uint8, YSize) // One row per unit of y.
// Allocate one large slice to hold all the pixels.
pixels := make([]uint8, XSize*YSize) // Has type []uint8 even though picture is [][]uint8.
// Loop over the rows, slicing each row from the front of the remaining pixels slice.
for i := range picture {
    picture[i], pixels = pixels[:XSize], pixels[XSize:]
}
```

#### map

- Golang 标准库中 map 的底层数据结构是什么样子的？
- Map 的查询时间复杂度如何分析？
- 极端情况下有很多哈希冲突，Golang 标准库如何去避免最坏的查询时间复杂度？
- Golang map Rehash 的策略是怎样的？什么时机会发生 Rehash？
- Rehash 具体会影响什么？哈希结果会受到什么影响？
- Rehash 过程中存放在旧桶的元素如何迁移？

https://www.cnblogs.com/qcrao-2018/p/10903807.html

hamp 结构

![map overview|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/golangmap_overview.6t6zf6bi43.webp)

##### key 定位

![map internal|400](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/golangmap_internal.26lceh82bk.webp)

key 经过哈希计算后得到哈希值，共 64 个 bit 位，计算它到底要落在哪个桶时，只会用到最后 B 个 bit 位。如果 B = 5，那么桶的数量，也就是 buckets 数组的长度是 2^5 = 32。

例如，现在有一个 key 经过哈希函数计算后，得到的哈希结果是：

> 10010111 | 000011110110110010001111001010100010010110010101010 │ 01010

用最后的 5 个 bit 位，也就是 01010，值为 10，也就是 10 号桶。在 bucket 中从前往后找到第一个空位，记录下哈希值的高 8 位，。查找某个 key 时，先找到对应的桶，再去遍历 bucket 中的 key
##### 扩容 & 迁移

插入新 key，会检测是否满足下面 2 个条件：
1. 装载因子（`loadFactor := count / (2^B)`）超过阈值，源码里定义的阈值是 6.5
2. overflow 的 bucket 数量过多：当 B 小于 15，也就是 bucket 总数 2^B 小于 2^15 时，如果 overflow 的 bucket 数量超过 2^B；当 B >= 15，也就是 bucket 总数 2^B 大于等于 2^15，如果 overflow 的 bucket 数量超过 2^15

非等量扩容，新的 key 会被分到新桶的前一半或者后一半（因为新桶数目是 2 倍）

渐进式扩容，添加/删除会判断是否正在扩容，会辅助迁移 **2 个桶**，「当前访问」的和「搬迁进度」`nevacuate` 桶（`nevacuate` 可能已经被迁移，会向后查找 1024 个桶），将所有 key（包括 overflow）都迁移到新桶里

##### race 检测 panic

在查找、赋值、遍历、删除的过程中都会检测写标志 flags，一旦发现写标志置位 (等于 1)，则直接 panic。赋值和删除函数载检测完标志是复位状态 (等于 0) 之后，先将写标志位置位，才会进行之后的操作

#### sync.Map
https://www.cnblogs.com/qcrao-2018/p/12833787.html

- sync.Map 比加锁的方案好在哪里，它的底层数据结构是怎样的？
- sync.Map 的 Load() 方法流程？
- sync.Map Store() 如何保持缓存层和底层 Map 数据是相同的? 是不是每次执行修改都需要去加锁？

##### 数据结构
![go sync map|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/go_sync_map.3d4nenqux9.webp)

`sync.Map` 数据结构
```go
type Map struct {
	mu Mutex
	read atomic.Value // readOnly
	dirty map[interface{}]*entry
	misses int
}
```

互斥量 mu 保护 read 和 dirty。

read 是 `atomic.Value` 类型，可以并发地读。但如果需要更新 read，则需要加锁保护。对于 read 中存储的 entry 字段，可能会被并发地 CAS 更新

dirty 是一个非线程安全的原始 map。包含新写入的 key，并且包含 read 中的所有未被删除的 key。这样，可以快速地将 dirty 提升为 read 对外提供服务。如果 dirty 为 nil，那么下一次写入时，会新建一个新的 dirty，这个初始的 dirty 是 read 的一个拷贝，但除掉了其中已被删除的 key

每当从 read 中读取失败，都会将 misses 的计数值加 1，当加到一定阈值以后，需要将 dirty 提升为 read，以期减少 miss 的情形

`readOnly` 数据结构
```go
// readOnly is an immutable struct stored atomically in the Map.read field.
type readOnly struct {
	m       map[interface{}]*entry
	amended bool // true if the dirty map contains some key not in m.
}
```

read 和 dirty 里存储的东西都包含 entry
```go
type entry struct {
	p unsafe.Pointer // *interface{}
}
```

- 当 p == nil 时，代表被删除，`dirty == nil` 或者 `dirty[k]` 指向该 entry
- 当 p == expunged 时，read 独有，代表被删除，`dirty != nil` 并且 `dirty` 中没有这个 key，当 `dirty->read` 完成后，又有新 key 写入时，此时 read 中的 amended 为 false，就会调用 dirtyLocked() 方法，此时会发生 `read->dirty` 的转变，此时会循环 read 数据，将 p 不为 nil 的值写到 dirty 中，如果 p 为 nil 则将 nil 转为 expunged
- 其他情况，p 指向一个正常的值，表示实际 interface{} 的地址，并且被记录在 `m.read.m[key]` 中。如果这时 m.dirty 不为 nil，那么它也被记录在 `m.dirty[key]` 中。两者实际上指向的是同一个值
##### Ops

1. `sync.map` 是线程安全的，读取，插入，删除也都保持着常数级的时间复杂度
2. 通过读写分离，降低锁时间来提高效率，read 相当于快照，key 集合不变但是 value 可以 CAS，提高读性能
3. 当 dirty 为 nil 的时候，read 就代表 map 所有的数据；当 dirty 不为 nil 的时候，dirty 才代表 map 所有的数据
4. Range 操作需要提供一个函数，参数是 `k,v`，返回值是一个布尔值：`f func(key, value interface{}) bool`
5. 调用 Load 或 LoadOrStore 函数时，如果在 read 中没有找到 key，则会将 misses 值原子地增加 1，当 misses 增加到和 dirty 的长度相等时，会将 dirty 提升为 read。以期减少读 miss
6. 新写入 key 会保存到 dirty 中，如果这时 dirty 为 nil，就会先新创建一个 dirty，并将 read 中未被删除的元素拷贝到 dirty

- Load
    - read 中有，直接读取返回，`p == 任何值` 都会返回
    - read 没有
      - `amended` 为 fasle，直接 false
      - `amended` 为 true，`misses++`，进入 dirty 中查找
- Store
    - read 当中存在
        - 不等于 `expunged`，则直接 CAS 更新，更新 `value`
        - 等于 `expunged`，将 p 修改为 nil，然后插入到 `m.dirty` 中，更新 `value`
    - dirty 中有，read 没有
        - 写入到 dirty
    - read 和 dirty 都没有
        - 创建 dirty，拷贝 read 未删除的到 dirty 中
        - 更新 read `amended` 字段，表示 dirty 有 read 中没有的 kv
        - 写入到 dirty 中
- Delete
    - read 找到 key，设置为 nil
    - 没有找到，并且 `amended` 为 true，进入 dirty 中查找，找到直接删除（map delete）
- For Range
    - 如果 `amended` 为 true，直接将 `dirty` 提升为 `read`，然后清空 `dirty`，然后遍历返回

#### channel

https://speakerd.s3.amazonaws.com/presentations/10ac0b1d76a6463aa98ad6a9dec917a7/GopherCon_v10.0.pdf

channel 中的读写都是发送的拷贝「副本」
##### hchan

```go
type hchan struct {
	// chan 里元素数量
	qcount   uint
	// chan 底层循环数组的长度
	dataqsiz uint
	// 指向底层循环数组的指针（只针对有缓冲的 channel）
	buf      unsafe.Pointer
	// chan 中元素大小
	elemsize uint16
	// chan 是否被关闭的标志
	closed   uint32
	// chan 中元素类型
	elemtype *_type // element type
	// 已发送元素在循环数组中的索引
	sendx    uint   // send index
	// 已接收元素在循环数组中的索引
	recvx    uint   // receive index
	// 等待接收的 goroutine 队列
	recvq    waitq  // list of recv waiters
	// 等待发送的 goroutine 队列
	sendq    waitq  // list of send waiters

	// 保护 hchan 中所有字段
	lock mutex
}
```
##### 读写 channel

| 操作      | nil channel | closed channel | not nil, not closed channel                           |
| ------- | ----------- | -------------- | ----------------------------------------------------- |
| close   | panic       | panic          | 正常关闭                                                  |
| 读 <- ch | 阻塞          | 读到对应类型的零值      | 阻塞或正常读取数据。缓冲型 channel 为空或非缓冲型 channel 没有等待发送者时会阻塞     |
| 写 ch <- | 阻塞          | panic          | 阻塞或正常写入数据。非缓冲型 channel 没有等待接收者或缓冲型 channel buf 满时会被阻塞 |

#### defer

https://go.cyub.vip/feature/defer.html

1. defer 函数是按照后进先出的顺序执行
2. defer 函数的传入参数在定义时就已经明确
3. defer 函数可以读取和修改函数的命名返回值

![go defer|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/go_defer_internal.6f0jd9wno1.webp)

> [!NOTE]
> 延迟调用的实参是在此延迟调用被推入延迟调用队列时被估值的
> 
> 函数体内的表达式是在此函数被执行的时候才会被逐渐估值的，不管此函数是被普通调用还是延迟/协程调用。

```go
package main

import "fmt"

func main() {
	fmt.Println(test1()) // 0, 0
	fmt.Println(test2()) // 3, 3
	fmt.Println(test3()) // 0, 4
	fmt.Println(test4()) // 0, 5
}

func test1() (v int) {
	defer fmt.Println(v)
	return v
}

func test2() (v int) {
	defer func() {
		fmt.Println(v)
	}()
	return 3
}

func test3() (v int) {
	defer fmt.Println(v)
	v = 3
	return 4
}

func test4() (v int) {
	defer func(n int) {
		fmt.Println(n)
	}(v)
	return 5
}
```

#### mutex

Go 语言的 `sync.Mutex` 由两个字段 `state` 和 `sema` 组成。其中 `state` 表示当前互斥锁的状态，而 `sema` 是用于控制锁状态的信号量。
```go
type Mutex struct {
	state int32
	sema  uint32
}
```

- 正常模式
    - 新来的 goroutine 在尝试获得锁时，会首先进行**自旋**，在一定次数之后，如果仍获取失败，则会进入等待队列
    - 当正在持有锁的 goroutine 释放锁后，并不会直接将锁传递给等待队列的第一个 goroutine。而是第一个尝试获得锁的 goroutine
    - 如果队列头部的 goroutine 获取失败，它并不会去到队列尾部，而是继续在头部等待
    - **非公平锁**，在竞争少的情况下，拥有**高吞吐量**，但是它会导致 goroutine 饥饿。而当队列中的 goroutine 的等待时间超过 1 ms 时，转变为饥饿模式
- 饥饿模式
    - 新来的 goroutine 直接到后面进行排队
    - 当前 goroutine 释放锁之后，它会将锁直接传递给位于等待队列头部的 goroutine
    - 而当队列全部清空，或者有一个 goroutine 的等待时间小于 1 ms 就获得了锁，**转变为正常模式**
    - **公平锁**，所有新来的 goroutine 都会排队，相比正常模式吞吐量下降，但是优化了线程饥饿

#### panic/recover

`panic` 能够改变程序的控制流，调用 `panic` 后会立刻停止执行当前函数的剩余代码，并在当前 Goroutine 中递归执行调用方的 `defer`；
`recover` 可以中止 `panic` 造成的程序崩溃。它是一个只能在 `defer` 中发挥作用的函数，在其他作用域中调用不会发挥作用；

- `panic` 只会触发当前 Goroutine 的 `defer`；
- `recover` 只有在 `defer` 中调用才会生效；
- `panic` 允许在 `defer` 中嵌套多次调用；

#### for range

语法糖
```
// The loop we generate:
//   for_temp := range
//   len_temp := len(for_temp)
//   for index_temp = 0; index_temp < len_temp; index_temp++ {
//           value_temp = for_temp[index_temp]
//           index = index_temp
//           value = value_temp
//           original body
//   }
```

### GMP

![gmp model|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/gmp_model.7sn2vqpy7g.webp)

- G - Goroutine，Go  协程，是参与调度与执行的最小单位
- M - Machine，指的是系统级线程
- P - Processor，指的是逻辑处理器，P 关联了的本地可运行 G 的队列 (也称为 LRQ)，最多可存放 256 个 G

> 线程 M 想运行任务就需得获取 P，即与 P 关联

1. 首先从 P 的本地队列 (LRQ) 获取 G
2. 若 LRQ 中没有可运行的 G，M 会尝试从全局队列 (GRQ) 拿一批 G 放到 P 的本地队列
3. 若全局队列也未找到可运行的 G 时候，M 会随机从其他 P 的本地队列偷一半放到自己 P 的本地队列
4. 拿到可运行的 G 之后，M 运行 G，G 执行之后，M 会从 P 获取下一个 G，不断重复下去

#### 数量

- G 的数量：理论上没有数量上限限制的。查看当前G的数量可以使用 `runtime.NumGoroutine()`
- P 的数量：由启动时环境变量 `$GOMAXPROCS` 或者是由 `runtime.GOMAXPROCS()` 决定。这意味着在程序执行的任意时刻都只有 `$GOMAXPROCS` 个 goroutine 在同时运行
- M 的数量：
	- go 语言本身的限制：go 程序启动时，会设置 M 的最大数量，默认 10000. 但是内核很难支持这么多的线程数，所以这个限制可以忽略
	- runtime/debug 中的 SetMaxThreads 函数，设置 M 的最大数量一个 M 阻塞了，会创建新的 M。M 与 P 的数量没有绝对关系，一个 M 阻塞，P 就会去创建或者切换另一个 M，所以，即使 P 的默认数量是 1，也有可能会创建很多个 M 出来

#### Goroutine

Goroutine 创建时函数参数是在当前 Goroutine 评估的，然后创建新的 Goroutine，拷贝到新的 Goroutine 中执行代码，最开始栈空间分配比较小，如果发现不够用，会产生中断，然后创建更大的空间，拷贝栈空间到新空间，然后恢复执行，执行完成，缩小空间

#### 调度

![gmp schedule|800](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/go_schedule.8ojkb73e5w.webp)

1. 我们通过 go func () 来创建一个 goroutine；
2. 有两个存储 G 的队列，一个是局部调度器 P 的本地队列、一个是全局 G 队列。新创建的 G 会先保存在 P 的本地队列中，如果 P 的本地队列已经满了就会保存在全局的队列中；
3. G 只能运行在 M 中，一个 M 必须持有一个 P，M 与 P 是 1：1 的关系。M 会从 P 的本地队列弹出一个可执行状态的 G 来执行，如果 P 的本地队列为空，就会想其他的 MP 组合偷取一个可执行的 G 来执行；
4. 一个 M 调度 G 执行的过程是一个循环机制；
5. 当 M 执行某一个 G 时候如果发生了 syscall 或则其余阻塞操作，M 会阻塞，如果当前有一些 G 在执行，runtime 会把这个线程 M 从 P 中摘除 (detach)，然后再创建一个新的操作系统的线程 (如果有空闲的线程可用就复用空闲线程) 来服务于这个 P；
6. 当 M 系统调用结束时候，这个 G 会尝试获取一个空闲的 P 执行，并放入到这个 P 的本地队列。如果获取不到 P，那么这个线程 M 变成休眠状态，加入到空闲线程中，然后这个 G 会被放入全局队列中。

### Memory
![go heap structure|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/go_heap_structure.7i09anzpss.webp)

> go 内存分配核心思想就是把内存分为多级管理，从而降低锁的粒度。它将可用的堆内存采用二级分配的方式进行管理：每个 P 都会自行维护一个独立的内存池，进行内存分配时优先从该内存池中分配，当内存池不足时才会向全局内存池申请，以避免不同线程对全局内存池的频繁竞争

#### mspan

`mspan` 由一组连续的页组成，按照一定大小划分成 `object`(8b -> 32k)

#### mcache

`mcache` 每个 P 绑定一个 mcache，本地缓存可用的 mspan 资源，这样就可以直接给 G 分配，因为不存在多个 G 竞争的情况，所以不会消耗锁资源

#### mcentral

`mcentral` 为所有 mcache 提供切分好的 mspan 资源。每个 central 保存一种特定大小的全局 mspan 列表，包括已分配出去的和未分配出去的。每个 mcentral 对应一种 mspan，而 mspan 的种类导致它分割的 object 大小不同。当工作线程的 mcache 中没有合适（也就是特定大小的）的 mspan 时就会从 mcentral 获取。

mcentral 被所有的工作线程共同享有，存在多个 Goroutine 竞争的情况，因此会消耗锁资源

#### mheap

代表 Go 程序持有的所有堆空间，Go 程序使用一个 mheap 的全局对象 mheap 来管理堆内存。

当 mcentral 没有空闲的 mspan 时，会向 mheap 申请。而 mheap 没有资源时，会向操作系统申请新内存。mheap 主要用于大对象的内存分配，以及管理未切割的 mspan，用于给 mcentral 切割成小对象

#### Allocate

Go 在程序启动时，会向操作系统申请一大块内存，之后自行管理

变量是在栈上分配还是在堆上分配，是由逃逸分析的结果决定的。通常情况下，编译器是倾向于将变量分配到栈上的，因为它的开销小，最极端的就是 `zero garbage`，所有的变量都会在栈上分配，这样就不会存在内存碎片，垃圾回收之类的东西。

Go 的内存分配器在分配对象时，根据对象的大小，分成三类：小对象（小于等于 16 B）、一般对象（大于 16 B，小于等于 32 KB）、大对象（大于 32 KB）。

大体上的分配流程：

- <=16B 的对象使用 mcache 的 tiny 分配器分配（会分配在一个 object 中）；
- > 32 KB 的对象，直接从 mheap 上分配；
- `(16B,32KB]` 的对象，首先计算对象的规格大小，然后使用 mcache 中相应规格大小的 mspan 分配；
	- 如果 mcache 没有相应规格大小的 mspan，则向 mcentral 申请
	- 如果 mcentral 没有相应规格大小的 mspan，则向 mheap 申请
	- 如果 mheap 中也没有合适大小的 mspan，则向操作系统申请

### GC

https://www.luozhiyun.com/archives/475

#### GC 触发
1. 主动触发，通过调用 runtime.GC 来触发 GC，此调用阻塞式地等待当前 GC 运行完毕
2. 被动触发，分为两种方式：
    - 使用系统监控（sysmon），当超过 forcegcperiod(2m) 没有产生任何 GC 时，强制触发 GC
    - 使用步调（Pacing）算法，判断当前 Heap 分配是否达到了阈值，其核心思想是控制内存增长的比例

#### Process

![go gc mark process|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/go_gc_mark_process.8ad4k8sizh.webp)

1. sweep termination（清理终止）
   1. 触发 `STW`，所有的 P（处理器） 都会进入 safe-point（安全点）
   2. 扫描任何未扫描的 span。只有在此 GC 周期在预期时间之前被强制执行时才会有未扫描的 span。
2. the mark phase（标记阶段）
   1. 将 `_GCoff` GC 状态改成 `_GCmark`，开启 Write Barrier（写入屏障）、mutator assists（协助线程），将根对象入队；在所有 P 启用写屏障之前，不能扫描任何对象，这是通过 `STW` 完成的。
   2. `start the world`，所有 mark workers（标记进程）和 mutator assists（协助线程）会开始并发标记内存中的对象。对于任何指针写入和新的指针值，都会被写屏障覆盖，而所有新创建的对象都会被直接标记成黑色；
   3. GC 执行根节点的标记，这包括扫描所有的栈、全局对象以及不在堆中的运行时数据结构。扫描 goroutine 栈绘导致 goroutine 停止，并对栈上找到的所有指针加置灰，然后继续执行 goroutine。
   4. GC 在遍历灰色对象队列的时候，会将灰色对象变成黑色，并将该对象指向的对象置灰；
   5. GC 会使用分布式终止算法（distributed termination algorithm）来检测何时不再有根标记作业或灰色对象，如果没有了 GC 会转为 mark termination（标记终止）；
3. mark termination（标记终止）
   1. `STW`
   2. 将 gcphase 设置为 `_GCmarktermination`，关闭 GC 工作线程以及 mutator assists（协助线程）
   3. 执行清理，如 `flush mcache`
4. mark termination（标记终止）
   1. 将 GC 状态转变至 `_GCoff`，初始化清理状态并关闭 Write Barrier（写入屏障）；
   2. `start the world`。从这一点开始，新分配的对象是白色的，并在使用之前对分配进行扫描。
   3. 后台并发清理所有的内存管理单元
5. 当进行了充分的分配后，重放从上述 1 开始的序列

> 因为 GC 标记的工作是分配 25% 的 CPU 来进行 GC 操作，所以有可能 GC 的标记工作线程比应用程序的分配内存慢，导致永远标记不完，那么这个时候就需要应用程序的线程来协助完成标记工作

#### 并发清扫

清扫阶段与正常程序执行同时进行。堆按 span 逐个进行清扫，既懒惰地（当 goroutine 需要另一个 span 时）也在后台 goroutine 中并发进行（这有助于那些不受 CPU 限制的程序）。在 STW 标记终止结束时，所有 span 都被标记为「需要清扫」

> 后台 goroutine 逐一清扫 span

为了避免在有未清扫的 span 时请求更多的 OS 内存，当一个 goroutine 需要另一个 span 时，它首先尝试回收那么多内存通过清扫
- 当 goroutine 需要从堆中分配新的小对象 span 时，它为相同对象大小的小对象 span 进行清扫，直到至少释放一个对象
- 当 goroutine 需要从堆中分配大对象 span 时，它清扫 span 直到至少释放那么多页到堆中

有一个情况下这可能不够：如果 goroutine 清扫并释放了两个非相邻的一页 span 到堆中，它将分配一个新的两页 span，但仍可能有其他未清扫的一页 span 可以合并为一个两页 span

至关重要的是确保在未清扫的 span 上不进行任何操作（这将损坏 GC 位图中的标记位）。

在 GC 期间，所有 mcaches 都被刷新到中央缓存中，因此它们是空的
- 当 goroutine 抓取一个新的 span 到 mcache 时，GC 会清理它
- 当 goroutine 显式释放一个对象或设置一个 finalizer 时，GC 会确保 span 被清理（通过清理，或者通过等待并发清理来实现）
  
仅当所有 span 都被清扫时，finalizer goroutine 才会被启动。当下一次 GC 开始时，它将清扫所有尚未清扫的 span（如果有）

#### GC 速率

当所分配的堆大小达到一定比例（由控制器计算的触发堆的大小）时，将会触发 GC，具体的公式是已经使用的内存量和额外使用内存量的比例，比例由 GOGC 环境变量控制（默认为 100）

如果 GOGC=100 并且我们正在使用 4M，我们将达到 8M 时再次进行 GC（该标记在 gcController.heapGoal 变量中进行跟踪）

这可以保持 GC 的成本和分配内存的成本保持线性比例。调整 GOGC 只会改变线性常数（以及使用的额外内存量）

#### Oblets

为了防止在清理大型对象时出现长时间的停顿，同时为了提高并行性，GC 将大于 maxObletBytes 的对象的清理工作分解为最多为 maxObletBytes 的 oblets。

当清理遇到一个大对象的开头时，它只清理第一个 oblet 并将剩余的 oblet 作为新的清理作业排入队列

#### 三色抽象
- 白色对象 — 潜在的垃圾，其内存可能会被垃圾收集器回收；
- 黑色对象 — 活跃的对象，包括不存在任何引用外部指针的对象以及从根对象可达的对象；
- 灰色对象 — 活跃的对象，因为存在指向白色对象的外部指针，垃圾收集器会扫描这些对象的子对象；

![go gc|400](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/golang_gc_obj_pic.2krrubepqx.webp)

在垃圾收集器开始工作时，程序中不存在任何的黑色对象，垃圾收集的根对象会被标记成灰色，垃圾收集器只会从灰色对象集合中取出对象开始扫描，当灰色集合中不存在任何对象时，标记阶段就会结束。
#### 屏障技术
https://readit.site/a/6eFeD/17503-eliminate-rescan.md

<https://liqingqiya.github.io/golang/gc/%E5%9E%83%E5%9C%BE%E5%9B%9E%E6%94%B6/%E5%86%99%E5%B1%8F%E9%9A%9C/2020/07/24/gc5.html>

想要在并发或者增量的标记算法中保证正确性，我们需要达成以下两种三色不变性（Tri-color invariant）中的一种：

- 强三色不变性 — 黑色对象不会指向白色对象，只会指向灰色对象或者黑色对象；
- 弱三色不变性 — 黑色对象指向的白色对象必须包含一条从灰色对象经由多个白色对象的可达路径；

垃圾收集中的屏障技术更像是一个钩子方法，它是在用户程序读取对象、创建新对象以及更新对象指针时执行的一段代码，根据操作类型的不同，我们可以将它们分成读屏障（Read barrier）和写屏障（Write barrier）两种，因为读屏障需要在读操作中加入代码片段，对用户程序的性能影响很大，所以编程语言往往都会采用写屏障保证三色不变性。

##### Dijkstra 插入写屏障

操作：在 A 对象引用 B 对象的时候，B 对象被标记为灰色。(将 B 挂在 A 下游，B 必须被标记为灰色)

满足：强三色不变式 (不存在黑色对象引用白色对象的情况了，因为白色会强制变成灰色)

```go
writePointer(slot, ptr):
    shade(ptr)
    *slot = ptr
```
每当执行类似 `*slot = ptr` 的表达式时，我们会执行上述写屏障通过 shade 函数尝试改变指针的颜色。如果 ptr 指针是白色的，那么该函数会将该对象设置成灰色，其他情况则保持不变。
Dijkstra 插入屏障的好处在于可以立刻开始并发标记。但存在两个缺点：
1. 由于 Dijkstra 插入屏障的「保守」，在一次回收过程中可能会残留一部分对象没有回收成功，只有在下一个回收过程中才会被回收；
2. 在标记阶段中，每次进行指针赋值操作时，都需要引入写屏障，这无疑会增加大量性能开销；为了避免造成性能问题，Go 团队在最终实现时，没有为所有栈上的指针写操作，启用写屏障，而是当发生栈上的写操作时，将栈标记为灰色，将会需要标记终止阶段 STW 时对这些栈进行重新扫描。

##### Yuasa 删除写屏障

操作：被删除的对象，如果自身为灰色或者白色，那么被标记为灰色

满足：弱三色不变式 (保护灰色对象到白色对象的路径不会断)

```go
writePointer(slot, ptr)
    shade(*slot)
    *slot = ptr
```
会在老对象的引用被删除时，将白色的老对象涂成灰色，这样删除写屏障就可以保证弱三色不变性，老对象引用的下游对象一定可以被灰色对象引用。

Yuasa 屏障在标记开始时需要 STW 来扫描或快照堆栈，因为删除屏障同样不被应用与对栈指针的写入操作上，故初始栈指针指向的堆节点不能被 `*slot` 保护到，需要被提前保护

Yuasa 删除屏障的优势则在于不需要标记结束阶段的重新扫描，结束时候能够准确的回收所有需要回收的白色对象。缺陷是 Yuasa 删除屏障会拦截写操作，进而导致波面的退后，产生「冗余」的扫描。
删除写屏障（基于起始快照的写屏障）有一个前提条件，就是起始的时候，把整个根部扫描一遍，让所有的可达对象全都在灰色保护下（根黑，下一级在堆上的全灰），之后利用删除写屏障捕捉内存写操作，确保弱三色不变式不被破坏，就可以保证垃圾回收的正确性。

##### 混合写屏障

具体操作:
1. GC 开始将栈上的对象全部扫描并标记为黑色(之后不再进行第二次重复扫描，无需 STW)
2. GC 期间，任何在栈上创建的新对象，均为黑色
3. 被删除的对象标记为灰色
4. 被添加的对象标记为灰色

满足: 变形的弱三色不变式

论文里的伪代码：
```go
writePointer(slot, ptr):
    shade(*slot)
    if current stack is grey:
        shade(ptr)
    *slot = ptr
```

golang 实际实现的伪代码：
```go
writePointer(slot, ptr):
    shade(*slot)
    shade(ptr)
    *slot = ptr
```
1. 混合写屏障继承了插入写屏障的优点，起始无需 STW 打快照，直接并发扫描垃圾即可；
2. 混合写屏障继承了删除写屏障的优点，赋值器是黑色赋值器，扫描过一次就不需要扫描了，这样就消除了插入写屏障时期最后 STW 的重新扫描栈；
3. 混合写屏障扫描精度继承了删除写屏障，比插入写屏障更低，随着带来的是 GC 过程全程无 STW；
4. 混合写屏障扫描栈虽然没有 STW，但是扫描某一个具体的栈的时候，还是要停止这个 goroutine 赋值器的工作的（针对一个 goroutine 栈来说，是暂停扫的，要么全灰，要么全黑哈，原子状态切换），GC 开始将栈上的对象全部扫描，并将全部可达对象标记为黑色（之后不再进行第二次重复扫描，无需 STW），GC 期间，任何在栈上创建的新对象，均为黑色。

##### 总结
> 自我总结

因为 STW 影响性能，为了缩短 STW 时间使用并发三色标记法，防止并发写堆/栈操作影响，添加了写屏障（读屏障影响太大）
插入写屏障保障了强三色不变性，将修改的目标指针赋值为灰色，保障黑色只会指向灰色，比较保守，A->C 改为 A->B，C 指针可能会下次才会被回收，并且因为写屏障不对栈内存开启，栈新增对象都用灰色，所以结束后需要再次 rescan 栈
删除写屏障保障了弱三色不变性，删除时将被删除引用的指针赋值为灰色，同样不保护栈，需要开始前 STW 预先快照堆栈

混合写屏障，则栈新增变量都用黑色，不需要最后再次扫描，精度不高，但是 GC 过程没有 STW（不包括 STW 准备和收尾工作）
#### Mark Assist
目前的 Go 实现中，当 GC 触发后，会首先进入并发标记的阶段。并发标记会设置一个标志，并在 mallocgc 调用时进行检查。当存在新的内存分配时，会暂停分配内存过快的那些 goroutine，并将其转去执行一些辅助标记（Mark Assist）的工作，从而达到放缓继续分配、辅助 GC 的标记工作的目的。

编译器会分析用户代码，并在需要分配内存的位置，将申请内存的操作翻译为 mallocgc 调用，而 mallocgc 的实现决定了标记辅助的实现，其伪代码思路如下：
```go
func mallocgc(t typ.Type, size uint64) {
	if enableMarkAssist {
		// 进行标记辅助，此时用户代码没有得到执行
		(...)
	}
	// 执行内存分配
	(...)
}
```

### How to avoid GC

> [!NOTE]
> Go 里每个 P 提供了本地线程缓存（Local Thread Cache）称作 mcache，P 需要内存能直接从 mcache 中获取，由于在同一时间只有一个 Goroutine 运行在 p 上，所以中间不需要任何锁的参与
> 
> 每个大小的内存规格都有两种 scan 和 noscan，noscan 代表不含有指针的对象，noscan GC 时不需要扫描

1. 避免使用指针，例如 string(string header 里有指针)、time.Time、string key 的 map、slice values 的 map
2. 减少分配内存，例如 `func (r *Reader) Read(buf []byte) (int, error)`，传入 buf 更好
3. `sync.Pool` 复用对象，减少新对象的分配（`sync.Pool` 里的数据会随时被回收，需要区分场景，适合作为 buf 使用）
4. 预分配，防止扩容浪费造成复制

### CGO

https://www.cockroachlabs.com/blog/the-cost-and-complexity-of-cgo/


## Redis

https://www.xiaolincoding.com/redis/

https://medium.com/nerd-for-tech/understanding-redis-in-system-design-7a3aa8abc26a

https://github.com/gyoomi/redis/blob/master/src/main/java/com/gyoomi/redis/note/0_%E5%BC%80%E7%AF%87%EF%BC%9A%E6%8E%88%E4%BA%BA%E4%BB%A5%E9%B1%BC%E4%B8%8D%E8%8B%A5%E6%8E%88%E4%BA%BA%E4%BB%A5%E6%B8%94%20%E2%80%94%E2%80%94%20Redis%20%E5%8F%AF%E4%BB%A5%E7%94%A8%E6%9D%A5%E5%81%9A%E4%BB%80%E4%B9%88%EF%BC%9F.md

### 数据结构 

#### dict rehash

```c
typedef struct dictht {
    //哈希表数组
    dictEntry **table;
    //哈希表大小
    unsigned long size;  
    //哈希表大小掩码，用于计算索引值
    unsigned long sizemask;
    //该哈希表已有的节点数量
    unsigned long used;
} dictht;
```

dict 存储 key->value，当容量不够时会扩容 rehash

redis 采用渐进式 hash，底层挂载 2 个 hashtable，hashtable 底层分桶存储（先分桶，然后链表串起来）

底层会存储 rehashindex 代表搬迁进度，del/set/get 操作判断到当前正在扩容会先辅助扩容，将 rehashindex 位置的数据搬迁到新的 hashtable 中，并且 get/set/del 操作会在两个 hashtable 中进行，查找会在 ht[0] 先进行，没找到会继续查找 ht[1]，添加会在 ht[1] 中进行，删除和更新依赖查找

还会在周期函数中定时执行辅助 rehash（周期函数默认 1s）

**带来的问题？**
rehash 过程中内存占用翻倍

### Event loop

- Redis 主要的处理流程包括接收请求、执行命令，以及周期性地执行后台任务（serverCron），这些都是由这个事件循环驱动的。
- 当请求到来时，I/O 事件被触发，事件循环被唤醒，根据请求执行命令并返回响应结果；
- 同时，后台异步任务（如回收过期的 key）被拆分成若干小段，由 timer 事件所触发，夹杂在 I/O 事件处理的间隙来周期性地运行。
- 这种执行方式允许仅仅使用一个线程来处理大量的请求，并能提供快速的响应时间。当然，这种实现方式之所以能够高效运转，除了事件循环的结构之外，还得益于系统提供的异步的 I/O 多路复用机制 (I/O multiplexing)。
- 事件循环利用 I/O 多路复用机制，对 CPU 进行时分复用 (多个事件流将 CPU 切割成多个时间片，不同事件流的时间片交替进行)，使得多个事件流就可以并发进行。
- 而且，使用单线程事件机制可以避免代码的并发执行，在访问各种数据结构的时候都无需考虑线程安全问题，从而大大降低了实现的复杂度。

### 6.0 多线程 

https://cloud.tencent.com/developer/article/1940123

![redis compare single/multiplethead|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/redis_multithread.41xwwvxj5e.webp)

网络 I/O 读写使用多线程，执行部分单线程，并发读写 IO 的时候，主线程会阻塞

等待读队列没满的时候，主线程直接将 socket 读放入等待队列里。写 socket 完成后，等待队列清空
### Cluster
https://redis.io/docs/reference/cluster-spec/#main-properties-and-rationales-of-the-design

#### Codis

主要解决 redis 单点和水平扩展的问题

codis 使用 proxy 协调请求，默认分为 1024 个槽位（可以增加），使用 zk/etcd 在集群中同步持久化 meta 信息（槽位信息）
扩容 redis 实例时，会把需要迁移的槽位的 key 都复制到新的 redis 实例上，魔改的单 key 原子迁移，比较慢
如果当前访问的 key 所属槽位正在迁移，就会先强制迁移当前 key 到新的 redis 实例上，然后再执行操作

优点：扩缩容简单，proxy 服务本身是无状态的，不会耦合数据存储，只做分布式逻辑，etcd 存储节点数据
缺点：增加一层 proxy->redis 集群的耗时，魔改了 redis，没有最新支持。涉及主从同步，还需要部署哨兵节点。dashboard/proxy/group 部署复杂

#### Redis cluster
redis cluster 则是去中心化的，使用 smartclient 来路由请求到不同的 redis 实例，如果集群发生了变更导致 key 移动了，会返回告知 client 正确的 redis 实例 (所以扩缩容的情况延迟会高一些)

redis cluster 节点之间使用 gossip 协议通信

优点：一层请求，延迟低（除非集群变更），支持新版本 redis，all in one，部署简单
缺点：sdk 要更新，扩缩容没有 codis 简单，并且同步迁移 key 是比较慢的

### 持久化（RDB & AOF）

#### AOF 

AOF 会记录每个写操作

WAL（Write Ahead Log）：先写日志，再写数据，一般记录修改后的数据
AOF: Append Only File：先写数据，再写日志，Redis 收到的每一条写命令

好处：
- 避免出现记录错误命令的情况
- 不阻塞当前操作
风险：
- 日志丢失
- AOF 日志也是在主线程中执行的，阻塞下一个命令

写回策略
- Always，同步写回：每个写命令执行完，立马同步地将日志写回磁盘（高可靠性）（每次写入 AOF 文件数据后，就执行 `fsync()` 函数）
- No，操作系统控制的写回：每个写命令执行完，只是先把日志写到 AOF 文件的内存缓冲区，由操作系统决定何时将缓冲区内容写回磁盘（高性能）（永不执行 `fsync()` 函数）
- Everysec，每秒写回：每个写命令执行完，只是先把日志写到 AOF 文件的内存缓冲区，每隔一秒把缓冲区中的内容写入磁盘（创建一个异步任务来执行 `fsync()` 函数）

##### AOF 重写

到达阈值后，压缩 AOF 文件，**读取当前数据库中的所有键值对**，然后将每一个键值对用一条命令记录到「**新的 AOF 文件**」，等到全部记录完后，就将新的 AOF 文件替换掉现有的 AOF 文件（使用新的 AOF 文件，避免重写流程失败，污染现有的 AOF 文件）

##### AOF 后台重写

- 子进程进行 AOF 重写期间，主进程可以继续处理命令请求，从而避免阻塞主进程；
- 子进程带有主进程的数据副本（copy on write）
	- 多线程之间会共享内存，修改共享内存数据需要通过加锁，降低性能）
	- 创建子进程时，父子进程是共享内存数据的，不过这个共享的内存只能以只读的方式，而当父子进程任意一方修改了该共享内存，就会发生「写时复制」，操作写的一方分配新内存并拷贝数据，于是父子进程就有了独立的数据副本，就不用加锁来保证数据安全。

有两个阶段会导致阻塞父进程：
- 创建子进程的途中，由于要复制父进程的页表等数据结构，阻塞的时间跟页表的大小有关，页表越大，阻塞的时间也越长；
- 创建完子进程后，如果子进程或者父进程修改了共享数据，就会发生写时复制，这期间会拷贝物理内存，如果内存越大，自然阻塞的时间也越长；

重写 AOF 日志过程中，如果主进程修改了已经存在 key-value，此时这个 key-value 数据在子进程的内存数据就跟主进程的内存数据不一致了，这时要怎么办呢？

在 `bgrewriteaof` 子进程开始后，使用 AOF 重写缓冲区，在重写 AOF 期间，当 Redis 执行完一个写命令之后，它会同时将这个写命令写入到 「AOF 缓冲区」和 「AOF 重写缓冲区」。

![redis aof rewrite|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/redis_aof_rewrite.2velsqlblv.webp)

也就是说，在 bgrewriteaof 子进程执行 AOF 重写期间，主进程需要执行以下三个工作:

1. 执行客户端发来的命令；
2. 将执行后的写命令追加到 「AOF 缓冲区」；
3. 将执行后的写命令追加到 「AOF 重写缓冲区」；

当子进程完成 AOF 重写工作（扫描数据库中所有数据，逐一把内存数据的键值对转换成一条命令，再将命令记录到重写日志）后，会向主进程发送一条信号，信号是进程间通讯的一种方式，且是异步的

主进程收到该信号后，会调用一个信号处理函数，该函数主要做以下工作：
- 将 AOF 重写缓冲区中的所有内容追加到新的 AOF 的文件中，使得新旧两个 AOF 文件所保存的数据库状态一致
- 新的 AOF 的文件进行改名，覆盖现有的 AOF 文件

#### RDB
内存快照，redis 某一时刻的数据状态以文件的形式写到磁盘上，做数据恢复时，可以直接把文件数据读入内存

Redis 提供了两个命令来生成 RDB 文件，分别是 save 和 bgsave，他们的区别就在于是否在「主线程」里执行：

- 执行了 save 命令，就会在主线程生成 RDB 文件，由于和执行操作命令在同一个线程，所以如果写入 RDB 文件的时间太长，会阻塞主线程；
- 执行了 bgsave 命令，会创建一个子进程来生成 RDB 文件，这样可以避免主线程的阻塞；
### 过期策略

主动过期+被动过期

1. 被动过期：只有当访问某个 key 时，才判断这个 key 是否已过期，如果已过期，则从实例中删除
2. 主动过期：Redis 内部维护了一个定时任务，定时从全局的过期哈希表中随机取出 20 个 key，然后删除其中过期的 key，如果过期 key 的比例超过了 25%，则继续重复此过程，直到过期 key 的比例下降到 25% 以下，或者这次任务的执行耗时超过了 25 毫秒，才会退出循环
### 问题

#### 穿透/雪崩/击穿

- 缓存穿透：是指恶意请求或者不存在的 key 频繁访问缓存，数据库数据不存在也无法写回，导致请求直接绕过缓存访问数据库，增加数据库负担。解决方法包括使用布隆过滤器、空值缓存等
- 缓存击穿：是指某个热点 key 过期时，大量请求同时访问该 key，导致缓存失效，请求直接访问数据库。解决方法包括设置较长有效期、使用互斥锁等
- 缓存雪崩：是指大量缓存数据同时失效或过期，导致大量请求直接访问数据库，造成数据库压力激增，甚至导致系统崩溃。解决方法包括设置不同的过期时间、二级缓存、使用熔断机制等

#### 延迟高的原因

- 集中过期
- 网络 IO 的问题，带宽过载

#### 热点 key 问题

- 导致 redis 节点挂掉
- 缓存击穿，影响 DB
- 占有过高主机带宽，导致机器其它服务问题

SellerGateway 使用 Redis 实现分布式令牌桶限流算法，发现某台机器evalsha1 出现最多次

是请求用户本身 QPS 比较高

解决方案：
1. 本地一次拉取多个 token，然后不够了再去拉取
2. 分片限流，每秒限流改为每 100ms 一个 key
2. 多级限流，流量小分布式限流，流量高单机限流，需要动态更新单机限流值

当时的解决方案：
错误的解决方案，只分片了 lua script，实际上还是同一台 redis


SellerGateway 的 API Cache 也是单个 key，随着业务扩大，导致 redis 节点的流量不平衡（占用 4mb，随机 10~30 秒去拉取一次，总共 4mb * 150 * 4worker，最高 240MB/s 了，随着业务扩大，还会增大）

临时解决，将 worker 更新间隔扩大，减少一倍流量
解决方案：根据 API group 划分 key，更新代码，为了灰度期间不影响业务，做 API 发布的时候双写 key
### Cache/DB consistency

1. 想要提高应用的性能，可以引入「缓存」来解决
2. 引入缓存后，需要考虑缓存和数据库一致性问题，可选的方案有：「更新数据库 + 更新缓存」、「更新数据库 + 删除缓存」
3. 更新数据库 + 更新缓存方案，在「并发」场景下无法保证缓存和数据一致性，解决方案是加「分布锁」，但这种方案存在「缓存资源浪费」和「机器性能浪费」的情况
4. 采用「先删除缓存，再更新数据库」方案，在「并发」场景下依旧有不一致问题，解决方案是「延迟双删」，但这个延迟时间很难评估
5. 采用「先更新数据库，再删除缓存」方案，为了保证两步都成功执行，需配合「消息队列」或「订阅变更日志」的方案来做，本质是通过「重试」的方式保证数据最终一致
6. 采用「先更新数据库，再删除缓存」方案，「读写分离 + 主从库延迟」也会导致缓存和数据库不一致，缓解此问题的方案是「延迟双删」，凭借经验发送「延迟消息」到队列中，延迟删除缓存，同时也要控制主从库延迟，尽可能降低不一致发生的概率

### Redis Lua 脚本

`EVAL script numkeys [key [key ...]] [arg [arg ...]]`

`SCRIPT LOAD` 添加 lua 脚本会用 sha 1 算法生成一个 id，可以使用 evalsha 来执行

或者 EVAL 执行过一次之后也会生成 sha 1 id

> [!NOTE]
> 单个 Lua 脚本操作的 key 必须在同一个节点上

### 缺/优点

纯内存操作，快，需要淘汰超过内存限制的数据，可以使用 qdb 之类的东西，超过内存限制的存储在 LSM-Tree 里（类似冷热分离）

### Use Cases

多级缓存，内存缓存会有数据一致性的问题，可以通过消息队列来更新内存缓存，但是只能用于一致性不敏感的地方，比如评论之类的

api_cache
- partner_id % 100 -> randomint 打散 （partner_id 实际上也会存在单点）

修改频率比较低，增加内存缓存，并且任务定时更新

seller centre 的后台更新 sidebar 缓存

#### Distributed Lock
获取锁：setnx key value expire（超时时间防止异常没有 del 锁）
释放锁：del key

**超时问题**
如果获取锁到释放锁之间时间过长，超过锁的 ttl，被再次持有（违背互斥条件），并且可能被误删除

setnx 的时候添加「唯一值」，delete 的时候 delete if `value == 唯一值`（用 lua 脚本实现原子性）

Shopee DLock 的实现：
- Mutex 和上面一样，加上了「Renew」功能（和 delete 一样，需要先 get 判断 value 是否相等，然后 `PEXPIRE`）
- Semaphore（Accquire 和 Release），使用 zset 实现
- XLock/SLock，使用 zset 实现 X/S Lock 2 个队列，FIFO 实现
#### zset
push 成功率计算（六个小时成功率）

通过 timestamp 作为 score 高位，自然有序，每 5 分钟一个 window 计数
`<member, score> => <member: timestamp / 300, score: member << 30>`

最后计算当前成功率是：

获取全部数据 `zrangebyscore xxx（lastborder) (lastborder + 1)`
`score & 0x3fffffff` 取低 32 位，然后累加

删除过期的 record 也很方便（当前存储 6 个小时）

好处：统计数据直接是排序的，获取返回很容易，有变更也很简单，hash 结构需要一个个 hget
坏处：zset 本身额外的内存占用，在这里不需要排序，所以时间复杂度忽略不计

1. 计数，如上
2. 排名
3. 延迟任务列表
#### hash
featuretoggle 的 cache 都是使用的 hash，减少缓存删除的时候的 workload，只删除部分

openapi 统计 api 调用用的 hash
`api_path <-> call count`

#### Rate Limit

- 固定窗口计数器
- 滑动窗口计数器
- 漏桶
- 令牌桶

SellerGateway 的实现
Hash 结构，先检查 key 是否存在，不存在就初始化 brust

获取其中上次填充的时间
- 计算当前与上次填充的时间间隔
- 如果时间间隔大于 interval（1 分钟），就重新填充 brust tokens
- 否则，计算填充的 tokens，加到 tokens 里面

最后检查是否还有 tokens 供获取，如果没有，则返回 0，否则返回 1
## MySQL DB

https://www.xiaolincoding.com/mysql/

- 索引：B+ tree 索引、哈希索引、全文索引、覆盖索引等
- 存储引擎简单介绍，如 InnoDB, MyISAM
- 主从复制
- 事务特性简单介绍

## Kafka MQ
https://jack-vanlightly.com/blog/2023/11/14/the-architecture-of-serverless-data-systems

https://mdnice.com/writing/c1d01d8793154629a82a9eb1bc0d1318

https://engineering.linkedin.com/kafka/benchmarking-apache-kafka-2-million-writes-second-three-cheap-machines


![kafka producer write|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/kaf_producer_write.13ln5f5s7x.webp)

![kafka consumer read|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/image.2h869gj07a.webp)

### 索引
InnoDB 的索引是以 B+树的结构存储的，一颗 M 阶（M≥3）的 B+树有以下特点：

- m 个 Key 的内部节点有 m+1 颗子树，Key 在节点内有序排列，Key 左边的子树的所有值小于 Key 值
- 叶节点的容量必须半满以上，即节点内 key 的个数在 `M/2(向上取整)~M` 之间, 叶子节点在同一层，并用指针连接成链表
- N 个节点的 M 阶的 B+ Tree，高度为 logm(N/2), 二叉树的高度为 log2(N)
	- 例: 实际应用中 M 值可能达到 500，如果 B 树为 3 层，则最少可以容纳 `250*250*250=15,625,000`  个节点，而容纳同样多节点的二叉树高度超过 23 层（log 215625000），对于查询操作， B+树节省了昂贵的磁盘 IO (7500 转 SATA 硬盘 1 个随机 IO 耗费 8~10 ms，内存操作是纳秒级别)
- B+ Tree 索引只能找到 key 值所属的范围，要找到具体的 key 需要在节点内继续做二分查找 (在内存中)
- B+ Tree 叶节点存储的数据是以 InnoDB 的页 (16 KB) 为单位，由于 B+ Tree 的数据结构特点，至少半满 M/2(M≥3), 所以一个叶子节点最少存放两条数据。这就是推荐一条记录小于 8 K 的原因，大于 8 K  需要额外的存储空间，增加 IO 次数

https://use-the-index-luke.com/sql/anatomy/the-tee

在创建表时，InnoDB 存储引擎会根据不同的场景选择不同的列作为**聚簇索引**的索引键：
- 如果有主键，默认会使用主键作为聚簇索引的索引键
- 如果没有主键，就选择第一个不包含 NULL 值的唯一列作为聚簇索引的索引键
- 在上面两个都没有的情况下，InnoDB 将自动生成一个隐式自增 id 列作为聚簇索引的索引键

其它索引都属于辅助索引，也称为二级索引或非聚簇索引。创建的主键索引和二级索引默认使用的是 B+Tree 索引

主键索引的 B+Tree 和二级索引的 B+Tree 区别如下：
- 主键索引的 B+Tree 的叶子节点存放的是实际数据，所有完整的记录都存放在主键索引的 B+Tree 的叶子节点里
- 二级索引的 B+Tree 的叶子节点存放的是主键值，而不是实际数据
- 获取主键值，然后再通过主键索引中的 B+Tree 树查询到对应的叶子节点，然后获取整行数据。这个过程叫「**回表**」，也就是说要查两个 B+Tree 才能查到数据

- 从物理存储的角度来看，索引分为聚簇索引（主键索引）、二级索引（辅助索引）
- 从字段特性的角度来看，索引分为主键索引、唯一索引、普通索引、前缀索引
- 从字段个数的角度来看，索引分为单列索引、联合索引（复合索引）
    - 建立在单列上的索引称为单列索引，比如主键索引
    - 建立在多列上的索引称为联合索引

#### 联合索引
https://www.xiaolincoding.com/mysql/index/index_interview.html#%E8%81%94%E5%90%88%E7%B4%A2%E5%BC%95%E8%8C%83%E5%9B%B4%E6%9F%A5%E8%AF%A2

1. 使用联合索引时，存在**最左匹配原则**，也就是按照最左优先的方式进行索引的匹配（联系 B+ Tree 特性）
2. 联合索引的最左匹配原则会一直向右匹配直到遇到「范围查询」就会停止匹配。范围查询的字段可以用到联合索引，但是在范围查询字段之后的字段无法用到

我们也可以在执行计划中的 key_len 知道这一点，在使用联合索引进行查询的时候，通过 key_len 我们可以知道优化器具体使用了多少个字段的搜索条件来形成扫描区间的边界条件。

> 联合索引的最左匹配原则，在遇到范围查询（如 >、<）的时候，就会停止匹配，也就是范围查询的字段可以用到联合索引，但是在范围查询字段的后面的字段无法用到联合索引
> 
> 注意，对于 >=、<=、BETWEEN、like 前缀匹配的范围查询，并不会停止匹配

#### 索引下推

对于联合索引（a, b），在执行 `select * from table where a > 1 and b = 2` 语句的时候，只有 a 字段能用到索引，那在联合索引的 B+Tree 找到第一个满足条件的主键值（ID 为 2）后，还需要判断其他条件是否满足（看 b 是否等于 2）

- 在 MySQL 5.6 之前，只能从 ID 2 （主键值）开始一个个回表，到「主键索引」上找出数据行，再对比 b 字段值。
- 而 MySQL 5.6 引入的**索引下推优化**，可以在联合索引遍历过程中，对联合索引中包含的字段先做判断，直接过滤掉不满足条件的记录，减少回表次数

当 `Explain` 里，出现了 Extra 为 `Using index condition`，那么说明使用了索引下推的优化

#### 索引失效

- 当我们使用左或者左右模糊匹配的时候，也就是 `like %xx` 或者 `like %xx%` 这两种方式都会造成索引失效；
- 当我们在查询条件中对索引列做了计算（col 1 + col 2 > xxx, col 1 + 1 = 10 不行，col 1 = 10 - 1）、函数（DATE）、类型转换操作（MySQL 比较时是默认是字符串转数字，如果 id = "1" 可以，phone_num = 132 xxxxx 不行）
- 联合索引要能正确使用需要遵循最左匹配原则，也就是按照最左优先的方式进行索引的匹配，否则就会导致索引失效。
- 在 WHERE 子句中，如果在 OR 前的条件列是索引列，而在 OR 后的条件列不是索引列，那么索引会失效。

#### 索引原则

- 索引区分度要高（status 不好，timestamp 好）
- 主键索引最好是自增的，非自增插入位置随机，导致页分裂
- 索引最好设置为 not null（null 会占用额外信息，并且判断是否使用索引逻辑更为复杂）
- 经常 group by 和 order by 字段建索引（group by 在前）
- 类型小的字段建索引

#### Count 区别

https://www.xiaolincoding.com/mysql/index/count.html

`count(1)` 和 `count(*)` 没区别，有优化

`count(1)`、 `count(*)`、 `count(主键字段)` 在执行的时候，如果表里存在二级索引，优化器就会选择二级索引进行扫描。

所以，如果要执行 `count(1)`、 `count(*)`、` count(主键字段)` 时，尽量在数据表上建立二级索引，这样优化器会自动采用 key_len 最小的二级索引进行扫描，相比于扫描主键索引效率会高一些。

再来，就是不要使用 `count(字段)` 来统计记录个数，因为它的效率是最差的，会采用全表扫描的方式来统计。如果你非要统计表中该字段不为 NULL 的记录个数，建议给这个字段建立一个二级索引。

### Log

#### redo log
重做日志用来实现事务持久性，主要有两部分文件组成，重做日志缓冲（redo log buffer）以及重做日志文件（redo log），前者是在内存中，后者是在磁盘中

物理格式的日志，记录的是物理数据页面的修改的信息，其 `redo log` 是顺序写入 `redo log file` 的物理文件中去的，物理文件是有最大限制的 4G，环状写

#### undo log
回滚日志，记录数据被修改前的信息

正好跟前面的重做日志进行相反操作。undo log 主要记录的是数据的逻辑变化，为了在发生错误时回滚之前的操作，需要将之前的操作都记录下来，然后在发生错误时才可以回滚

逻辑格式的日志，在执行 undo 的时候，仅仅是将数据从逻辑上恢复至事务之前的状态，而不是从物理页面上操作实现的，这一点是不同于 redo log 的

逻辑格式的日志，可以简单认为就是执行过的事务中的 sql 语句。包括了执行的 sql 语句（增删改）反向的信息
- delete 对应着反向的 insert
- update 对应着 update 执行前后的版本的信息
- insert 对应着 delete 和 insert 本身的信息
#### bin log
归档日志，记录了所有的 DDL 和 DML 语句（除查询语句外），以事件形式记录，是事务安全型，Server 层

- statement 模式记录 SQL
- row 记录行修改
- mixed 根据不同 SQL 选择 row 还是 statement
### 事务

https://www.xiaolincoding.com/mysql/transaction/mvcc.html

https://blog.csdn.net/SnailMann/article/details/94724197

#### ACID

- **原子性（Atomicity）**：一个事务中的所有操作，要么全部完成，要么全部不完成，不会结束在中间某个环节。事务在执行过程中发生错误，会被回滚到事务开始前的状态（undo log）
- **一致性（Consistency）**：是指事务操作前和操作后，数据满足完整性约束，数据库保持一致性状态（持久性+原子性+隔离性）
- **隔离性（Isolation）**：数据库允许多个并发事务同时对其数据进行读写和修改的能力，隔离性可以防止多个事务并发执行时由于交叉执行而导致数据的不一致，因为多个事务同时使用相同的数据时，不会相互干扰，每个事务都有一个完整的数据空间，对其他并发事务是隔离的（MVCC 或者锁机制）
- **持久性（Durability）**：事务处理结束后，对数据的修改就是永久的，即便系统故障也不会丢失（redo log）

#### 隔离级别
https://cloud.tencent.com/developer/article/1450773

并行化事务带来的问题

- **脏读**：读到其他事务未提交的数据；如果一个事务「读到」了另一个「未提交事务修改过的数据」
- **不可重复读**：前后读取的数据不一致；在一个事务内多次读取同一个数据，如果出现前后两次读到的数据不一样的情况
- **幻读**：前后读取的记录数量不一致；在一个事务内多次查询某个符合查询条件的「记录数量」，如果出现前后两次查询到的记录数量不一样的情况

严重程度：
![并行化事务问题严重程度|600](https://cdn.xiaolincoding.com//mysql/other/d37bfa1678eb71ae7e33dc8f211d1ec1.png)

解决办法，隔离性的四个级别

- 读未提交（read uncommitted），指一个事务还没提交时，它做的变更就能被其他事务看到；
- 读提交（read committed），指一个事务提交之后，它做的变更才能被其他事务看到，防止脏读；
- 可重复读（repeatable read），指一个事务执行过程中看到的数据，一直跟这个事务启动时看到的数据是一致的，避免不可重复读；（**MySQL InnoDB 引擎的默认隔离级别**）
- 串行化（serializable ）；会对记录加上读写锁，在多个事务对这条记录进行读写操作时，如果发生了读写冲突的时候，后访问的事务必须等前一个事务执行完成，才能继续执行；

按隔离水平高低排序如下：
![隔离级别水平|600](https://cdn.xiaolincoding.com//mysql/other/cce766a69dea725cd8f19b90db2d0430.png)

#### 锁机制
> [!note]
> 乐观并发控制 `Optimistic concurrency control`：OCC 对数据修改持有乐观态度，假设数据不会被并发修改，不需要锁，提交修改之前，各个事务检查数据是否被修改，如果被修改，则回滚事务
> 
> 悲观并发控制 `Pessimistic concurrency control`：PCC 对数据修改持有悲观态度，假设数据会被并发修改，在事务处理前需要先获取锁，执行完释放锁，其它事务访问数据需要等待锁释放

- 乐观锁
    - 使用版本标识来确定读到的数据与提交时的数据是否一致。在每一行记录的后面增加两个隐藏列，记录创建版本号和删除版本号，而每一个事务在启动的时候，都有一个唯一的递增的版本号，提交时校验版本号。适用于低数据争用，写冲突比较少的环境
- 悲观锁
    - 依靠数据库提供的锁机制实现，比如 mysql 的排他锁，`select … for update` 来实现悲观锁
    - 共享锁 (S 锁、读锁): 如果事务 T 对数据 A 加上共享锁后，则其他事务只能对 A 再加共享锁，不能加排他锁。获准共享锁的事务只能读数据，不能修改数据（`select ... lock in share mode`）
    - 排它锁 (X 锁、写锁)：如果事务 T 对数据 A 加上排他锁后，则其他事务不能再对 A 加任任何类型的封锁。获准排他锁的事务既能读数据，又能修改数据。分为行锁和表锁

MySQL 常用的两种引擎 MyISAM 和 InnoDB，MyISAM 默认使用表锁，InnoDB 默认使用行锁

> 注意：使用 InnoDB 引擎，如果筛选条件里面没有索引字段，就会锁住整张表，否则的话，锁住相应的行

##### 当前读

- `select lock in share mode`（共享锁）
- `select for update`、`update`、`insert`、`delete`(排他锁)

这些操作都是一种当前读，就是它读取的是记录的最新版本，读取时还要保证其他并发事务不能修改当前记录，会对读取的记录进行加锁。

- `record lock`：记录锁, 仅仅锁住索引记录的一行
- `gap Lock`：间隙锁，锁定一个范围，但不包括记录本身。GAP 锁的目的，是为了防止同一事务的两次当前读，出现幻读的情况
- `next-key Lock`：是 record lock 和 gap lock 的组合，用于锁定一个范围，并且锁定记录本身。对于行的查询，都是采用该方法，主要目的是解决幻读的问题

##### 快照读
像不加锁的 select 操作就是快照读，即不加锁的非阻塞读

快照读的前提是隔离级别不是「读未提交」和「串行化」级别，因为未提交读总是读取最新的数据行，而不是符合当前事务版本的数据行。而串行化则会对所有读取的行都加锁

在 `RR` 级别下，快照读是通过 MVVC(多版本控制) 和 undo log 来实现的，当前读是通过加 record lock(记录锁) 和 gap lock(间隙锁) 来实现的。

#### MVCC 实现
<https://notes.eatonphil.com/2024-05-16-mvcc.html>

>[!note]
> MVCC 的实现，是通过保存数据在某个时间点的快照来实现的。每个事务读到的数据项都是一个历史快照，被称为快照读，不同于当前读的是快照读读到的数据可能不是最新的，但是快照隔离能使得在整个事务看到的数据都是它启动时的数据状态。而写操作不覆盖已有数据项，而是创建一个新的版本，直至所在事务提交时才变为可见

MVCC 只在 `READ COMMITTED` 和 `REPEATABLE READ` 两个隔离级别下工作。其他两个隔离级别够和 MVCC 不兼容, 因为 `READ UNCOMMITTED` 总是读取最新的数据行（包含事务未提交的数据）, 而不是符合当前事务版本的数据行。而 `SERIALIZABLE` 则会对所有读取的行都加锁。

快照读，读取的是记录的可见版本 (有可能是历史版本)，不用加锁。主要应用于无需加锁的普通查询（select）操作。快照读的意思是，数据有多个版本，当事务并发执行时，某一事务读取的数据来自其中一个版本（快照）。快照读的前提是隔离级别不是串行级别，串行级别下的快照读会退化成当前读。

##### 隐式字段

每行记录除了我们自定义的字段外，还有数据库隐式定义的 `DB_TRX_ID`, `DB_ROLL_PTR`, `DB_ROW_ID` 等字段

- `DB_TRX_ID`：6 byte，最近 「修改/插入」的事务 ID：记录创建这条记录/最后一次修改该记录的事务 ID
- `DB_ROLL_PTR`：7 byte，回滚指针，指向这条记录的上一个版本（存储于 rollback segment 里）
- `DB_ROW_ID`：6 byte，隐含的自增 ID（隐藏主键），如果数据表没有主键，InnoDB 会自动以 `DB_ROW_ID` 产生一个聚簇索引
- 实际还有一个删除 flag 隐藏字段, 既记录被更新或删除并不代表真的删除，而是删除 flag 变了

##### undo log

undo log 主要分为两种：
- **insert undo log**：代表事务在 `insert` 新记录时产生的 `undo log`, 只在事务回滚时需要，并且在事务提交后可以被立即丢弃
- **update undo log**：事务在进行 `update` 或 `delete` 时产生的 `undo log`；不仅在事务回滚时需要，在快照读时也需要；所以不能随便删除，只有在快速读或事务回滚不涉及该日志时，对应的日志才会被 `purge` 线程统一清除

##### Read View

事务进行 `快照读` 操作的时候生产的 `读视图` (Read View)，在该事务执行的快照读的那一刻，会生成数据库系统当前的一个快照，记录并维护系统当前活跃事务的 ID (**当每个事务开启时，都会被分配一个 ID , 这个 ID 是递增的，所以最新的事务，ID 值越大**)

### 复制
https://www.cnblogs.com/f-ck-need-u/p/9155003.html

![mysql master_slave|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/mysql_master_slave.6t6zklp4x7.webp)



#### GTID
https://www.cnblogs.com/f-ck-need-u/p/9164823.html

### 查询优化器

![mysql internal|700](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/.3rb3hxmtnc.webp)

1. 常量折叠：在编译时简化常量表达式。例如，将 `WHERE 3*2=6` 替换为 `WHERE 6=6` ，然后替换为 `WHERE TRUE`
2. 谓词下推：将 `WHERE` 条件移至尽可能靠近数据检索阶段，这样可以最大限度地减少查询执行后期需要处理的行数。
3. 连接重新排序：根据表的大小和连接条件查找最有效的连接表顺序，这会显着影响查询的性能。
4. 索引使用：确定何时使用索引来加速数据检索。这包括在有多个索引可用时选择最佳索引，并确定表扫描是否比索引访问更有效。
5. 消除不必要的联接：删除满足查询不需要的联接，例如在联接表但未从中选择任何列并且没有条件使用它时。
6. 子查询扁平化：将子查询转换为联接或应用其他转换以提高其效率。
7. 具体化：通过将中间结果存储在临时表中来避免不必要的计算，然后可以在查询中重用该结果。这对于多次引用的子查询特别有用
8. 查询重写：以更优化的形式重写查询，而不改变语义。例如，将 `IN` 转换为 `EXISTS` 或反之亦然（如果可以更有效地完成）。
9. 分区修剪：在分区表中，根据 `WHERE` 子句条件跳过与查询无关的分区。
10. 列修剪：从查询计划中排除未使用的列，从而减少从磁盘读取的数据。
11. 批处理：批量处理行以减少上下文切换的开销并提高某些操作（如连接和聚合）的效率。
12. 并行执行：利用多个 CPU 核心并行执行查询的不同部分，这可以显着减少复杂查询的总执行时间。
13. 聚合下推：在执行计划中尽早进行聚合，这往往可以减少后期需要传输和处理的数据量。
14. 索引合并：在同一个表上组合多个索引来过滤行，比使用任何单个索引更有效。
15. 延迟全行数据的检索，直到绝对必要时，如果最初只需要几列或者有可以首先应用的过滤器，这可以减少 IO。
16. 公共子表达式消除：识别并重用在查询中多次计算的表达式的结果。
17. 使用覆盖索引：当查询可以完全由索引中的数据满足时，从而避免访问主表数据的需要。
18. 表消除：从查询中删除不影响结果的表，例如使用 `LEFT JOIN` 时，不使用连接表的列，并且连接条件始终为 true。

#### 同步

如何做双向复制同步？

设置不同的自增主键范围

#### Deadlock

## Middlewares

### Clickhouse
官网介绍
https://clickhouse.com/docs/zh

适用场景
- 绝大多数是读请求
- 数据以相当大的批次 (> 1000 行) 更新，而不是单行更新; 或者根本没有更新。
- 已添加到数据库的数据不能修改。
- 对于读取，从数据库中提取相当多的行，但只提取列的一小部分。
- 宽表，即每个表包含着大量的列
- 查询相对较少 (通常每台服务器每秒查询数百次或更少)
- 对于简单查询，允许延迟大约 50 毫秒
- 列中的数据相对较小：数字和短字符串 (例如，每个 URL 60 个字节)
- 处理单个查询时需要高吞吐量 (每台服务器每秒可达数十亿行)
- 事务不是必须的
- 对数据一致性要求低
- 每个查询有一个大表。除了他以外，其他的都很小。
- 查询结果明显小于源数据。换句话说，数据经过过滤或聚合，因此结果适合于单个服务器的 RAM 中

https://clickhouse.com/docs/en/optimize/sparse-primary-indexes

https://clickhouse.com/blog/clickhouse-faster-queries-with-projections-and-primary-indexes

#### 为什么快？

1. 分析场景中往往需要读大量行但是少数几个列。在行存模式下，数据按行连续存储，所有列的数据都存储在一个 block 中，不参与计算的列在 IO 时也要全部读出，读取操作被严重放大。而列存模式下，只需要读取参与计算的列即可，极大的减低了 IO cost，加速了查询。  
2. 同一列中的数据属于同一类型，压缩效果显著。列存往往有着高达十倍甚至更高的压缩比，节省了大量的存储空间，降低了存储成本。 
3. 更高的压缩比意味着更小的 data size，从磁盘中读取相应数据耗时更短。  
4. 自由的压缩算法选择。不同列的数据具有不同的数据类型，适用的压缩算法也就不尽相同。可以针对不同列类型，选择最合适的压缩算法。  
高压缩比，意味着同等大小的内存能够存放更多数据，系统 cache 效果更好。

#### 查询过程

https://clickhouse.com/docs/en/optimize/sparse-primary-indexes

clickhouse 索引结构可以看成一个连续的序列，使用二分搜索进行查找

主索引每组行（granule 个）一个索引 mark（稀疏索引），存储列的 key colum values

clickhouse 中每个列一个 .bin 文件存储，初步通过稀疏索引查找到需要扫描的行范围

然后再去查找每个列单独的 mark 文件（存储 block_offset 和 granule offset），最后定位到符合条件的行后，会去 .bin 里解压对应的 compress block，然后使用 granule offset 找到数据

传输到 clickhouse 引擎处理

#### 优化
1. 避免字段空值（例如在 log report 时上传设置的默认值）
2. 创建中间表（partner_id, shop_id, path）聚合为每天的 (partner_id, shop_id, success_calls...) 表

### Elasticsearch

https://javapub.blog.csdn.net/article/details/123761794

https://blog.csdn.net/weixin_35688430/article/details/110545234

https://pdai.tech/md/db/nosql-es/elasticsearch.html

ES 和关系型数据库概念对照

| 概念                | Elasticsearch         | 关系型数据库        |
|---------------------|-----------------------|---------------------|
| 索引 (Index)         | 存储文档的单元，类似于数据库 | 数据库              |
| 类型 (Type)         | 已废弃，一个索引只能包含一个类型 | 表                 |
| 文档 (Document)      | 最小的信息单元，用 JSON 表示 | 行                 |
| 字段 (Field)        | 存储文档属性           | 列                 |
| 映射 (Mapping)       | 定义字段和数据类型       | 模式               |
| 分片和副本 (Shards and Replicas) | 数据分片和副本 | 分区和复制        |
| 搜索 (Search)        | 全文搜索和过滤操作       | 查询               |
| 聚合 (Aggregations)  | 数据汇总、分组和统计信息 | 聚合函数           |

#### 文档写入
![es write index|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/es_write_index.5mno8g10ay.webp)

1. 客户端向 Node 1 发送新建、索引或者删除请求。
2. 节点使用文档的 `_id` 确定文档属于分片 0 。请求会被转发到 Node 3，因为分片 0 的主分片目前被分配在 Node 3 上。
3. Node 3 在主分片上面执行请求。如果成功了，它将请求并行转发到 Node 1 和 Node 2 的副本分片上。一旦所有的副本分片都报告成功, Node 3 将向协调节点报告成功，协调节点向客户端报告成功。
##### 持久化过程

###### write

一个新文档过来，会存储在 in-memory buffer 内存缓存区中，顺便会记录 Translog（Elasticsearch 增加了一个 translog ，或者叫事务日志，在每一次对 Elasticsearch 进行操作时均进行了日志记录）。这时候数据还没到 segment ，是搜不到这个新文档的。数据只有被 refresh 后，才可以被搜索到。

###### refresh

refresh 默认 1 秒钟执行一次。ES 是支持修改这个值的，通过 index.refresh_interval 设置 refresh （冲刷）间隔时间。refresh 流程大致如下：
1. in-memory buffer 中的文档写入到新的 segment 中，但 segment 是存储在文件系统的缓存中。此时文档可以被搜索到
2. 最后清空 in-memory buffer。注意: Translog 没有被清空，为了将 segment 数据写到磁盘
3. 文档经过 refresh 后，segment 暂时写到文件系统缓存，这样避免了性能 IO 操作，又可以使文档搜索到。refresh 默认 1 秒执行一次，性能损耗太大。一般建议稍微延长这个 refresh 时间间隔，比如 5 s。因此，ES 其实就是准实时，达不到真正的实时。

###### flush

每隔一段时间—​例如 translog 变得越来越大—​索引被刷新（flush）；一个新的 translog 被创建，并且一个全量提交被执行

上个过程中 segment 在文件系统缓存中，会有意外故障文档丢失。那么，为了保证文档不会丢失，需要将文档写入磁盘。那么文档从文件缓存写入磁盘的过程就是 flush。写入磁盘后，清空 translog。具体过程如下：

1. 所有在内存缓冲区的文档都被写入一个新的段。
2. 缓冲区被清空。
3. 一个 Commit Point 被写入硬盘。
4. 文件系统缓存通过 fsync 被刷新（flush）。
5. 老的 translog 被删除。

###### merge

由于自动刷新流程每秒会创建一个新的段，这样会导致短时间内的段数量暴增。而段数目太多会带来较大的麻烦。每一个段都会消耗文件句柄、内存和 cpu 运行周期。更重要的是，每个搜索请求都必须轮流检查每个段；所以段越多，搜索也就越慢。

Elasticsearch 通过在后台进行 Merge Segment 来解决这个问题。小的段被合并到大的段，然后这些大的段再被合并到更大的段。

当索引的时候，刷新（refresh）操作会创建新的段并将段打开以供搜索使用。合并进程选择一小部分大小相似的段，并且在后台将它们合并到更大的段中。这并不会中断索引和搜索。

一旦合并结束，老的段被删除：
1. 新的段被刷新（flush）到了磁盘。写入一个包含新段且排除旧的和较小的段的新提交点。
2. 新的段被打开用来搜索。
3. 老的段被删除。
#### 查询

![es write doc|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/es_read_doc.6wqleryvwo.webp)

1. 客户端向 Node 1 发送获取请求。
2. 节点使用文档的 `_id` 来确定文档属于分片 0 。分片 0 的副本分片存在于所有的三个节点上。在这种情况下，它将请求转发到 Node 2 。
3. Node 2 将文档返回给 Node 1 ，然后将文档返回给客户端。

在处理读取请求时，协调结点在每次请求的时候都会通过轮询所有的副本分片来达到负载均衡。

在文档被检索时，已经被索引的文档可能已经存在于主分片上但是还没有复制到副本分片。在这种情况下，副本分片可能会报告文档不存在，但是主分片可能成功返回文档。一旦索引请求成功返回给用户，文档在主分片和副本分片都是可用的。
#### 倒排索引

单词到文档的映射

term index 存储在内存，

##### 优化

## System

https://www.xiaolincoding.com/os

- 进程与线程简单介绍，区别，以及进程间通信方式，线程同步方式
- 用户态和内核态
- 内存管理：分页分段，虚拟内存，空闲地址管理方法
- 死锁：死锁的必要条件，死锁的检测与恢复，死锁的预防，死锁的避免

### 进程
https://www.geeksforgeeks.org/difference-between-process-and-thread/

#### Address Space

```text
+-----------------------+ 0xFFFFFFFF (Top of memory)
|      Kernel Space     |
+-----------------------+
| Memory-Mapped Region  |
+-----------------------+
|         Stack         |
|   (Grows downward)    |
+-----------------------+
|                       |
|          Heap         |
|   (Grows upward)      |
|                       |
+-----------------------+
|         BSS           |
+-----------------------+
|         Data          |
+-----------------------+
|         Text          |
+-----------------------+
|   Command-line args   |
|   Environment vars    |
+-----------------------+ 0x00000000 (Bottom of memory)
```

- **文本段 (Text Segment)**：包含程序的可执行代码。这个段通常是只读的，并且在运行相同程序的进程之间共享以节省内存。
- **数据段 (Data Segment)**：包含程序员初始化的全局和静态变量。这个段是可写的。
- **BSS段 (Block Started by Symbol)**：包含未初始化的全局和静态变量。这个段也是可写的。
- **堆 (Heap)**：用于动态内存分配。随着程序的执行，分配更多内存时它向上增长（通过 `malloc` 等函数）。
- **栈 (Stack)**：用于函数调用管理、本地变量和控制流（函数调用/返回）。它向下增长。
- **内存映射区域 (Memory-Mapped Region)**：包含内存映射文件和共享内存。这个区域用于文件 I/O 和进程间通信。
- **内核空间 (Kernel Space)**：保留给内核使用，运行在受保护的内存区域。用户进程无法直接访问该区域。

#### 堆/栈

1. 栈的内存管理简单，分配比堆上快
2.  栈的内存不需要回收，而堆需要，无论是主动 free，还是被动的垃圾回收，这都需要花费额外的 CPU。
3. 栈上的内存有更好的局部性，堆上内存访问就不那么友好了，CPU 访问的 2 块数据可能在不同的页上，导致 CPU 访问耗时高

#### 进程间通信
- 管道
	- 匿名管道（父子进程），有名管道
- 消息队列
- 共享内存（shm 和 mmap）
- 信号量
- 信号（kill 或者 alarm 发送）
- socket（TCP/UDP/UDS）

#### 进程/线程/协程

#### `Background/Daemon` Process

| 特点        | 后台进程                     | 守护进程                  |
| --------- | ------------------------ | --------------------- |
| 启动方式      | 在终端中启动，并使用 `&` 符号放入后台运行  | 系统启动时启动，并在系统关闭时停止     |
| 终端关联      | 与启动它的终端会话相关联             | 与控制终端无关               |
| 标准输入/输出   | 通常与终端相关联，除非显式更改          | 通常被重定向到 `/dev/null`   |
| SIGHUP 信号 | 终端退出时会收到 SIGHUP 信号并可能被终止 | 不受终端影响                |
| 父进程       | 通常有父进程，可能受父进程退出影响        | 没有父进程，父进程通常是系统级别的守护程序 |
Daemon process 特点
1. 继承当前 session （对话）的标准输出（stdout）和标准错误（stderr）。因此，后台任务的所有输出依然会同步地在命令行下显示
2. 不再继承当前 session 的标准输入（stdin）。你无法向这个任务输入指令了。如果它试图读取标准输入，就会暂停执行（halt）

##### 创建 background process

- 使用 `&` 符号：在 Linux 终端中，可以通过在命令后面加上&符号来将进程放入后台运行。例如，`command &` 会将 command 这个进程放入后台运行，不会阻塞当前终端
- 使用 nohup 命令：使用 nohup 命令可以创建一个忽略 SIGHUP 信号的进程，即在终端关闭后仍然继续运行。通过结合 nohup 和&符号，可以创建一个后台运行的进程，并且关闭当前终端不会影响该进程的执行。示例：`nohup command &`

##### 创建 daemon process

https://github.com/pasce/daemon-skeleton-linux-c

1. 创建子进程，父进程退出
	1. 首先，使用 fork() 函数创建一个子进程。如果 fork() 返回值大于 0，则表示当前进程是父进程
	2. 此时父进程应该调用 exit(0) 退出。这样做的目的是让子进程成为孤儿进程，从而被 init 进程（PID 为 1）收养，确保子进程不会成为进程组的组长，这是后续步骤中调用 setsid() 函数的前提条件
2. 在子进程中创建新会话
	1. 子进程调用 setsid() 函数创建一个新的会话，并成为新会话的首进程和进程组的组长。这一步使得进程脱离原有的终端控制，确保守护进程不会意外地获得控制终端
3. 改变当前目录为根目录
	1. 通过调用 chdir ("/") 函数将当前工作目录改变为根目录。这是因为守护进程通常需要在系统运行期间一直运行，如果守护进程的工作目录是一个挂载的文件系统，那么这个文件系统就无法被卸载
4. 重新设置文件权限掩码
	1. 调用 umask(0) 函数清除文件创建掩码。这样做是为了确保守护进程创建的文件和目录具有合适的权限，不受继承自父进程的 umask 值的影响
5. 关闭文件描述符
	1. 子进程从父进程继承了打开的文件描述符。为了防止守护进程无意中使用这些文件描述符，应该关闭它们。可以通过 getdtablesize() 函数获取进程打开的文件描述符数目，然后遍历并关闭这些文件描述符

### Buffer-Cache/Page-Cache

> Cache 名为缓存，Buffer 名为缓冲
> 
> Cache 和 Buffer 的出现就是为了弥补高速设备和低速设备之间的矛盾而设立的中间层。

- page cache: 页缓存, 负责缓存逻辑数据
- buffer cache: 块缓存, 负责缓存物理数据

- Cache 会将低速设备中常被访问的数据缓存起来，当高速设备需要再次访问这些数据时，会命中 Cache 中的数据，以减少对低速设备的访问
- Buffer 用于缓和高速设备要把数据回写到低速设备时带来的冲击，当数据量比较大时，Buffer 能将数据分割成合适的大小，分批回写到磁盘；当数据量比较小的时候，Buffer 能将分散的写操作集中进行，减少磁盘碎片和硬盘的反复寻道，通过「流量整形」提高系统性能

### 共享内存

- POSIX 共享内存 (结合内存映射 mmap 使用)
    - mmap 映射的内存是非持久化的，随着进程关闭，映射会随即失效
    - mmap 内存映射机制标准的系统调用，分为：
        - 匿名映射
        - 文件映射，又分为：
            - MAP_PRIVATE
            - MAP_SHARED
- System V 共享内存 (经典方案)
    - sysv shm 是持久化的，除非被进程明确的删除，否则在系统关机前始终存在于内存中
### Swap

内存不够时，允许内存换页到磁盘上，达到缓冲作用

强内存依赖的组件不能开这个东西，redis 慢可能是因为这个

### Kafka 为什么快？

#### 顺序磁盘 I/O

> [!NOTE]
> **随机读写**：指的是当存储器中的消息被读取或写入时，所需要的时间与这段信息所在的位置无关。当读取第一个 block 时，要经历寻道、旋转延迟、传输三个步骤才能读取完这个 block 的数据。而对于下一个 block，如果它在磁盘的其他任意位置，访问它会同样经历寻道、旋转、延时、传输才能读取完这个 block 的数据，我们把这种方式叫做随机读写
> 
> **顺序读写**：是一种按记录的逻辑顺序进行读、写操作的存取方法，即按照信息在存储器中的实际位置所决定的顺序使用信息。如果这个 block 的起始扇区刚好在刚才访问的 block 的后面，磁头就能立刻遇到，不需等待直接传输，这种就叫顺序读写

Kafka 利用磁盘顺序读写的高效性，将消息持久化到本地磁盘中。与随机读写相比，顺序读写的性能要高出几个数量级。Kafka 的消息是不断追加到日志文件的末尾，这种顺序写的方式大大提高了写入吞吐量

#### Sendfile (Zero-Copy) + Page Cache
优化 Consumer 读取。通过直接在内核空间和网络缓冲区之间传输数据，避免了 CPU 的额外负担，从而提高了数据传输的效率

![kafka no zerocopy|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/kafka_no_dma.6wqleqicdj.webp)

- 第一次传输：从硬盘上将数据读到操作系统内核的缓冲区里，这个传输是通过 DMA 搬运的。
- 第二次传输：从内核缓冲区里面的数据复制到分配的内存里面，这个传输是通过 CPU 搬运的。
- 第三次传输：从分配的内存里面再写到操作系统的 Socket 的缓冲区里面去，这个传输是由 CPU 搬运的。
- 第四次传输：从 Socket 的缓冲区里面写到网卡的缓冲区里面去，这个传输是通过 DMA 搬运的。

利用了 zerocopy 后，实际上只会有 2 次传输（使用 sendfile 调用）
![kafka zerocopy|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/kafka_zerocopy.13ln5fzkhh.webp)

- 第一次传输：通过 DMA 从硬盘直接读到操作系统内核的读缓冲区里面。
- 第二次传输：根据 Socket 的描述符信息直接从读缓冲区里面写入到网卡的缓冲区里面。

而且如果 consume/produce 速率相差不大的情况，几乎都是从操作系统 Page Cache 读取（ I/O stat 可以观察到）
#### 批量处理
Kafka 支持批量读取发送和批量压缩消息，这减少了网络 I/O 的次数和数据的体积，从而提高了效率
#### 分区和文件格式
Kafka 的 Topic 被分为多个 Partition，Partition 分布在不同的服务器上以目录形式独立存在，实现数据的并行处理和负载均衡

Kafka 的数据文件是一系列可追加的日志段，每个段都有索引，使得数据的读取非常高效
#### Pull 模式
Kafka 采用消费者拉取（pull）模式，消费者根据自己的消费能力从 Broker 拉取数据，这种方式使得消费者可以更灵活地控制数据的消费速率，避免了生产者推送（push）模式可能导致的消费者处理不过来而拖慢整体处理速度的问题

### Rebalance

### 可靠性

#### Producer acks 参数
这个参数用来指定分区中有多少个副本收到这条消息，生产者才认为这条消息是写入成功的，这个参数有三个值：
- acks = 1，默认为 1。生产者发送消息，**只要 leader 副本成功写入消息，就代表成功**。这种方案的问题在于，当返回成功后，如果 leader 副本和 follower 副本**还没有来得及同步**，leader 就崩溃了，那么在选举后新的 leader 就没有这条**消息，也就丢失了**。
- acks = 0。生产者发送消息后直接算写入成功，不需要等待响应。这个方案的问题很明显，**只要服务端写消息时出现任何问题，都会导致消息丢失**。
- acks = -1 或 acks = all。生产者发送消息后，需要等待 ISR 中的所有副本都成功写入消息后才能收到服务端的响应。毫无疑问这种方案的**可靠性是最高的**，但是如果 ISR 中只有 leader 副本，那么就和 acks = 1 毫无差别了。

#### Consumer offset 提交策略
默认情况下，当消费者消费到消息后，就会自动提交位移。但是如果消费者消费出错，没有进入真正的业务处理，那么就可能会导致这条消息消费失败，从而丢失。我们可以开启手动提交位移，等待业务正常处理完成后，再提交 offset。

## Network

- 七层结构，简单介绍一下每一层。
- 输入 URL 后，将发生什么？这个问题会涉及到很大一部分的计算机网络基础。
- HTTP 和 HTTPS，DNS 解析
- TCP、UDP、拥塞控制、三次握手、四次挥手、滑动窗口
- IP 和 ARP 协议

**互联网协议套件**
https://en.wikipedia.org/wiki/Internet_protocol_suite
### OSI 七层

https://www.freecodecamp.org/chinese/news/osi-model-networking-layers/

![OSI 七层|900](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/osi_7.7w6of10sew.webp)

### HTTP status code
https://www.loggly.com/blog/http-status-code-diagram/

https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Status
#### 3xx
- 301 Moved Permanently（永久移动）
	- 搜素引擎 SEO
- 302 Found（临时重定向）
	- 推荐仅在响应 `GET` 或 `HEAD` 方法时采用 302 状态码，而在其他时候使用 `307` Temporary Redirect 来替代
- 307 Temporary Redirect
	- 可以确保请求方法和消息主体不会发生变化，其它和 302 一样
- 308 Permanent Redirect
	- 可以确保请求方法和消息主体不会发生变化，其它和 301 一样

#### 4xx
- 403 Forbidden
	- 指的是服务器端有能力处理该请求，但是拒绝授权访问，例如无权限访问
- 404 Not Found
- 405 Method Not Allowed
- 413 Content Too Large
- 429 Too Many Requests
- 431 Request Header Fields Too Large
#### 5xx
- 500 Internal Server Error

### TCP

![tcp state change|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/tcp_state_change.4g4dkq7uvh.webp)

#### TCP 三次握手

![tcp 3way|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/tcp_3way.8dwqjdpa7z.webp)

使用 3 次握手的原因
- 防止「历史 SYN 连接」初始化了连接
	- 2 次握手的情况，客户端初次发送 SYN seq=90 的包，然后客户端重启并发送 seq=100 的包，服务端收到后就直接 ESTABLISHED，然后就会发送数据，但是这次数据客户端会 RST 终止连接，所以浪费服务端一次连接和资源
	- 3 次握手的情况，就可以避免服务端初始化历史连接
- 交换序列号
- 避免资源浪费
	- 2 次握手的情况，客户端 SYN 报文如果阻塞重发，会浪费服务端初始化多次连接

#### TCP 四次挥手

![tcp 4way|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/tcp_4way.4ckr500y0v.webp)

- 关闭连接时，客户端向服务端发送 `FIN` 时，仅仅表示客户端不再发送数据了但是还能接收数据。
- 服务端收到客户端的 `FIN` 报文时，先回一个 `ACK` 应答报文，而服务端可能还有数据需要处理和发送，等服务端不再发送数据时，才发送 `FIN` 报文给客户端来表示同意现在关闭连接。

#### 可靠性

- 滑动窗口
- 流量控制
- 拥塞控制
- 重传机制

#### TIME_WAIT

MSL 是 Maximum Segment Lifetime 英文的缩写，中文可以译为“报文最大生存时间”，他是任何报文在网络上存在的最长时间，超过这个时间报文将被丢弃

RTT 是客户到服务器**往返**所花时间（round-trip time），TCP 含有动态估算 RTT 的算法。TCP 还持续估算一个给定连接的 RTT，这是因为 RTT 受网络传输拥塞程序的变化而变化。表示从发送端发送数据开始，到发送端收到来自接收端的确认（接收端收到数据后便立即发送确认），总共经历的时延

2 MSL 即两倍的 MSL，TCP 的 TIME_WAIT 状态也称为 2 MSL 等待状态，当 TCP 的一端发起主动关闭，在发出最后一个 ACK 包后，即第 3 次挥手完成后发送了第四次挥手的 ACK 包后就进入了 TIME_WAIT 状态，必须在此状态上停留两倍的 MSL 时间

等待 2 MSL 时间主要目的是怕最后一个  ACK 包对方没收到，那么对方在超时后将重发第三次握手的 FIN 包，主动关闭端接到重发的 FIN 包后可以再发一个 ACK 应答包。在 TIME_WAIT 状态时两端的端口不能使用，要等到 2 MSL 时间结束才可继续使用。当连接处于 2 MSL 等待阶段时任何迟到的报文段都将被丢弃。不过在实际应用中可以通过设置 SO_REUSEADDR 选项达到不必等待 2 MSL 时间结束再使用此端口


**出现 TIME_WAIT 的原因**

可能是连接池，minidle 太小，导致连接创建后被大量关闭（压测场景下出现过）

#### 全/半连接队列

>[!NOTE]
>客户端调用 connect 会在内核里随机选择一个端口，因为四元组确定一条 TCP 连接
>
>所以连接同一个服务端: PORT，是会有端口资源被 TIME_WAIT 占用耗尽的情况，可以打开 `net.ipv4.tcp_tw_reuse` 参数，会在 connect 的时候允许复用 TIME_WAIT


![tcp sockets connect accept|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/sockts_tcp_connect_accept.b8rattz9v.webp)


##### syncookies
开启 syncookies 功能就可以在不使用 SYN 半连接队列的情况下成功建立连接

syncookies 是这么做的：服务器根据当前状态计算出一个值，放在己方发出的 SYN+ACK 报文中发出，当客户端返回 ACK 报文时，取出该值验证，如果合法，就认为连接建立成功
#### TCP security
https://heimdalsecurity.com/blog/what-is-tcp/

#### TCP 粘包

1. 应用程序写入的数据大于套接字缓冲区大小，这将会发生拆包。
2. 应用程序写入数据小于套接字缓冲区大小，网卡将应用多次写入的数据发送到网络上，这将会发生粘包。
3. 进行 MSS（最大报文长度）大小的 TCP 分段，当 TCP 报文长度-TCP 头部长度>MSS 的时候将发生拆包。
4. 接收方法不及时读取套接字缓冲区数据，这将发生粘包。
### UDS vs TCP

> UDS 通常比 TCP  要快，主要原因是它绕过了网络协议栈，减少了系统调用的数量，避免了 TCP 协议的一些限制，并使用了文件路径而非网络地址进行通信

1. UDS 比 TCP 要快
	1. 绕过网络协议栈，当使用 Unix Domain Socket 时，数据传输不经过网络协议栈，而是直接在内核空间中进行，减少了在用户空间和内核空间之间的拷贝次数，同时也避免了 TCP/IP 协议栈的开销，如头部的添加、校验和的计算、封装和解封装的过程等
	2. 相比之下，即使是在同一台机器上使用 TCP loopback 连接，数据仍然需要通过网络协议栈，增加了处理时间
2. 更少的系统调用
	1. Unix Domain Socket 通常需要更少的系统调用来发送相同量的数据。
	2. 在 TCP 连接中，每次传输数据都涉及到更多的系统调用，如 `connect`、`send`、`recv` 等，这些都会增加 CPU 的使用率和延迟
3. 避免了 TCP 的一些限制
	1. TCP 连接受到其协议特性的限制，如慢启动、拥塞控制、流量控制等，这些特性在跨网络通信时非常有用，但在同一台机器上的进程间通信（IPC）中可能会导致不必要的延迟
	2. Unix Domain Socket 不受这些限制，因此在本地通信时可以提供更低的延迟和更高的吞吐量
4. 地址使用的是文件路径而非 IP 端口
	1. Unix Domain Socket 使用文件系统中的路径作为地址，而不是网络地址和端口号。这意味着它们不需要管理网络端口号，也不受本地端口数量的限制，而且配置和管理相对简单

### UDP

### HTTP & HTTPS
- HTTP 是超文本传输协议，信息是明文传输，存在安全风险的问题。HTTPS 则解决 HTTP 不安全的缺陷，在 TCP 和 HTTP 网络层之间加入了 SSL/TLS 安全协议，使得报文能够加密传输。
- HTTP 连接建立相对简单， TCP 三次握手之后便可进行 HTTP 的报文传输。而 HTTPS 在 TCP 三次握手之后，还需进行 SSL/TLS 的握手过程，才可进入加密报文传输。
- 两者的默认端口不一样，HTTP 默认端口号是 80，HTTPS 默认端口号是 443。
- HTTPS 协议需要向 CA（证书权威机构）申请数字证书，来保证服务器的身份是可信的。


![https tls | 800](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/https.lyd5v73z.webp)

### HTTP/2 & HTTP/3

#### HTTP/2
1. 头部压缩，静态表/动态表/Huffman 编码，使用二进制数据传输内容
2. 并发传输，多个 Stream 复用 1 个 TCP 连接，节约了 TCP 和 TLS 握手时间，以及减少了 TCP 慢启动阶段对流量的影响
3. 服务器支持主动推送资源

问题：
- 队头阻塞
- 连接迁移
- TCP 和 TLS 握手时延

HTTP/2(cleartext) 是 HTTP/2 但是不包含 TLS（GRPC 使用）

#### HTTP/3

QUIC 协议
- 分为多个传输队列，避免了队头阻塞

### IO 多路复用

https://www.xiaolincoding.com/os/8_network_system/selete_poll_epoll.html#%E6%9C%80%E5%9F%BA%E6%9C%AC%E7%9A%84-socket-%E6%A8%A1%E5%9E%8B

https://moonbingbing.gitbooks.io/openresty-best-practices/content/base/web_evolution.html

将 CPU 时分复用
#### select/poll
```cpp
int select(int n, fd_set *readfds, fd_set *writefds,
        fd_set *exceptfds, struct timeval *timeout);

int poll(struct pollfd *fds, unsigned int nfds, int timeout);
```

select 函数监视的文件描述符分 3 类，分别是 writefds、readfds 和 exceptfds。调用后 select 函数会阻塞，直到有描述符就绪（有数据 可读、可写、或者有 except），或者超时（timeout 指定等待时间，如果立即返回设为 null 即可）。当 select 函数返回后，通过遍历 fd_set，来找到就绪的描述符

`select` 存在文件描述符数量限制（1024）

`select` 和 `poll` 都存在用户态与内核态之间的拷贝开销，都通过遍历文件描述符集合来实现，随着并发数增加，性能损耗显著
#### epoll
```cpp
int epoll_create(int size)
int epoll_ctl(int epfd, int op, int fd, struct epoll_event *event)

typedef union epoll_data {
    void *ptr;
    int fd;
    __uint32_t u32;
    __uint64_t u64;
} epoll_data_t;

struct epoll_event {
    __uint32_t events; /* Epoll events */
    epoll_data_t data; /* User data variable */
};

int epoll_wait(int epfd, struct epoll_event *events, int maxevents, int timeout);
```

- 监视的描述符数量不受限制
- IO 的效率不会随着监视 fd 的数量的增长而下降。epoll 不同于 select 和 poll 轮询的方式，而是通过每个 fd 定义的回调函数来实现的。只有就绪的 fd 才会执行回调函数
- 支持水平触发和边沿触发两种模式：
    - 水平触发模式，文件描述符状态发生变化后，如果没有采取行动，它将后面反复通知，这种情况下编程相对简单，libevent 等开源库很多都是使用的这种模式
    - 边沿触发模式，只告诉进程哪些文件描述符刚刚变为就绪状态，只说一遍，如果没有采取行动，那么它将不会再次告知。理论上边缘触发的性能要更高一些，但是代码实现相当复杂（Nginx 使用的边缘触发）
- mmap 加速内核与用户空间的信息传递。epoll 是通过内核与用户空间 mmap 同一块内存，避免了无谓的内存拷贝

#### Network I/O models

https://gao-xiao-long.github.io/2017/04/20/network-io/

- 阻塞 IO：当用户程序执行 `read` ，线程会被阻塞，一直等到内核数据准备好，并把数据从内核缓冲区拷贝到应用程序的缓冲区中，当拷贝过程完成，`read` 才会返回（等待的是「内核数据准备好」和「数据从内核态拷贝到用户态」这两个过程）
- 非阻塞 IO：`read` 后立刻返回，然后不断轮询，但是还是会等待数据拷贝
- 异步 IO：在「内核数据准备好」和「数据从内核空间拷贝到用户空间」这两个过程都不用等待，拷贝完成会通知

#### Reactor 模型

非阻塞同步模型
Reactor 收到事件后，根据事件类型分配（dispatch）给某个进程 / 线程

Reactor 模式主要由 Reactor 和处理资源池这两个核心部分组成
- Reactor 负责监听和分发事件，事件类型包含连接事件、读写事件
- 处理资源池负责处理事件，如 read -> 业务逻辑 -> send（同步）

单 reactor 单线程/进程：redis
单 reactor 多线程：reactor 可能成为瓶颈
多 reactor 多线程：nginx/netty

#### Proactor
非阻塞异步模型

##### Nginx

通过 `accept_mutex` 锁控制惊群现象

### 平滑迁移/热重启

https://goteleport.com/blog/golang-ssh-bastion-graceful-restarts/

https://ms2008.github.io/2019/12/28/hot-upgrade/

https://www.hitzhangjie.pro/blog/2020-08-28-go%E7%A8%8B%E5%BA%8F%E5%A6%82%E4%BD%95%E5%AE%9E%E7%8E%B0%E7%83%AD%E9%87%8D%E5%90%AF/

#### scm_rights

https://zhuanlan.zhihu.com/p/405620115

SCM_RIGHTS 协议类型的套接字通常用于在 Linux 或 UNIX 系统上的进程之间传递文件描述符。通过使用 SCM_RIGHTS 和 sendmsg/recvmsg 函数，可以实现在进程之间传递文件描述符的功能。这种机制允许一个进程将打开的文件描述符传递给另一个进程，从而实现进程间的文件描述符共享。SCM_RIGHTS 协议类型的套接字提供了一种有效的方式来处理文件描述符的传递，使得进程间通信更加灵活和高效

可以用来做 TCP 连接平滑迁移，Cloudflare 写过文章
## Backend

### Connection Pool

Redis 和 Mysql 的池化技术，是类似的（https://go.dev/doc/database/manage-connections）

| Term          | Meaning                                                                                   |
| ------------- | ----------------------------------------------------------------------------------------- |
| **MinIdle**   | 最小空闲连接数，它决定了池中存在的最小连接数。增大则有助于减少流量突然增加时创建新连接所花费的时间，过大则浪费资源，过小则可能会导致连接频繁创建然后关闭（`TIME_WAIT`） |
| MaxIdle       | 最大空闲连接数，它决定了可以放回池的最大连接数。它等价于 `pool_size`                                                  |
| **MaxActive** | 最大打开连接数，决定可以使用的最大连接数。MaxActive 等于 `pool_size + overflow_size`                             |
| OverflowSIze  | 超过最大空闲连接数后可以创建的连接                                                                         |

![cache pool lifetime|800](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/redis_pool_phase.2krrwu8vqr.webp) 

### 灰度发布

按照地区灰度：坏处，有些地区流量较小

按照 Container 灰度：好处，更容易发现问题（蓝绿消耗更多资源）

蓝绿发布流程
1. 部署 2 套集群
2. 新集群引入流量（非实时流量先测试，然后引入实时流量）
3. 切断旧集群流量，切换为新集群（可以分阶段进行）

金丝雀发布：逐步部署集群中节点为新版本

### SLA 保障

http://kaito-kidd.com/2021/10/15/what-is-the-multi-site-high-availability-design/

https://www.abelsun.tech/arch/base/arch-y-ensure-high-availability.html#_4-6-%E5%AE%9E%E6%97%B6%E7%9B%91%E6%8E%A7%E5%92%8C%E5%BA%A6%E9%87%8F

流程规范
- 发版规范
- code review + golangci-lint

服务本身
- 异地多活
- DR 灾备
    - 方案 1，gateway 配置两份路径，DR 直接 gateway 切换
    - 方案 2，DNS 刷新，出现问题刷新到灾备机房

DR PPT: https://docs.google.com/presentation/d/1qDdHDknLlPOzmprWHKKKRMtNfDmtAQmY6NakkJXFTQU/edit#slide=id.g120be379d5f_0_55

发现问题
- tracing
    - 查看链路（跨 cid）
- 监控
	- 灰度监控，错误率对比
	- 异常告警，alert 区分核心和非核心告警群
- log
    - 通过 requestid 串联起 log，制定 log 级别

解决
- 服务降级
- 熔断/限流

### OAuth
https://www.ruanyifeng.com/blog/2014/05/oauth_2_0.html

https://juejin.cn/post/7195762258962219069

https://dev.mi.com/console/doc/detail?pId=711

### API 鉴权

使用 hmac-sha 256 散列算法，对请求数据进行签名，然后传递到服务端，服务端拼接同样的字符串，然后用同样的 key 计算 sign，验证 sign 是否正确

使用非对称加密的鉴权
用户使用私钥签名用 sha 256 算出来的签名摘要，服务端使用公钥解密，得到 sign，然后同样算法计算出 sign 是否正确，可以知道调用者是谁

### 中间件

接入公司的框架 EKL/Gas/Ucache/Hardy 都是为了更好的「可观测性」（tracing,log,metrics），以及 split by market，以及全链路压测 FCST
#### Gin

![gin start|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/gin-start.1lbpg5blb4.webp)

- radix 树实现路由匹配
- middleware 处理

#### Samara/EKL

Samara
- Consumer 基本上就是「拉取」消息后将消息放到 Chan 里，然后实现 Conusme 逻辑消费即可
- Producer 是将消息「推送」到 Chan，sync 就是直接发送，async 就是等待再发送

EKL 
- Tracing 支持
- 全链路压测支持（shadow topic）
- Hot retry & Cold retry（原地重试和延迟 topic 重试）
- Advance Consume Model，拉取消息后，放到多个 Chan 里，然后并发的消费，可以完全并发，想要保留顺序消费可以基于 Partition（不同 partition 并发，相同 partition 顺序），可以基于基于消息某字段，也可以自己实现 Dispatcher（例如 Traffic limit，基于（ShopId + Region）来 dispatch 的）
- Producer async 发送增加 Callback 通知
#### GAS

中间件 IOC 框架，每个中间件分为 Interface 和 Impl 两层

业务只对接 Interface 层，不直接引用 Impl 层

- 利于公司库中间件维护，比如修改 Tracing，Logger，Config 之类的
- 提高开发效率，可以基于 Proto 生成 Client/Server 代码
- 统一的一套规范，将中间件生命周期给 Gas 管理

#### Mates

其实就是异步任务和事件系统，async task & event system 

内部实现了 DB + Cronjob Scan + MQ + Worker Consumer + Worker call downstream 

 #### HSpex

实现了 Transport，内部实现 http.RoundTripper，在内部转换成 spex 然后请求，然后返回，再转化成 Response

Security HTTP SDK 也是类似，为了防止 SSRF 攻击，验证判断是否是内部 IP ，是的话就阻断

#### GDBC / SDDL

SDDL（Shopee Distributed Data Layer）是一个统一的数据访问层，屏蔽底层配置变更、读写分离、数据同步、分库分表、数据源切换、流量切换、监控

SDDL 还支持缩放
- 双写缩放
- binlog 缩放

公司的数据库 SDK，提供分片逻辑，提供页面配置，统一的路由规则和 Tracing 和影子库支持

### 微服务

#### Service Mesh

- Control Plane
- Data Plane
#### 服务治理

#### API Gateway

- API 管理（分组/发布/插件）
- 流量管理（熔断，限流，降级，拦截器）
	- 分布式限流（API + account, API + IP,  custom key, API mode）
- 动态路由（根据参数或者 header 转发 API  到不同的）
- 灰度管理
- 流量重放（STP 平台）
- 服务 Mock
### Observability

Understand service dependency and topology (who calls who?)
了解服务依赖性和拓扑（谁调用谁？）

Latency identification and performance tuning (who is slow?)
延迟识别和性能调优（谁慢？）

Root cause analysis and troubleshooting (who went wrong?)
根本原因分析和故障排除（谁错了？）

#### Tracing 

在分布式链路跟踪中有两个重要的概念：跟踪（trace）和跨度（ span）
trace 是请求在分布式系统中的整个链路视图，span 则代表整个链路中不同服务内部的视图，span 组合在一起就是整个 trace 的视图

https://www.jaegertracing.io/docs/1.55/architecture/

trace id 由几部分组成
1. trace id = `<instance + timestamp + random + 一些 flags(debug,tailbase,sampling)>`
2. span id = `<level> + <span context seq> + <random>`
3. parent id

tracing client -> (log agent -> jaeger collector) -> kafka -> jaeger ingester -> DB

##### 采样方法

**动态采样**：根据一定采样率采样，入口服务将标记写入到 tracing span id 中

**尾采样**：

1. 预采样（例如 10%，防止消耗过多资源）
2. 采样策略

![tail_base_sampling_policy|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/tail_base_policy.9gwfphpg72.webp)

### 分布式协议

#### Raft

#### Gossip

可视化：https://rrmoelker.github.io/gossip-visualization/

- 种子节点在 Gossip 周期内散播消息
- 被感染节点随机选择 N 个邻接节点散播消息
- 每次散播消息都选择尚未发送过的节点进行散播
### Indexing & Searching

https://blugelabs.com/blog/bluge-code-walkthrough-indexing/

#### Bloom filter

https://findingprotopia.org/posts/how-to-write-a-bloom-filter-cpp/

https://en.wikipedia.org/wiki/Bloom_filter

https://ethereum.stackexchange.com/questions/3418/how-does-ethereum-make-use-of-bloom-filters

https://ricardoanderegg.com/posts/understanding-bloom-filters-by-building-one/

https://codapi.org/embed/?sandbox=go&src=gist:e7bde93f98c5e47ca38d359a5104fd88:bloom.go

原版本不支持删除

计数式 bloomfilter

bit 数组改为计数数组，set 每位 +1，del 每位 -1，检测则判断是否全 > 0，如果有 =0，则一定不在

#### Cuckoo filter
布谷鸟过滤器是一个存储桶数组，使用两个哈希函数，将每个元素的指纹存储到两个可能的位置之一

如果两个位置都已被占用，则会进行「踢出」操作，将其中一个移到其另一个可能的位置，并重复这一过程，直到找到一个空闲位置或达到最大踢出次数

查找时查看是否存了当前元素的指纹，存在则返回 true

删除时先查找，然后存在就删除
#### Count–min sketch

Count–min sketch 是一种概率数据结构，用于高效地估计大型数据流中的元素频率。它在处理大数据集时尤其有用，能够在有限内存和计算资源的情况下提供快速和近似的查询结果。

Count–min sketch 由一个二维数组（或矩阵）和一组独立的哈希函数组成。其工作方式如下：
1. **初始化**：创建一个大小为 `d x w` 的二维数组，所有元素初始为零。这里 d 是哈希函数的数量，w 是每个哈希函数对应的数组长度。
2. **哈希函数**：定义 d 个独立的哈希函数 (h_1, h_2, ..., h_d)，每个哈希函数将输入元素映射到 `[0, w-1]` 的范围内。
3. **更新（插入）**：当一个元素 e 出现在数据流中时，使用每个哈希函数计算该元素的哈希值，并在相应的位置上增加计数。
4. **查询**：要查询某个元素 e 的频率估计值，计算该元素在每个哈希函数下的哈希值，并取这些位置上计数的最小值。

优点
- **空间效率**：Count–min sketch 使用的空间是固定的，与数据流的大小无关。它只需 O (d x w) 的空间。
- **时间效率**：插入和查询操作的时间复杂度都是 O (d)，其中 d 是哈希函数的数量。

缺点
- **近似估计**：由于哈希冲突的存在，Count–min sketch 提供的频率估计值可能会比实际值大，但不会比实际值小。
- **误差**：估计值的误差与哈希函数的数量 d 和数组长度 w 相关。通过增加 d 和 w 可以减少误差，但也会增加空间和计算资源的开销。

应用
Count–min sketch 在许多需要处理大数据流的场景中有广泛应用，例如：
- 网络流量监控
- 数据库查询优化
- 分布式系统中的频率估计
- 机器学习中的特征选择
#### Bitmaps (Roaring Bitmaps)

Roaring bitmaps
https://www.vikramoberoi.com/a-primer-on-roaring-bitmaps-what-they-are-and-how-they-work/

https://www.elastic.co/blog/frame-of-reference-and-roaring-bitmaps

https://vikramoberoi.com/posts/using-bitmaps-to-run-interactive-retention-analyses-over-billions-of-events-for-less-than-100-mo/

https://dgraph.io/blog/post/serialized-roaring-bitmaps-golang/

https://news.ycombinator.com/item?id=32937930

Roaring Bitmap 将一个 32 位的整数分为两部分，一部分是高 16 位，另一部分是低 16 位。对于高 16 位，Roaring Bitmap 将它存储到一个有序数组中，这个有序数组中的每一个值都是一个“桶”；而对于低 16 位，Roaring Bitmap 则将它存储在一个 2^16 的位图中，将相应位置置为 1。这样，每个桶都会对应一个 2^16 的位图。

![geektime_roaring_bitmaps|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/geektime_roaring_bitmaps.9nzn9xk5at.webp)

相比于位图法，这种设计方案就是通过，将不存在的桶的位图空间全部省去这样的方式，来节省存储空间的。

而代价就是将高 16 位的查找，从位图的 O (1) 的查找转为有序数组的 log (n) 查找。

Roaring Bitmap 对低 16 位的位图部分进行了优化：
- 如果一个桶中存储的数据少于 4096 个，我们就不使用位图，而是直接使用 short 型的有序数组存储数据。
- 同时，我们使用可变长数组机制，让数组的初始化长度是 4，随着元素的增加再逐步调整数组长度，上限是 4096。这样一来，存储空间就会低于 8K，也就小于使用位图所占用的存储空间了
#### TF-IDF
term frequency–inverse document frequency 词频-逆文档频率

TF (t, d)=在文档 d 中词条 t 出现的次数​ / 文档 d 中所有的词条数目
IDF (t, D)=log (文档集合 D 的总文档数/包含词条 t 的文档数+1​)

TF-IDF = (TF) * (IDF)

#### 搜索推荐

索引构建、检索召回候选集和排序返回

#### B+Tree & LSM-Tree

### Load balancing

常见的负载均衡算法

1. 轮询算法（Round Robin）：依次将请求分配给每个后端服务器，循环进行。
2. 最小连接数算法（Least Connections）：将请求分配给当前连接数最少的后端服务器。
3. 加权轮询算法（Weighted Round Robin）：根据后端服务器的权重来进行轮询分配请求。
4. 加权最小连接数算法（Weighted Least Connections）：根据后端服务器的权重将请求分配给当前连接数最少的服务器。
5. 随机算法（Random）：随机选择一个后端服务器来处理请求。
#### Consistent Hash
采用环状减少节点数量变更带来的映射变更 workload

`hash(key) % len(nodes) -> hash(key) % len(ring length)`

使用虚拟节点，缩小节点 work 范围粒度，减少节点变更出现的「倾斜」，使得结果更均衡（虚拟节点位置一般是 `a-1`, `a-2`, `b-1`, `b-2` 做 hash 得到的，也是均衡分散在环上的），虚拟节点还可以支持权重配置
### Security

#### SSRF

服务器端请求伪造（也称为 SSRF）是一种常见的 Web 安全漏洞，允许攻击者诱导服务器端应用程序发出非预期的网络请求。

在典型的 SSRF 攻击中，攻击者可能会导致存在 SSRF 漏洞的服务器连接到组织内的内网服务。在其他情况下，他们可能会强制服务器连接到任意外部系统，从而可能泄露授权凭证等敏感数据，甚至可能导致命令执行。

典型的 SSRF 攻击如下，直接请求了用户可控的 URL 地址，

```go
// SECURITY RISK!
imgUrl := c.Query("image_url")
resp, _ := http.GET(imgUrl)
```

危害
SSRF 漏洞可以导致：
访问到存在漏洞服务器的数据，或者未授权访问其他内网服务的数据。

在某些情况下，SSRF 漏洞可能允许攻击者执行任意命令。
导致与外部第三方系统连接的 SSRF 漏洞利用可能会导致恶意向前攻击，这些攻击似乎源自托管易受攻击应用程序的组织。

修改和删除数据库记录：数据一致性是非常重要的，尤其是对于金融和银行平台。修改或者删除这些信息会导致业务严重中断。

代码执行：在数据库所在的机器执行任意命令，甚至接管整个机器，获取除了数据库以外的数据。攻击者可以利用这个机器作为跳板进一步攻击内网的其他机器。

解决方案：加上黑名单验证
#### XSS

攻击者利用应用程序没有对用户的输入，以及页面的输出进行严格地过滤，从而使恶意攻击者能往 Web 页面里插入恶意代码，当用户浏览该页面时，嵌入其中 Web 里面的恶意代码会被执行，从而达到恶意攻击者的特殊目的

建议:
1. 对用户输入的数据进行严格过滤，包括但不限于以下字符及字符串 `javascript script src img onerror alert < >` 
2. 根据页面的输出背景环境，对输出特殊字符进行编码/转义
3. 针对富文本编辑器，除了针对数据源的严格过滤之外，更加有效的方法就是梳理需要使用的安全 HTML 标签和属性白名单 (白名单即可，不需要的标签属性都干掉)，对 href 和 src 进行参数校验。如使用 `sanitize-html` 可参考: <https://www.npmjs.com/package/sanitize-html>
4. 在 Cookie 上设置 HTTPOnly 标志，从而禁止客户端脚本访问 Cookie

#### SQL Injection

1. 验证用户所有的输入数据，进行严格的过滤 (包括但不限于以下字符及字符串: `’、”、<、>、/、*、;、+、-、&、|、(、)、and、or、select、 union`)，某个数据被接受之前，使用标准输入验证机制，验证所有输入数据的长度、类型、语法以及业务规则，有效检测攻击;
2. 使用参数化查询 (`PreparedStatement`)，避免将未经过滤的输入直接拼接到 SQL 查询语句中;

### 密码学

#### 散列算法（Hashing Algorithm）

- MD（Message-Digest Algorithm）
- SHA（Secure Hash Algorithm）

#### 消息认证码（Message Authentication Code）

- HMAC（Hash-based Message Authentication Code）

##### 对称加密

对称加密，顾名思义就是加密和解密都是使用同一个密钥，常见的对称加密算法有 DES、3 DES 和 AES 等，其优缺点如下：
- 优点：算法公开、计算量小、加密速度快、加密效率高，适合加密比较大的数据。
- 缺点：
    1. 交易双方需要使用相同的密钥，也就无法避免密钥的传输，而密钥在传输过程中无法保证不被截获，因此对称加密的安全性得不到保证。
    2. 每对用户每次使用对称加密算法时，都需要使用其他人不知道的惟一密钥，这会使得发收信双方所拥有的钥匙数量急剧增长，密钥管理成为双方的负担。对称加密算法在分布式网络系统上使用较为困难，主要是因为密钥管理困难，使用成本较高。

##### 非对称加密

非对称加密，顾名思义，就是加密和解密需要使用两个不同的密钥：公钥（public key）和私钥（private key）。

公钥与私钥是一对，如果用公钥对数据进行加密，只有用对应的私钥才能解密；如果用私钥对数据进行加密，那么只有用对应的公钥才能解密。

非对称加密算法实现机密信息交换的基本过程是：

1. 甲方生成一对密钥并将其中的一把作为公钥对外公开；
2. 得到该公钥的乙方使用公钥对机密信息进行加密后再发送给甲方；
3. 甲方再用自己保存的私钥对加密后的信息进行解密。

常用的非对称加密算法是 RSA 算法，其优缺点如下：

- 优点：算法公开，加密和解密使用不同的钥匙，私钥不需要通过网络进行传输，安全性很高。
- 缺点：计算量比较大，加密和解密速度相比对称加密慢很多。

### 幂等性设计

调用接口有多个可能性返回，成功/失败/超时，其中「超时」难以处理，比如订单创建支付重试可能会有问题

- 一种是需要下游系统提供相应的查询接口。上游系统在 timeout 后去查询一下。如果查到了，就表明已经做了，成功了就不用做了，失败了就走失败流程。
- 另一种是通过幂等性的方式。也就是说，把这个查询操作交给下游系统，我上游系统只管重试，下游系统保证一次和多次的请求结果是一样的。

对于第一种方式，需要对方提供一个查询接口来做配合。

而第二种方式则需要下游的系统提供支持幂等性的交易接口。则需要全局 ID，要做到幂等性的交易接口，需要有一个唯一的标识，来标志交易是同一笔交易

### Unique ID generator

#### Design

- ID 需要按时间粗略排序
- ID 必须严格单增
- ID 不能泄露商业机密，例如每分钟创建的订单数量
- ID 是否会用于分片

```
+-------------------------------------------------------------+
| xx Bit Timestamp |  xx Bit MachineID  |  xx Bit Sequence ID |
+-------------------------------------------------------------+
```

确保 MachineID 全局唯一性，还需要结合 Distributed Lock 来使用

如果需要随机 ID，可以使用增量 ID 算法先生成，然后 reverse bit（`1011` 变为 `1101`）来实现

#### snowflake
- 41 bits 作为毫秒数。大概可以用 69.7 年。
- 10 bits 作为机器编号（5 bits 是数据中心，5 bits 的机器 ID），支持 1024 个实例。
- 12 bits 作为毫秒内的序列号。一毫秒可以生成 4096 个序号。

![snowflow uuid|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/snowflow_uuid.32hu0r96nl.webp)

存在时钟回拨的问题
- 可以等待到上次获取的时间点
- 预生成一部分 id
- 不依赖时间戳，而是利用 redis/mysql 做

#### partnervoucher/svs orderid

时间戳（second）+ random（5 位）

使用 redis setnx 保障唯一

利用类似凯撒密码来加密为 ordersn，对外都是 ordersn，可以根据 ordersn 反解

不需要递增，简单快速生成即可，使用 redis 保障了唯一，或者也可使用 machineid，保障分布式唯一（但是需要启动时增加 machine id try lock 逻辑）

而且 svs 使用 orderid 分表，这里不需要从 0 递增

### 限流设计

>[!NOTE]
> pXX 代表请求耗时的百分位，p 90 代表 90% 的请求都比当前快

#### Rate Limit

##### 固定窗口计数器
将时间划分为多个窗口，在每个窗口内每有一次请求就将计数器加一；如果计数器超过了限制数量，则本窗口内所有的请求都被丢弃当时间到达下一个窗口时，计数器重置。

优点：实现简单 
缺点：不够平滑，并且存在「两倍配置速率」问题。

**两倍配置速率**问题，考虑如下情况：
限制 1 秒内最多通过 5 个请求，在第一个窗口的最后半秒内通过了 5 个请求，第二个窗口的前半秒内又通过了 5 个请求。这样看来就是在 1 秒内通过了 10 个请求。

##### 滑动窗口计数器
将时间划分为多个区间，在每个区间内每有一次请求就将计数器加一维持一个时间窗口，占据多个区间；每经过一个区间的时间，则抛弃最老的一个区间，并纳入最新的一个区间。如果当前窗口内区间的请求计数总和超过了限制数量，则本窗口内所有的请求都被丢弃。

优点：避免双倍突发请求问题；时间区间的精度足够高时可以做到平滑
缺点：时间区间的精度越高，算法所需的空间容量就越大；依然存在突刺的情况

##### 漏桶
将每个请求视作「水滴」放入「漏桶」进行存储，「漏桶」以固定速率向外「漏」出请求来执行如果「漏桶」空了则停止「漏水」；如果「漏桶」满了则多余的「水滴」会被直接丢弃。

漏桶算法多使用队列实现，服务的请求会存到队列中，服务的提供方则按照固定的速率从队列中取出请求并执行，过多的请求则放在队列中排队或直接拒绝。

优点：解决了突刺现象
缺点：当短时间内有大量的突发请求时，即便此时服务器没有任何负载，每个请求也得在队列当中等一段时间才能被响应

场景：
**总量控制**：漏桶的保护是尽量缓存请求（缓存不下才丢），令牌桶的保护主要是丢弃请求（即使系统还能处理，只要超过指定的速率就丢弃，除非此时动态提高速率）。

##### 令牌桶
令牌以固定速率生成；生成的令牌放入令牌桶中存放，如果令牌桶满了则多余的令牌会直接丢弃。当请求到达时，会尝试从令牌桶中取令牌，取到了令牌的请求可以执行；如果桶空了，那么尝试取令牌的请求会被直接丢弃。

优点：既平滑分布，又能够承受范围内的突发请求，可以动态调整处理速度
缺点：每个请求必须从令牌桶取令牌，有性能要求

场景：
**速率控制**：控制自己的处理速度，控制访问第三方的速度
#### 动态限流

类似 TCP 拥塞控制算法，TCP 使用 RTT - Round Trip Time 来探测网络的延时和性能，从而设定相应的「滑动窗口」的大小，以让发送的速率和网络的性能相匹配

在业务场景中，可以使用 P 90/P 95 延时来探测网路情况

记录每个请求的耗时然后排序（浪费 CPU 资源）
- 可以采样进行
- 蓄水池抽样
### Container

VM
- 能在同一台机器上运行不同类型的实例，将物理资源转化为可共享的形式，有效利用了物理资源
- 与主机完全隔离，仅共享物理资源

Container
- 是一个进程（或一组进程），但与普通进程相比，与操作系统的隔离程度更高
- 但与虚拟机相比，其隔离性较差，因此安全性较低
### Golang 缓存库实现

#### GroupCache

一致性 Hash 选择节点，LRU 淘汰策略，不支持 set，也不支持过期，需要给 hook function，访问 cache 失败从 function 里获取并更新，singleflight 机制防止太多 function 执行

如何做过期？
加个定时的后台任务，扫描过期，粒度不好控制，可以每个 key 一个 cron fn，优化可以仿照 Ristretto 过期时间分桶，减少 ticker 数量
#### BigCache

无指针结构，免 GC，高速内存缓存
#### Ristretto

Ristretto 支持两种配置计算 memory 使用量
- total usage，每次 `set/remove` 都去 check 内存
- total key，不用每次都去 check，只用计算 key 总数（内存使用量不确定）
底层是 mutex shared map，`runtime.memhash` 计算分片
get 操作是加入到 shared 队列，批量 mutex 1 个执行
set 操作是加入到 channel，但是超过容量则 drop 这个操作

ttl 是将 expire 的 kyes 分桶存储（5 s），一次 check 一个桶的 key，对比每个 key 一个 ticker，减少 ticker 数量
### 前后端分离

#### 跨域
https://developer.mozilla.org/zh-CN/docs/Web/HTTP/CORS

面对浏览器的校验，A 域名 javascript 访问 B 域名会产生跨域
- A 发送请求 Origin `https://a.com`
- B 响应返回 Access-Control-Allow-Origin `https://a.com`（或者 `*`）

- Access-Control-Allow-Methods
- Access-Control-Allow-Headers
### Serverless

https://www.sqlite.org/serverless.html

- Classic Serverless：数据库引擎与应用程序在相同的进程、线程和地址空间中运行。没有消息传递或网络活动
- Neo-Serverless：数据库引擎在与应用程序不同的命名空间中运行，可能在一台单独的机器上，但数据库由托管提供商作为服务提供，不需要应用程序所有者进行管理或管理
### 时间轮算法

- Hierarchical Wheel Timer algorithms

## WORKed

sg -> br 的 dts 任务平均延迟是 1s 左右，网络差或者集群状况差的情况会到半个小时以上

feature toggle QPS 90k

openapi QPS 60k，上报 QPS 50k+

kafka 同步延迟 1s 左右

push 系统推送 4000 QPS，大促或者活动是 3 倍
业务方推送的 QPS 是 11k 左右吧
目前 96 个 topic，平均到每个 topic 200 的 QPS

app 数量 2w+，开了 push 的数量 4000+
partner shop 表 1400w 记录，这些基本上是活跃的
partner merchant 70w 记录
### Openplatform

OPENAPI 日志上报 + 搜索优化 + 几个面板

- Opservice 好些需求，santistic，search 优化 (es function score + stop word)，对接了 datasuite 的 clickhouse 写 API，帮他们查了好几个 bug 了
- Opservice 的日志上报插件，我加了 senstive 的功能，基本上一半多我都改了，也算我实现的吧，日志上报插件功能其实也没啥，收集日志，处理+规范格式，发送到 kafka。lua kafka 调优之类的，修改了 buffer size.
- Opservice 的日常几个需求，其实记不太得了，改改 bug，没学到什么
- EKL 全组都是我接入的，虽然没学到什么（消费者模型？）
- Commercial metrics 的需求，也是对接的 data servcice（dataservice 的 common 是我设计的，个人觉得还不错）
- 几个面板也算是我做的，service partner，security dashboard，API santistic，log search.
- Partnership 完全我实现，跨团队合作

security 面板，安全团队扫描开发者填写的域名和 ip，然后发消息通知 opservice 这边。开发者处理好发工单回复，ops 会和 security 确认，并且流转状态，一些漏洞例子
- wordpress/tomcat/cisco/springboot h2 某个版本安全漏洞
- ES 组件暴露在外网，Redis/Druid Monitor 没有密码校验

### Sellerplatform

主要还是 miscellaneous + featuretoggle

- Shopsetting 页面，基本都是我加的
- CNSC Exchange rate，我写的，decimal 也是我设计的
- Seller center 网关插件修改，写得不多，sop 可以当作全是我写的
- Seller gateway 整体翻译成英文（脚本先翻译了一遍，后边又慢慢补遗漏的）
- 还做了点主子账号的需求，kafka + 主子账号之类的，其实啥也没有，接收到 sip 开店消息 kafka，然后选择合适的主子账号绑定
- CNSC 统一弹窗，我自己设计和实现的，现在有点臃肿，但是在当时挺合理
- Make checker 审批，我实现的，跨团队+kafka，保留拓展性（kafka state 字段返回，很多可以配置的地方）
- CNSC 侧边栏优化，主要是之前太慢了，feature toggle 对于有五千个以上店铺的 merchant 来说，就特别慢，所以做了优化，如果判断到存在最近 30/15 分钟的缓存，那就直接返回，如果没有，获取的时候如果超过 5 s，那就直接返回全部 sidebar，然后后台慢慢算
- 还有很多其他业务的开发
- FeatureToggle，主要是做了版本回滚，然后还有加了 merchant/subaccount 维度的 feature toggle，feature toggle 的难点主要是 qps 比较高，对于 db 查询压力比较大？（是吗），所以有带缓存的接口
- 网关做了些需求，缝缝补补，可以看下网关发布

https://www.cnblogs.com/upyun/p/17463973.html

#### featuretoggle

`featuretoggle_tag_mapping_<region>_tab`

`shop/merchant/subaccount_tag_mapping_{idx}_tab`

`tag_info_<region>_tab`

查询接口
is_in_feature_toggle (70 k QPS)
get_shops_feature_toogles (2 k QPS) // 主要是首页

##### 缓存设计
缓存设计：mapping 数据都是用的 hget，便于更新

feature toggle key tag mapping cache（feature key 维度）
- 更新的时候会先拿分布式锁
- 有软过期时间判断，命中缓存未过期直接返回旧数据，过期直接更新缓存（这里没有异步，应该是数据量不大）

2kw 个 shop

shop_tags 缓存 = tagid (8 b) * 100 (every shop average tag) * 2000w (2kw shop) ~= 15 GB

没有存储 shop-> featuretoogle 的缓存，时间换空间，假设存储 shop->featurekey 的缓存，需要多出 20 GB 左右

shop->tagids 长时间硬过期+短时间软过期（+random，防止雪崩），首先命中缓存判断软过期，未过期直接返回旧数据，过期开启异步任务更新缓存
类似于 CNSC 侧边栏

优化点
- 优化 cache 存储，对于 toggle 只存 id
- 优化内部 batch 接口编码，优化数据返回，get shops feature toogles（feature toggle key 编码为 idx 数组）

#### miscellaneous

高度使用异步加载和并发加载（CBSC 主子账号）

侧边栏软过期 + 异步加载 + 降级

make cheacker 审批流抽象，类型 + 类型关联 toggle 开关 + kafka 通知下游

内容框架，history announcement 后台加载，redis hash 储存 shop info，拉取失败 backoff 重试

统一弹窗，抽象出 channel 的概念，将弹窗的抽象为 内容 + 属性，属性控制弹窗种类，target user，站点 等等，内容则分为前端静态资源 component key 和静态内容以及 iframe(survey) 以及其它三方（加载和消失逻辑由他们控制）

consumer 对大卖做侧边栏缓存更新

consumer 做三方统一弹窗的消费逻辑，来取消弹窗

#### sellergateway

- 请求转发（spex, hspex, http）
- 流量控制（限流）
- 身份认证（业务自行实现）
- 灰度转发
- mock

hspex 实现 2 种方案
2 种都需要将 gin router 传给 hsepx，获取 router 并且做 proxy（相当于复制一份流量，一份给 gin，一份给自己）

1. 使用纯原生 spex 启动注册，view 层相当于复制一份，专门给 spex 使用，然后启动 spex server，改动很多，view 层都需要去处理，并且涉及 3 方 http 调用都需要改成 spex
2. 实现中间层，中间层负责将流量转为 spex，然后中间层自己注册到 spex，获取结果，中间层也负责将结果转为 http response

openresty 开发，插件是洋葱模型，content_by_lua 实现逻辑，log_by_lua 记录日志+上报 kafka+metrics

content_by_lua 变量生命周期会在请求结束终结，所以使用 ngx.ctx 来存储

每个 worker 存储路由表，每次更新会生成新的 prefix radix 和 regex radix 和完整匹配列表

1. admin 发送请求更新路由表 cache，会路由到随机一个 worker，然后计算更新，然后存储到 cache 中，30-60s 随机事件会去拉取（output 流量高）
2. 发布之后，worker 更新，存储到 share cache 中，30-60s 随机去拉取（减少到 1/4 output 流量）
3. http 广播告诉 worker 更新，local/share cache 做 fallback（调用 n 次接口，增加注册服务中间件，比较麻烦，好处就是更新及时并且快）
4. 写 cache 路由表冗余 n 份，worker 读取任意一份，分担集群 output 流量到每个 codis 节点（不太好，反而引入了 n 次更新，增加了不一致的可能）
5. 使用 version 更新，并引入 redis 由 worker 上报自己版本，admin 可以看到发布是否成功（好处是不强依赖 redis，只用来判断发布进度，并且拉取粒度不大，缺点还是需要等 30-60s 才会生效）

转发协议支持
1. http
2. hspex，组装转发出 spex 请求，返回 spex response 到 gateway，然后获取信息，主要 hspex 不需要以来 nginx 转发，而是通过 spex 做服务发现（利于 split by market），流量完全是 spex，对于业务自己的区别就是需要启动 hsepx server。spexheader 的信息会写到 http request context 中
3. spex，组装出 spex 请求，然后转发，返回 spex response 到 gateway，然后获取信息，业务不需要自己实现 spex 逻辑，可以使用 gateway proto header

接入 spex 的问题 cosocket 不支持获取特定连接和获取全部连接，而 agent 和 client 有超时逻辑，导致频繁新建连接，register 会产生 etcd boardcast，被 spex 团队要求修改
使用 go proxy 服务代理请求，cosocket 还是创建这么多请求，但是 go proxy 和 spex agent 之间就可以复用请求了

### 难点/亮点
1. log report 插件，从 0 到 1
	- content by lua 阶段实现
	- 洋葱模型 plugin (request, response, log)
	- kafka 日志上报，出现内存占用升高的问题，解决：调整 ringbuffer 以及 maxretry 的参数
	- 实现高效的打码规则，类似 jsonpath，不过简化版
	- 日志上报占用内存高的问题，优化方案：
		- 抽象出新的服务，让新的服务承载着部分流量，lua resty kafka 库本身的缺点，占用比 golang 同 qps 的服务高
2. seller-centre 首页
   1. 侧边栏，缓存，异步加载，cbsc 因为账号系统特殊，加载会很慢，所以超过阈值，会直接返回全量菜单，后台继续算
   2. 弹窗，消息队列使用，监听消息，写入弹窗
   3. make checker 审批流
3. partnership
	1. 个人完成，并推进开发测试进度
	2. 多个团队参与
	3. 项目带来收益

#### Openplatform 项目
##### 平滑迁移 partner key
难点：
移除 rabbitMQ 组件，改用存储+定时任务扫表实现
任务执行间隔 5 s

##### 推送优化
push system 是这样的

原有流程：br/sg 都是发送到 sg-pushgateway，然后再发到不同 topic（br 的再从 sg topic 同步到 br topic）sg/br consumer 消费

问题：br 的开发者有部署在 sg 的 APP，对于这部分目前现状是从 br consumer 回调，几百 ms 延迟，并且还有镜像任务的延迟。sg 开发者也有 br  APP，从 sg 推送也是一样的问题

修改后流程：
biz team 发送到 sg/br pushgateway 的 API，根据消息 region 分拣到 push consumer 的 topic 里，然后对于 br 消息的 sg APP 会分拣到 sg，再从 sg 消费发送到用户 callback 地址

实现流程
1. 部署了 br pushgateway 服务，添加 push APP 的 region 字段，默认 sg（逐步切换到 br）
2. 申请了 br 的 sg topic，负责将消息同步到 sg，申请了 sg 的 br topic，将消息同步到 br，添加同步任务（分保序/非保序消息 topic）
3. biz team 逐步切换域名
4. 上线后，提供 push APP region 测速功能，刷新存量开发者 push region 字段

效果？
原来的 push 消息，限制了消息推送方只能调用 sg API，所以如果从同一个 erp 提供商 br 消息里的 sg 开发者，br 的消息调用延迟 300~400 ms，sg 10 ms
导致消息堆积
修改后，接口调用延迟降低，20 ms

难点？
技术方案的设计，代码处理，涉及发送的地方都要判断，region 测速时超时等 error 的处理

##### Penalty
惩罚系统，主要是下面几个组件
- Rule
	- API call （数据来源于 log report）
	- Security vulnerabitlity
	- DPP 签署
- Punishment
	- API call limit（限制为同期上月 n% 或者 fixable）
	- APP/Account suspend
	- APP restricted
	- PII masking（写个 header 标记到 openapi 里）
- Noti
	- 每日发送邮件

实现上
- object selector
	- 过滤筛选 app，condition 串起来
- punishment indicator generate
	- 根据参数生成 indicator
	- condition 串联起来
- do punishment
	- API limit 就加入限制，调用时检查
	- APP/Account suspend 之类的，就写入表字段
##### log report
###### 难点
1. 发掘到高效的打码规则，类似 json path，比知道 key 每个层级字符都遍历要简单高效，实现解释器，类似 json 解析器，状态转移的实现
2. 内存占用高的排查和优化，代码本身的处理

###### 存在的问题和可优化的地方
实现耦合在网关，不好做其它优化，必须依附于 openresty

lua resty kafka 不好的地方
1. 社区活跃度较低，不支持消息压缩（这一点影响小）
2. 内存使用不节制（影响大）

我们有多个上报 log 的服务（push, api call），消息发送不统一
###### 内存占用升高
```lua
    KAFKA_PRODUCER_CONFIG = {
        producer_type = "async",
        request_timeout = 2000, --ms
        required_acks = 1, --## 1: master ack, 0: no ack, -1: all acks
        max_retry = 1,
        retry_backoff = 500, -- ms
        api_version = 2,
        keepalive_size = 1000, -- max qps per worker
        keepalive_timeout = 59000, --ms; less than kafka. we close the connection first.  doc=> https://kafka.apache.org/documentation/#brokerconfigs_connections.max.idle.ms
        error_handle = on_send_kafka_msg_error,
        batch_num = 1000, -- big enough for one coroutine to get all the data(others will quit if get no lock) to avoid buffer overflow
        max_buffering = 10000, -- pool size
        flush_time = 500, --flush frequency. make sure the ringbuffer data is consumed faster then producer producing
    },
```

主要是
max_buffering: queue.buffering.max.messages, ringbuffer 的最大消息容量
batch_num: message 将首先写入 ringbuffer。当缓冲区超过 batch_num 或每次 flush_time 刷新缓冲区时，它将发送到 kafka 服务器。

消息平均大小 25 kb（244 k/10）

memory 消耗问题
4 c 4 g 机器，占用到了 3.5 g，平常是 2 g 以下，原因

不上报的话是 1.5 g，正常上报是 2.5 g

memory usage = max_buffering * message_size * 4 (worker) = 10000 * 15 kb（按照 15 kb 算） * 4（worker 数量） = 600 m

后续需求添加了上报的数据，消息体接近翻倍，内存占用到了 3 g 以上，控制在 70%

于是修改参数优化，第一次尝试
max_buffering 50000 -> 10000
flush_time 1000 -> 500
batch_num 10000 -> 1000

降低了 buffer 的大小，减少了一点 memory 占用
新的 usage = max_buffering * message_size * 4(worker) = 1000 * 30kb * 4 = 120m
但是因为加快了 flush 的速度，所以内存分配变多了，差不多能保持在 2.6g 的样子

再后来，了解到 lua gc，于是扩容了机器内存，下个季度会做优化

而且存在千分之 1 的 API 是大于 1M 的，翻倍就是 2M

优化：
可以拆分出新的服务，实现 golang 的 kafka 上报，增加了网络开销，但是降低了维护成本和机器占有

其中 lua(LuaJIT) 的 gc 策略是占用内存需要达到当前内存的 2 倍才会 gc，所以会有很多占用不会回收，go 会基于分配策略来调整 gc 触发时机
而且 golang 的 samara 的吞吐量 qps 大于 lua resty kafka，所以 momory 占有会减少，cpu 利用率会上升，将上报和网关拆分，内存出现不稳定情况，不会影响到网关

nginx worker 进程中的内存分为两部分，一部分是由 lua 管理, 另一部分则是通过 glibc 通过 malloc 申请。通过 pmap 工具发现，RSS 增长的部分主要是由 lua 管理的 object 占用的内存。glibc 管理分配的内存没看到有泄漏，另外在 sellergateway 的配置中，有加了 lua_malloc_trim 10000 (设置清理内存的指令周期), 保证 glibc 的内存被释放。因此 lua 管理的 object 是内存上涨的主要原因。查看 lua GC 的触发时机，默认情况下，lua 触发 gc 的机制是当前内存的使用量是上一次 GC 后的 2 倍，也就是说，假如在某次 gc 后，nginx worker 进程的当前内存使用是 1.1 GB, 那么随着更多的 lua object 创建，内存达到了 1.9 GB,  lua 不会触发 GC，内存会一直上涨到 2 GB, 触发文初提到的 2 GB 的上限问题，导致了 panic

ELK 里的 logstash，为什么不使用
- 我们这边是公司开发的 log agent，没办法实现一些过滤和业务逻辑
#### seller center 侧边栏/弹窗
##### 侧边栏
因为 CBSC 绑定店铺很多，而且每个 shop 都需要判断 feature toggle 和 authcode，所以采用异步加载的方案，检查缓存是否上次设置时间是否是 15 分钟之内，是的话直接返回
不是的话，进行加载，加载时间超过 3s 就直接返回上次计算出的，或者没有计算出的数据的话，就返回全量数据，在后台继续计算，然后设置回去
计算过程也并行的家在，加载 feature toggle，加载 shop 列表

消息
监听 subaccount 和 shop 变化关系，提前做计算

定时任务：定时更新大卖的侧边栏
##### 弹窗/内容
难点
业务本身，设计一套弹窗规范，数据表结构，target user mapping 关系存储，多个团队对接，设置和通知弹窗状态

内容 announcement 在 CBSC 可以查看 history 的 announcement，也是存在 shop 数很多的情况，也是类似的处理，先将计算好的结果直接返回，然后还有没算完的后台继续计算缓存

#### partner voucher 难点
partner voucher 本身业务不难，因为虚拟物品没有库存的概念，注意 balance 扣减
审批流，状态流转，并且有补偿手段，添加 cli 可以手动执行状态变更，手动操作等

定时任务扫描执行待发货的
### Tracing 总结

采样率可以通过设置，反映在 span id 上
尾部采样是支持的，可以上报特殊的 flag 信息就行

出现的问题

- span id 断开的问题
	- gin 的问题

### 问题/事故

##### gin context 涉及的问题
因为接入 tracing，需要上报 span id，在我们的系统内部，传递信息是使用 context
默认的 tracing sdk 是写入到 `c.Request.Context`，在下面是拿不到的，因为后续使用和上报都是用的 gin.Context
导致问题：tracing 信息上报失败，没法取到 tracing，导致每次都生成新的 span，最后 jaeger 上看到的就是断开的链路

于是设置 ContextWithFallback 为 true，这样 context 取值的时候会 fallback 回 request context

但是这样还会有问题
现象：上述修改上线后，发现有些异步任务访问 db/redis 会出现 context canceled 的情况，也就是 ctx 关闭了

异步任务是使用外部传入的 context，那也就是 gin.Context done 了，因为设置了 ContextWithFallback 参数，导致请求结束后被 done，收到了 done 消息导致的

还会导致 panic，如果没有 handle 到 panic，就会导致容器退出，gin.Context 是复用的，每次都会清空重新设置 c.Request，如果有复用，并且其它协程正在使用这个 context，就会导致 nil pointer panic

##### DB rows 没有正确关闭问题
现象：console 页面打不开，登录失败

登录失败，也就是请求没有处理，看到协程数和 CPU 数不断增加（增加很缓慢，开始没有发现）
只能使用 go 的 pprof 看了下，看到协程数很多，看到调用的 func 很多都是创建连接

gorm 中调用 `Rows()` 函数进行查询的时候，需要获取一个连接。策略是：
1. 如果连接池中有空闲连接，返回一个空闲的
2. 如果连接池中没有空的连接，且没有超过最大创建的连接数，则创建一个新的返回
3. 如果连接池中没有空的连接，且超过最大创建的连接数，则等待连接释放后，返回这个空闲连接

所以也就是超过连接数了，接口响应失败
频繁创建也就是说我们没有正确释放这个连接导致的
##### metrics 导致内存上涨

发版后几个小时后，看到容器内存上涨很多，已经有容器 oom 了

内存上涨可能是 
- QPS 上涨
- Kafka 消息突增

以上两点都排查了，没有问题
查看修改点，看到引入了新的 cache key，但是 cache 不会影响服务内存
查看 cache 监控发现上报的 cache key 不对，带上了 shopid 后缀，导致 metrics 数据非常多，没有做聚合

##### feature toggle 缓存穿透

feature toggle 被压测流量压爆

VN db 压测流量导致，缓存没有命中，请求到 RDS，但是 RDS 性能比较差，导致 DB 挂掉

##### cache 连接数配置错误

ucache 连接数配置得太高了

subaccount 批量解绑绑定任务，执行时触发重复执行的 bug，导致创建太多与 codis 的连接，codis ha proxy 负载增加然后挂掉了
### Split by market

##### Push

如上「Push 优化」

其实就是之前的发送粒度是 developer，推送方推送过来只是根据 partner region 分拣到 push consumer br/sg
但是，因为存在 br 开发者，但是它业务是在 sg 的，并且授权了 sg 的店铺，所以其实最终还是要分拣到 push consumer sg

所以，拆分了 sg consumer 的 br topic 和 br consumer 的 sg topic，然后同步到对应 region 消费，其实这一步就是根据 app 的配置来分拣
然后发送

延迟降低了很多，对于绝大多数请求推送延迟都降低到了 20ms 以下，对比之前推送延迟 200ms+

还有一些相关的修改，请求一些请求成功率/惩罚关停 也改为根据 app 的推送配置来请求获取

##### Openplatform

问题：br 的开发者，他授权有 sg/br 的 shop，请求是被转发到 sg，所以最少一次跨 zone，对于 br shop，还会跨 zone 获取店铺信息

优化这部分逻辑，添加 br 的部署，包括 br gateway，br 授权鉴权接口及其页面

1. 搭一套 br 容器，br 域名的流量转发到 br 服务
2. 授权鉴权信息使用对应 db，对授权信息 partner_shop_tab 和 partner_merchant_tab 做忽略主键 sg/br 双向同步
3. 添加 log report 的 br topic，以及同步任务，同步回 sg 消费
4. 推进 biz team 部署一套 br 容器，spex 接口则添加路由配置，对于个别还是依旧存在跨 zone 请求

### DR

DR 是灾备处理，SG 出现过机房失火的情况，所以在 US 提供同样一套服务，出现问题时，切换到 US 机房，平时 US 不一定会对外提供服务

主要面临的问题
1. 网络问题，专线单向延迟在 240ms 左右，DB 同步在次延迟下，单集群最高只有 5MB/s 的速率，所以可能会出现 1h 的延迟
2. 数据问题，切换后的数据恢复，对于 US，在数据同步过程中未被同步的部分数据，如何做修复
3. 

#### Kafka

DR 同步方案
- 对于消息类型的 Kafka 集群，使用 MirrorMaker2 官方组件同步消息和 offset
- 对于 Binlog 类型的，默认只开启 binlog 处理，当作 slave 库，kafka 本身不同步；DR 时被切换为 master 库，然后消费 kafka（工具去手动设置 offset）

两种方案都需要处理幂等消息

DR 时业务还有 DR toggle，监听到 DR az 开启的配置，就关闭容器处理（consumer/cronjob 之类的）

![mirrormaker2|700](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/kafka_mirrormaker.4n7lrl51xv.webp)

## Algorithm

二进制枚举
- [x]  [78. 子集](https://leetcode-cn.com/problems/subsets/)
- [ ]  [1178. 猜字谜](https://leetcode-cn.com/problems/number-of-valid-words-for-each-puzzle/)
- [ ]  [1255. 得分最高的单词集合](https://leetcode-cn.com/problems/maximum-score-words-formed-by-letters/)
- [ ]  [1601. 最多可达成的换楼请求数目](https://leetcode-cn.com/problems/maximum-number-of-achievable-transfer-requests/)
- [ ]  [2002. 两个回文子序列长度的最大乘积](https://leetcode-cn.com/problems/maximum-product-of-the-length-of-two-palindromic-subsequences/)
- [ ]  [5992. 基于陈述统计最多好人数](https://leetcode-cn.com/problems/maximum-good-people-based-on-statements/)

SegmentTree（线段树）
https://leetcode.com/tag/segment-tree/

Dynamic program
**Dynamic Programming Related Problems (718. Maximum Length of Repeated Subarray)**
- [x]  [3. Unique Paths](https://leetcode.com/problems/unique-paths/)
- [ ]  [5. Pascal's Triangle](https://leetcode.com/problems/pascals-triangle/)
- [ ]  [6. Maximum Product Subarray](https://leetcode.com/problems/maximum-product-subarray/)
- [ ]  [7. House Robber](https://leetcode.com/problems/house-robber/)
- [ ]  [8. Longest Increasing Subsequence](https://leetcode.com/problems/longest-increasing-subsequence/)
- [ ]  [9. Coin Change](https://leetcode.com/problems/coin-change/)
- [ ]  [10. Counting Bits](https://leetcode.com/problems/counting-bits/)
- [ ]  [11. Ones and Zeroes](https://leetcode.com/problems/ones-and-zeroes/)
- [ ]  [12. Target Sum](https://leetcode.com/problems/target-sum/)

### ByteDance Top 100

1. **[Number of Atoms](https://leetcode.com/problems/number-of-atoms)**
2. **[Game](https://leetcode.com/problems/24-game)**
3. **[Splitting a String Into Descending Consecutive Values](https://leetcode.com/problems/splitting-a-string-into-descending-consecutive-values)**
4. **[Shortest Path to Get All Keys](https://leetcode.com/problems/shortest-path-to-get-all-keys)**
5. **[Add Bold Tag in String](https://leetcode.com/problems/add-bold-tag-in-string)**
7. **[Course Schedule III](https://leetcode.com/problems/course-schedule-iii)**
8. **[Binary Tree Longest Consecutive Sequence](https://leetcode.com/problems/binary-tree-longest-consecutive-sequence)**
9. **[Russian Doll Envelopes](https://leetcode.com/problems/russian-doll-envelopes)**
13. **[Binary Tree Right Side View](https://leetcode.com/problems/binary-tree-right-side-view)**
14. **[Find Median from Data Stream](https://leetcode.com/problems/find-median-from-data-stream)**
15. **[Maximum Product of Three Numbers](https://leetcode.com/problems/maximum-product-of-three-numbers)**
16. **[Subsets](https://leetcode.com/problems/subsets)**
17. **[Knight Dialer](https://leetcode.com/problems/knight-dialer)**
18. **[Number of Distinct Islands](https://leetcode.com/problems/number-of-distinct-islands)**
19. **[Lowest Common Ancestor of a Binary Tree](https://leetcode.com/problems/lowest-common-ancestor-of-a-binary-tree)**
20. **[Robot Room Cleaner](https://leetcode.com/problems/robot-room-cleaner)**
21. **[Merge Intervals](https://leetcode.com/problems/merge-intervals)**
22. **[Course Schedule II](https://leetcode.com/problems/course-schedule-ii)**
23. **[Subarray Sum Equals K](https://leetcode.com/problems/subarray-sum-equals-k)**
24. **[Random Pick Index](https://leetcode.com/problems/random-pick-index)**
25. **[Search a 2D Matrix II](https://leetcode.com/problems/search-a-2d-matrix-ii)**
26. **[Reorganize String](https://leetcode.com/problems/reorganize-string)**
27. **[Longest Increasing Subsequence](https://leetcode.com/problems/longest-increasing-subsequence)**
29. **[N-Queens](https://leetcode.com/problems/n-queens)**
30. **[Merge k Sorted Lists](https://leetcode.com/problems/merge-k-sorted-lists)**
31. **[LRU Cache](https://leetcode.com/problems/lru-cache)**
32. **[Basic Calculator](https://leetcode.com/problems/basic-calculator)**
33. **[Palindrome Partitioning](https://leetcode.com/problems/palindrome-partitioning)**
34. **[Construct Binary Tree from Preorder and Inorder Traversal](https://leetcode.com/problems/construct-binary-tree-from-preorder-and-inorder-traversal)**
35. **[Meeting Rooms II](https://leetcode.com/problems/meeting-rooms-ii)**
36. **[Longest String Chain](https://leetcode.com/problems/longest-string-chain)**
37. **[Coin Change 2](https://leetcode.com/problems/coin-change-2)**
38. **[Target Sum](https://leetcode.com/problems/target-sum)**
39. **[Insert Delete GetRandom O(1)](https://leetcode.com/problems/insert-delete-getrandom-o1)**
40. **[Convert Binary Search Tree to Sorted Doubly Linked List](https://leetcode.com/problems/convert-binary-search-tree-to-sorted-doubly-linked-list)**
41. **[Word Search](https://leetcode.com/problems/word-search)**
42. **[Course Schedule](https://leetcode.com/problems/course-schedule)**
43. **[Reverse Words in a String](https://leetcode.com/problems/reverse-words-in-a-string)**
44. **[Word Ladder](https://leetcode.com/problems/word-ladder)**
45. **[Minimum Size Subarray Sum](https://leetcode.com/problems/minimum-size-subarray-sum)**
46. **[Longest Palindromic Substring](https://leetcode.com/problems/longest-palindromic-substring)**
47. **[Serialize and Deserialize Binary Tree](https://leetcode.com/problems/serialize-and-deserialize-binary-tree)**
48. **[Happy Number](https://leetcode.com/problems/happy-number)**
49. **[Find First and Last Position of Element in Sorted Array](https://leetcode.com/problems/find-first-and-last-position-of-element-in-sorted-array)**
50. **[Merge Two Sorted Lists](https://leetcode.com/problems/merge-two-sorted-lists)**
51. **[Sliding Window Maximum](https://leetcode.com/problems/sliding-window-maximum)**
52. **[Minimum Window Substring](https://leetcode.com/problems/minimum-window-substring)**
54. **[Search in Rotated Sorted Array](https://leetcode.com/problems/search-in-rotated-sorted-array)**
55. **[Move Zeroes](https://leetcode.com/problems/move-zeroes)**
56. **[Binary Tree Maximum Path Sum](https://leetcode.com/problems/binary-tree-maximum-path-sum)**
57. **[Search a 2D Matrix](https://leetcode.com/problems/search-a-2d-matrix)**
58. **[Top K Frequent Elements](https://leetcode.com/problems/top-k-frequent-elements)**
59. **[Generate Parentheses](https://leetcode.com/problems/generate-parentheses)**
60. **[Word Break](https://leetcode.com/problems/word-break)**
61. **[Next Permutation](https://leetcode.com/problems/next-permutation)**
62. **[Permutations](https://leetcode.com/problems/permutations)**
63. **[Plus One](https://leetcode.com/problems/plus-one)**
64. **[Validate Binary Search Tree](https://leetcode.com/problems/validate-binary-search-tree)**
65. **[Best Time to Buy and Sell Stock](https://leetcode.com/problems/best-time-to-buy-and-sell-stock)**

### 题单
#### Krahets 笔面试精选 88 题

https://leetcode.cn/studyplan/selected-coding-interview/

### 优先队列/单调队列/滑动窗口

#### [239. 滑动窗口最大值](https://leetcode.cn/problems/sliding-window-maximum/)

##### 优先队列
```go
func parent(i int) int {
	return (i - 1) / 2
}

func leftchild(i int) int {
	return i*2 + 1
}

func rightchild(i int) int {
	return i*2 + 2
}

type Element struct {
	Val int
	Idx int
}

type PriorityQueue []Element

func (p *PriorityQueue) Push(v Element) {
	*p = append(*p, v)

	p.swim(p.Len() - 1)
}

func (p *PriorityQueue) Len() int {
	return len(*p)
}

func (p *PriorityQueue) Top() Element {
	return (*p)[0]
}

func (p *PriorityQueue) Pop() {
	(*p)[0], (*p)[p.Len()-1] = (*p)[p.Len()-1], (*p)[0]
	(*p) = (*p)[:p.Len()-1]

	p.sink(0)
}

func (p *PriorityQueue) sink(i int) {
	for i < p.Len() {
		leidx, riidx := leftchild(i), rightchild(i)
		maxidx := i

		if leidx < p.Len() && (*p)[leidx].Val > (*p)[maxidx].Val {
			maxidx = leidx
		}

		if riidx < p.Len() && (*p)[riidx].Val > (*p)[maxidx].Val {
			maxidx = riidx
		}

		if maxidx == i {
			break
		}

		(*p)[maxidx], (*p)[i] = (*p)[i], (*p)[maxidx]
		i = maxidx
	}
}

func (p *PriorityQueue) swim(i int) {
	for i >= 0 {
		pidx := parent(i)

		if pidx < 0 {
			break
		}

		if (*p)[pidx].Val >= (*p)[i].Val {
			break
		}

		(*p)[pidx], (*p)[i] = (*p)[i], (*p)[pidx]
		i = pidx
	}
}

func maxSlidingWindow(nums []int, k int) (ans []int) {
	pq := PriorityQueue{}

	for i := 0; i < k; i++ {
		pq.Push(Element{nums[i], i})
	}

	ans = append(ans, pq.Top().Val)

	for i := k; i < len(nums); i++ {
		pq.Push(Element{nums[i], i})

		for pq.Top().Idx <= i-k {
			pq.Pop()
		}

		ans = append(ans, pq.Top().Val)
	}

	return ans
}
```

#### [3. 无重复字符的最长子串](https://leetcode.cn/problems/longest-substring-without-repeating-characters/)

```go
func lengthOfLongestSubstring(s string) (ans int) {
	pos := make(map[byte]int)

	st := 0

	for i := 0; i < len(s); i++ {
		if _, ok := pos[s[i]]; ok {
			st = max(st, pos[s[i]]+1)
		}

		ans = max(ans, i-st+1)

		pos[s[i]] = i
	}

	return
}
```
#### [128. 最长连续序列](https://leetcode.cn/problems/longest-consecutive-sequence/)

计数处理，不要重复计算 num-1 的序列

```go
func longestConsecutive(nums []int) int {
	cntmap := make(map[int]int)

	ans := 0

	for _, v := range nums {
		cntmap[v]++
	}

	for _, v := range nums {
		if _, ok := cntmap[v-1]; ok {
			continue
		}

		len := 1

		for {
			if _, ok := cntmap[v+1]; !ok {
				break
			}

			v++
			len++
		}

		ans = max(ans, len)
	}

	return ans
}
```

#### [76. 最小覆盖子串](https://leetcode.cn/problems/minimum-window-substring/)

滑动窗口，首先滑动到符合条件的右边界，然后滑动左边界直到不符合条件

这里还需要注意是判断 `wcnt[sc] < tcnt[sc]`，而不是「不相等」

```go
func minWindow(s string, t string) string {
	le := 0
	ans := ""

	tcnt, wcnt := make(map[rune]int), make(map[rune]int)

	for _, c := range t {
		tcnt[c]++
	}

	num := 0

	for ed, c := range s {
		wcnt[c]++
		if wcnt[c] == tcnt[c] {
			num++
		}

		for le <= ed && num == len(tcnt) {
			if ed-le+1 < len(ans) || ans == "" {
				ans = s[le : ed+1]
			}

			sc := rune(s[le])
			wcnt[sc]--
			if wcnt[sc] < tcnt[sc] {
				num--
			}
			le++
		}
	}

	return ans
}
```

#### [209. 长度最小的子数组](https://leetcode.cn/problems/minimum-size-subarray-sum/)
##### 标准滑窗
```go
func minSubArrayLen(target int, nums []int) (ans int) {
	le, sum := 0, 0

	ans = len(nums) + 1

	for i, num := range nums {
		sum += num
		for sum >= target {
			ans = min(ans, i-le+1)
			sum -= nums[le]
			le++
		}
	}

	if ans > len(nums) {
		return 0
	}

	return ans
}
```

##### 二分法
```go
func minSubArrayLen(target int, nums []int) int {
	n := len(nums)
	ans := math.MaxInt32
	psum := make([]int, n+1)
	for i := 1; i <= n; i++ {
		psum[i] = psum[i-1] + nums[i-1]
	}

	for i := 1; i <= n; i++ {
		t := target + psum[i-1]

		p := sort.SearchInts(psum, t)

		if p <= n {
			ans = min(ans, p-(i-1))
		}
	}

	if ans > len(nums) {
		return 0
	}

	return ans
}
```

#### [1658. 将 x 减到 0 的最小操作数](https://leetcode.cn/problems/minimum-operations-to-reduce-x-to-zero/)

这里需要反向思考怎么得到结果
1. 首先知道每次只能从开头或者结尾拿数字，就知道拿取的数字，在首尾两端形成长度 >= 0 的子数组
2. 反向考虑到未拿取的数字形成子数组，得到求 target 的最大子数组

这里滑动窗口的时候要使用 `for sum > target` 然后缩减左区间的方法
因为如果使用 `for sum >= target` 然后在内部判断结果并且缩减区间，就会导致问题，因为如果 `target == 0`，也就是每次都要 `le = ri + 1` 的时候才会计算结果，这样的话就会后者就会漏掉这个判断
```go
func minOperations(nums []int, x int) int {
	target := 0
	for _, v := range nums {
		target += v
	}
	target = target - x

	le, sum := 0, 0
	ans := math.MaxInt
	for i, v := range nums {
		sum += v
		for le <= i && sum > target {
			sum -= nums[le]
			le++
		}

		if sum == target {
			ans = min(ans, len(nums)-(i-le+1))
		}
	}

	if ans == math.MaxInt {
		return -1
	}

	return ans
}
```
### 单调栈

#### [503. 下一个更大元素 II](https://leetcode.cn/problems/next-greater-element-ii/)

如果当前元素比栈顶元素大，那么栈顶元素的下一个更大元素就是当前元素，所以出栈，如果没有则将当前元素入栈

使用 `2*n` 的方式，相当于构造出一个拼接的数组
```go
func nextGreaterElements(nums []int) (ans []int) {
	n := len(nums)

	ans = make([]int, n)
	for i := range ans {
		ans[i] = -1
	}

	sta := make([]int, 0)

	statop := func() int {
		return sta[len(sta)-1]
	}

	stapop := func() {
		sta = sta[:len(sta)-1]
	}

	for i := 0; i < n*2; i++ {
		for len(sta) > 0 && nums[statop()] < nums[i%n] {
			ans[statop()] = nums[i%n]
			stapop()
		}
		sta = append(sta, i%n)
	}

	return ans
}
```
### 数组

#### [56. 合并区间](https://leetcode.cn/problems/merge-intervals/)
```cpp
class Solution {
public:
    vector<vector<int> > merge(vector<vector<int> >& intervals) {
        sort(intervals.begin(), intervals.end());

		vector<vector<int> > ret;
		vector<int> cur = intervals[0];

		for (int i = 1; i < intervals.size(); i++) {
			auto interval = intervals[i];

			if (interval[0] <= cur[1]) {
				cur = {cur[0], max(cur[1], interval[1])};
			} else {
				ret.emplace_back(cur);
				cur = interval;
			}
		}

        ret.emplace_back(cur);

		return ret;
    }
};
```

#### [31. 下一个排列](https://leetcode.cn/problems/next-permutation/)
```cpp
class Solution {
public:
    void nextPermutation(vector<int>& nums) {
        int idx = nums.size() - 1;
        while (idx > 0 && nums[idx - 1] >= nums[idx]) idx--;

        reverse(nums.begin() + idx, nums.end());

        if (idx == 0) {
            return ;
        }

        for (int i = idx; i < nums.size(); i++) {
            if (nums[i] > nums[idx - 1]) {
                swap(nums[i], nums[idx - 1]);
                break;
            }
        }
    }
};
```

##### [Bigger is Greater](https://www.hackerrank.com/challenges/bigger-is-greater/problem?isFullScreen=true)

和上面的 31 题是一样的，在 hackerrank 上，只是是排列字符串，并且不要求没答案时排列成最小的
所以察觉到已经是最大排列时，直接返回即可

```cpp
string biggerIsGreater(string s) {
    int idx = s.size() - 1;
    while (idx > 0 && s[idx - 1] >= s[idx]) idx--;
    
    if (idx == 0) {
        return "no answer";
    }
    
    reverse(s.begin() + idx, s.end());
    
    for (int i = idx; i < s.size(); i++) {
        if (s[i] > s[idx - 1]) {
            swap(s[i], s[idx - 1]);
            break;
        }
    }
    
    return s;
}
```
### 贪心

#### [53. 最大子数组和](https://leetcode.cn/problems/maximum-subarray/)
```cpp
class Solution {
public:
    int maxSubArray(vector<int>& nums) {
        int ans = INT_MIN, sum = 0;

        for (int i = 0; i < nums.size(); i++) {
            sum += nums[i];
            ans = max(ans, sum);

            if (sum < 0) sum = 0;
        }


        return ans;
    }
};
```

#### [面试题 17.24. 最大子矩阵](https://leetcode.cn/problems/largest-submatrix-with-repeated-number/)

直接想到可以将「求子矩阵的最大值」转换成「求一维数组子数组的最大值」，但是这样复杂度是 `O(n^3 * m)`

所以这里的 `arr` 数组是在遍历时更新的，减少一个 `n` 的时间复杂度

这里很重要的一点是，`i` 和 `j` 和 `k` 其实是固定的，我们只能知道 `st`，所以当 `sum` < 0 的时候，我们只更新 `st`

```go
func getMaxMatrix(matrix [][]int) (ans []int) {
	n, m := len(matrix), len(matrix[0])
	maxVal := int(-1e8)

	for i := 0; i < n; i++ {
		arr := make([]int, m)

		for j := i; j < n; j++ {
			sum, st := 0, 0
			for k := 0; k < m; k++ {
				arr[k] += matrix[j][k]
				sum += arr[k]

				if sum > maxVal {
					maxVal = sum
					ans = []int{i, st, j, k}
				}

				if sum < 0 {
					sum = 0
					st = k + 1
				}
			}
		}
	}

	return ans
}
```

#### [45. 跳跃游戏 II](https://leetcode.cn/problems/jump-game-ii/)

理解 `[起跳，结束]` 这个范围就可以了，不用知道每次到达的坐标，只用知道一次起跳能到达的范围，超过 `结束` 就需要次数 + 1

有点类似合并区间

```cpp
class Solution {
public:
    int jump(vector<int>& nums) {
        int end = 0, maxp = 0, ans = 0;

        for (int i = 0; i < nums.size() - 1; i++) {
            maxp = max(maxp, nums[i] + i);

            if (i == end) {
                ans++;
                end = maxp;
            }
        }

        return ans;
    }
};
```

### 买卖股票

https://leetcode.cn/circle/discuss/qiAgHn/

#### I (k = 1)
[121. 买卖股票的最佳时机](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock/)

```cpp
class Solution {
public:
    int maxProfit(vector<int>& prices) {
        vector<vector<int>> dp(prices.size(), vector<int> (2, 0));

        // dp[i][0] 代表第 i 天是没有持有股票的收益，dp[i][1] 代表是持有股票的收益

        dp[0][0] = 0, dp[0][1] = -prices[0];

        for (int i = 1; i < prices.size(); i++) {
            dp[i][0] = max(dp[i - 1][1] + prices[i], dp[i - 1][0]);
            dp[i][1] = max(dp[i - 1][1], - prices[i]);
        }

        return dp[prices.size() - 1][0];
    }
};
```

简化成两个变量存储
```cpp
class Solution {
public:
    int maxProfit(vector<int>& prices) {
        int buy = -prices[0], sell = 0;

        for (int i = 1; i < prices.size(); i++) {
            sell = max(buy + prices[i], sell);
            buy = max(-prices[i], buy);
        }

        return sell;
    }
};
```

##### 贪心
也可以贪心的判断
```cpp
class Solution {
public:
    int maxProfit(vector<int>& prices) {
        int minPrice = INT_MAX, profit = 0;

        for (auto& e : prices) {
            minPrice = min(e, minPrice);

            profit = max(profit, e - minPrice);
        }

        return profit;
    }
};
```


#### II (k = 任意, 贪心)
[122. 买卖股票的最佳时机 II](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-ii/)

```cpp
class Solution {
public:
    int maxProfit(vector<int>& prices) {
        vector<vector<int>> dp(prices.size(), vector<int> (2, 0));

        int ans = 0;

        dp[0][0] = -prices[0]; // hold 
        dp[0][1] = 0; // sell 

        for (int i = 1; i < prices.size(); i++) {
            dp[i][0] = max(dp[i - 1][1] - prices[i], dp[i - 1][0]);
            dp[i][1] = max(dp[i - 1][0] + prices[i], dp[i - 1][1]);

            // ans = max(dp[i][1], ans); 因为可以证明 dp[i][1] 收益肯定是单调递增的，所以可以去掉 ans
        }

        return dp[prices.size() - 1][1];
    }
};
```

##### 贪心
```cpp
class Solution {
public:
    int maxProfit(vector<int>& prices) {
        int ans = 0;

        for (int i = 1; i < prices.size(); i++) {
            auto cur = prices[i] - prices[i - 1];
            if (cur > 0) {
                ans += cur;
            }
        }

        return ans;
    }
};
```

#### III (k = 2)
[123. 买卖股票的最佳时机 III](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-iii/)

```go
func maxProfit(prices []int) (ans int) {
	// dp[i][j][k] 代表第 i 天在 j 状态（0: 持有股票，1: 不持有股票），处于第 k 次交易时的最大收益
	dp := make([][][]int, len(prices))
	for i := 0; i < len(prices); i++ {
		dp[i] = make([][]int, 2)
		for j := 0; j < 2; j++ {
			dp[i][j] = make([]int, 3)
		}
	}

	for k := 1; k < 3; k++ {
		dp[0][0][k] = -prices[0]
	}

	for i := 1; i < len(prices); i++ {
		for k := 1; k < 3; k++ {
			dp[i][0][k] = max(dp[i-1][1][k-1]-prices[i], dp[i-1][0][k])
			dp[i][1][k] = max(dp[i-1][0][k]+prices[i], dp[i-1][1][k])
		}
	}

	return dp[len(prices)-1][1][2]
}
```

#### IIII (k = n)
[188. 买卖股票的最佳时机 IV](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-iv/)

```go
func maxProfit(k int, prices []int) (ans int) {
	if k >= len(prices)/2 {
		return maxWithoutLimit(prices)
	}

	// dp[i][j][k] 代表第 i 天在 j 状态（0: 持有股票，1: 不持有股票），处于第 k 次交易时的最大收益
	dp := make([][][]int, len(prices))
	for i := 0; i < len(prices); i++ {
		dp[i] = make([][]int, 2)
		for j := 0; j < 2; j++ {
			dp[i][j] = make([]int, k+1)
		}
	}

	for j := 1; j <= k; j++ {
		dp[0][0][j] = -prices[0]
	}

	for i := 1; i < len(prices); i++ {
		for j := 1; j <= k; j++ {
			dp[i][0][j] = max(dp[i-1][1][j-1]-prices[i], dp[i-1][0][j])
			dp[i][1][j] = max(dp[i-1][0][j]+prices[i], dp[i-1][1][j])
		}
	}

	return dp[len(prices)-1][1][k]
}

func maxWithoutLimit(prices []int) (ans int) {
	for i := 1; i < len(prices); i++ {
		if prices[i]-prices[i-1] > 0 {
			ans += prices[i] - prices[i-1]
		}
	}

	return ans
}
```

#### [714. 买卖股票的最佳时机含手续费](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-with-transaction-fee/)

```cpp
class Solution {
public:
    int maxProfit(vector<int>& prices, int fee) {
        vector<vector<int>> dp(prices.size(), vector<int> (2, 0));

        // dp[i][0]: sell or not buy, dp[i][1]: hold or not sell;

        dp[0][0] = 0, dp[0][1] = -prices[0] - fee;

        for (int i = 1; i < prices.size(); i++) {
            dp[i][0] = max(dp[i - 1][0], dp[i - 1][1] + prices[i]);
            dp[i][1] = max(dp[i - 1][1], dp[i - 1][0] - prices[i] - fee);
        }

        return dp[prices.size() - 1][0];
    }
};
```

### 子序列和问题

##### 拆分成 2 个子数组

##### [1049. 最后一块石头的重量 II](https://leetcode.cn/problems/last-stone-weight-ii/)

###### 深搜记忆化
`dfs(stones, idx, cursum, sum)` 这样的深搜，但是这样会超时（在这道题的复杂度下），所以需要做记忆化

```cpp
class Solution {
public:
	map<pair<int, int>, int> cache;
    int lastStoneWeightII(vector<int>& stones) {
		return dfs(stones, 0, 0, accumulate(stones.begin(), stones.end(), 0));
	}

	int dfs(vector<int>& stones, int idx, int cur, int sum) {
		if (idx == stones.size()) {
			return abs(sum - 2 * cur);
		}

		if (cache.count({idx, cur})) {
			return cache[{idx, cur}];
		}

		int misum = min(dfs(stones, idx + 1, cur + stones[idx], sum),
					dfs(stones, idx + 1, cur, sum));

		cache[{idx, cur}] = misum;

		return misum;
	}
};
```

###### 动态规划

`dp[i][j]` 代表前 i 个 stone 在 j 的容量下能组成的最大值

```go
func lastStoneWeightII(stones []int) (ans int) {
	sum := 0
	for _, v := range stones {
		sum += v
	}

	// dp[i][j] 代表前 i 个 stone 在 j 的容量下能组成的最大值
	dp := make([][]int, len(stones)+1)
	for i := 0; i <= len(stones); i++ {
		dp[i] = make([]int, sum+1)
	}

	for i := 1; i <= len(stones); i++ {
		dp[i] = make([]int, sum+1)
		for j := 1; j <= sum; j++ {
			if j < stones[i-1] {
				dp[i][j] = dp[i-1][j]
				continue
			}
			dp[i][j] = max(dp[i-1][j], dp[i-1][j-stones[i-1]]+stones[i-1])
		}
	}

	return sum - 2*dp[len(stones)][sum/2]
}
```

#### [2035. 将数组分成两个数组并最小化数组和的差](https://leetcode.cn/problems/partition-array-into-two-arrays-to-minimize-sum-difference/)

#### [1755. 最接近目标值的子序列和](https://leetcode.cn/problems/closest-subsequence-sum/)

### 动态规划

#### [1143. 最长公共子序列](https://leetcode.cn/problems/longest-common-subsequence/)

```cpp
class Solution {
public:
    int longestCommonSubsequence(string text1, string text2) {
        int n = text1.size(), m = text2.size();
        vector<vector<int>> dp(n + 1, vector<int> (m + 1, 0));

        for (int i = 1; i <= text1.size(); i++) {
            for (int j = 1; j <= text2.size(); j++) {
                if (text1[i - 1] == text2[j - 1]) {
                    dp[i][j] = dp[i - 1][j - 1] + 1;
                } else {
                    dp[i][j] = max(dp[i - 1][j], dp[i][j - 1]);
                }
            }
        }

        return dp[n][m];
    }
};
```
#### [718. 最长重复子数组](https://leetcode.cn/problems/maximum-length-of-repeated-subarray/)

和 1143 很像的一道题，但是 1143 是子序列，定义和子数组不一样，子序列可以「删除」字符，所以子序列的答案允许通过上一级 `max(dp[i-1][j], dp[i][j-1])` 转移，子数组不行，如果当前不一样，那计数只能归零

```go
func findLength(nums1 []int, nums2 []int) int {
	n, m := len(nums1), len(nums2)

	dp := make([][]int, n+1)
	for i := 0; i <= n; i++ {
		dp[i] = make([]int, m+1)
	}

	ans := 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if nums1[i-1] == nums2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			}

			ans = max(dp[i][j], ans)
		}
	}

	return ans
}
```
#### [300. 最长递增子序列](https://leetcode.cn/problems/longest-increasing-subsequence/)

常规的 DP 题，思考好递增序列是如何生成的
```go
func lengthOfLIS(nums []int) (ans int) {
	dp := make([]int, len(nums))

	for i := 0; i < len(nums); i++ {
		dp[i] = 1
	}

	for i := 0; i < len(nums); i++ {
		for j := 0; j < i; j++ {
			if nums[i] > nums[j] {
				dp[i] = max(dp[i], dp[j]+1)
			}
		}
		ans = max(dp[i], ans)
	}

	return
}
```
#### [329. 矩阵中的最长递增路径](https://leetcode.cn/problems/longest-increasing-path-in-a-matrix/)

这道题挺难的，需要思考到递增序列如何产生，然后想到只有存在大小关系的相邻格才会对结果有影响，思考方式个人觉得很像「2D 接雨水」

这里还需要注意，如果通过二维 dp 的方式来获取最长递增子序列，这里会超时，需要使用一维 dp，为什么可以使用 1 维 dp，因为这里是可以枚举出需要比较的其它位置

##### 二维 DP

> 二维 DP 勉强没超时

```go
func longestIncreasingPath(matrix [][]int) (ans int) {
	type item struct {
		x, y int
		val  int
	}

	n, m := len(matrix), len(matrix[0])
	nums := make([]item, 0, n*m)

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			nums = append(nums, item{i, j, matrix[i][j]})
		}
	}

	sort.Slice(nums, func(i, j int) bool {
		return nums[i].val < nums[j].val
	})

	dp := make([]int, len(nums))

	for i := 0; i < len(dp); i++ {
		dp[i] = 1
	}

2

	for i := 0; i < len(nums); i++ {
		for j := 0; j < i; j++ {
			if nums[i].val == nums[j].val {
				continue
			}
			ix, iy := nums[i].x, nums[i].y
			jx, jy := nums[j].x, nums[j].y

			for k := 0; k < 4; k++ {
				if jx+dx[k] == ix && jy+dy[k] == iy {
					dp[i] = max(dp[j]+1, dp[i])
				}
			}
		}
		ans = max(dp[i], ans)
	}

	return ans
}
```
##### 一维 DP

这里最后简单的把 ans + 1，就算加上序列默认值 1 了

需要注意，这里如果写成上面的 `dp[x]` 一维的形式，就会导致错误，因为我们知道了 `jx` 和 `jy` 但是我们不知道 `dp[j]` 中的 `j` 在哪里，所以相当于信息不够了，需要改为用二维 `dp` 数组，只是还是遍历 `O(n)` 复杂度

```go
func longestIncreasingPath(matrix [][]int) (ans int) {
	type item struct {
		x, y int
		val  int
	}

	n, m := len(matrix), len(matrix[0])

	nums := make([]item, 0, n*m)

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			nums = append(nums, item{i, j, matrix[i][j]})
		}
	}

	sort.Slice(nums, func(i, j int) bool {
		return nums[i].val < nums[j].val
	})

	dp := make([][]int, n)
	for i := 0; i < n; i++ {
		dp[i] = make([]int, m)
	}

	dx, dy := []int{1, 0, -1, 0}, []int{0, 1, 0, -1}

	for i := 0; i < len(nums); i++ {
		ix, iy, v := nums[i].x, nums[i].y, nums[i].val

		for k := 0; k < 4; k++ {
			jx := ix + dx[k]
			jy := iy + dy[k]

			if jx < 0 || jx >= n || jy < 0 || jy >= m || matrix[jx][jy] >= v {
				continue
			}

			dp[ix][iy] = max(dp[jx][jy]+1, dp[ix][iy])
		}

		ans = max(dp[ix][iy], ans)
	}

	return ans + 1
}
```

##### 拓扑排序
偶然看到还可以用拓扑排序来做


```go
func longestIncreasingPath(matrix [][]int) (ans int) {
	n, m := len(matrix), len(matrix[0])
	edges := make([][]int, n*m)

	degs := make([]int, n*m)

	dx, dy := []int{1, 0, -1, 0}, []int{0, 1, 0, -1}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			for k := 0; k < 4; k++ {
				nx, ny := i+dx[k], j+dy[k]
				if nx < 0 || nx >= n || ny < 0 || ny >= m ||
					matrix[nx][ny] <= matrix[i][j] {
					continue
				}

				edges[i*m+j] = append(edges[i*m+j], nx*m+ny)
				degs[nx*m+ny]++
			}
		}
	}

	queue := make([]int, 0, n*m)
	for i := 0; i < n*m; i++ {
		if degs[i] == 0 {
			queue = append(queue, i)
		}
	}

	for len(queue) > 0 {
		size := len(queue)

		ans++

		for i := 0; i < size; i++ {
			u := queue[0]
			queue = queue[1:]

			for _, v := range edges[u] {
				degs[v]--
				if degs[v] == 0 {
					queue = append(queue, v)
				}
			}
		}
	}

	return ans
}
```

#### [115. 不同的子序列](https://leetcode.cn/problems/distinct-subsequences/)
和编辑距离很类似
```go
func numDistinct(s string, t string) int {
	// dp[i][j] => t[:j] 在 s[:i] 中出现的个数
	// result => dp[len(s)][len(t)]

	// s[i-1] == t[j-1] 求得 dp[i][j] = dp[i - 1][j - 1] + dp[i - 1][j]
	// s[i-1] != t[j-1] 求得 dp[i][j] = dp[i - 1][j]

	n, m := len(s), len(t)

	dp := make2dSlice(n+1, m+1)

	for i := 0; i <= n; i++ {
		dp[i][0] = 1
	}

	for j := 1; j <= m; j++ {
		dp[0][j] = 0
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			dp[i][j] = dp[i-1][j]
			if s[i-1] == t[j-1] {
				dp[i][j] += dp[i-1][j-1]
			}
		}
	}

	return dp[n][m]
}

func make2dSlice(row, col int) [][]int {
	arr := make([][]int, row)

	for i := 0; i < row; i++ {
		arr[i] = make([]int, col)
	}

	return arr
}
```

#### [887. 鸡蛋掉落](https://leetcode.cn/problems/super-egg-drop/)

https://leetcode.cn/problems/super-egg-drop/solutions/197163/ji-dan-diao-luo-by-leetcode-solution-2/

#### [322. 零钱兑换](https://leetcode.cn/problems/coin-change/)

`dp[0] = 0` 这个很重要，初始值 `INT_MAX` 也很重要，这里使用 `long long` 避免溢出

```cpp
class Solution {
public:
    int coinChange(vector<int>& coins, int amount) {
        vector<long long> dp(amount + 1, INT_MAX);
        sort(coins.begin(), coins.end());

        dp[0] = 0;

        // dp[i] = k: 拼凑到 i 使用 k 个硬币
        for (int i = 1; i <= amount; i++) {
            for (auto& c : coins) {
                if (i < c) break;

                dp[i] = min(dp[i], dp[i - c] + 1);
            }
        }
        
        return dp[amount] >= INT_MAX ? -1 : dp[amount];
    }
};
```

#### [518. 零钱兑换 II](https://leetcode.cn/problems/coin-change-ii/)

这道题非常重要，很容易出错

应该是外层遍历 `coins` 而不是 `amount`，因为是求「组合」而不是「排列」，组合中的 `coins` 组合不能重复的，对于 3 来说 `[2, 1]` 和 `[1, 2]` 是一样，如果外层是 `amount` 就会出现取到重复组合的情况

> 为什么外层是 `coins` 就不会取到重复组合呢？

因为每个外层 `coins` 只会遍历一次，不会取到重复的


```cpp
class Solution {
public:
    int change(int amount, vector<int>& coins) {
        vector<int> dp(amount + 1, 0);

        dp[0] = 1;

        for (auto& c : coins) {
            for (int i = 1; i <= amount; i++) {
                if (i < c) continue;

                dp[i] += dp[i - c];
            }
        }
        
        // for (auto& e : dp) cout << e << " " << endl;

        return dp[amount];
    }
};
```
#### [91. 解码方法](https://leetcode.cn/problems/decode-ways/)

> 这里为什么是 `0.. i-1` 序列？

这道题的关键是：对于序列 `0.. i` 的解码方式来说，如果当前 `i` 只是 `1-9` 那么，解码方式就是 `f (i) += f (i - 1)`，如果上一个 `i - 1` 和当前能组合成 `10-26`，那么就是 `f (i) += f (i - 2)`
##### 解法 1
```go
func numDecodings(s string) (ans int) {
	if s[0] == '0' {
		return 0
	}

	dp := make([]int, len(s)+1)
	dp[0], dp[1] = 1, 1

	for i := 2; i <= len(s); i++ {
		if s[i-1] >= '1' && s[i-1] <= '9' {
			dp[i] = dp[i-1]
		}

		if s[i-2] == '1' || (s[i-2] == '2' && s[i-1] <= '6') {
			dp[i] += dp[i-2]
		}
	}

	return dp[len(s)]
}
```
#### [64. 最小路径和](https://leetcode.cn/problems/minimum-path-sum/)

```cpp
class Solution {
public:
    int minPathSum(vector<vector<int>>& grid) {
        int n = grid.size(), m = grid[0].size();
        vector<vector<int>> dp(n, vector<int>(m, 0));

        dp[0][0] = grid[0][0];
        for (int i = 1; i < n; i++) {
            dp[i][0] += grid[i][0] + dp[i - 1][0];
        }
        
        for (int i = 1; i < m; i++) {
            dp[0][i] += grid[0][i] + dp[0][i - 1];
        }

        for (int i = 1; i < n; i++) {
            for (int j = 1; j < m; j++) {
                dp[i][j] = min(dp[i - 1][j], dp[i][j - 1]) + grid[i][j];
            }
        }

        return dp[n - 1][m - 1];
    }
};
```

#### [100290. 使矩阵满足条件的最少操作次数](https://leetcode.cn/problems/minimum-number-of-operations-to-satisfy-conditions/)

这里是 394 周赛题，写的时候开始没想到怎么去判断当冲突的时候 `dp[i][j]` 的值，看了题解才知道，直接枚举前后的选择就行，因为这里数据范围很小
```cpp
class Solution {
public:
    int minimumOperations(vector<vector<int>>& grid) {
		int n = grid.size(), m = grid[0].size();

		vector<vector<int>> cnt(m, vector<int>(10, 0));

		for (int j = 0; j < m; ++j) {
			for (int i = 0; i < n; ++i) {
				cnt[j][grid[i][j]]++;
			}
		}

		vector<vector<int>> dp(m, vector<int>(10, INT_MAX));

		// dp[i][j] 代表第 i 列如果选择刷成 j 时的最少操作数
		for (int j = 0; j <= 9; ++j) {
			dp[0][j] = n - cnt[0][j];
		}

		for (int i = 1; i < m; ++i) {
			for (int j = 0; j <= 9; ++j) {
				for (int k = 0; k <= 9; ++k) {
					if (k == j) continue ;
					dp[i][j] = min(dp[i][j], dp[i - 1][k] + n - cnt[i][j]);
				}
			}
		}

        return *min_element(dp[m - 1].begin(), dp[m - 1].end());
    }
};
```


#### 221. 最大正方形
```go
func maximalSquare(matrix [][]byte) (ans int) {
	n, m := len(matrix), len(matrix[0])
	dp := make([][]int, n)
	for i := 0; i < n; i++ {
		dp[i] = make([]int, m)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if matrix[i][j] == '1' {
				if i == 0 || j == 0 {
					dp[i][j] = 1
				} else {
					dp[i][j] = min(dp[i-1][j], dp[i][j-1], dp[i-1][j-1]) + 1
				}

				ans = max(ans, dp[i][j])
			}
		}
	}

	return ans * ans
}
```

#### [198. 打家劫舍](https://leetcode.cn/problems/house-robber/)

##### 二维 DP
```go
func rob(nums []int) (ans int) {
	// dp[i][k] 代表 nums[0:i+1] 所能获得的最大收益，k = 0 代表不偷窃，k = 1 偷窃
	if len(nums) == 0 {
		return 0
	}
	dp := make([][]int, len(nums))
	for i := 0; i < len(nums); i++ {
		dp[i] = make([]int, 2)
	}

	dp[0][1] = nums[0]

	for i := 1; i < len(nums); i++ {
		dp[i][0] = max(dp[i-1][1], dp[i-1][0])
		dp[i][1] = dp[i-1][0] + nums[i]
	}

	return max(dp[len(nums)-1][0], dp[len(nums)-1][1])
}
```

观察到可以化简为 2 个变量
```go
func rob(nums []int) (ans int) {
	if len(nums) == 0 {
		return 0
	}

	preSteal, PreNotSteal := nums[0], 0

	for i := 1; i < len(nums); i++ {
		curSteal := PreNotSteal + nums[i]
		curNotSteal := max(preSteal, PreNotSteal)

		preSteal, PreNotSteal = curSteal, curNotSteal
	}

	return max(preSteal, PreNotSteal)
}
```

##### 一维 DP
```go
func rob(nums []int) (ans int) {
	if len(nums) == 0 {
		return 0
	}

	if len(nums) == 1 {
		return nums[0]
	}

	dp := make([]int, len(nums))

	dp[0] = nums[0]
	dp[1] = max(nums[0], nums[1])

	for i := 2; i < len(nums); i++ {
		dp[i] = max(dp[i-2]+nums[i], dp[i-1])
	}

	return dp[len(nums)-1]
}
```

#### [213. 打家劫舍 II](https://leetcode.cn/problems/house-robber-ii/)

就是分类讨论下 `max(rob1(nums[:n-1]), rob1(nums[1:]))`，取开头不取结尾，和取结尾不取开头

```go
func rob(nums []int) (ans int) {
	if len(nums) == 0 {
		return 0
	}

	if len(nums) == 1 {
		return nums[0]
	}

	n := len(nums)

	dp := make([]int, n)
	for i := 0; i < n-1; i++ {
		a, b := 0, 0

		if i >= 1 {
			a = dp[i-1]
		}
		if i >= 2 {
			b = dp[i-2]
		}

		dp[i] = max(a, b+nums[i])
	}

	ans = dp[n-2]

	dp[0] = 0
	for i := 1; i < n; i++ {
		a, b := 0, 0

		if i >= 1 {
			a = dp[i-1]
		}
		if i >= 2 {
			b = dp[i-2]
		}
		dp[i] = max(a, b+nums[i])
	}

	ans = max(ans, dp[n-1])

	return ans
}

```
### 二叉树

####  [230. 二叉搜索树中第K小的元素](https://leetcode.cn/problems/kth-smallest-element-in-a-bst/)
##### 递归

```cpp
/**
 * Definition for a binary tree node.
 * struct TreeNode {
 *     int val;
 *     TreeNode *left;
 *     TreeNode *right;
 *     TreeNode() : val(0), left(nullptr), right(nullptr) {}
 *     TreeNode(int x) : val(x), left(nullptr), right(nullptr) {}
 *     TreeNode(int x, TreeNode *left, TreeNode *right) : val(x), left(left), right(right) {}
 * };
 */
class Solution {
public:
    vector<int> vals;

    int kthSmallest(TreeNode* root, int k) {
        traval(root);

        if (k - 1 >= vals.size()) return -1;

        return vals[k - 1];
    }
    
    void traval(TreeNode* root) {
        if (!root) return ;

        traval(root->left);
        vals.push_back(root->val);
        traval(root->right);
    }
};
```

##### Stack
```cpp
/**
 * Definition for a binary tree node.
 * struct TreeNode {
 *     int val;
 *     TreeNode *left;
 *     TreeNode *right;
 *     TreeNode() : val(0), left(nullptr), right(nullptr) {}
 *     TreeNode(int x) : val(x), left(nullptr), right(nullptr) {}
 *     TreeNode(int x, TreeNode *left, TreeNode *right) : val(x), left(left), right(right) {}
 * };
 */
class Solution {
public:
    int kthSmallest(TreeNode* root, int k) {
        stack<TreeNode*> sta;

        while (root || !sta.empty()) {
            while (root) {
                sta.push(root);
                root = root->left;
            }

            root = sta.top(); sta.pop();

            if (k == 1) {
                return root->val;
            }

            k--;

            root = root->right;
        }

        return -1;
    }
};
```

#### [1110. 删点成林](https://leetcode.cn/problems/delete-nodes-and-return-forest/)

```cpp
/**
 * Definition for a binary tree node.
 * struct TreeNode {
 *     int val;
 *     TreeNode *left;
 *     TreeNode *right;
 *     TreeNode() : val(0), left(nullptr), right(nullptr) {}
 *     TreeNode(int x) : val(x), left(nullptr), right(nullptr) {}
 *     TreeNode(int x, TreeNode *left, TreeNode *right) : val(x), left(left), right(right) {}
 * };
 */
class Solution {
public:
    vector<TreeNode*> ret;
    unordered_set<int> s;
    vector<TreeNode*> delNodes(TreeNode* root, vector<int>& to_delete) {
        for (auto e : to_delete) {
            s.insert(e);
        }

        if (helper(root)) {
            ret.push_back(root);
        }

        return ret;
    }


    TreeNode* helper(TreeNode* root) {
        if (root == nullptr) return nullptr;

        root->left = helper(root->left);
        root->right = helper(root->right);

        if (!s.count(root->val)) return root;

        if (root->left) ret.push_back(root->left);
        if (root->right) ret.push_back(root->right);

        return nullptr;
    }
};
```
#### [98. 验证二叉搜索树](https://leetcode.cn/problems/validate-binary-search-tree/)

用范围来界定的话会简单很多，注意数据范围
因为「范围」本身就带有从上往下遍历的历史记录
```cpp
class Solution {
public:
    bool isValidBST(TreeNode* root) {
        return traval(root, LONG_MIN, LONG_MAX);
    }

    bool traval(TreeNode* root, long min, long max) {
        if (root == nullptr) return true ;

        if (root->val <= min || root->val >= max) return false ;

        return traval(root->left, min, root->val) &&
                traval(root->right, root->val, max);
    }
};
```

#### [687. 最长同值路径](https://leetcode.cn/problems/longest-univalue-path/)
```cpp
/**
 * Definition for a binary tree node.
 * struct TreeNode {
 *     int val;
 *     TreeNode *left;
 *     TreeNode *right;
 *     TreeNode() : val(0), left(nullptr), right(nullptr) {}
 *     TreeNode(int x) : val(x), left(nullptr), right(nullptr) {}
 *     TreeNode(int x, TreeNode *left, TreeNode *right) : val(x), left(left), right(right) {}
 * };
 */
class Solution {
public:
    int ans = 0;
    int longestUnivaluePath(TreeNode* root) {
        traval(root);

        return ans;
    }

    int traval(TreeNode* root) {
        if (!root) return 0;

        int le = traval(root->left);
        int ri = traval(root->right);

        bool leb = root->left && root->left->val == root->val;
        bool rib = root->right && root->right->val == root->val;

        if (rib && leb) {
            ans = max(ans, le + ri + 2);
            return max(le, ri) + 1;
        }

        if (rib) {
            ans = max(ans, ri + 1);
            return ri + 1;
        }

        if (leb) {
            ans = max(ans, le + 1);
            return le + 1;
        }

        return 0;
    }
};
```

#### [297. 二叉树的序列化与反序列化](https://leetcode.cn/problems/serialize-and-deserialize-binary-tree/)

先序遍历序列化二叉树
```go
type Codec struct{}

func Constructor() (ans Codec) {
	return
}

func (c *Codec) serialize(root *TreeNode) string {
	if root == nil {
		return "#_"
	}

	res := strconv.Itoa(root.Val) + "_"

	res += c.serialize(root.Left)
	res += c.serialize(root.Right)

	return res
}

func (c *Codec) deserialize(data string) (ans *TreeNode) {
	vals := strings.Split(data, "_")
	if vals[len(vals)-1] == "" {
		vals = vals[:len(vals)-1]
	}

	idx := 0
	return helper(vals, &idx)
}

func helper(vals []string, idx *int) (ans *TreeNode) {
	if len(vals) == 0 || vals[*idx] == "#" {
		*idx++
		return nil
	}

	ans = &TreeNode{
		Val: formatVal(vals[*idx]),
	}

	*idx++
	ans.Left = helper(vals, idx)
	ans.Right = helper(vals, idx)

	return ans
}

func formatVal(val string) int {
	valInt, _ := strconv.Atoi(val)

	return valInt
}
```

#### [124. 二叉树中的最大路径和](https://leetcode.cn/problems/binary-tree-maximum-path-sum/)

```go
func maxPathSum(root *TreeNode) int {
	ans := math.MinInt

	var traval func(*TreeNode) int

	traval = func(root *TreeNode) int {
		if root == nil {
			return 0
		}

		le := max(traval(root.Left), 0)
		ri := max(traval(root.Right), 0)

		ans = max(le+ri+root.Val, ans)

		return max(le, ri) + root.Val
	}

	traval(root)

	return ans
}
```
### 栈与队列

#### [20. 有效的括号](https://leetcode.cn/problems/valid-parentheses/)

```cpp
class Solution {
public:
    bool isValid(string s) {
        stack<int> sta;

        unordered_map<int, int> map{
            {'}', '{'},
            {')', '('}, 
            {']', '['}
        };

        for (auto& c : s) {
            if (isRight(c)) {
                if (sta.empty() || map[sta.top()] != c) {
                    return false;
                }
                sta.pop();
            } else {
                sta.push(c);
            }
        }

        return sta.empty();
    }

    bool isRight(char c) {
        return c == ')' || c == '}' || c == ']';
    }
};
```

#### [155. 最小栈](https://leetcode.cn/problems/min-stack/) & [232. 用栈实现队列](https://leetcode.cn/problems/implement-queue-using-stacks/)

155: main 栈储存数据，help 栈储存当前最小值，`pop` 时判断 `help.top() == main.top()`，如果是需要一起弹出
232: main 栈和 help 栈，help 负责 `reverse` main 栈的内容

#### [394. 字符串解码](https://leetcode.cn/problems/decode-string/)

##### 栈解法

> 要考虑到 `num stack` 和 `string stack` 分别塞入的是什么值？

`num stack` 是保存 `[` 之前的要 `repeat` 的次数，`string stack` 储存的是之前部分产生的 res 值
当前的 `res` 值代表，还没计算倍数的序列 string（也是结果，结果就当作不会 * num 的序列就可以了），而且存在有嵌套 `[]` 的情况，所以这里的 `res` 很重要，递归也是同样的想法

```cpp
class Solution {
public:
    string decodeString(string s) {
        stack<int> nums;
        stack<string> strs;

        string res = "";
        int num = 0;
        for (int i = 0; i < s.size(); i++) {
            if (isdigit(s[i])) {
                num = num * 10 + s[i] - '0';
            } else if (isalpha(s[i])) {
                res += s[i];
            } else if (s[i] == '[') {
                nums.push(num); strs.push(res);
                num = 0; res = "";
            } else {
                string saved = strs.top(); strs.pop();
                int size = nums.top(); nums.pop();

                for (int j = 0; j < size; j++) {
                    saved += res; 
                }

                res = saved;
            }
        }

        return res;
    }
};
```
##### 递归解法
主要是要传递 idx 的引用

```cpp
class Solution {
public:
    string decodeString(string s) {
        int idx = 0;
        return recursion(s, idx);
    }

    string recursion(string s, int& i) {
        string res = "";

        while (i < s.size() && s[i] != ']') {
            if (!isdigit(s[i])) {
                res += s[i++];
            } else {
                int num = 0;
                while (i < s.size() && isdigit(s[i])) {
                    num = num * 10 + s[i++] - '0';
                }

                i += 1;
                string curs = recursion(s, i);
                i += 1;
                while (num--) {
                    res += curs;
                }
            }
        }

        return res;
    }
};
```

#### [1190. 反转每对括号间的子串](https://leetcode.cn/problems/reverse-substrings-between-each-pair-of-parentheses/)

和 394 是差不多的题
##### 递归解法

```cpp
class Solution {
public:
    string reverseParentheses(string s) {
		auto res = help(s, 0);

		return res.first;
    }

	pair<string, int> help(string s, int st) {
		string cur = "";
		for (int i = st; i < s.size(); i++) {
			char c = s[i];
			if (c == '(') {
				auto ns = help(s, i + 1);
				cur += ns.first;
				i = ns.second;
			} else if (c == ')') {
				return {reverse(cur), i};
			} else {
				cur += c;
			}
		}

		return {cur, s.size() - 1};
	}

	string reverse(string s) {
		for (int i = 0; i < s.size() / 2; i++) {
			auto tmp = s[i];
			s[i] = s[s.size() - 1 - i];
			s[s.size() - 1 - i] = tmp;
		}

		return s;
	}
};
```

##### 栈解法

```cpp
class Solution {
public:
    string reverseParentheses(string s) {
		stack<string> sta;

		string cur = "";
		for (auto& c : s) {
			if (c == '(') {
				sta.push(cur);
				cur = "";
			} else if (c == ')') {
				auto pre = sta.top(); sta.pop();
				cur = pre + reverse(cur);
			} else {
				cur += c;
			}
		}

		return cur;
    }

	string reverse(string s) {
		for (int i = 0; i < s.size() / 2; i++) {
			auto tmp = s[i];
			s[i] = s[s.size() - 1 - i];
			s[s.size() - 1 - i] = tmp;
		}

		return s;
	}
};
```
#### [295. 数据流的中位数](https://leetcode.cn/problems/find-median-from-data-stream/)

画个图看看就行，规定好`奇数`位的时候拿哪个部分里的数据
```cpp
class MedianFinder {
public:
    priority_queue<int> q1; // 大顶堆
    priority_queue<int, vector<int>, greater<int>> q2; // 小顶堆
    int count = 0;
    MedianFinder() {}

    void addNum(int num) {
        if (q1.size() == q2.size()) {
            q1.push(num);
            q2.push(q1.top()); q1.pop();
        } else {
            q2.push(num);
            q1.push(q2.top()); q2.pop();
        }
    }
    
    double findMedian() {
        if (q1.size() == q2.size()) {
            return (q2.top() + q1.top()) / 2.0;
        }

        return q2.top();
    }
};
```

#### [32. 最长有效括号](https://leetcode.cn/problems/longest-valid-parentheses/)

##### 栈
这个写法，不需要入栈 `)` 的坐标，所以我觉得更好理解一点
主要是要考虑到，遇到 `)` 后从栈拿出一个匹配之后：
1. 如果栈还有值，那么代表我们只能暂时确定 `i - sta.Top()` 是长度
2. 如果没有值，那么从上次的 st 开始，我们都是完整的匹配了每个括号，计数 `i-st+1`

```go
func longestValidParentheses(s string) (ans int) {
	sta := Stack{}

    st := 0

    for i, c := range s {
        if c == '(' {
            sta.Push(i)
            continue
        }

        if sta.IsEmpty() {
            st = i + 1
            continue
        }

        sta.Pop()

        if sta.IsEmpty() {
            ans = max(i - st + 1, ans)
        } else {
            ans = max(i - sta.Top(), ans)
        }
    }

    return ans
}

type Stack []int

func (s *Stack) Push(x int) {
	*s = append(*s, x)
}

func (s *Stack) Pop() int {
	x := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return x
}

func (s *Stack) Top() int {
    return (*s)[len(*s) - 1]
}

func (s *Stack) IsEmpty() bool {
	return len(*s) == 0
}
```

##### 计数法
直接写一个循环了，其实没必要
```go
func longestValidParentheses(s string) (ans int) {
    le, ri, rle, rri := 0, 0, 0, 0

    for i, c := range s {
        if c == '(' {
            le++
        } else {
            ri++
        }

        if s[len(s)-1-i] == '(' {
            rle++
        } else {
            rri++
        }

        if ri > le {
            le, ri = 0, 0
        } else if le == ri {
            ans = max(ans, 2 * le)
        }

        if rle > rri {
            rri, rle = 0, 0
        } else if rle == rri {
            ans = max(ans, 2 * rle)
        }
    }

    return ans
}
```

### 计算器

「计算器」基本上都是运用栈原理
#### [224. 基本计算器](https://leetcode.cn/problems/basic-calculator/)

这里的原理是，因为没有 `* /`，所以只需要考虑加减法，遇到左括号相当于就是一次压栈，保存上次的结果和当前括号前的正负号，出栈时先计算正负，然后加上上次的结果

因为是全加减运算，所以只用考虑正负号，但是这里还比较重要的一点是，`+` 和 `-` 其实相当于一个运算符 `* (1/-1)`

```go
func calculate(s string) int {
	sta := make([]int, 0)

	pop := func() int {
		x := sta[len(sta)-1]
		sta = sta[:len(sta)-1]
		return x
	}

	ans := 0
	flag := 1

	for i := 0; i < len(s); i++ {
		c := s[i]

		if s[i] >= '0' && s[i] <= '9' {
			num := 0

			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				num = num*10 + int(s[i]) - '0'
				i++
			}

			ans += flag * num
			flag = 1

			i--
			continue
		}

		switch c {
		case '+':
			flag = 1
		case '-':
			flag = -1
		case '(':
			sta = append(sta, ans)
			sta = append(sta, flag)

			flag = 1
			ans = 0
		case ')':
			ans *= pop()
			ans += pop()
		}
	}

	return ans
}
```

#### [227. 基本计算器 II](https://leetcode.cn/problems/basic-calculator-ii/)

这里要注意，`c == ' '` 这种情况，这种情况可能会在最后一个字符上
```go
func calculate(s string) (ans int) {
	pre := '+'
	num := 0

	sta := make([]int, 0)

	for i, c := range s {
		isdigit := c >= '0' && c <= '9'

		if isdigit {
			num = num*10 + int(c) - '0'
		}

		if !isdigit && c != ' ' || i == len(s)-1 {
			switch pre {
			case '+':
				sta = append(sta, num)
			case '-':
				sta = append(sta, -num)
			case '*':
				sta[len(sta)-1] *= num
			case '/':
				sta[len(sta)-1] /= num
			}

			pre = c
			num = 0
		}
	}

	for _, v := range sta {
		ans += v
	}

	return ans
}
```

#### [150. 逆波兰表达式求值](https://leetcode.cn/problems/evaluate-reverse-polish-notation/)

比较简单，注意出栈的顺序，是 y 先出
```go
func evalRPN(tokens []string) (ans int) {
	sta := make([]int, 0)

	push := func(x int) {
		sta = append(sta, x)
	}

	pop := func() int {
		x := sta[len(sta)-1]
		sta = sta[:len(sta)-1]
		return x
	}

	for _, token := range tokens {
		if token != "+" && token != "-" &&
			token != "*" && token != "/" {

			num, _ := strconv.Atoi(token)
			push(num)

			continue
		}

		y := pop()
		x := pop()

		switch token {
		case "+":
			push(x + y)
		case "-":
			push(x - y)
		case "*":
			push(x * y)
		case "/":
			push(x / y)
		}
	}

	return sta[0]
}
```

### 杂项
#### [409. 最长回文串](https://leetcode.cn/problems/longest-palindrome/)

记录奇数个数，最后只能选一个奇数存在
```cpp
class Solution {
public:
    int longestPalindrome(string s) {
        unordered_map<int, int> map;

        for (auto& c : s) map[c]++;

        int odd = 0;
        for (auto& [key, val] : map) {
            if (val % 2 == 1) odd++;
        }

        if (odd == 0) return s.size();

        return s.size() - odd + 1;
    }
};
```

### 双指针

#### [283. 移动零](https://leetcode.cn/problems/move-zeroes/)
```cpp
class Solution {
public:
    vector<vector<string>> groupAnagrams(vector<string>& strs) {
        unordered_map<string, vector<string>> map;

        for (auto& s : strs) {
            string sp = s;
            sort(sp.begin(), sp.end());

            map[sp].emplace_back(s);
        }

        vector<vector<string>> ret;

        for (auto &e : map) {
            ret.emplace_back(e.second);
        }

        return ret;
    }
};
```

#### [11. 盛最多水的容器](https://leetcode.cn/problems/container-with-most-water/)

`con = min(a, b) * (ib - ia)`
其中 ib - ia 是递减的，则要尝试增大 min(a, b) 才可能有更大的 con
```cpp
class Solution {
public:
    int maxArea(vector<int>& height) {
        int le = 0, ri = height.size() - 1, ans = 0;

        while (le < ri) {
            ans = max(min(height[le], height[ri]) * (ri - le), ans);

            if (height[le] < height[ri]) {
                le++;
            } else {
                ri--;
            }
        }

        return ans; 
    }
};
```

#### [15. 三数之和](https://leetcode.cn/problems/3sum/)
去除重复答案的部分比较重要，还有注意边界条件
```go
func threeSum(nums []int) (ans [][]int) {
	n := len(nums)

	sort.Slice(nums, func(i, j int) bool {
		return nums[i] < nums[j]
	})

	for i := 0; i < n-2; {
		j, k := i+1, n-1

		for j < k {
			psum := nums[j] + nums[k]
			if psum+nums[i] == 0 {
				ans = append(ans, []int{nums[i], nums[j], nums[k]})
				k--
				j++
				for k > j && nums[k+1] == nums[k] {
					k--
				}
				for k > j && nums[j-1] == nums[j] {
					j++
				}
			} else if psum+nums[i] > 0 {
				k--
				for k > j && nums[k+1] == nums[k] {
					k--
				}
			} else {
				j++
				for k > j && nums[j-1] == nums[j] {
					j++
				}
			}
		}
		i++
		for i < n-2 && nums[i-1] == nums[i] {
			i++
		}
	}

	return ans
}
```

#### [438. 找到字符串中所有字母异位词](https://leetcode.cn/problems/find-all-anagrams-in-a-string/)

```cpp
class Solution {
public:
    vector<int> findAnagrams(string s, string p) {
        if (s.size() < p.size()) return {};
        vector<int> cnts(26, 0), cntp(26, 0);

        for (auto& c : p) {
            cntp[c - 'a']++;
        }

        vector<int> ret;
        for (int i = 0; i < p.size(); i++) {
            cnts[s[i] - 'a']++;
        }

        if (cnts == cntp) {
            ret.push_back(0);
        }
        
        for (int i = p.size(); i < s.size(); i++) {
            cnts[s[i - p.size()] - 'a']--;
            cnts[s[i] - 'a']++;

            if (cnts == cntp) {
                ret.push_back(i - p.size() + 1);
            }
        } 

        return ret;
    }
};
```
### 回溯

#### [46. 全排列](https://leetcode.cn/problems/permutations/)
```cpp
class Solution {
public:
    vector<vector<int>> res;
    vector<vector<int>> permute(vector<int>& nums) {
        vector<bool> vis(nums.size(), false);
        vector<int> path;
        backtrack(nums, vis, path);

        return res;
    }

    void backtrack(const vector<int>& nums, vector<bool>& vis, vector<int>& path) {
        if (path.size() == nums.size()) {
            res.emplace_back(path);
            return ;
        }

        for (int i = 0; i < nums.size(); i++) {
            if (vis[i]) continue;

            vis[i] = true;
            path.push_back(nums[i]);
            backtrack(nums, vis, path);
            vis[i] = false;
            path.pop_back();
        }
    }
};
```

#### [47. 全排列 II](https://leetcode.cn/problems/permutations-ii/)

这段可以理解下
```cpp
if (vis[i] || (i > 0 && nums[i] == nums[i - 1] && !vis[i - 1])) {
    continue;
}
```

```cpp
class Solution {
public:
    vector<vector<int>> res;
    vector<vector<int>> permuteUnique(vector<int>& nums) {
        sort(nums.begin(), nums.end());

        vector<int> path;
        vector<bool> vis(nums.size(), false);

        backtrack(nums, path, vis);
	
        return res;
    }

    void backtrack(const vector<int>& nums, vector<int>& path, vector<bool>& vis) {
        if (path.size() == nums.size()) {
            res.emplace_back(path);

            return ;
        }

        for (int i = 0; i < nums.size(); i++) {
            if (vis[i] || (i > 0 && nums[i] == nums[i - 1] && !vis[i - 1])) {
                continue;
            }

            path.push_back(nums[i]);
            vis[i] = true;
            backtrack(nums, path, vis);
            vis[i] = false;
            path.pop_back();
        }
    }
};
```

#### [39. 组合总和](https://leetcode.cn/problems/combination-sum/)
```cpp
class Solution {
public:
    vector<vector<int>> res;
    vector<vector<int>> combinationSum(vector<int>& candidates, int target) {
        vector<int> path;
        sort(candidates.begin(), candidates.end());
        backtrack(candidates, path, target, 0, 0);

        return res;
    }

    void backtrack(const vector<int>& candidates, vector<int>& path, int target, int cursum, int idx) {
        if (cursum == target) {
            res.push_back(path);
        }

        for (int i = idx; i < candidates.size(); i++) {
            if (cursum + candidates[i] > target) continue;

            path.push_back(candidates[i]);
            backtrack(candidates, path, target, cursum + candidates[i], i);
            path.pop_back();
        }
    }
};
```

#### [40. 组合总和 II](https://leetcode.cn/problems/combination-sum-ii/)

```cpp
if (cursum + candidates[i] > target || (i > idx && candidates[i] == candidates[i - 1])) continue;
```

这行剪枝算法很重要，为什么是 `i > idx && (i > idx && candidates[i] == candidates[i - 1])`

是因为只有盘算到 `i > idx` 的时候判断 `candidates[i] == candidates[i - 1]` 才会有意义，`idx` 是实际上选择的数字

```cpp
class Solution {
public:
    vector<vector<int>> res;
    vector<vector<int>> combinationSum2(vector<int>& candidates, int target) {
        sort(candidates.begin(), candidates.end());
        vector<int> path;

        backtrack(candidates, path, 0, target, 0);
        return res;
    }

    void backtrack(const vector<int>& candidates, vector<int>& path, int idx, int target, int cursum) {
        if (cursum == target) {
            res.emplace_back(path);

            return;
        }

        for (int i = idx; i < candidates.size(); i++) {
            if (cursum + candidates[i] > target || (i > idx && candidates[i] == candidates[i - 1])) continue;
            path.push_back(candidates[i]);
            backtrack(candidates, path, i + 1, target, cursum + candidates[i]);
            path.pop_back();
        }
    }
};
```

#### [216. 组合总和 III](https://leetcode.cn/problems/combination-sum-iii/)
```cpp
class Solution {
public:
    vector<vector<int>> ret;
    vector<vector<int>> combinationSum3(int k, int n) {
        vector<int> path;
        backtrack(path, k, n, 1);

        return ret;
    }

    void backtrack(vector<int>& path, int k, int n, int st) {
        if (k == 0 && n == 0) {
            ret.emplace_back(path);
            return ;
        }

        for (int i = st; i <= 9; i++) {
            path.push_back(i);
            backtrack(path, k - 1, n - i, i + 1);
            path.pop_back();
        }
    }
};
```

#### [37. 解数独](https://leetcode.cn/problems/sudoku-solver/)

```go
func solveSudoku(board [][]byte) {
	rowsCnt, colsCnt, boxesCnt := [9][9]int{}, [9][9]int{}, [9][9]int{}
	emptys := make([][2]int, 0, 81)

	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if board[i][j] == '.' {
				emptys = append(emptys, [2]int{i, j})
				continue
			}

			v := int32(board[i][j]) - '1'

			rowsCnt[i][v]++
			colsCnt[j][v]++
			boxesCnt[i/3*3+j/3][v]++
		}
	}

	dfs(board, rowsCnt, colsCnt, boxesCnt, emptys, 0)
}

func dfs(board [][]byte, rowsCnt, colsCnt, boxesCnt [9][9]int, emptys [][2]int, idx int) bool {
	if idx == len(emptys) {
		return true
	}

	i, j := emptys[idx][0], emptys[idx][1]
	for v := 0; v < 9; v++ {
		if rowsCnt[i][v] == 0 && colsCnt[j][v] == 0 && boxesCnt[i/3*3+j/3][v] == 0 {
			rowsCnt[i][v] = 1
			colsCnt[j][v] = 1
			boxesCnt[i/3*3+j/3][v] = 1
			board[i][j] = '1' + byte(v)
			if dfs(board, rowsCnt, colsCnt, boxesCnt, emptys, idx+1) {
				return true
			}
			rowsCnt[i][v] = 0
			colsCnt[j][v] = 0
			boxesCnt[i/3*3+j/3][v] = 0
			board[i][j] = '.'
		}
	}

	return false
}
```

#### [679. 24 点游戏](https://leetcode.cn/problems/24-game/)

注意这里看题时容易思考括号的位置，其实括号就是选 card 的不同顺序而已，剩下的就是枚举回溯了

```go
func judgePoint24(cards []int) bool {
	var dfs func([]float64) bool

	EPSILON := 1e-6

	ops := []string{"+", "-", "*", "/"}

	dfs = func(nums []float64) bool {
		n := len(nums)

		if n == 0 {
			return false
		}

		if n == 1 {
			return abs(nums[0]-float64(24)) < EPSILON
		}

		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if i == j {
					continue
				}

				temp := make([]float64, 0, n)

				for k := 0; k < n; k++ {
					if k != i && k != j {
						temp = append(temp, nums[k])
					}
				}

				ni, nj := nums[i], nums[j]
				for _, op := range ops {
					if (op == "+" || op == "*") && i < j {
						continue
					}
					switch op {
					case "+":
						temp = append(temp, ni+nj)
					case "-":
						temp = append(temp, ni-nj)
					case "*":
						temp = append(temp, ni*nj)
					case "/":
						if abs(nj) < EPSILON {
							continue
						}
						temp = append(temp, ni/nj)
					}

					if dfs(temp) {
						return true
					}

					temp = temp[:len(temp)-1]
				}
			}
		}

		return false
	}

	nums := make([]float64, 0, len(cards))

	for _, card := range cards {
		nums = append(nums, float64(card))
	}

	return dfs(nums)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
```

### 搜索 (BFS/DFS)

#### [200. 岛屿数量](https://leetcode.cn/problems/number-of-islands/)

几乎完全一样的题
- [695. 岛屿的最大面积](https://leetcode.cn/problems/max-area-of-island/)
- [130. 被围绕的区域](https://leetcode.cn/problems/surrounded-regions/)

```go
func numIslands(grid [][]byte) int {
    n, m := len(grid), len(grid[0])

    var dfs func(int, int)

    dx, dy := []int{1, 0, -1, 0}, []int{0, 1, 0, -1}

    dfs = func(i, j int) {
        if i < 0 || i >= n || j < 0 || j >= m || grid[i][j] != '1' {
            return 
        }

        grid[i][j] = '0'

        for k := 0; k < 4; k++ {
            dfs(dx[k] + i, dy[k] + j)
        }
    }


    ans := 0
    for i := 0; i < n; i++ {
        for j := 0; j < m; j++ {
            if grid[i][j] == '1' {
                dfs(i, j)
                ans++
            }
        }
    }

    return ans
}
```

#### [994. 腐烂的橘子](https://leetcode.cn/problems/rotting-oranges/)
```go
var dx, dy = []int{1, 0, -1, 0}, []int{0, 1, 0, -1}

func orangesRotting(grid [][]int) (ans int) {
	n, m := len(grid), len(grid[0])
	q := list.New()

	fresh := 0

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if grid[i][j] == 1 {
				fresh++
			} else if grid[i][j] == 2 {
				q.PushBack([]int{i, j})
			}
		}
	}

	for q.Len() > 0 && fresh > 0 {
		size := q.Len()

		for size > 0 {
			front := q.Front()
			q.Remove(front)

			i, j := front.Value.([]int)[0], front.Value.([]int)[1]
			for k := 0; k < 4; k++ {
				x, y := i+dx[k], j+dy[k]

				if x >= 0 && x < n && y >= 0 && y < m && grid[x][y] == 1 {
					grid[x][y] = 2
					fresh--
					q.PushBack([]int{x, y})
				}
			}

			size--
		}

		ans++
	}

	if fresh > 0 {
		return -1
	}

	return ans
}
```
### 链表
#### [206. 反转链表](https://leetcode.cn/problems/reverse-linked-list/)

```cpp
/**
 * Definition for singly-linked list.
 * struct ListNode {
 *     int val;
 *     ListNode *next;
 *     ListNode() : val(0), next(nullptr) {}
 *     ListNode(int x) : val(x), next(nullptr) {}
 *     ListNode(int x, ListNode *next) : val(x), next(next) {}
 * };
 */
class Solution {
public:
    ListNode* reverseList(ListNode* head) {
        if (head == nullptr || head->next == nullptr) return head;
        
        auto nhead = reverseList(head->next);

        // reverse 2 node
        auto cur = head->next;
        cur->next = head;
        head->next = nullptr;

        return nhead;
    }
};
```

#### [92. 反转链表 II](https://leetcode.cn/problems/reverse-linked-list-ii/)

头插法
```go
func reverseBetween(head *ListNode, left int, right int) *ListNode {
	dummy := &ListNode{Next: head}

	pre := dummy

	for i := 0; i < left-1; i++ {
		pre = pre.Next
	}

	cur := pre.Next

	for i := 0; i < right-left; i++ {
		tmp := cur.Next
		cur.Next = tmp.Next
		tmp.Next = pre.Next
		pre.Next = tmp
	}

	return dummy.Next
}
```
#### [25. K 个一组翻转链表](https://leetcode.cn/problems/reverse-nodes-in-k-group/)

##### 递归法

这里要注意 ed 只能取到开区间，如果取到闭区间值
```go
nhead := reverse(head, ed.Next)
head.Next = reverseKGroup(ed.Next, k)
```
这两行就会有问题，因为 `reverse` 中改变了 ed，导致 `ed.Next` 不是原来的值了

```go
func reverseKGroup(head *ListNode, k int) (ans *ListNode) {
	ed := head

	for i := 0; i < k; i++ {
		if ed == nil {
			return head
		}
		ed = ed.Next
	}

	nhead := reverse(head, ed)

	head.Next = reverseKGroup(ed, k)

	return nhead
}

func reverse(st, ed *ListNode) *ListNode {
	pre, cur := (*ListNode)(nil), st

	for cur != ed {
		next := cur.Next
		cur.Next = pre
		pre = cur
		cur = next
	}

	return pre
}
```
##### O(1) 空间

这里写法有几种，这种我感觉从逻辑上来说要简单些
```go
func reverseKGroup(head *ListNode, k int) (ans *ListNode) {
	dummy := &ListNode{}
	cur, pre := head, dummy

	for cur != nil {
		ed := cur
		for i := 0; i < k; i++ {
			if ed == nil {
				for cur != nil {
					pre.Next = cur
					cur = cur.Next
					pre = pre.Next
				}

				return dummy.Next
			}
			ed = ed.Next
		}

		st := cur

		for cur != ed {
			next := cur.Next
			cur.Next = pre.Next
			pre.Next = cur
			cur = next
		}

		pre = st
		cur = ed
	}

	return dummy.Next
}
```

下面这个为什么简单？
因为在反转的过程中，它没有断裂和下个节点的连接，所以不需要特殊处理不满足 k 个时维护下一层节点
但是这种方法的反转太抽象不好理解
```go
func reverseKGroup(head *ListNode, k int) *ListNode {
	dummy := &ListNode{Next: head}
	pre, cur := dummy, head

	for cur != nil {
		ed := cur
		for i := 0; i < k-1 && ed != nil; i++ {
			ed = ed.Next
		}

		if ed == nil {
			break
		}

		nextSt := ed.Next

		for i := 0; i < k-1; i++ {
			next := cur.Next
			cur.Next = next.Next
			next.Next = pre.Next
			pre.Next = next
		}

		pre = cur
		cur = nextSt
	}

	return dummy.Next
}
```

#### [148. 排序链表](https://leetcode.cn/problems/sort-list/)

归并排序
```go
func sortList(head *ListNode) *ListNode {
	return mergeSort(head)
}

func mergeSort(head *ListNode) *ListNode {
	if head == nil || head.Next == nil {
		return head
	}

	fast, slow := head.Next, head
	for fast != nil && fast.Next != nil {
		fast = fast.Next.Next
		slow = slow.Next
	}

	right := slow.Next
	slow.Next = nil

	le := mergeSort(head)
	ri := mergeSort(right)

	return mergeTwo(le, ri)
}

func mergeTwo(l1 *ListNode, l2 *ListNode) *ListNode {
	l3 := &ListNode{}
	cur := l3

	for l1 != nil && l2 != nil {
		if l1.Val < l2.Val {
			cur.Next = l1
			l1 = l1.Next
		} else {
			cur.Next = l2
			l2 = l2.Next
		}

		cur = cur.Next
	}

	if l1 != nil {
		cur.Next = l1
	}

	if l2 != nil {
		cur.Next = l2
	}

	return l3.Next
}
```
#### [143. 重排链表](https://leetcode.cn/problems/reorder-list/)

要注意的是，连接 l1/l2 的时候，可以直接用插入 l2 节点的方法，而不用申请新的链表
```go
func reorderList(head *ListNode) {
	fast, slow := head, head

	for fast != nil && fast.Next != nil {
		fast = fast.Next.Next
		slow = slow.Next
	}

	l2 := reverse(slow.Next)
	slow.Next = nil
	l1 := head

	for l1 != nil && l2 != nil {
		l2next := l2.Next
		l1next := l1.Next

		l2.Next = l1.Next
		l1.Next = l2

		l1 = l1next
		l2 = l2next
	}
}

func reverse(head *ListNode) *ListNode {
	pre, cur := (*ListNode)(nil), head

	for cur != nil {
		next := cur.Next
		cur.Next = pre
		pre = cur
		cur = next
	}

	return pre
}
```

#### [82. 删除排序链表中的重复元素 II](https://leetcode.cn/problems/remove-duplicates-from-sorted-list-ii/)
```go
func deleteDuplicates(head *ListNode) *ListNode {
	dummy := &ListNode{}
	pre, cur := dummy, head

	for cur != nil {
		next := cur.Next
		if next != nil && cur.Val == next.Val {
			for next != nil && cur.Val == next.Val {
				next = next.Next
			}
			pre.Next = next
			cur = next
		} else {
			pre.Next = cur
			pre = cur
			cur = next
		}
	}

	return dummy.Next
}
```
### 二分

#### 二分查找
##### [34. 在排序数组中查找元素的第一个和最后一个位置](https://leetcode.cn/problems/find-first-and-last-position-of-element-in-sorted-array/)

理不清楚就画个图
```cpp
class Solution {
public:
    vector<int> searchRange(vector<int>& nums, int target) {
		int le = 0, ri = nums.size() - 1;

		while (le < ri) {
			int mid = le + (ri - le) / 2;
			if (nums[mid] == target) {
				ri = mid;
			} else if (nums[mid] > target) {
				ri = mid - 1;
			} else {
				le = mid + 1;
			}
		}

		if (le < 0 || le == nums.size() || nums[le] != target) return {-1, -1};

        vector<int> res = {le};

		ri = nums.size() - 1;

		while (le < ri) {
			int mid = le + (ri - le + 1) / 2;
			if (nums[mid] == target) {
				le = mid;
			} else if (nums[mid] < target) {
				le = mid + 1;
			} else {
				ri = mid - 1;
			}
		}

		res.push_back(le);

		return res;
    }
};
```

##### [153. 寻找旋转排序数组中的最小值](https://leetcode.cn/problems/find-minimum-in-rotated-sorted-array/)

可以列举出所有情况，然后合并同样处理的分支，就得到了下面的结论
```go
func findMin(nums []int) (ans int) {
	le, ri := 0, len(nums)-1

	for le < ri {
		mid := le + (ri-le)/2

		if nums[mid] < nums[ri] {
			ri = mid
		} else {
			le = mid + 1
		}
	}

	return nums[le]
}
```

##### [33. 搜索旋转排序数组](https://leetcode.cn/problems/search-in-rotated-sorted-array/)
```go
func search(nums []int, target int) int {
	le, ri := 0, len(nums)-1

	for le <= ri {
		mid := le + (ri-le)/2

		if target == nums[mid] {
			return mid
		}

		if nums[mid] <= nums[ri] {
			if nums[mid] < target && nums[ri] >= target {
				le = mid + 1
			} else {
				ri = mid - 1
			}
		} else {
			if nums[mid] > target && nums[le] <= target {
				ri = mid - 1
			} else {
				le = mid + 1
			}
		}
	}

	return -1
}
```

#### 二分答案 （最大最小值，最小最大值）
##### [1011. 在 D 天内送达包裹的能力](https://leetcode.cn/problems/capacity-to-ship-packages-within-d-days/)

```cpp
class Solution {
public:
    int shipWithinDays(vector<int>& weights, int days) {
		int le = 1, ri = accumulate(weights.begin(), weights.end(), 0);

		while (le < ri) {
			int mid = (ri - le) / 2 + le;
			if (check(weights, days, mid)) {
				ri = mid;
			} else {
				le = mid + 1;
			}
		}

		return le;
    }

	int check(const vector<int>& weights, int rest, int p) {
		int cur = 0;

		for (int i = 0; i < weights.size(); i++) {
			if (weights[i] > p) return false;
			if (cur + weights[i] > p) {
				rest--;
				cur = 0;
				if (rest < 0) return false;
			}

			cur += weights[i];
		}

		if (cur) rest--;
		return rest >= 0;
	}
};

```
##### [2226. 每个小孩最多能分到多少糖果](https://leetcode.cn/problems/maximum-candies-allocated-to-k-children/)

二分查找的边界很重要
```cpp
class Solution {
public:
    int maximumCandies(vector<int>& candies, long long k) {
        int le = 0, ri = *max_element(candies.begin(), candies.end());

		while (le < ri) {
			int mid = le + (ri - le + 1) / 2;

			if (check(candies, mid, k)) {
				le = mid;
			} else {
				ri = mid - 1;
			}
		}

		return ri;
    }

	bool check(const vector<int>& candies, int n, long long k) {
		if (n == 0) return true;

 		long long cnt = 0;

		for (auto& ca : candies) {
			cnt += ca / n;
			if (cnt >= k) {
				return true;
			}
		}

		return false;
	}
};
```
#### [153. 寻找旋转排序数组中的最小值](https://leetcode.cn/problems/find-minimum-in-rotated-sorted-array/)

可以列举出所有情况，然后合并同样处理的分支，就得到了下面的结论
```go
func findMin(nums []int) (ans int) {
	le, ri := 0, len(nums)-1

	for le < ri {
		mid := le + (ri-le)/2

		if nums[mid] < nums[ri] {
			ri = mid
		} else {
			le = mid + 1
		}
	}

	return nums[le]
}
```

### 分治

#### [215. 数组中的第K个最大元素](https://leetcode.cn/problems/kth-largest-element-in-an-array/)

类似于「快速排序」的思想，快速选择然后判断下一步搜索的方向
```go
func findKthLargest(nums []int, k int) (ans int) {
	p := nums[rand.Intn(len(nums))]

	large, small, equal := make([]int, 0, len(nums)), make([]int, 0, len(nums)), make([]int, 0, len(nums))

	for _, v := range nums {
		if v > p {
			large = append(large, v)
		} else if v < p {
			small = append(small, v)
		} else {
			equal = append(equal, v)
		}
	}

	if len(large) >= k {
		return findKthLargest(large, k)
	}

	if k > len(large)+len(equal) {
		return findKthLargest(small, k-len(large)-len(equal))
	}

	return p
}
```

快速排序选择 `partition` 方法，不需要额外的空间，这里很重要的是运用 2 分查找的技巧，每次排除一半，如果不使用二分，逻辑会复杂一些
```go
func findKthLargest(nums []int, k int) (ans int) {
	left, right := 0, len(nums)-1
	for {
		pivot := partition(nums, left, right)
		if pivot == len(nums)-k {
			return nums[pivot]
		} else if pivot < len(nums)-k {
			left = pivot + 1
		} else {
			right = pivot - 1
		}
	}
}

func partition(nums []int, st, ed int) int {
	if st == ed {
		return st
	}

	pivot := nums[st]

	left, right := st, ed

	for left < right {
		for left < right && nums[right] >= pivot {
			right--
		}

		for left < right && nums[left] <= pivot {
			left++
		}

		nums[left], nums[right] = nums[right], nums[left]
	}

	nums[st], nums[left] = nums[left], nums[st]

	return left
}
```

#### [53. 最大子数组和](https://leetcode.cn/problems/maximum-subarray/)

```go
func maxSubArray(nums []int) (ans int) {
	ans = maxHelper(nums, 0, len(nums)-1)
	return
}

func maxHelper(nums []int, le, ri int) int {
	if le == ri {
		return nums[le]
	}

	mid := (le + ri) / 2

	les := maxHelper(nums, le, mid)
	ris := maxHelper(nums, mid+1, ri)

	lesum, lecur := int(-1e4), 0

	for i := mid; i >= le; i-- {
		lecur += nums[i]
		lesum = max(lesum, lecur)
	}

	risum, ricur := int(-1e4), 0
	for i := mid + 1; i <= ri; i++ {
		ricur += nums[i]
		risum = max(risum, ricur)
	}

	return max(lesum+risum, les, ris)
}
```

#### [152. 乘积最大子数组](https://leetcode.cn/problems/maximum-product-subarray/)

和「53. 最大子数组和」差不多，分治的思想，但是这里的计算 `cross 序列` 要复杂一些
`lemax, lemin, rimax, rimin := nums[mid], nums[mid], 1, 1` 这里的初始值也要考虑到

```go
func maxProduct(nums []int) (ans int) {
	if len(nums) == 1 {
		return nums[0]
	}

	mid := len(nums) / 2

	lemax := maxProduct(nums[0:mid])
	rimax := maxProduct(nums[mid:])

	midmax := extendMidMaxProduct(nums, mid)

	return max(lemax, rimax, midmax)
}

func extendMidMaxProduct(nums []int, mid int) int {
	lemax, lemin, rimax, rimin := nums[mid], nums[mid], 1, 1

	base := 1

	for i := mid; i >= 0; i-- {
		base *= nums[i]

		lemax = max(base, lemax)
		lemin = min(base, lemin)
	}

	base = 1
	for i := mid + 1; i < len(nums); i++ {
		base *= nums[i]

		rimax = max(base, rimax)
		rimin = min(base, rimin)
	}

	return max(lemax*rimax, lemin*rimin)
}
```
#### [14. 最长公共前缀](https://leetcode.cn/problems/longest-common-prefix/)
```go
func longestCommonPrefix(strs []string) string {
	if len(strs) == 1 {
		return strs[0]
	}

	return helper(strs, 0, len(strs)-1)
}

func helper(strs []string, le, ri int) string {
	if le >= ri {
		return strs[le]
	}

	mid := le + (ri-le)/2

	return findTwo(helper(strs, le, mid), helper(strs, mid+1, ri))
}

func findTwo(a, b string) string {
	i := 0

	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}

	return a[0:i]
}
```

#### [4. 寻找两个正序数组的中位数](https://leetcode.cn/problems/median-of-two-sorted-arrays/)
这道题比较困难
##### `log(m + n)` +  `kth`

将寻找中位数的问题化为找两个数组的第 k 小的数，然后利用数组本身的有序性，比较数组 k / 2 的位置，得到下一个查找的区间
这里复杂度不如二分的方法，但是更通用更好理解一些

```go
func findMedianSortedArrays(nums1 []int, nums2 []int) (ans float64) {
	n, m := len(nums1), len(nums2)

	if (n+m)%2 == 0 {
		return (findK(nums1, nums2, (n+m)/2) + findK(nums1, nums2, (n+m)/2+1)) / 2
	}

	return findK(nums1, nums2, (n+m)/2+1)
}

func findK(nums1 []int, nums2 []int, k int) (ans float64) {
	if len(nums1) == 0 {
		return float64(nums2[k-1])
	}

	if len(nums2) == 0 {
		return float64(nums1[k-1])
	}

	if k == 1 {
		return float64(min(nums1[0], nums2[0]))
	}

	l := k / 2

	l1 := l
	if l > len(nums1) {
		l1 = len(nums1)
	}

	l2 := l
	if l > len(nums2) {
		l2 = len(nums2)
	}

	if nums1[l1-1] < nums2[l2-1] {
		return findK(nums1[l1:], nums2, k-l1)
	}

	return findK(nums1, nums2[l2:], k-l2)
}
```

##### `log(min(m + n))`
```go
func findMedianSortedArrays(nums1 []int, nums2 []int) (ans float64) {
	// 假设 nums1 中存在下标 i, nums2 中存在下标 j
	// 使得 nums1[i-1] < nums[j] && nums[j-1] < nums[i]
	n1, n2 := len(nums1), len(nums2)

	if n1 > n2 {
		return findMedianSortedArrays(nums2, nums1)
	}

	le, ri := 0, n1

	for le <= ri {
		i := le + (ri-le)/2
		j := (n1+n2+1)/2 - i

		if i > 0 && nums1[i-1] > nums2[j] {
			ri = i - 1
		} else if i < n1 && nums2[j-1] > nums1[i] {
			le = i + 1
		} else {
			maxle := int(-1e9)

			if i == 0 {
				maxle = nums2[j-1]
			} else if j == 0 {
				maxle = nums1[i-1]
			} else {
				maxle = max(nums1[i-1], nums2[j-1])
			}

			if (n1+n2)%2 == 1 {
				return float64(maxle)
			}

			minri := int(1e9)

			if i == n1 {
				minri = nums2[j]
			} else if j == n2 {
				minri = nums1[i]
			} else {
				minri = min(nums1[i], nums2[j])
			}

			return float64(maxle+minri) / 2
		}
	}

	return
}
```

### 前缀和

#### [面试题 17.05. 字母与数字](https://leetcode.cn/problems/find-longest-subarray-lcci/)
```cpp
class Solution {
public:
    vector<string> findLongestSubarray(vector<string>& array) {
        int n = array.size(); 
        vector<int> psum(n + 1, 0);

        for (int i = 1; i <= n; i++) {
            psum[i] = psum[i - 1] + (isalpha(array[i - 1][0]) ? 1 : -1);
        }

        unordered_map<int, int> map;

        int maxl = 0, st = 0;

        for (int i = 0; i <= n; i++) {
            if (!map.count(psum[i])) {
                map[psum[i]] = i;
                continue;
            }

            auto pi = map[psum[i]];
            if (i - pi > maxl) {
                maxl = i - pi;
                st = pi;
            }
        }

        if (maxl == 0) {
            return {};
        }

        return vector<string> (array.begin() + st, array.begin() + st + maxl);
    }
};
```

#### [560. 和为 K 的子数组](https://leetcode.cn/problems/subarray-sum-equals-k/)
```go
func subarraySum(nums []int, k int) (ans int) {
	pres := make([]int, len(nums)+1)

	for i, v := range nums {
		pres[i+1] = pres[i] + v
	}

	for i := 0; i < len(nums)+1; i++ {
		for j := i + 1; j < len(nums)+1; j++ {
			if pres[j]-pres[i] == k {
				ans++
			}
		}
	}

	return ans
}
```

一次遍历
```go
func subarraySum(nums []int, k int) (ans int) {
	cntMap := make(map[int]int, len(nums))
	cntMap[0] = 1

	sum := 0

	for _, v := range nums {
		sum += v

		if cnt, ok := cntMap[sum-k]; ok {
			ans += cnt
		}

		cntMap[sum]++
	}

	return ans
}
```
### 其它

#### 排序

##### 快速排序
https://en.wikipedia.org/wiki/Quicksort

这里非常重要的是
1. 将 `nums[p]` 交换到 `nums 头部`，这样保证了，其它数据都是在其单侧，即「右侧」，交换时不会破坏特性
2. 要首先从右往左找小于 `nums[p]` 的值
	- 因为我们第一次交换到头部，引入了一个未知的变量值，这个值被交换到了 `p` 的位置，从左往右，如果这个变量本来就小于等于 `nums[p]`，那么不会导致问题，如果大于 `nums[p]`，并且，从右往左没有找到合适的值，就会得到错误的结果，又交换回去了
	- 例如 `[3, 1, 2, 4, 5]`，选择 `3` 作为划分值，交换到头部后 `[2, 1, 3, 4, 5]`，然后推导下就知道问题了
3. 循环里比较值的时候，使用 `<= or >=`是为什么

```go
func quickSort(nums []int, st, ed int) {
	if st >= ed {
		return
	}

	p := rand.Intn(ed-st+1) + st
	le, ri := st, ed

	nums[le], nums[p] = nums[p], nums[le]
	p = le

	for le < ri {
		for le < ri && nums[ri] >= nums[p] {
			ri--
		}

		for le < ri && nums[le] <= nums[p] {
			le++
		}

		nums[le], nums[ri] = nums[ri], nums[le]
	}

	nums[le], nums[p] = nums[p], nums[le]

	quickSort(nums, st, le)
	quickSort(nums, le+1, ed)
}
```

沙雕写法，但是我觉得很好，具有完整无误的正确性，其它太难看懂了
```go
func quickSort(nums []int, st, ed int) {
	if st >= ed {
		return
	}

	p := rand.Intn(ed-st+1) + st

	small, large, equal := make([]int, 0), make([]int, 0), make([]int, 0)

	for i := st; i <= ed; i++ {
		if nums[i] < nums[p] {
			small = append(small, nums[i])
		} else if nums[i] > nums[p] {
			large = append(large, nums[i])
		} else {
			equal = append(equal, nums[i])
		}
	}

	copy(nums[st:ed+1], append(append(small, equal...), large...))

	quickSort(nums, st, st+len(small)-1)
	quickSort(nums, st+len(small)+len(equal), ed)
}
```

##### 归并
```cpp
class Solution {
public:
    vector<int> sortArray(vector<int>& nums) {
        return mergesort(nums, 0, nums.size() - 1);
    }

    vector<int> mergesort(vector<int>& nums, int le, int ri) {
        if (le == ri) return {nums[le]};

        int mid = le + (ri - le) / 2;

        auto learr = mergesort(nums, le, mid);
        auto riarr = mergesort(nums, mid + 1, ri);

        return merge(learr, riarr);
    }

    vector<int> merge(vector<int> nums1, vector<int> nums2) {
        vector<int> nums(nums1.size() + nums2.size(), 0);

        int c1 = 0, c2 = 0, c = 0;

        while (c1 < nums1.size() && c2 < nums2.size()) {
            if (nums1[c1] <= nums2[c2]) {
                nums[c++] = nums1[c1++];
            } else {
                nums[c++] = nums2[c2++];
            }
        }

        while (c1 < nums1.size()) nums[c++] = nums1[c1++];
        while (c2 < nums2.size()) nums[c++] = nums2[c2++];

        return nums;
    }
};

```

#### 接雨水

##### [42. 接雨水](https://leetcode.cn/problems/trapping-rain-water/)

###### 最大值数组法
```go
func trap(height []int) (ans int) {
	n := len(height)
	leMaxArr, riMaxArr := make([]int, n), make([]int, n)

	maxHeight := 0
	for i := 0; i < n; i++ {
		maxHeight = max(maxHeight, height[i])
		leMaxArr[i] = maxHeight
	}

	maxHeight = 0
	for i := n - 1; i >= 0; i-- {
		maxHeight = max(maxHeight, height[i])
		riMaxArr[i] = maxHeight
	}

	for i := 0; i < n; i++ {
		ans += min(leMaxArr[i], riMaxArr[i]) - height[i]
	}

	return
}
```
###### 双指针法

每个格子能蓄的最大水量为 `min(left max, right max) - height[i]`

如果 `height[le] < height[ri]`，则可以知道 rightmax 肯定 > leftmax，可以确定其当前的 `left max` 和 `right max` 中的`较小`值

```cpp
class Solution {
public:
    int trap(vector<int>& height) {
		if (height.size() <= 2) return 0;

		int n = height.size(), ans = 0;
		int le = 0, ri = n - 1;
		int lemax = height[le], rimax = height[ri];

		while (le < ri) {
			if (height[le] <= height[ri]) {
				ans += lemax - height[le];
				le++;
				lemax = max(lemax, height[le]);
			} else {
				ans += rimax - height[ri];
				ri--;
				rimax = max(rimax, height[ri]);
			}
		}

		return ans;
    }
};
```

##### [407. 接雨水 II](https://leetcode.cn/problems/trapping-rain-water-ii/)

通过优先队列找到当前「最低」的格子，然后 bfs search
```go
func trapRainWater(heightMap [][]int) (ans int) {
	pq := PriorityQueue{}

	n, m := len(heightMap), len(heightMap[0])

	vis := make([][]bool, n)
	for i := 0; i < n; i++ {
		vis[i] = make([]bool, m)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if i == 0 || i == n-1 || j == 0 || j == m-1 {
				vis[i][j] = true
				pq.Push(Cell{i, j, heightMap[i][j]})
			}
		}
	}

	for !pq.IsEmpty() {
		cell := pq.Pop()

		for _, dir := range [][]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := cell.col+dir[0], cell.row+dir[1]

			if nx < 0 || nx >= n || ny < 0 || ny >= m || vis[nx][ny] {
				continue
			}

			vis[nx][ny] = true
			ans += max(0, cell.height-heightMap[nx][ny])
			pq.Push(Cell{nx, ny, max(cell.height, heightMap[nx][ny])})
		}
	}

	return ans
}

type Cell struct {
	col, row, height int
}

type PriorityQueue []Cell

func (pq *PriorityQueue) IsEmpty() bool { return len(*pq) == 0 }

func (pq *PriorityQueue) Pop() Cell {
	minIdx, minHeight := 0, (*pq)[0].height

	for i := 1; i < len(*pq); i++ {
		if (*pq)[i].height < minHeight {
			minIdx = i
			minHeight = (*pq)[i].height
		}
	}

	minCell := (*pq)[minIdx]
	*pq = append((*pq)[:minIdx], (*pq)[minIdx+1:]...)

	return minCell
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(Cell))
}
```


#### 计数
##### [Non-Divisible Subset](https://www.hackerrank.com/challenges/non-divisible-subset/problem?isFullScreen=true)

挺巧妙的
```cpp
int nonDivisibleSubset(int k, vector<int> s) {
    vector<int> cnt(k, 0);
    
    for (auto& e : s) {
        cnt[e % k]++;
    }
    
    int ans = 0;
    
    ans += cnt[0] > 0 ? 1 : 0;
    ans += k % 2 == 0 && cnt[k % 2] > 0 ? 1 : 0;
    
    for (int i = 1; i < (k + 1) / 2; i++) {
        ans += max(cnt[i], cnt[k - i]);
    }
    
    return ans;
}
```
#### 线段树

#### 并查集

##### [128. 最长连续序列](https://leetcode.cn/problems/longest-consecutive-sequence/)
```go
type UF struct {
	fa   map[int]int
	size map[int]int
}

func (u *UF) Find(x int) int {
	if u.fa[x] != x {
		u.fa[x] = u.Find(u.fa[x])
	}

	return u.fa[x]
}

func (u *UF) Union(x, y int) {
	fx, fy := u.Find(x), u.Find(y)

	if fx == fy {
		return
	}

	if u.size[fx] > u.size[fy] {
		fy, fx = fx, fy
	}

	u.size[fy] += u.size[fx]
	u.fa[fx] = fy
}

func longestConsecutive(nums []int) (ans int) {
	uf := UF{
		fa:   make(map[int]int),
		size: make(map[int]int),
	}

	for _, num := range nums {
		uf.fa[num] = num
		uf.size[num] = 1
	}

	for _, num := range nums {
		if _, ok := uf.fa[num+1]; ok {
			uf.Union(num, num+1)
		}
	}

	for _, v := range uf.size {
		ans = max(v, ans)
	}

	return ans
}
```
#### 最短路

##### [743. 网络延迟时间](https://leetcode.cn/problems/network-delay-time/)

djikstra 算法
```go
func networkDelayTime(times [][]int, n int, k int) (ans int) {
	graph := make([][][2]int, n+1)
	distances := make([]int, n+1)

	for i := 1; i <= n; i++ {
		distances[i] = 1e9
	}

	distances[k] = 0

	for _, edge := range times {
		u, v, w := edge[0], edge[1], edge[2]
		graph[u] = append(graph[u], [2]int{v, w})
	}

	pq := PriorityQueue{}
	pq.Push(Cell{distance: 0, node: k})

	for !pq.IsEmpty() {
		cell := pq.Pop()
		if distances[cell.node] < cell.distance {
			continue
		}

		for _, item := range graph[cell.node] {
			v, w := item[0], item[1]

			newDistance := cell.distance + w
			if newDistance < distances[v] {
				distances[v] = newDistance
				pq.Push(Cell{distance: newDistance, node: v})
			}
		}
	}

	for i := 1; i <= n; i++ {
		if distances[i] == 1e9 {
			return -1
		}
		ans = max(ans, distances[i])
	}

	return
}

type Cell struct {
	distance, node int
}

type PriorityQueue []Cell

func (pq *PriorityQueue) IsEmpty() bool { return len(*pq) == 0 }

func (pq *PriorityQueue) Pop() Cell {
	minIdx, minDistance := 0, (*pq)[0].distance

	for i := 1; i < len(*pq); i++ {
		if (*pq)[i].distance < minDistance {
			minIdx = i
			minDistance = (*pq)[i].distance
		}
	}

	minCell := (*pq)[minIdx]
	*pq = append((*pq)[:minIdx], (*pq)[minIdx+1:]...)

	return minCell
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(Cell))
}
```

#### 拓扑排序
##### [207. 课程表](https://leetcode.cn/problems/course-schedule/)
```go
func canFinish(numCourses int, prerequisites [][]int) bool {
	graph := make2DArray(numCourses, 0)
	indgrees := make([]int, numCourses)

	for _, p := range prerequisites {
		a, b := p[0], p[1]
		graph[b] = append(graph[b], a)
		indgrees[a]++
	}

	queue := make([]int, 0)

	for i := 0; i < numCourses; i++ {
		if indgrees[i] == 0 {
			queue = append(queue, i)
		}
	}

	if len(queue) == 0 {
		return false
	}

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for _, v := range graph[u] {
			indgrees[v]--
			if indgrees[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	for i := 0; i < numCourses; i++ {
		if indgrees[i] != 0 {
			return false
		}
	}

	return true
}

func make2DArray(n, m int) [][]int {
	res := make([][]int, n)
	for i := 0; i < n; i++ {
		res[i] = make([]int, m)
	}
	return res
}
```

##### [210. 课程表 II](https://leetcode.cn/problems/course-schedule-ii/)
```cpp
class Solution {
public:
    vector<int> findOrder(int n, vector<vector<int>>& pres) {
        vector<vector<int>> grid(n);
        vector<int> inds(n, 0);

        for (auto& e : pres) {
            auto a = e[0], b = e[1];
            grid[b].push_back(a);
            inds[a]++;
        }

        queue<int> que;
        for (int i = 0; i < inds.size(); i++) {
            if (inds[i] == 0) {
                que.push(i);
            }
        }

        if (que.size() == 0) {
            return {};
        }

        vector<int> ret;
        while (!que.empty()) {
            int cur = que.front(); que.pop();
            ret.push_back(cur);
            for (auto& node : grid[cur]) {
                inds[node]--;
                if (inds[node] == 0) {
                    que.push(node);
                }
            }
        }

        return ret.size() == n ? ret : vector<int>();
    }
};
```

#### [208. 实现 Trie (前缀树)](https://leetcode.cn/problems/implement-trie-prefix-tree/)
```go
type Trie struct {
	childs map[rune]*Trie
	word   string
}

func Constructor() Trie {
	return Trie{
		childs: make(map[rune]*Trie),
		word:   "",
	}
}

func NewTrie() *Trie {
	return &Trie{
		childs: make(map[rune]*Trie),
		word:   "",
	}
}

func (t *Trie) Insert(word string) {
	cur := t

	for _, c := range word {
		_, ok := cur.childs[c]
		if !ok {
			cur.childs[c] = NewTrie()
		}

		cur = cur.childs[c]
	}

	cur.word = word
}

func (t *Trie) Search(word string) bool {
	cur := t

	for _, c := range word {
		v, ok := cur.childs[c]
		if !ok {
			return false
		}
		cur = v
	}

	return cur.word == word
}

func (t *Trie) StartsWith(prefix string) bool {
	cur := t

	for _, c := range prefix {
		v, ok := cur.childs[c]
		if !ok {
			return false
		}
		cur = v
	}

	return true
}

```

#### MATH

##### rand(x) implement rand(y)

需要常见均匀的分布
`x * rand (x) + rand (x)` 就是均匀的

rand5 实现 rand7
```go
func rand7() int {
	for {
		n := 5*rand5() + rand5()

		if n < 21 {
			return n % 7
		}
	}
}
```

rand7 实现 rand10 (rand7 范围是 `[1, 7]`)
```cpp
class Solution {
public:
    int rand10() {
        while (1) {
            int n = (rand7() - 1) * 7 + rand7();

            if (n > 10) continue;

            return n;
        }

        return -1;
    }
};
```

#### [146. LRU 缓存](https://leetcode.cn/problems/lru-cache/)

淘汰算法：名称是最近最少使用（get / put 都会将 item 移到最近使用），也就是淘汰最久没有使用的

```go
type LRUCache struct {
	list *list.List
	m    map[int]*list.Element
	cap  int
}

type Element struct {
	Key   int
	Value int
}

func Constructor(capacity int) LRUCache {
	return LRUCache{
		list: list.New(),
		m:    make(map[int]*list.Element),
		cap:  capacity,
	}
}

func (l *LRUCache) Get(key int) int {
	v, ok := l.m[key]
	if !ok {
		return -1
	}

	l.list.Remove(v)
	l.list.PushFront(v.Value)
	l.m[key] = l.list.Front()

	return v.Value.(Element).Value
}

func (l *LRUCache) Put(key int, value int) {
	ele := Element{key, value}

	v, ok := l.m[key]
	if !ok {
		if l.list.Len() == l.cap {
			back := l.list.Back()
			l.list.Remove(back)
			delete(l.m, back.Value.(Element).Key)
		}
	} else {
		l.list.Remove(v)
	}

	l.list.PushFront(ele)
	l.m[key] = l.list.Front()
}
```

#### 荷兰国旗

##### [75. 颜色分类](https://leetcode.cn/problems/sort-colors/)

画一遍图应该能清楚了，要不然比较难理解 `le` 和 `i` 的正确性是如何维护的

这里还需要注意 `nums[i] == 2` 的情况，不能把 `i++`，因为从后边交换过来的数并不能确定大小，可能是 `0` 或者 `1`

而与前边交换得到的数大小是能确定的，肯定是 `1` 或者 `0`，所以不用移动位置，直接向后移动就行了，而且这里因为可能会拿到 `0`，所以需要自己移动，而不是依赖于下次判断 `i++`

```go
func sortColors(nums []int) {
	le, ri, i := 0, len(nums)-1, 0

	for i <= ri {
		if nums[i] == 0 {
			nums[le], nums[i] = nums[i], nums[le]
			le++
            i++
		} else if nums[i] == 2 {
			nums[ri], nums[i] = nums[i], nums[ri]
			ri--
		} else {
			i++
		}

	}
}
```
#### 洗牌算法

洗牌算法 knuth-shuffle，生成的 `n!` 种全排列，每种排列出现的概率是相等的，即每个元素放在每个位置的概率都是随机的

主要思想，将牌组分为已排序和未排序两部分，每次从未排序中选择一个放在待排序的位置

原理推导，对于某个元素，假设选择到了第 i 轮，他在第一轮没被选中的概率是 `(n-1)/n`，第二轮没被选中的概率是 `(n-2)/(n-1)` 一直到 i 轮 `(n-i-1)/(n-i)`，乘起来后得到 `(n-i-1)/n`，当前被选中的概率是 `1/(n-i-1)`，乘后算下来元素被选中的概率是 `1/n`

#### 蓄水池抽样
给出一个数据流，我们需要在此数据流中随机选取 k 个数。由于这个数据流的长度很大，因此需要边遍历边处理，而不能将其一次性全部加载到内存。

请写出一个随机选择算法，使得数据流中所有数据被**等概率**选中。

其基本思路是：
- 构建一个大小为 k 的数组，将数据流的前 k 个元素放入数组中。
- 对数据流的前 k 个数**先**不进行任何处理。
- 从数据流的第 k + 1 个数开始，在 `[1, i]` 之间选一个数 rand，其中 i 表示当前是第几个数。
- 如果 rand 大于等于 k 什么都不做
- 如果 rand 小于 k，将 rand 和 i 交换，也就是说选择当前的数代替已经被选中的数。
- 最终返回幸存的元素即可

这种算法的核心在于先以某一种概率选取数，并在后续过程以另一种概率换掉之前已经被选中的数。因此实际上每个数被最终选中的概率都是**被选中的概率 * 不被替换的概率**。

假设抽样集合大小为 50，那么前 50 个数据会直接进入集合（不足 50 的话不需要抽样）。从第 51 个数据开始，获得一个 1 ～51 的随机数 n，然后把第 51 个数据与第 n 个数据替换（如果 n 不在抽样集合中，则跳过）。用该方法可以分批将数据进行抽样并保持概率相等。

证明：
只有 50 个数据时，每个数据被抽中概率的概率为 100%，概率相等。

到第 51 个数据，该数据进入抽样集合概率为 50/51，集合内数据被抽出去的概率为 1/51，最终集合内每个数据存活的概率为 50/51。

到第 52 个数据，该数据进入抽样集合概率为 50/52，集合内的数据被抽出去的概率为 1/52，第二轮存活概率为第一轮存活概率 50/51 * （1 - 1/52） = 50/52，与进入集合概率相等。