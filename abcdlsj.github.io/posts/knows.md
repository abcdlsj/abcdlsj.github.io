---
title: "Knows"
date: 2024-03-31T22:30:59+08:00
tags:
  - Interview
hide: true
wip: false
tocPosition: left-sidebar
---

## Golang
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

### GMP

https://morsmachine.dk/go-scheduler
https://go.cyub.vip/gmp/gmp-model/

runtime
https://www.purewhite.io/2019/11/28/runtime-hacking-translate/

### 内存管理
https://povilasv.me/go-memory-management/#
https://povilasv.me/go-memory-management-part-2/
https://povilasv.me/go-memory-management-part-3/

内存分配
https://luozhiyun.com/archives/434

### 垃圾回收（GC）

https://www.luozhiyun.com/archives/475
https://www.ardanlabs.com/blog/2018/12/garbage-collection-in-go-part1-semantics.html
https://golang.design/go-questions/memgc/principal/
https://tonybai.com/2023/06/13/understand-go-gc-overhead-behind-the-convenience/


![go gc|400](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/golang_gc_obj_pic.2krrubepqx.webp)

#### 三色抽象
- 白色对象 — 潜在的垃圾，其内存可能会被垃圾收集器回收；
- 黑色对象 — 活跃的对象，包括不存在任何引用外部指针的对象以及从根对象可达的对象；
- 灰色对象 — 活跃的对象，因为存在指向白色对象的外部指针，垃圾收集器会扫描这些对象的子对象；

在垃圾收集器开始工作时，程序中不存在任何的黑色对象，垃圾收集的根对象会被标记成灰色，垃圾收集器只会从灰色对象集合中取出对象开始扫描，当灰色集合中不存在任何对象时，标记阶段就会结束。

#### 屏障技术
<https://liqingqiya.github.io/golang/gc/%E5%9E%83%E5%9C%BE%E5%9B%9E%E6%94%B6/%E5%86%99%E5%B1%8F%E9%9A%9C/2020/07/24/gc5.html>

想要在并发或者增量的标记算法中保证正确性，我们需要达成以下两种三色不变性（Tri-color invariant）中的一种：

- 强三色不变性 — 黑色对象不会指向白色对象，只会指向灰色对象或者黑色对象；
- 弱三色不变性 — 黑色对象指向的白色对象必须包含一条从灰色对象经由多个白色对象的可达路径7；

垃圾收集中的屏障技术更像是一个钩子方法，它是在用户程序读取对象、创建新对象以及更新对象指针时执行的一段代码，根据操作类型的不同，我们可以将它们分成读屏障（Read barrier）和写屏障（Write barrier）两种，因为读屏障需要在读操作中加入代码片段，对用户程序的性能影响很大，所以编程语言往往都会采用写屏障保证三色不变性。

- Dijkstra 插入写屏障
```go
writePointer(slot, ptr):
    shade(ptr)
    *slot = ptr
```
每当执行类似 `*slot = ptr` 的表达式时，我们会执行上述写屏障通过 shade 函数尝试改变指针的颜色。如果 ptr 指针是白色的，那么该函数会将该对象设置成灰色，其他情况则保持不变。
Dijkstra 插入屏障的好处在于可以立刻开始并发标记。但存在两个缺点：
1. 由于 Dijkstra 插入屏障的「保守」，在一次回收过程中可能会残留一部分对象没有回收成功，只有在下一个回收过程中才会被回收；
2. 在标记阶段中，每次进行指针赋值操作时，都需要引入写屏障，这无疑会增加大量性能开销；为了避免造成性能问题，Go 团队在最终实现时，没有为所有栈上的指针写操作，启用写屏障，而是当发生栈上的写操作时，将栈标记为灰色，但此举产生了灰色赋值器，将会需要标记终止阶段 STW 时对这些栈进行重新扫描。

- Yuasa 删除写屏障
```go
writePointer(slot, ptr)
    shade(*slot)
    *slot = ptr
```
会在老对象的引用被删除时，将白色的老对象涂成灰色，这样删除写屏障就可以保证弱三色不变性，老对象引用的下游对象一定可以被灰色对象引用。
Yuasa 屏障在标记开始时需要 STW 来扫描或快照堆栈，因为删除屏障同样不被应用与对栈指针的写入操作上，故初始栈指针指向的堆节点不能被 `*slot` 保护到，需要被提前保护
Yuasa 删除屏障的优势则在于不需要标记结束阶段的重新扫描，结束时候能够准确的回收所有需要回收的白色对象。缺陷是 Yuasa 删除屏障会拦截写操作，进而导致波面的退后，产生「冗余」的扫描。
删除写屏障（基于起始快照的写屏障）有一个前提条件，就是起始的时候，把整个根部扫描一遍，让所有的可达对象全都在灰色保护下（根黑，下一级在堆上的全灰），之后利用删除写屏障捕捉内存写操作，确保弱三色不变式不被破坏，就可以保证垃圾回收的正确性。

- 混合写屏障
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

#### 垃圾收集器阶段

|阶段|说明|赋值器状态|
|:-:|:--|:-:|
|清扫终止|为下一个阶段的并发标记做准备工作，启动写屏障|STW|
|标记|与赋值器并发执行，写屏障处于开启状态|并发|
|标记终止|保证一个周期内标记任务完成，停止写屏障|STW|
|内存清扫|将需要回收的内存归还到堆中，写屏障处于关闭状态|并发|
|内存归还|将过多的内存归还给操作系统，写屏障处于关闭状态|并发|

1. 清理终止阶段；
	1. 暂停程序，所有的处理器在这时会进入安全点（Safe point）；
	2. 如果当前垃圾收集循环是强制触发的，我们还需要处理还未被清理的内存管理单元；
2. 标记阶段；
	1. 将状态切换至 `_GCmark`、开启写屏障、用户程序协助（Mutator Assists）并将根对象入队；
	2. 恢复执行程序，标记进程和用于协助的用户程序会开始并发标记内存中的对象，写屏障会将被覆盖的指针和新指针都标记成灰色，而所有新创建的对象都会被直接标记成黑色；
	3. 开始扫描根对象，包括所有 Goroutine 的栈、全局对象以及不在堆中的运行时数据结构，扫描 Goroutine 栈期间会暂停当前处理器；
	4. 依次处理灰色队列中的对象，将对象标记成黑色并将它们指向的对象标记成灰色；
	5. 使用分布式的终止算法检查剩余的工作，发现标记阶段完成后进入标记终止阶段；
3. 标记终止阶段；
	1. 暂停程序、将状态切换至 `_GCmarktermination` 并关闭辅助标记的用户程序；
	2. 清理处理器上的线程缓存；
4. 清理阶段；
	1. 将状态切换至 `_GCoff` 开始清理阶段，初始化清理状态并关闭写屏障；
	2. 恢复用户程序，所有新创建的对象会标记成白色；
	3. 后台并发清理所有的内存管理单元，当 Goroutine 申请新的内存管理单元时就会触发清理；

##### GC 触发机制
1. 主动触发，通过调用 runtime.GC 来触发 GC，此调用阻塞式地等待当前 GC 运行完毕。

2. 被动触发，分为两种方式：
    - 使用系统监控，当超过两分钟没有产生任何 GC 时，强制触发 GC。
    - 使用步调（Pacing）算法，其核心思想是控制内存增长的比例。

##### mark assist
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

### 标准库

#### slice 和数组
slice 的底层数据是数组，slice 是对数组的封装，它描述一个数组的片段。两者都可以通过下标来访问单个元素。

数组是定长的，长度定义好之后，不能再更改。在 Go 中，数组是不常见的，因为其长度是类型的一部分，限制了它的表达能力，比如 `[3]int` 和 `[4]int` 就是不同的类型。

而切片则非常灵活，它可以动态地扩容。切片的类型和长度无关。

