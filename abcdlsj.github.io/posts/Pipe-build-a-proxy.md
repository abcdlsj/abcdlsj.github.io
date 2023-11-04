---
title: "Pipe - Build a tunnel tool like `frp` or `ngrok`"
date: 2023-11-02T18:47:37+08:00
tags:
  - Network
  - Weekly
  - Tunnel
hide: false
---
## Background
**简单的转发工具**
公司内部的服务框架通信都是通过 `UNIX domain` 连接机器本地的 `Agent` 实现，在公司容器集群环境，都是会启动 `Agent`，但是作为本地环境，没有 `Agent` 的条件，所以本地启动服务都是使用 `socat` 将本地 `UNIX domain` 流量转发到远程的一个 `TCP agent` 来启动服务的
类似于：`socat unix:/xxx.sock,fork tcp:agent.xxx.io:5443`

> `socat` 是一个瑞士军刀类的工具，非常强大

借助 `io.Copy()` 以及 `net` 包，动手实现了符合需求的转发工具，代码量很少，加上 `flag` 代码不超过 50 行，但是却非常实用，启动速度很快（实测因为 `socat` 会对流量做加解密，所以直接 `copy` 会快一点点）

**远程端口转发**
`frp` 很适用于内网转发，不过功能很多用不上，`ngrok` 则适用于做一些临时暴露本地服务之类的。
因为去年的尝试，所以准备实现一个类似工具，实现过程中没有借鉴太多其它项目的代码，很多地方都是遇到问题再去查，所以写一篇博客记录下是很值得的，这里写一下思路以及核心代码。