数组就是一片连续的内存， slice 实际上是一个结构体，包含三个字段：长度、容量、底层数组。

```go
// runtime/slice.go
type slice struct {
	array unsafe.Pointer // 元素指针
	len   int // 长度
	cap   int // 容量
}
```
#### map

map 的 panic 无法被 recovery

#### sync.Map

https://www.cnblogs.com/qcrao-2018/p/12833787.html

![go sync map|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/go_sync_map.3d4nenqux9.webp)


1. `sync.map` 是线程安全的，读取，插入，删除也都保持着常数级的时间复杂度。
2. 通过读写分离，降低锁时间来提高效率，适用于读多写少的场景。
3. Range 操作需要提供一个函数，参数是 `k,v`，返回值是一个布尔值：`f func(key, value interface{}) bool`。
4. 调用 Load 或 LoadOrStore 函数时，如果在 read 中没有找到 key，则会将 misses 值原子地增加 1，当 misses 增加到和 dirty 的长度相等时，会将 dirty 提升为 read。以期减少“读 miss”。
5. 新写入的 key 会保存到 dirty 中，如果这时 dirty 为 nil，就会先新创建一个 dirty，并将 read 中未被删除的元素拷贝到 dirty。
6. 当 dirty 为 nil 的时候，read 就代表 map 所有的数据；当 dirty 不为 nil 的时候，dirty 才代表 map 所有的数据。
#### defer

https://go.cyub.vip/feature/defer.html

1. defer 函数是按照后进先出的顺序执行
2. defer 函数的传入参数在定义时就已经明确
3. defer 函数可以读取和修改函数的命名返回值

![go defer|500](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/go_defer_internal.6f0jd9wno1.webp)

延迟调用的实参是在此延迟调用被推入延迟调用队列时被估值的
函数体内的表达式是在此函数被执行的时候才会被逐渐估值的，不管此函数是被普通调用还是延迟/协程调用。

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