> 开始的版本只实现 `TCP` 转发，含有 `Caddy` 来做 `Auto Subdomain https`，代码不到 `1000` 行。后边优化了下，现在支持 `TCP/UDP` 协议，所以本文只涉及 `TCP/UDP` 实现（不过其它协议也大都类似
> 另外顺带一提，`GitHub` 有非常多类似的实现，比如 [ekzhang/bore](https://github.com/ekzhang/bore) 和 [rapiz1/rathole](https://github.com/rapiz1/rathole/)（`Tokio` 的功能太强大了，忍不住想用 `Rust` 重写 :P）

所有的代码都在 [abcdlsj/pipe](https://github.com/abcdlsj/pipe/tree/484084da8b9edb99fb39e5d7561cc94d16d7031c) 里（本文纂写时的版本）

## How it works

实现**远程端口转发**

假设有一个服务器（Server）和一个客户端（Client）。其中，服务器的 IP 可以直接从公网访问，而客户端的 IP 则不行，并且客户端可以访问服务器。

我们希望有一种方法来建立服务器端口和客户端端口之间的关联，将对服务器端口的访问转发到客户端的对应端口，通过公网访问服务器的端口就相当于访问客户端的端口。

**如何实现这个转发？**
首先 Server 端应该和 Client 端进行通信，对于 Server 端的入站请求，将请求和 Client 端进行**绑定**，Client 则对目标端口和 Server 端通信连接进行**绑定**。**绑定** 的意思是对两个连接进行 `io.Copy()`。

假设我们 Server 通信端口是 8910，要将 Client 的 3000 端口穿透到 Server 的 9000 端口。
最后结构差不多就是这样：
```d2
Flow: {
  server: {
    remoteport 9000
  }
  client: {
    localport 3000
  }
  client <-> server: 1. Prepare(handshark, auth, request forward...)
  user -> server: 2. View remote port 9000
  server -> server: 3. io.Copy() control connection 8910 and user connection
  server -> client: 4. Send start proxy request
  client -> client: 5. io.Copy() control connection 8910 and local connection
}
```

这里省略了一些值得思考的东西，比如：
1. Client 端和 Server 端建立连接 `Auth/Handshake` 的部分，如何「鉴权」？
2. `Control` 连接里的消息是否固定长度？如果不固定，怎么处理「边界」问题？
3. Client 端接收到 `Proxy` 请求后，进行 `io.Copy()` 的是哪个连接，Server 端又怎样处理呢？

> 对于 1 和 2 可以看下面的详细实现
> 对于 2，如果没有实现过类似的工具可能不太清楚为什么会有这个问题，看了下面详细的流程大概就清楚了（忽略「鉴权」部分

1. 首先启动 Server 端，「监听」 8910 端口
2. 启动 Client, Client 端和 Server 端建立 `Control` 连接，然后发送一条 `Forward` 接口告诉 Server 端将要转发到 9000 端口
3. Server 端从 `Control` 连接接收到 `Forward` 消息，开始对 9000 端口进行「监听」，准备接收来自用户端的请求
> 此时 Server 是否要 `Copy` 用户连接和 `Control` 连接呢？
> 答案是不应该也不能，因为 `Control` 连接还会有来自 Server 或者 Client 的「其它」的流量，例如 `Close`、`Heartbeat` 消息等，这些流量如果直接 `Copy` 到用户连接上，那就会产生问题。
4. 当有新的用户请求到来时，Server 端通过 `Control` 连接发送 `Exchange` 消息，告诉 Client 端：有新的用户连接，准备开始对流量进行 `Copy`
5. Client 端接收到 `Exchange` 消息，建立连接到 `Local 3000` 端口，准备 `Copy` 流量
> Client 端也不能直接 `Copy` `Control` 连接和 `Local 3000` 连接，和 3 是一样的情况

那么也就是我们遇到了「连接复用」的问题，这个问题在对多端流量进行处理的时候很常见，而且因为这里是直接 `io.Copy` 没办法区分流量的不同。
解决这个问题有很多方法：
1. 可以使用连接复用库，例如 [hashicorp/yamux](https://github.com/hashicorp/yamux)，`frp` 默认使用 `yamux`
2. 对报文在应用层自行区分，同时 `Copy` 的部分也要做处理（ps. `yamux` 就是对报文做了处理） 
3. 最简单的方法，也是大多数内网转发工具用的方法，就是如果需要 `Copy` 就新建一个连接，简单有效
> 方法 3 可能存在的问题是，端口的连接总数是有限的，但是正常都足够的（只要实现上连接有正常 `Close`，在 Client 不是很多的情况下是没有太大问题的

方法 1 和 方法 3 是最适合的，而且 `yamux` 接入并不复杂，我选择方法 3 来实现（后续会抽空加上 `yamux` 支持

选择方法 3，因为 Server 端并不能新建通信连接，所以需要告诉 Client 新建连接，因为 Client 会 `Copy` `Local 3000` 流量到这个新建的连接上，所以对于「主分支」的 Server 来说，它需要判断是 `Forward` 还是 `Exchange` 消息，然后如果是 `Exchange`，需要**拿出**用户连接 `Copy` 到此 `Exchange` 消息的连接上。

所以步骤 3，Server 需要保存用户请求，创建对应的 `Connection UUID`，然后带上发送 `Exchange` 消息到 Client
步骤 5，Client 需要接收到 `Exchange` 消息，新建 Server 连接，然后首先发送带上同样 `UUID` 的 `Exchange` 消息到 Server，然后 `Copy` `Local 3000` 流量到此新建的 Server 连接上
步骤 6. Server 接收到 `Exchange` 消息，通过 `UUID` 取出对应的用户连接，然后 `Copy` 用户连接和此连接上

至此，用户访问 Server 端 9000 端口的「一个连接」的访问流程已经完成了
流程很简单，那么接下来，我会写一下每个流程的代码实现

## Codes

### Structure
结构大概是这样
```
.
├── client
│   ├── cfg.toml
│   ├── cmd.go
│   ├── config.go
│   └── serve.go
├── cmd
│   └── cmd.go
├── example
│   └── udp_forward
│       ├── README.md
│       └── echo.go
├── logger
│   └── log.go
├── main.go
├── pio
│   ├── encrypt.go
│   └── limit.go
├── proto
│   ├── error.go
│   ├── msg.go
│   └── packet.go
├── proxy
│   ├── buf.go
│   ├── proxy.go
│   └── udp.go
└── server
    ├── cfg.toml
    ├── cmd.go
    ├── config.go
    ├── conn
    │   ├── constant.go
    │   ├── tcp.go
    │   └── udp.go
    └── serve.go
```
简要描述下：
- cmd: `cmd` 入口，`cobra`
- proxy：对两个连接进行 `Copy` 的部分
- proto：`Control` 发送的消息结构体，以及序列化封装
- client：client 处理流程
- server：server 处理流程
- pio: `io.Reader` 和 `io.Writer` 的封装，实现限速（`Speed limit`）的功能

### Auth
因为打算通过 `Message` 发送来写一个流程，所以就不分别写 `Client` 和 `Server` 的结构体了

`Auth` 采用简单的 Token 校验，消息里有 `Token` 以及 `Timestamp` 字段，收到消息会 `md5(Token + Timestamp)` 进行校验（最开始我的实现 Client 和 Server 每个收发消息都会带上校验字段，好处是少一次 Auth 的发送时间，后来看到很多实现都只是在建立连接的时候校验，所以也改成连接创建时校验）

`Login message` 结构体
[proto/msg.go#L56](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/proto/msg.go#L56)
```go
type MsgLogin struct {
	Token     string `json:"token"`
	Version   string `json:"client_version"`
	Timestamp int64  `json:"timestamp"`
}
```

**Client** `Dial` 创建和 `Auth` 部分差不多是这样
[client/serve.go#L75](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/client/serve.go#L75)
```go
func authDialSvr(svraddr string, token string) (net.Conn, error) {
	conn, err := net.Dial("tcp", svraddr)
	if err != nil {
		return nil, err
	}

	if err = proto.Send(conn, proto.NewMsgLogin(token)); err != nil {
		return nil, err
	}

	return conn, nil
}
```

**Server 部分**
Server 会 `Listening 8910`，端口等待新的连接到来
[server/serve.go#L38-L62](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/server/serve.go#L38-L62)
```go
func (s *Server) Run() {
  ...
  listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		logger.Fatalf("Error listening: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Infof("Error accepting: %v", err)
			return
		}

		go s.handle(conn)
	}
}
```

Server handle 部分对 `MsgLogin` 进行校验，校验不通过直接断开连接
[server/serve.go#L64-L79](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/server/serve.go#L64-L79)
```go
func (s *Server) handle(conn net.Conn) {
	loginMsg := proto.MsgLogin{}
	if err := proto.Recv(conn, &loginMsg); err != nil {
		logger.Errorf("Error reading from connection: %v", err)
		conn.Close()
		return
	}

	hash := md5.New()
	hash.Write([]byte(s.cfg.Token + fmt.Sprintf("%d", loginMsg.Timestamp)))

	if fmt.Sprintf("%x", hash.Sum(nil)) != loginMsg.Token {
		logger.Errorf("Invalid token, client addr: %s", conn.RemoteAddr().String())
		conn.Close()
		return
	}
  ...
}
```
> 这里最开始是想的返回 `MsgLoginResp` 但是发现其实没有必要，直接断开也是可以的

这里用到 `proto.Send`，接下来会介绍 `message` 的实现
### Message
`TCP` 是 `Stream` 协议，我们并不知道需要读取多少字节，每个消息的长度也都是不固定的，所以需要实现自己的序列化规则
#### format
规定 `Message` 的 `format` 如下：

```
|<1 byte>|<2 byte>|<length byte>|
|PacketType|Length| Json Message|
```

#### pack/unpack
[proto/packet.go#L42](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/proto/packet.go#L42)
```go
func packet(typ PacketType, msg interface{}) ([]byte, error) {
	buf, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return packet0(typ, buf)
}

func packet0(typ PacketType, buf []byte) ([]byte, error) {
	if len(buf) > 65535 {
		return nil, ErrMsgLength
	}
	ret := make([]byte, 3+len(buf))
	ret[0] = byte(typ)
	ret[1] = byte(len(buf) >> 8)
	ret[2] = byte(len(buf))
	copy(ret[3:], buf)
	return ret, nil
}

func read(r io.Reader) (PacketType, []byte, error) {
	typ, buf, err := read0(r)
	if err != nil {
		return PacketUnknown, nil, err
	}
	return PacketType(typ), buf, nil
}

func read0(r io.Reader) (typ byte, buf []byte, err error) {
	buf = make([]byte, 1)
	_, err = r.Read(buf)
	if err != nil {
		return
	}

	typ = buf[0]

	buf = make([]byte, 2)
	_, err = r.Read(buf)
	if err != nil {
		err = ErrMsgRead
		return
	}
	l := int(buf[0])<<8 + int(buf[1])
	buf = make([]byte, l)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return
	}

	if n != l {
		err = ErrMsgLength
		return
	}

	return
}
```
#### send/recv
[proto/msg.go#L16](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/proto/msg.go#L16)
```go
func Send(w io.Writer, msg Msg) error {
	buf, err := packet(msg.Type(), msg)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

func Recv(r io.Reader, msg Msg) error {
	p, buf, err := read(r)
	if err != nil {
		return err
	}

	if p != msg.Type() {
		return ErrInvalidMsg
	}

	if err := json.Unmarshal(buf, msg); err != nil {
		return err
	}

	return nil
}
```

到这里 `Message` 的序列化和解析已经完成了，之后使用 `Msg` 或者添加新的 `Msg` 都不用关注这部分

### Forward
client 部分就是发送 `Forward` 消息，接收返回的 `ForwardResp`
[client/serve.go#L88](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/client/serve.go#L88)
```go
func (f *Forwarder) Run() {
	rConn, err := authDialSvr(f.svraddr, f.token)
	...

	if err = proto.Send(rConn, proto.NewMsgForward(f.proxyName, f.subdomain,
		f.proxyType, f.remotePort)); err != nil {

		f.logger.Fatalf("Error send forward msg to remote: %v", err)
	}

	frdResp := &proto.MsgForwardResp{}
	if err = proto.Recv(rConn, frdResp); err != nil {
		f.logger.Fatal("Error reading forward resp msg from remote, please check your config")
	}

	if frdResp.Status != "success" {
		f.logger.Fatalf("Forward failed, status: %s, remote port: %d", frdResp.Status, f.remotePort)
	}

	for {
    ...
  }
}
```
发送后，如果检验成功，Client 端会在 `for` 循环里接收来自 Server 端的消息

**Server 端处理 `Forward` 消息**
[server/serve.go#L83-L107](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/server/serve.go#L83-L107)
```go
	pt, buf, err := proto.Read(conn)
	if err != nil {
		logger.Errorf("Error reading from connection: %v", err)
		return
	}

	switch pt {
	case proto.PacketForwardReq:
		failChan := make(chan struct{})
		defer close(failChan)

		go func() {
			<-failChan
			if err := proto.Send(conn, proto.NewMsgForwardResp("", "failed")); err != nil {
				logger.Errorf("Error sending forward failed resp message: %v", err)
			}
		}()

		msg := &proto.MsgForwardReq{}
		if err := json.Unmarshal(buf, msg); err != nil {
			logger.Errorf("Error unmarshalling message: %v", err)
			return
		}

		s.handleForward(conn, msg, failChan)
  ...
  }
```

`handleForward` 函数
[server/serve.go#L133](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/server/serve.go#L133)
```go
func (s *Server) handleForward(cConn net.Conn, msg *proto.MsgForwardReq, failChan chan struct{}) {
	uPort := msg.RemotePort
	if !s.availablePort(uPort) {
		failChan <- struct{}{}
		return
	}
	from := cConn.RemoteAddr().String()
	switch msg.ProxyType {
  ... // udp
  case "tcp":
		uListener, err := net.Listen("tcp", fmt.Sprintf(":%d", uPort))
		if err != nil {
			failChan <- struct{}{}
			return
		}
		defer uListener.Close()

		s.addForward(Forward{
			To:           uPort,
			From:         from,
			Subdomain:    msg.Subdomain,
			listenCloser: uListener,
		})

		domain := fmt.Sprintf("%s.%s", msg.Subdomain, s.cfg.Domain)
		if !s.cfg.DomainTunnel {
			domain = ""
		}

		if err = proto.Send(cConn, proto.NewMsgForwardResp(domain, "success")); err != nil {
			failChan <- struct{}{}
			return
		}

		for {
			userConn, err := uListener.Accept()
			if err != nil {
				return
			}
			go func() {
				uid := conn.NewUuid()
				s.tcpConnMap.Add(uid, userConn)
				if err := proto.Send(cConn, proto.NewMsgExchange(uid, msg.ProxyType)); err != nil {
					logger.Errorf("Error sending exchange message: %v", err)
				}
			}()
		}
	}
}
```
大概流程就是：
1. check port available
2. 添加到 forwards 里（admin 展示, forward close）
3. 发送 `ForwardResp` 消息
4. 创建 `uListener` 并且等待用户连接
5. 收到用户连接，创建 `uuid`，发送 `Exchange` 消息

### Exchange

[client/serve.go#L124](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/client/serve.go#L124)
Client 端从发送 `Forward` 消息后的 `for` 里不断获取消息，然后如果是 `Exchange` 消息
```go
	for {
		p, buf, err := proto.Read(rConn)
		if err != nil {
			f.logger.Errorf("Error reading msg from remote: %v", err)
			return
		}

		nlogger := f.logger.CloneAdd(p.String())
		switch p {
    ... // heartbeat
		case proto.PacketExchange:
			msg := &proto.MsgExchange{}
			if err := json.Unmarshal(buf, msg); err != nil {
				cancelForward(f.token, f.svraddr, f.proxyName, f.localPort, f.remotePort)
				return
			}

			switch msg.ProxyType {
			case "tcp":
				go func() {
					nRconn, err := authDialSvr(f.svraddr, f.token)
					if err != nil {
						nlogger.Errorf("Error connecting to remote: %v", err)
						cancelForward(f.token, f.svraddr, f.proxyName, f.localPort, f.remotePort)
						return
					}
					if err = proto.Send(nRconn, proto.NewMsgExchange(msg.ConnId, f.proxyType)); err != nil {
						nlogger.Infof("Error sending exchange msg to remote: %v", err)
					}
					lConn, err := net.Dial(msg.ProxyType, fmt.Sprintf(":%d", f.localPort))
					if err != nil {
						nlogger.Errorf("Error connecting to local: %v, will close forward, %s:%d", err, f.proxyType, f.localPort)
						return
					}

					proxy.Stream(lConn, nRconn)
				}()
			}
    }
}
```
这里可以看到逻辑很简单
1. 接收到 `Exchange` 消息后
2. 创建新的 Server 连接
3. 根据 `ProxyType` 来创建 `Local` 连接
4. 调用 `proxy.Stream` 进行流量 `Copy`

Server 端接收到 `Exchange` 消息就很简单了，从 `tcpConnMap` 里拿出对应的连接，然后同样的 `proxy.Stream` 进行流量 `Copy`
[server/serve.go#L254](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/server/serve.go#L254)
```go
func (s *Server) handleExchange(conn net.Conn, msg *proto.MsgExchange) {
	switch msg.ProxyType {
  ... // udp
  case "tcp":
		uConn, ok := s.tcpConnMap.Get(msg.ConnId)
		if !ok {
			return
		}

		defer s.tcpConnMap.Del(msg.ConnId)
		proxy.Stream(conn, uConn)
	}
}
```
### Conclusion
到此，`Pipe` 的实现已经差不多，基本上列出了完整的流程，接下来会写下 `Pipe` 所实现的 `Feature`

## Feat
### Auto subdomain https
目标是实现一个自动 `Subdomain` 分配并且支持 `Https`
也就是加入 Server 运行在 `example.com` 机器，Client 开启转发 `Local 3000` 到 `Server 9000` 端口
Server 会生成 `xxx.example.com` 的 `Subdomain`，可以通过 `https://xxx.example.com` 来访问 Client `Local 3000`
因为不太想过于麻烦的实现 `Https`，所以借助 `Caddy` 来做 `Https` 的部分
代码在 [server/caddy_service.go#L22](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/server/caddy_service.go#L22)
借助 `Caddy` 的 `API` 添加 `https` 的 `route`
```go
func addCaddyRouter(host string, port int) {
	tunnelId := fmt.Sprintf("%s.%d", host, port)
	resp, err := http.Post(caddyAddRouteUrl, "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(caddyAddRouteF, tunnelId, host, port))))
	if err != nil {
		logger.Errorf("Tunnel creation failed, err: %v", err)
		return
	}
	defer resp.Body.Close()

	resp, err = http.Post(caddyAddTlsSubjectsUrl, "application/json", bytes.NewBuffer([]byte(fmt.Sprintf("\"%s\"", host))))
	if err != nil {
		logger.Errorf("Tunnel creation failed, err: %v", err)
		return
	}
	defer resp.Body.Close()
	logger.Infof("Tunnel created successfully, id: %s, host: %s", tunnelId, cr.PWhiteUnderline(host))
}
```
前置准备：
1. 要先设置好域名 `DNS` 解析，要设置两条记录 `A *.example.com <your server ip>` 和 `A example.com <your server ip>`
2. 运行 `Caddy`（如果是 `Cloudflare DNS` 还需要自己编译支持 `Cloudflare DNS plugin` 的 `Caddy` 版本，以及配置里填写 `Cloudflare KEY`，具体流程如有需要网上找下应该可以找到）
3. Server 端带上支持 `Subdomain` 的参数，可以看项目 `README.md`

### Deploy at `fly.io`
因为 `fly.io` 支持 `Dockerfile`，所以只用简单的写个 `Dockerfile` 即可
关键是 `fly.toml`
```toml
app = "pipefly"
primary_region = "hkg"

[build]

# Control
[[services]]
  internal_port = 8910
  protocol = "tcp"

  [[services.ports]]
    port = 8910
  
# Admin
[[services]]
  internal_port = 8911
  protocol = "tcp"

  [[services.ports]]
    handlers = ["http"]
    port = 80

  [[services.ports]]
    handlers = ["tls", "http"]
    port = 443

# Forward TCP
[[services]]
  internal_port = 9000
  protocol = "tcp"

  [[services.ports]]
    handlers = ["tls", "http"]
    port = 9000
```

`Control` 和 `Admin` 因为都是 `TCP`，所以 `protocol` 是 `tcp`，然后 `Admin` 希望直接从 <https://pipefly.fly.dev/> 访问，就需要加上 `handlers = ["http"]` 以及 `https` 的 `handlers = ["tls", "http"]`

然后这里需要在配置里指定出 `Forward` 的端口，这样运行 Server 和 Client 后
访问 <https://pipefly.fly.dev:9000> 就会访问到 Client `Local 3000` 了

> ps. `UDP` 的配置，`fly.io` 也是支持的，可以看 `fly` 的文档，或者可以看这个例子 [AnimMouse/frp-flyapp](https://github.com/AnimMouse/frp-flyapp)
### `UDP`
`UDP` 的支持，因为 `UDP` 没有连接的概念，只有 `Packet` 概念，所以我们可以「封装」`UDP` 流量为 `MsgUDPDatagram`，然后做流量的 `Copy`

[proxy/udp.go#L1-L77](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/proxy/udp.go#L1-L77)
```go
package proxy

import (
	"io"
	"net"
	"strings"

	"github.com/abcdlsj/pipe/logger"
	"github.com/abcdlsj/pipe/proto"
)

func UDPClientStream(token string, tcp, udp io.ReadWriteCloser) error {
	go func() {
		for {
			msg := proto.MsgUDPDatagram{}
			if err := proto.Recv(tcp, &msg); err != nil {
				return
			}
			n, err := udp.Write(msg.Payload)
			if err != nil {
				return
			}

			if n != len(msg.Payload) {
				return
			}
		}
	}()

	for {
		buf := make([]byte, 4096)
		n, err := udp.Read(buf)
		if err != nil {
			return err
		}
		if err = proto.Send(tcp, proto.NewMsgUDPDatagram(nil, buf[:n])); err != nil {
			return err
		}
	}
}
func UDPDatagram(token string, tcp io.ReadWriteCloser, udp *net.UDPConn) error {
	for {
		buf := make([]byte, 4096)
		n, addr, err := udp.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		if err = proto.Send(tcp, proto.NewMsgUDPDatagram(addr, buf[:n])); err != nil {
			return err
		}

		go func() {
			msg := proto.MsgUDPDatagram{}
			if err := proto.Recv(tcp, &msg); err != nil {
				return
			}
			_, err := udp.WriteTo(msg.Payload, addr)
			if err != nil {
				return
			}
		}()
	}
}
```

相当于 `TCP` 转发里的 `proxy.Stream` 替代

### Speed limit
得益于 `io.Reader` 和 `io.Writer` 接口，以及 `rate` 包，实现限速其实也很简单

```go
type LimitStream struct {
	rw       io.ReadWriteCloser
	ctx      context.Context
	wlimiter *rate.Limiter
	rlimiter *rate.Limiter
}

func NewLimitStream(rw io.ReadWriteCloser, limit int) *LimitStream {
	return &LimitStream{
		rw:       rw,
		ctx:      context.Background(),
		wlimiter: rate.NewLimiter(rate.Limit(limit), limit), // set burst = limit
		rlimiter: rate.NewLimiter(rate.Limit(limit), limit), // set burst = limit
	}
}

func (s *LimitStream) Read(p []byte) (int, error) {
	if s.rlimiter == nil {
		return s.rw.Read(p)
	}

	do := func(r *LimitStream, p []byte) (int, error) {
		n, err := r.rw.Read(p)
		if err != nil {
			return n, err
		}
		if err := r.rlimiter.WaitN(r.ctx, n); err != nil {
			return n, err
		}
		return n, nil
	}

	if len(p) < s.rlimiter.Burst() {
		return do(s, p)
	}

	burst := s.rlimiter.Burst()
	var read int
	for i := 0; i < len(p); i += burst {
		end := i + burst
		if end > len(p) {
			end = len(p)
		}

		n, err := do(s, p[i:end])
		read += n
		if err != nil {
			return read, err
		}
	}

	return read, nil
}
```
原理就是，假如说我们想要限速到 `10k/s`，那么就初始化 `burst=10k` 的 `rate.Limiter`
`Read` 的时候，调用 `WaitN`，因为容量为 `10k`，所以 `WaitN` 每读取 `10k byte` 就会等待 `1s`
这样就实现了 `10k/s` 的限速，而且使用上非常简单，初始化一个 `LimitStream` 就可以了

## Done
写了下如何实现一个内网转发的小工具，代码本身还有很多可以优化的地方，比如
1. 完善「错误处理」「重试」，对于哪些错误需要重试，哪些错误直接退出
2. 支持更多转发协议，例如 `HTTP/Quic/WebSocket`，`Control` 协议也可以支持更多，目前是 `TCP`，可以支持 `UDP/KCP` 等
3. 完善监控采集，这部分可以用 `Prometheus`，但是对于小项目来说太麻烦了
4. `Load balance` 这部分一直在思考如何做，从上边 `fly.io` 的部署就能知道，`Server` 端访问只能是单机的

写这个小项目后感觉到，项目是很难「维护」的，而且因为个人的局限性，代码一开始很难做出合理的「抽象」，导致后续有代码改动的时候会变很困难。之前写一些几百行的小项目还不觉得，现在这个项目代码量变多后，感觉代码「结构」、「接口」还是不够「清晰」。

感谢阅读！
## refs
<https://pandaychen.github.io/2020/01/01/MAGIC-GO-IO-PACKAGE/>
<https://github.com/ekzhang/bore>
<https://github.com/rapiz1/rathole>
<https://github.com/AnimMouse/frp-flyapp>