Go 语言的 [`sync.Mutex`](https://draveness.me/golang/tree/sync.Mutex) 由两个字段 `state` 和 `sema` 组成。其中 `state` 表示当前互斥锁的状态，而 `sema` 是用于控制锁状态的信号量。
```go
type Mutex struct {
	state int32
	sema  uint32
}
```

##### 正常模式

在该模式下，goroutine 在尝试获得锁时，会首先进行**自旋**，在一定次数之后，如果仍获取失败，则会进入等待队列。当正在持有锁的 goroutine 释放锁后，并不会直接将锁传递给等待队列的第一个 goroutine。而是第一个尝试获得锁的 goroutine，也就是说：等待队列的第一个 goroutine，需要和其他新来的 goroutine 竞争。而这种竞争，往往是新来的 goroutine 更有优势，它们正在持有 CPU 时间，而队列头部的 goroutine 才刚被唤醒，并且新来的可能数量更多。如果队列头部的 goroutine 获取失败，它并不会去到队列尾部，而是继续在头部等待。可以看出，在正常模式下的 Mutex 是一种**非公平锁**，在竞争少的情况下，拥有**高吞吐量**，但是它会导致 goroutine 饥饿。而当队列中的 goroutine 的等待时间超过 1 ms 时，Mutex 会转变为饥饿模式。

##### 饥饿模式
在该模式下，所有新来的 goroutine 都不再会尝试直接进行锁的获取，而是直接到后面进行排队。当前正持有锁的 goroutine 释放锁之后，它会将锁直接传递给位于等待队列头部的 goroutine。而当队列全部清空，或者有一个 goroutine 的等待时间小于 1 ms 就获得了锁的时候，Mutex 又会重新转变为正常模式。可以看到，饥饿模式下，Mutex 为公平锁，所有新来的 goroutine 都会排队，相比正常模式吞吐量下降，但是优化了线程饥饿。

### panic/recover

`panic` 能够改变程序的控制流，调用 `panic` 后会立刻停止执行当前函数的剩余代码，并在当前 Goroutine 中递归执行调用方的 `defer`；
`recover` 可以中止 `panic` 造成的程序崩溃。它是一个只能在 `defer` 中发挥作用的函数，在其他作用域中调用不会发挥作用；

- `panic` 只会触发当前 Goroutine 的 `defer`；
- `recover` 只有在 `defer` 中调用才会生效；
- `panic` 允许在 `defer` 中嵌套多次调用；

### cgo

https://www.cockroachlabs.com/blog/the-cost-and-complexity-of-cgo/

## Redis

https://www.xiaolincoding.com/redis/
https://medium.com/nerd-for-tech/understanding-redis-in-system-design-7a3aa8abc26a

### Event loop

- Redis主要的处理流程包括接收请求、执行命令，以及周期性地执行后台任务（serverCron），这些都是由这个事件循环驱动的。
- 当请求到来时，I/O事件被触发，事件循环被唤醒，根据请求执行命令并返回响应结果；
- 同时，后台异步任务（如回收过期的key）被拆分成若干小段，由timer事件所触发，夹杂在I/O事件处理的间隙来周期性地运行。
- 这种执行方式允许仅仅使用一个线程来处理大量的请求，并能提供快速的响应时间。当然，这种实现方式之所以能够高效运转，除了事件循环的结构之外，还得益于系统提供的异步的I/O多路复用机制(I/O multiplexing)。
- 事件循环利用I/O多路复用机制，对 CPU 进行时分复用 (多个事件流将 CPU 切割成多个时间片，不同事件流的时间片交替进行)，使得多个事件流就可以并发进行。
- 而且，使用单线程事件机制可以避免代码的并发执行，在访问各种数据结构的时候都无需考虑线程安全问题，从而大大降低了实现的复杂度。

### 6.0 多线程 

![redis compare single/multiplethead|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/redis_multithread.41xwwvxj5e.webp)

### Cluster
https://redis.io/docs/reference/cluster-spec/#main-properties-and-rationales-of-the-design

codis 使用 proxy 协调请求，默认分为 1024 个槽位（可以增加），使用 zk/etcd 在集群中同步持久化 meta 信息（槽位信息）
扩容 redis 实例时，会把需要迁移的槽位的 key 都复制到新的 redis 实例上
如果当前访问的 key 所属槽位正在迁移，就会先强制迁移当前 key 到新的 redis 实例上，然后再执行操作

### 持久化（RDB & AOF）

#### AOF 
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

**AOF 重写**

到达阈值后，压缩 AOF 文件，读取当前数据库中的所有键值对，然后将每一个键值对用一条命令记录到「新的 AOF 文件」，等到全部记录完后，就将新的 AOF 文件替换掉现有的 AOF 文件（使用新的 AOF 文件，避免重写流程失败，污染现有的 AOF 文件）

**AOF 后台重写**
子进程进行 AOF 重写期间，主进程可以继续处理命令请求，从而避免阻塞主进程；
子进程带有主进程的数据副本（数据副本怎么产生的后面会说），这里使用子进程而不是线程，因为如果是使用线程，多线程之间会共享内存，那么在修改共享内存数据的时候，需要通过加锁来保证数据的安全，而这样就会降低性能。而使用子进程，创建子进程时，父子进程是共享内存数据的，不过这个共享的内存只能以只读的方式，而当父子进程任意一方修改了该共享内存，就会发生「写时复制」，于是父子进程就有了独立的数据副本，就不用加锁来保证数据安全。

子进程是怎么拥有主进程一样的数据副本的呢？

主进程在通过 fork 系统调用生成 bgrewriteaof 子进程时，操作系统会把主进程的「页表」复制一份给子进程，这个页表记录着虚拟地址和物理地址映射关系，而不会复制物理内存，也就是说，两者的虚拟空间不同，但其对应的物理空间是同一个。

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

当子进程完成 AOF 重写工作（扫描数据库中所有数据，逐一把内存数据的键值对转换成一条命令，再将命令记录到重写日志）后，会向主进程发送一条信号，信号是进程间通讯的一种方式，且是异步的。

主进程收到该信号后，会调用一个信号处理函数，该函数主要做以下工作：

将 AOF 重写缓冲区中的所有内容追加到新的 AOF 的文件中，使得新旧两个 AOF 文件所保存的数据库状态一致；
新的 AOF 的文件进行改名，覆盖现有的 AOF 文件。

#### RDB
内存快照，redis 某一时刻的数据状态以文件的形式写到磁盘上，做数据恢复时，可以直接把文件数据读入内存

Redis 提供了两个命令来生成 RDB 文件，分别是 save 和 bgsave，他们的区别就在于是否在「主线程」里执行：

- 执行了 save 命令，就会在主线程生成 RDB 文件，由于和执行操作命令在同一个线程，所以如果写入 RDB 文件的时间太长，会阻塞主线程；
- 执行了 bgsave 命令，会创建一个子进程来生成 RDB 文件，这样可以避免主线程的阻塞；

### 缓存问题

- 缓存穿透：是指恶意请求或者不存在的 key 频繁访问缓存，数据库数据不存在也无法写回，导致请求直接绕过缓存访问数据库，增加数据库负担。解决方法包括使用布隆过滤器、空值缓存等
- 缓存击穿：是指某个热点 key 过期时，大量请求同时访问该 key，导致缓存失效，请求直接访问数据库。解决方法包括设置较长有效期、使用互斥锁等
- 缓存雪崩：是指大量缓存数据同时失效或过期，导致大量请求直接访问数据库，造成数据库压力激增，甚至导致系统崩溃。解决方法包括设置不同的过期时间、二级缓存、使用熔断机制等

### 缓存更新

#### Cache Aside

先更新 db 后删除缓存（或者延迟双删）

### 缓存实现

GroupCache
BigCache

#### Ristretto

Ristretto 支持两种配置计算 memory 使用量
- total usage，每次 `set/remove` 都去 check 内存
- total key，不用每次都去 check，只用计算 key 总数（内存使用量不确定）
底层是 mutex shared map，`runtime.memhash` 计算分片
get 操作是加入到 shared 队列，批量 mutex 1 个执行
set 操作是加入到 channel，但是超过容量则 drop 这个操作

ttl 是将 expire 的 kyes 分桶存储（5s），一次 check 一个桶的 key，对比每个 key 一个 ticker，减少 ticker 数量
### Connection Pool

| Term         | Meaning                                                         |
| ------------ | --------------------------------------------------------------- |
| MinIdle      | 最小空闲连接数，它决定了池中存在的最小连接数。增大则有助于减少流量突然增加时创建新连接所花费的时间，过大则浪费资源       |
| MaxIdle      | 最大空闲连接数，它决定了可以放回池的最大连接数。它等价于 `pool_size` 。                      |
| MaxActive    | 最大打开连接数，决定可以使用的最大连接数。MaxActive 等于 `pool_size + overflow_size` 。 |
| OverflowSIze | 超过最大空闲连接数后可以创建的连接                                               |

![cache pool lifetime|800](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/redis_pool_phase.2krrwu8vqr.webp) 


### 缓存使用

多级缓存

api_cache
- partner_id % 100 -> randomint 打散 （partner_id 实际上也会存在单点）

修改频率比较低，增加内存缓存，并且任务定时更新

seller centre 的后台更新 sidebar 缓存

## Clickhouse

官网介绍
https://clickhouse.com/docs/zh

### OLAP 场景的关键特征
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


### 优化

1. 避免字段空值（例如在 log report 时上传设置的默认值）
2. 创建中间表（partner_id, shop_id, path）聚合为每天的 (partner_id, shop_id, success_calls...) 表

## MySQL

https://www.xiaolincoding.com/mysql/

- 索引：B+ tree 索引、哈希索引、全文索引、覆盖索引等
- 存储引擎简单介绍，如 InnoDB, MyISAM
- 主从复制
- 事务特性简单介绍

### 索引

#### 基础

https://use-the-index-luke.com/sql/anatomy/the-tee

在创建表时，InnoDB 存储引擎会根据不同的场景选择不同的列作为索引：
- 如果有主键，默认会使用主键作为聚簇索引的索引键（key）；
- 如果没有主键，就选择第一个不包含 NULL 值的唯一列作为聚簇索引的索引键（key）；
- 在上面两个都没有的情况下，InnoDB 将自动生成一个隐式自增 id 列作为聚簇索引的索引键（key）；

其它索引都属于辅助索引（Secondary Index），也被称为二级索引或非聚簇索引。创建的主键索引和二级索引默认使用的是 B+Tree 索引

B+Tree 存储千万级的数据只需要 3-4 层高度就可以满足，这意味着从千万级的表查询目标数据最多需要 3-4 次磁盘 I/O，所以 B+Tree 相比于 B 树和二叉树来说，最大的优势在于查询效率很高，因为即使在数据量很大的情况，查询一个数据的磁盘 I/O 依然维持在 3-4 次

主键索引的 B+Tree 和二级索引的 B+Tree 区别如下：
- 主键索引的 B+Tree 的叶子节点存放的是实际数据，所有完整的用户记录都存放在主键索引的 B+Tree 的叶子节点里；
- 二级索引的 B+Tree 的叶子节点存放的是主键值，而不是实际数据。
- 获取主键值，然后再通过主键索引中的 B+Tree 树查询到对应的叶子节点，然后获取整行数据。这个过程叫「回表」，也就是说要查两个 B+Tree 才能查到数据

从物理存储的角度来看，索引分为聚簇索引（主键索引）、二级索引（辅助索引）
- 主键索引的 B+Tree 的叶子节点存放的是实际数据，所有完整的用户记录都存放在主键索引的 B+Tree 的叶子节点里；
- 二级索引的 B+Tree 的叶子节点存放的是主键值，而不是实际数据

从字段特性的角度来看，索引分为主键索引、唯一索引、普通索引、前缀索引。

从字段个数的角度来看，索引分为单列索引、联合索引（复合索引）。
- 建立在单列上的索引称为单列索引，比如主键索引；
- 建立在多列上的索引称为联合索引；

使用联合索引时，存在**最左匹配原则**，也就是按照最左优先的方式进行索引的匹配。在使用联合索引进行查询的时候，如果不遵循「最左匹配原则」，联合索引会失效，这样就无法利用到索引快速查询的特性了。

#### 联合索引范围查询

https://www.xiaolincoding.com/mysql/index/index_interview.html#%E8%81%94%E5%90%88%E7%B4%A2%E5%BC%95%E8%8C%83%E5%9B%B4%E6%9F%A5%E8%AF%A2

联合索引有一些特殊情况，**并不是查询过程使用了联合索引查询，就代表联合索引中的所有字段都用到了联合索引进行索引查询**，也就是可能存在部分字段用到联合索引的 B+Tree，部分字段没有用到联合索引的 B+Tree 的情况。

这种特殊情况就发生在范围查询。联合索引的最左匹配原则会一直向右匹配直到遇到「范围查询」就会停止匹配。**也就是范围查询的字段可以用到联合索引，但是在范围查询字段的后面的字段无法用到联合索引**。

我们也可以在执行计划中的 key_len 知道这一点，在使用联合索引进行查询的时候，通过 key_len 我们可以知道优化器具体使用了多少个字段的搜索条件来形成扫描区间的边界条件。

**联合索引的最左匹配原则，在遇到范围查询（如 >、<）的时候，就会停止匹配，也就是范围查询的字段可以用到联合索引，但是在范围查询字段的后面的字段无法用到联合索引。注意，对于 >=、<=、BETWEEN、like 前缀匹配的范围查询，并不会停止匹配**

#### 索引下推

现在我们知道，对于联合索引（a, b），在执行 `select * from table where a > 1 and b = 2` 语句的时候，只有 a 字段能用到索引，那在联合索引的 B+Tree 找到第一个满足条件的主键值（ID 为 2）后，还需要判断其他条件是否满足（看 b 是否等于 2），那是在联合索引里判断？还是回主键索引去判断呢？

- 在 MySQL 5.6 之前，只能从 ID2 （主键值）开始一个个回表，到「主键索引」上找出数据行，再对比 b 字段值。
- 而 MySQL 5.6 引入的**索引下推优化**（index condition pushdown)， **可以在联合索引遍历过程中，对联合索引中包含的字段先做判断，直接过滤掉不满足条件的记录，减少回表次数**。

当你的查询语句的执行计划里，出现了 Extra 为 `Using index condition`，那么说明使用了索引下推的优化

#### 索引失效
- 当我们使用左或者左右模糊匹配的时候，也就是 `like %xx` 或者 `like %xx%`这两种方式都会造成索引失效；
- 当我们在查询条件中对索引列做了计算（col1 + col2 > xxx, col1 + 1 = 10 不行，col1 = 10 - 1）、函数（DATE）、类型转换操作（MySQL 比较时是默认是字符串转数字，如果 id = "1" 可以，phone_num = 132xxxxx 不行）
- 联合索引要能正确使用需要遵循最左匹配原则，也就是按照最左优先的方式进行索引的匹配，否则就会导致索引失效。
- 在 WHERE 子句中，如果在 OR 前的条件列是索引列，而在 OR 后的条件列不是索引列，那么索引会失效。

#### Other
- 索引区分度
- 主键索引最好是自增的
- 索引最好设置为 NOT NULL

#### `count(1)` / `count(*)` / `count(id)` / `count(col)`

https://www.xiaolincoding.com/mysql/index/count.html

`count(1)` 和 `count(*)` 没区别，有优化

`count(1)`、 `count(*)`、 `count(主键字段)` 在执行的时候，如果表里存在二级索引，优化器就会选择二级索引进行扫描。

所以，如果要执行 `count(1)`、 `count(*)`、` count(主键字段)` 时，尽量在数据表上建立二级索引，这样优化器会自动采用 key_len 最小的二级索引进行扫描，相比于扫描主键索引效率会高一些。

再来，就是不要使用 `count(字段)` 来统计记录个数，因为它的效率是最差的，会采用全表扫描的方式来统计。如果你非要统计表中该字段不为 NULL 的记录个数，建议给这个字段建立一个二级索引。

### 事务
https://www.xiaolincoding.com/mysql/transaction/mvcc.html

- **原子性（Atomicity）**：一个事务中的所有操作，要么全部完成，要么全部不完成，不会结束在中间某个环节，而且事务在执行过程中发生错误，会被回滚到事务开始前的状态，就像这个事务从来没有执行过一样，就好比买一件商品，购买成功时，则给商家付了钱，商品到手；购买失败时，则商品在商家手中，消费者的钱也没花出去。
- **一致性（Consistency）**：是指事务操作前和操作后，数据满足完整性约束，数据库保持一致性状态。比如，用户 A 和用户 B 在银行分别有 800 元和 600 元，总共 1400 元，用户 A 给用户 B 转账 200 元，分为两个步骤，从 A 的账户扣除 200 元和对 B 的账户增加 200 元。一致性就是要求上述步骤操作后，最后的结果是用户 A 还有 600 元，用户 B 有 800 元，总共 1400 元，而不会出现用户 A 扣除了 200 元，但用户 B 未增加的情况（该情况，用户 A 和 B 均为 600 元，总共 1200 元）。
- **隔离性（Isolation）**：数据库允许多个并发事务同时对其数据进行读写和修改的能力，隔离性可以防止多个事务并发执行时由于交叉执行而导致数据的不一致，因为多个事务同时使用相同的数据时，不会相互干扰，每个事务都有一个完整的数据空间，对其他并发事务是隔离的。也就是说，消费者购买商品这个事务，是不影响其他消费者购买的。
- **持久性（Durability）**：事务处理结束后，对数据的修改就是永久的，即便系统故障也不会丢失。

InnoDB 引擎通过什么技术来保证事务的这四个特性的呢？

- 持久性是通过 redo log （重做日志）来保证的；
- 原子性是通过 undo log（回滚日志） 来保证的；
- 隔离性是通过 MVCC（多版本并发控制） 或锁机制来保证的；
- 一致性则是通过持久性+原子性+隔离性来保证；

#### 隔离级别
https://cloud.tencent.com/developer/article/1450773
并行化事务带来的问题
- 脏读：读到其他事务未提交的数据；
	- 如果一个事务「读到」了另一个「未提交事务修改过的数据」，就意味着发生了「脏读」现象。
- 不可重复读：前后读取的数据不一致；
	- 在一个事务内多次读取同一个数据，如果出现前后两次读到的数据不一样的情况，就意味着发生了「不可重复读」现象。
- 幻读：前后读取的记录数量不一致
	- 在一个事务内多次查询某个符合查询条件的「记录数量」，如果出现前后两次查询到的记录数量不一样的情况，就意味着发生了「幻读」现象。

严重程度

![并行化事务问题严重程度](https://cdn.xiaolincoding.com//mysql/other/d37bfa1678eb71ae7e33dc8f211d1ec1.png)

解决办法，隔离性的四个级别

- **读未提交（_read uncommitted_）**，指一个事务还没提交时，它做的变更就能被其他事务看到；
- **读提交（_read committed_）**，指一个事务提交之后，它做的变更才能被其他事务看到；
- **可重复读（_repeatable read_）**，指一个事务执行过程中看到的数据，一直跟这个事务启动时看到的数据是一致的，**MySQL InnoDB 引擎的默认隔离级别**；
- **串行化（_serializable_ ）**；会对记录加上读写锁，在多个事务对这条记录进行读写操作时，如果发生了读写冲突的时候，后访问的事务必须等前一个事务执行完成，才能继续执行；

按隔离水平高低排序如下：
![隔离级别水平](https://cdn.xiaolincoding.com//mysql/other/cce766a69dea725cd8f19b90db2d0430.png)

### 查询优化器

TIDB: https://docs.pingcap.com/zh/tidb/stable/predicate-push-down

1. **Constant Folding**: Simplifying constant expressions at compile time. For example, replacing `WHERE 3*2=6` with `WHERE 6=6`, and then with `WHERE TRUE`. 常量折叠：在编译时简化常量表达式。例如，将 `WHERE 3*2=6` 替换为 `WHERE 6=6` ，然后替换为 `WHERE TRUE` 。
2. **Predicate Pushdown**: Moving the `WHERE` conditions as close as possible to the data retrieval stage, which minimizes the number of rows that need to be processed in the later stages of query execution. 谓词下推：将 `WHERE` 条件移至尽可能靠近数据检索阶段，这样可以最大限度地减少查询执行后期需要处理的行数。
3. **Join Reordering**: Finding the most efficient order to join tables based on their sizes and the join conditions, which can significantly affect the performance of the query. 连接重新排序：根据表的大小和连接条件查找最有效的连接表顺序，这会显着影响查询的性能。
4. **Index Usage**: Determining when to use indexes to speed up data retrieval. This includes choosing the best index if multiple are available and deciding whether a table scan might be more efficient than index access. 索引使用：确定何时使用索引来加速数据检索。这包括在有多个索引可用时选择最佳索引，并确定表扫描是否比索引访问更有效。
5. **Eliminating Unnecessary Joins**: Removing joins that are not needed to satisfy the query, such as when joining with a table but not selecting any columns from it and there are no conditions that use it. 消除不必要的联接：删除满足查询不需要的联接，例如在联接表但未从中选择任何列并且没有条件使用它时。
6. **Subquery Flattening**: Transforming subqueries into joins or applying other transformations to make them more efficient. 子查询扁平化：将子查询转换为联接或应用其他转换以提高其效率。
7. **Materialization**: Avoiding unnecessary computations by storing intermediate results in a temporary table, which can then be reused in the query. This is particularly useful for subqueries that are referenced multiple times. 具体化：通过将中间结果存储在临时表中来避免不必要的计算，然后可以在查询中重用该结果。这对于多次引用的子查询特别有用
8. **Query Rewrite**: Rewriting queries in a more optimal form without changing the semantics. For example, transforming `IN` to `EXISTS` or vice versa if it can be done more efficiently. 查询重写：以更优化的形式重写查询，而不改变语义。例如，将 `IN` 转换为 `EXISTS` 或反之亦然（如果可以更有效地完成）。
9. **Partition Pruning**: In a partitioned table, skipping the partitions that are not relevant to the query based on the `WHERE` clause conditions. 分区修剪：在分区表中，根据 `WHERE` 子句条件跳过与查询无关的分区。
10. **Column Pruning**: Excluding unused columns from the query plan so that less data is read from the disk. 列修剪：从查询计划中排除未使用的列，从而减少从磁盘读取的数据。
11. **Batch Processing**: Processing rows in batches to reduce the overhead of context switching and to improve the efficiency of certain operations like joins and aggregations. 批处理：批量处理行以减少上下文切换的开销并提高某些操作（如连接和聚合）的效率。
12. **Parallel Execution**: Taking advantage of multiple CPU cores to execute different parts of the query in parallel, which can significantly reduce the total execution time for complex queries. 并行执行：利用多个 CPU 核心并行执行查询的不同部分，这可以显着减少复杂查询的总执行时间。
13. **Aggregate Pushdown**: Performing aggregation as early as possible in the execution plan, which can often reduce the amount of data that needs to be transferred and processed in later stages. 聚合下推：在执行计划中尽早进行聚合，这往往可以减少后期需要传输和处理的数据量。
14. **Index Merge**: Combining multiple indexes on the same table to filter rows more efficiently than would be possible using any single index. 索引合并：在同一个表上组合多个索引来过滤行，比使用任何单个索引更有效。
15. **Late Materialization**: Delaying the retrieval of full row data until it is absolutely necessary, which can reduce IO if only a few columns are needed initially or if there are filters that can be applied first. (Late Materialization：延迟全行数据的检索，直到绝对必要时，如果最初只需要几列或者有可以首先应用的过滤器，这可以减少 IO。
16. **Common Subexpression Elimination**: Identifying and reusing the results of expressions that are computed more than once within a query. 公共子表达式消除：识别并重用在查询中多次计算的表达式的结果。
17. **Using Covering Indexes**: When a query can be satisfied entirely by the data in an index, thereby avoiding the need to access the main table data. 使用覆盖索引：当查询可以完全由索引中的数据满足时，从而避免访问主表数据的需要。
18. **Table Elimination**: Removing tables from the query that do not affect the result, such as when using `LEFT JOIN` where the joined table's columns are not used, and the join condition is always true. 表消除：从查询中删除不影响结果的表，例如使用 `LEFT JOIN` 时，不使用连接表的列，并且连接条件始终为 true。

## Elasticsearch

https://javapub.blog.csdn.net/article/details/123761794

https://blog.csdn.net/weixin_35688430/article/details/110545234

https://pdai.tech/md/db/nosql-es/elasticsearch.html

https://www.perplexity.ai/search/Elasticsearch-euZ.7A8MQo6pamoulLA7Cw

## Kafka/RabbitMQ
https://jack-vanlightly.com/blog/2023/11/14/the-architecture-of-serverless-data-systems

https://mdnice.com/writing/c1d01d8793154629a82a9eb1bc0d1318

https://engineering.linkedin.com/kafka/benchmarking-apache-kafka-2-million-writes-second-three-cheap-machines

### Kafka 为什么这么快？

Kafka 之所以速度快，主要归功于其设计和实现中的几个关键优化策略：

1. 顺序磁盘 I/O
Kafka 利用磁盘顺序读写的高效性，将消息持久化到本地磁盘中。与随机读写相比，顺序读写的性能要高出几个数量级。Kafka 的消息是不断追加到日志文件的末尾，这种顺序写的方式大大提高了写入吞吐量
2. 零拷贝技术（Zero-copy）
Kafka 使用零拷贝技术来优化网络传输过程中的数据复制操作。通过直接在内核空间和网络缓冲区之间传输数据，避免了 CPU 的额外负担，从而提高了数据传输的效率
3. 分区和并行处理
Kafka 的 Topic 被分为多个 Partition，这些 Partition 可以分布在不同的服务器上，从而实现数据的并行处理和负载均衡。每个 Partition 都是独立的，可以并行写入和读取，大大提高了系统的吞吐量
4. 批量处理
Kafka 支持批量发送和批量压缩消息，这减少了网络 I/O 的次数和数据的体积，从而提高了效率
5. Page Cache 和操作系统优化
Kafka 充分利用了操作系统的 Page Cache，通过内存映射（mmap）文件和 sendfile 系统调用，减少了数据在用户空间和内核空间之间的拷贝次数，提高了读写性能
6. 持久化和复制策略
尽管 Kafka 将数据持久化到磁盘，但其高效的存储格式和复制策略确保了数据的快速读写和高可用性。Kafka 的数据文件被组织成一系列可追加的日志段，每个段都有索引，使得数据的读取非常高效
7. 消费者拉取模式
Kafka 采用消费者拉取（pull）模式，消费者根据自己的消费能力从 Broker 拉取数据，这种方式使得消费者可以更灵活地控制数据的消费速率，避免了生产者推送（push）模式可能导致的消费者处理不过来而拖慢整体处理速度的问题

## System

https://www.xiaolincoding.com/os

- 进程与线程简单介绍，区别，以及进程间通信方式，线程同步方式
- 用户态和内核态
- 内存管理：分页分段，虚拟内存，空闲地址管理方法
- 死锁：死锁的必要条件，死锁的检测与恢复，死锁的预防，死锁的避免
- 数据库系统
### 进程线程

https://www.geeksforgeeks.org/difference-between-process-and-thread/

#### 进程隔离

#### PID 1 和父子进程

[操作系统内核](https://zh.wikipedia.org/wiki/%E6%A0%B8%E5%BF%83 "核心") 以[进程标识符](https://zh.wikipedia.org/wiki/%E8%BF%9B%E7%A8%8B%E6%A0%87%E8%AF%86%E7%AC%A6 "进程标识符")（_Process Identifier_，即 PID）来识别进程。进程 0 是系统[引导](https://zh.wikipedia.org/wiki/%E5%BC%95%E5%AF%BC "引导") 时创建的一个特殊进程，在其调用 fork 创建出一个子进程（即 PID=1 的进程 1，又称[init](https://zh.wikipedia.org/wiki/Init "Init")）后，进程 0 就转为[交换进程](https://zh.wikipedia.org/wiki/%E8%B0%83%E5%BA%A6 "调度")（有时也被称为[空闲进程](https://zh.wikipedia.org/w/index.php?title=Idle_(CPU)&action=edit&redlink=1)），而进程 1（init 进程）就是系统里其他所有进程的祖先。
#### 守护进程和后台进程
在 Linux 中，守护进程（Daemon process）和后台进程（Background process）之间有一些关键区别：

后台进程（Background process）

- 后台进程通常是在终端中启动的，可以通过在命令后面加上 `&` 符号来将其放入后台运行。
- 后台进程仍然与启动它的终端会话相关联，其标准输入/输出通常与终端相关联，除非显式更改。
- 当启动后台进程的终端退出时，后台进程会收到 SIGHUP 信号并可能被终止。
- 后台进程通常仍然有父进程，如果父进程退出，可能会影响到后台进程。

守护进程（Daemon process）

- 守护进程是在系统启动时启动并在系统关闭时停止的特殊进程。
- 守护进程在后台运行，并且与控制终端无关，通常是系统的初始线程的子进程。
- 守护进程的标准输入/输出通常被重定向到 `/dev/null`，以确保不受终端影响。
- 守护进程没有父进程，其父进程通常是 init 或 systemd 等系统级别的守护程序

Daemon process 特点

1. 继承当前 session （对话）的标准输出（stdout）和标准错误（stderr）。因此，后台任务的所有输出依然会同步地在命令行下显示。
2. 不再继承当前 session 的标准输入（stdin）。你无法向这个任务输入指令了。如果它试图读取标准输入，就会暂停执行（halt）。

#### 在 linux 创建 background process

使用&符号：在 Linux 终端中，可以通过在命令后面加上&符号来将进程放入后台运行。例如，command &会将 command 这个进程放入后台运行，不会阻塞当前终端

使用 nohup 命令：使用 nohup 命令可以创建一个忽略 SIGHUP 信号的进程，即在终端关闭后仍然继续运行。通过结合 nohup 和&符号，可以创建一个后台运行的进程，并且关闭当前终端不会影响该进程的执行。示例：`nohup command &`

#### 在 linux 上创建一个 daemon process

https://github.com/pasce/daemon-skeleton-linux-c

基本步骤

1. 创建子进程，父进程退出
首先，使用 fork () 函数创建一个子进程。如果 fork () 返回值大于 0，则表示当前进程是父进程，此时父进程应该调用 exit (0) 退出。这样做的目的是让子进程成为孤儿进程，从而被 init 进程（PID 为 1）收养，确保子进程不会成为进程组的组长，这是后续步骤中调用 setsid () 函数的前提条件

2. 在子进程中创建新会话
子进程调用 setsid () 函数创建一个新的会话，并成为新会话的首进程和进程组的组长。这一步使得进程脱离原有的终端控制，确保守护进程不会意外地获得控制终端

3. 改变当前目录为根目录
通过调用 chdir ("/") 函数将当前工作目录改变为根目录。这是因为守护进程通常需要在系统运行期间一直运行，如果守护进程的工作目录是一个挂载的文件系统，那么这个文件系统就无法被卸载

4. 重新设置文件权限掩码
调用 umask (0) 函数清除文件创建掩码。这样做是为了确保守护进程创建的文件和目录具有合适的权限，不受继承自父进程的 umask 值的影响

5. 关闭文件描述符
子进程从父进程继承了打开的文件描述符。为了防止守护进程无意中使用这些文件描述符，应该关闭它们。可以通过 getdtablesize () 函数获取进程打开的文件描述符数目，然后遍历并关闭这些文件描述符

## 网络编程

![tcp sockets connect accept|700](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/sockts_tcp_connect_accept.b8rattz9v.webp)
### IO 多路复用

https://www.xiaolincoding.com/os/8_network_system/selete_poll_epoll.html#%E6%9C%80%E5%9F%BA%E6%9C%AC%E7%9A%84-socket-%E6%A8%A1%E5%9E%8B

https://moonbingbing.gitbooks.io/openresty-best-practices/content/base/web_evolution.html

#### Select

https://learnku.com/docs/pymotw/select-wait-for-io-efficiently/3429

```cpp
int select(int n, fd_set *readfds, fd_set *writefds,
        fd_set *exceptfds, struct timeval *timeout);
```
#### Pool
```cpp
int poll(struct pollfd *fds, unsigned int nfds, int timeout);
```
#### Epoll

```cpp
int epoll_create(int size)；
int epoll_ctl(int epfd, int op, int fd, struct epoll_event *event)；
            typedef union epoll_data {
                void *ptr;
                int fd;
                __uint32_t u32;
                __uint64_t u64;
            } epoll_data_t;

            struct epoll_event {
                __uint32_t events;      /* Epoll events */
                epoll_data_t data;      /* User data variable */
            };

int epoll_wait(int epfd, struct epoll_event * events,
                int maxevents, int timeout);
```

### 平滑迁移/热重启

https://goteleport.com/blog/golang-ssh-bastion-graceful-restarts/

https://ms2008.github.io/2019/12/28/hot-upgrade/

https://www.hitzhangjie.pro/blog/2020-08-28-go%E7%A8%8B%E5%BA%8F%E5%A6%82%E4%BD%95%E5%AE%9E%E7%8E%B0%E7%83%AD%E9%87%8D%E5%90%AF/
#### Scm_rights

https://zhuanlan.zhihu.com/p/405620115

SCM_RIGHTS 协议类型的套接字通常用于在 Linux 或 UNIX 系统上的进程之间传递文件描述符。通过使用 SCM_RIGHTS 和 sendmsg/recvmsg 函数，可以实现在进程之间传递文件描述符的功能。这种机制允许一个进程将打开的文件描述符传递给另一个进程，从而实现进程间的文件描述符共享。SCM_RIGHTS 协议类型的套接字提供了一种有效的方式来处理文件描述符的传递，使得进程间通信更加灵活和高效

可以用来做 TCP 连接平滑迁移，Cloudflare 写过文章

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
	- 推荐仅在响应 [`GET`](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Methods/GET) 或 [`HEAD`](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Methods/HEAD) 方法时采用 302 状态码，而在其他时候使用 [`307`](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Status/307) Temporary Redirect 来替代
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

### TCP & UDP
https://www.xiaolincoding.com/network/3_tcp/tcp_interview.html

#### TCP security
https://heimdalsecurity.com/blog/what-is-tcp/

### UDS 比 TCP 速度快
是的，Unix Domain Socket（UDS）通常比 TCP 套接字（特别是在使用 TCP loopback 连接时）要快。这种性能差异的原因主要归结于几个关键因素：

#### 1. 绕过网络协议栈
当使用 Unix Domain Socket 时，数据传输不经过网络协议栈，而是直接在内核空间中进行，这减少了数据在用户空间和内核空间之间的拷贝次数，同时也避免了 TCP/IP 协议栈的开销，如头部的添加、校验和的计算、封装和解封装的过程等。相比之下，即使是在同一台机器上使用 TCP loopback 连接，数据仍然需要通过网络协议栈，增加了处理时间。

#### 2. 更少的系统调用
Unix Domain Socket 通常需要更少的系统调用来发送相同量的数据。在 TCP 连接中，每次传输数据都涉及到更多的系统调用，如 `connect`、`send`、`recv` 等，这些都会增加 CPU 的使用率和延迟

#### 3. 避免了 TCP 的一些限制
TCP 连接受到其协议特性的限制，如慢启动、拥塞控制、流量控制等，这些特性在跨网络通信时非常有用，但在同一台机器上的进程间通信（IPC）中可能会导致不必要的延迟。Unix Domain Socket 不受这些限制，因此在本地通信时可以提供更低的延迟和更高的吞吐量

#### 4. 地址使用的是文件路径而非 IP 端口
Unix Domain Socket 使用文件系统中的路径作为地址，而不是网络地址和端口号。这意味着它们不需要管理网络端口号，也不受本地端口数量的限制，而且配置和管理相对简单

#### 性能对比实例
一项性能测试显示，使用 Unix Domain Socket 的实现在一秒钟内可以发送和接收的消息数量是使用 IP 套接字实现的两倍多。这个比例在多次运行中是一致的，表明 Unix Domain Socket 在性能上具有明显的优势

#### 结论
综上所述，Unix Domain Socket 在本地进程间通信中通常比 TCP 套接字要快，主要原因是它绕过了网络协议栈，减少了系统调用的数量，避免了 TCP 协议的一些限制，并使用了文件路径而非网络地址进行通信。这些因素共同作用，使得 Unix Domain Socket 成为本地 IPC 的首选方法，尤其是在对性能有较高要求的场景中

## 后台技术

### OAuth
https://www.ruanyifeng.com/blog/2014/05/oauth_2_0.html

https://juejin.cn/post/7195762258962219069

https://dev.mi.com/console/doc/detail?pId=711

### 可观测性
#### Tracing 

https://www.jaegertracing.io/docs/1.55/architecture/

### Indexing/Search 技术

https://blugelabs.com/blog/bluge-code-walkthrough-indexing/

#### Bloomfilter

https://findingprotopia.org/posts/how-to-write-a-bloom-filter-cpp/

https://en.wikipedia.org/wiki/Bloom_filter

https://ethereum.stackexchange.com/questions/3418/how-does-ethereum-make-use-of-bloom-filters

https://ricardoanderegg.com/posts/understanding-bloom-filters-by-building-one/

https://codapi.org/embed/?sandbox=go&src=gist:e7bde93f98c5e47ca38d359a5104fd88:bloom.go

原版本不支持删除

计数式 bloomfilter

bit 数组改为计数数组，set 每位 +1，del 每位 -1，检测则判断是否全 > 0，如果有 =0，则一定不在

布谷鸟过滤器

#### Count–min sketch

#### Bitmaps(Roaring Bitmaps)

Roaring bitmaps
https://www.vikramoberoi.com/a-primer-on-roaring-bitmaps-what-they-are-and-how-they-work/

https://www.elastic.co/blog/frame-of-reference-and-roaring-bitmaps

https://vikramoberoi.com/posts/using-bitmaps-to-run-interactive-retention-analyses-over-billions-of-events-for-less-than-100-mo/

https://dgraph.io/blog/post/serialized-roaring-bitmaps-golang/

https://news.ycombinator.com/item?id=32937930

Roaring Bitmap 将一个 32 位的整数分为两部分，一部分是高 16 位，另一部分是低 16 位。对于高 16 位，Roaring Bitmap 将它存储到一个有序数组中，这个有序数组中的每一个值都是一个“桶”；而对于低 16 位，Roaring Bitmap 则将它存储在一个 2^16 的位图中，将相应位置置为 1。这样，每个桶都会对应一个 2^16 的位图。

![geektime_roaring_bitmaps|600](https://github.com/abcdlsj/picx-images-hosting/raw/master/knows/geektime_roaring_bitmaps.9nzn9xk5at.webp)

相比于位图法，这种设计方案就是通过，将不存在的桶的位图空间全部省去这样的方式，来节省存储空间的。而代价就是将高 16 位的查找，从位图的 O(1) 的查找转为有序数组的 log(n) 查找。那每个桶对应的位图空间，我们是否还能优化呢？前面我们说过，当位图中的元素太稀疏时，其实我们还不如使用链表。这个时候，链表的计算更快速，存储空间也更节省。Roaring Bitmap 就基于这个思路，对低 16 位的位图部分进行了优化：如果一个桶中存储的数据少于 4096 个，我们就不使用位图，而是直接使用 short 型的有序数组存储数据。同时，我们使用可变长数组机制，让数组的初始化长度是 4，随着元素的增加再逐步调整数组长度，上限是 4096。这样一来，存储空间就会低于 8K，“也就小于使用位图所占用的存储空间了

#### TF-IDF
term frequency–inverse document frequency 词频-逆文档频率

TF(t,d)=在文档 d 中词条 t 出现的次数​ / 文档 d 中所有的词条数目
IDF(t,D)=log(文档集合 D 的总文档数/包含词条 t 的文档数+1​)

TF-IDF = (TF) * (IDF)

#### LSM-Tree & LevelDB

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

#### XSS

攻击者利用应用程序没有对用户的输入，以及页面的输出进行严格地过滤，从而使恶意攻击者能往 Web 页面里插入恶意代码，当用户浏览该页面时，嵌入其中 Web 里面的恶意代码会被执行，从而达到恶意攻击者的特殊目的

建议:
1. 对用户输入的数据进行严格过滤，包括但不限于以下字符及字符串 `javascript script src img onerror alert < >` 
2. 根据页面的输出背景环境，对输出特殊字符进行编码/转义
3. 针对富文本编辑器，除了针对数据源的严格过滤之外，更加有效的方法就是梳理需要使用的安全 HTML 标签和属性白名单 (白名单即可，不需要的标签属性都干掉)，对 href 和 src 进行参数校验。如使用 `sanitize-html` 可参考: <https://www.npmjs.com/package/sanitize-html>
4. 在 Cookie 上设置 HTTPOnly 标志，从而禁止客户端脚本访问 Cookie

#### SQL Injection

1. 验证用户所有的输入数据，进行严格的过滤 (包括但不限于以下字符及字符串:`’、”、<、>、/、*、;、+、-、&、|、(、)、and、or、select、 union`)，某个数据被接受之前，使用标准输入验证机制，验证所有输入数据的长度、类型、语法以及业务规则，有效检测攻击;
2. 使用参数化查询 (`PreparedStatement`)，避免将未经过滤的输入直接拼接到 SQL 查询语句中;

## WORK

### Openplatform

OPENAPI 日志上报 + 搜索优化 + 几个面板

- Opservice 好些需求，santistic，search 优化 (es function score + stop word)，对接了 datasuite 的 clickhouse 写 API，帮他们查了好几个 bug 了
- Opservice 的日志上报插件，我加了 senstive 的功能，基本上一半多我都改了，也算我实现的吧，日志上报插件功能其实也没啥，收集日志，处理+规范格式，发送到 kafka。lua kafka 调优之类的，修改了 buffer size.
- Opservice 的日常几个需求，其实记不太得了，改改 bug，没学到什么
- EKL 全组都是我接入的，虽然没学到什么（消费者模型？）
- Commercial metrics 的需求，也是对接的 data servcice（dataservice 的 common 是我设计的，个人觉得还不错）
- 几个面板也算是我做的，service partner，security dashboard，API santistic，log search.
- Partnership 完全我实现，跨团队合作

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
- 网关做了些需求，缝缝补补，可以看下网关发布？说是我参与的？好像也行

https://www.cnblogs.com/upyun/p/17463973.html

#### featuretoggle

`featuretoggle_tag_mapping_<region>_tab` 表
`shop/merchant/subaccount_tag_mapping_{idx}_tab` 表
`tag_info_<region>_tab` 表

查询接口
is_in_feature_toggle(70k)
get_shops_feature_toogles(2k) // 主要是首页

缓存设计：mapping 数据都是用的 hget，便于更新

更新 toggle cache mapping cache 的时候会先拿分布式锁

shop->tagids 长时间硬过期+短时间软过期（+random，防止雪崩），首先命中缓存判断软过期，未过期直接返回旧数据，过期开启异步任务更新缓存
类似于 CNSC 侧边栏

优化点
- 优化 cache 存储，对于 toggle 只存 id
- 优化内部 batch 接口编码，优化数据返回，get shops feature toogles（feature toggle key 编码为 idx 数组）

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

#### OPENAPI 项目
##### 平滑迁移 partner key
难点：
移除 rabbitMQ 组件，改用存储+定时任务扫表实现
任务执行间隔 5s

##### 推送优化
push system 是这样的
业务流程
biz team 发送到 push gateway 的 API，根据消息 region producer 到 push consumer 的 topic 里，然后消费发送到用户 callback 地址
为了做 split by market，申请了 br 的 topic，部署了 br push gateway 服务

还有 cn，cn 只有 consumer 服务是在 cn
做分拣处理，根据用户的 region 选择，sg 的 br topic 同步到 br 的 br topic，br 的 sg topic 同步到 sg 的 sg topic，sg 和 br 的 consumer 再依次消费
cn 也是同样

region 选择，提供测速，选择延迟最低的 region 即可

效果？
原来的 push 消息，同一个 erp 提供商 br 消息里的 sg 开发者，br 的消息调用延迟 300~400ms，sg 10ms
导致消息堆积
修改后，接口调用延迟降低，20 ms

难点？
技术方案的设计，代码处理，涉及发送的地方都要判断，region 测速时超时等 error 的处理
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

消息平均大小 25kb（244k/10）

memory 消耗问题
4c4g 机器，占用到了 3.5g，平常是 2g 以下，原因

不上报的话是 1.5g，正常上报是 2.5g

memory usage = max_buffering * message_size * 4(worker) = 10000 * 15kb（按照 15kb 算） * 4（worker 数量） = 600m

后续需求添加了上报的数据，消息体接近翻倍，内存占用到了 3g 以上，控制在 70%

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
lua(LuaJIT) 的 gc 策略是占用内存需要达到当前内存的 2 倍才会 gc，所以会有很多占用不会回收，go 会基于分配策略来调整 gc 触发时机
而且 golang 的 samara 的吞吐量 qps 大于 lua resty kafka，所以 momory 占有会减少，cpu 利用率会上升，将上报和网关拆分，内存出现不稳定情况，不会影响到网关

#### seller center 侧边栏/弹窗
##### 侧边栏
因为 CBSC 绑定店铺很多，而且每个 shop 都需要判断 feature toggle 和 authcode，所以采用异步加载的方案，检查缓存是否上次设置时间是否是 15 分钟之内，是的话直接返回
不是的话，进行加载，加载时间超过 3s 就直接返回上次计算出的，或者没有计算出的数据的话，就返回全量数据，在后台继续计算，然后设置回去
计算过程也并行的家在，加载 feature toggle，加载 shop 列表

消息
监听 subaccount 和 shop 变化关系，提前做计算

定时更新大卖家的侧边栏

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
尾部采样是支持的，可以上报特殊的 tag 信息就行

出现的问题

- span id 断开的问题
	- gin 的问题
### 开发过程遇到的问题

##### gin context 涉及的问题
因为接入 tracing，需要上报 span id，在我们的系统内部，传递信息是使用 context
默认的 tracing sdk 是写入到 c.Request.Context，在下面是拿不到的，因为后续使用和上报都是用的 gin.Context
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

gorm 中调用 Rows() 函数进行查询的时候，需要获取一个连接。策略是：
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
func longestConsecutive(nums []int) (ans int) {
	set := make(map[int]bool)

	for _, num := range nums {
		set[num] = true
	}

	for _, num := range nums {
		if set[num-1] {
			continue
		}

		l := 1
		for set[num+1] {
			num++
			l++
		}

		ans = max(ans, l)
	}

	return
}
```
#### [128. 最长连续序列](https://leetcode.cn/problems/longest-consecutive-sequence/)

`unordered_set` 处理

**不要重复计算 num - 1 的序列**

```cpp
class Solution {
public:
    int longestConsecutive(vector<int>& nums) {
        unordered_set<int> set(nums.begin(), nums.end());

        int ans = 0;

        for (auto num : nums) {
            if (set.count(num - 1)) continue;

            int len = 1;
            while (set.count(num + 1)) {
                len++;
                num++;
            }

            ans = max(ans, len);
        }

        return ans;
    }
};
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
1. 首先知道每次只能从开头或者结尾拿数字，就知道拿取的数字，在首尾两端形成长度 >= 的子数组
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
这里的 `arr` 数组是在遍历时更新的，减少一个 `n` 的时间复杂度

**这里很重要的一点是，`i` 和 `j` 和 `k` 其实是固定的，我们只能知道 `st`**
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

和 1143 很像的一道题，但是 1143 是子序列，定义和子数组不一样，子序列可以「删除」字符，所以子序列的答案允许通过上一级`max(dp[i-1][j], dp[i][j-1])` 转移，子数组不行，如果当前不一样，那计数只能归零

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
```go
func longestIncreasingPath(matrix [][]int) (ans int) {
	n, m := len(matrix), len(matrix[0])
	if n == 1 && m == 1 {
		return 1
	}
	paths := make([][]int, n*m)

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			paths[i*m+j] = []int{i, j, matrix[i][j]}
		}
	}

	sort.Slice(paths, func(i, j int) bool {
		return paths[i][2] < paths[j][2]
	})

	dp := make([][]int, n)
	for i := 0; i < n; i++ {
		dp[i] = make([]int, m)
	}

	dx, dy := []int{1, 0, -1, 0}, []int{0, 1, 0, -1}
	for i := 0; i < n*m; i++ {
		x, y, v := paths[i][0], paths[i][1], paths[i][2]

		for k := 0; k < 4; k++ {
			nx, ny := x+dx[k], y+dy[k]
			if nx < 0 || nx >= n || ny < 0 || ny >= m || matrix[nx][ny] >= v {
				continue
			}

			dp[x][y] = max(dp[x][y], dp[nx][ny]+1)
			ans = max(ans, dp[x][y]+1) // dp[x][y] 的默认值是 1，这里加上
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

> 这里为什么是 `0..i-1` 序列？

这道题的关键是：对于序列 `0..i` 的解码方式来说，如果当前 `i` 只是 `1-9` 那么，解码方式就是 `f(i) += f(i - 1)`，如果上一个 `i - 1` 和当前能组合成 `10-26`，那么就是 `f(i) += f(i - 2)`
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

### 搜索 (BFS/DFS)

#### [200. 岛屿数量](https://leetcode.cn/problems/number-of-islands/)
```go
func numIslands(grid [][]byte) (ans int) {
	for i := 0; i < len(grid); i++ {
		for j := 0; j < len(grid[0]); j++ {
			if grid[i][j] == '1' {
				helper(grid, i, j)
				ans++
			}
		}
	}

	return ans
}

var dx, dy = []int{1, 0, -1, 0}, []int{0, 1, 0, -1}

func helper(grid [][]byte, i, j int) {
	if i < 0 || i >= len(grid) || j < 0 || j >= len(grid[0]) || grid[i][j] != '1' {
		return
	}

	grid[i][j] = '#'

	for k := 0; k < 4; k++ {
		helper(grid, i+dx[k], j+dy[k])
	}
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
#### [2226. 每个小孩最多能分到多少糖果](https://leetcode.cn/problems/maximum-candies-allocated-to-k-children/)

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
		return (findKth(nums1, nums2, (n+m)/2) + findKth(nums1, nums2, (n+m)/2+1)) / 2
	}

	return findKth(nums1, nums2, (n+m+1)/2)
}

func findKth(nums1 []int, nums2 []int, k int) float64 {
	l := k / 2

	if len(nums1) == 0 {
		return float64(nums2[k-1])
	}

	if len(nums2) == 0 {
		return float64(nums1[k-1])
	}

	if k == 1 {
		return float64(min(nums1[0], nums2[0]))
	}

	l1, l2 := 0, 0
	if l >= len(nums1) {
		l1 = len(nums1) - 1
	} else {
		l1 = l - 1
	}

	if l >= len(nums2) {
		l2 = len(nums2) - 1
	} else {
		l2 = l - 1
	}

	if nums1[l1] < nums2[l2] {
		return findKth(nums1[l1+1:], nums2, k-l1-1)
	}

	return findKth(nums1, nums2[l2+1:], k-l2-1)
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

这里通过判断 `height[le], height[ri]` 的大小，可以确定其当前的 `left max` 和 `right max` 中的`较小`值
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
#### 图的最短路

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

#### 洗牌算法

洗牌算法 knuth-shuffle，生成的 `n!` 种全排列，每种排列出现的概率是相等的，即每个元素放在每个位置的概率都是随机的

主要思想，将牌组分为已排序和未排序两部分，每次从未排序中选择一个放在待排序的位置

上次未被选中的概率是 `(n-1)/n`，被选中的概率是 `1/n-1`，相乘后 `(n-1)/n * (1/n-1) = 1/n`
