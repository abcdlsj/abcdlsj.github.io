---
title: "Pipe - Build a tunnel tool like frp/ngrok"
date: 2023-11-04T22:47:37+08:00
tags:
  - Network
  - Weekly
  - Tunnel
hide: true
---
## Background
**A simple forwarding tool**
The communication between services in the company's internal service framework is realized by connecting the `UNIX domain` on which each machine's `Agent` listens, and in the company's container cluster environment, the `Agent` is started.
But as a local environment, there is no `Agent` condition, so the local startup service is to use `socat` to forward the local `UNIX domain` traffic to the remote `TCP agent` to start the service (there is nothing special about the remote `TCP agent`, the traffic is also forwarded to the machine `Agent`).
We usually use `socat -d -d -d UNIX-LISTEN:/tmp/xxx.sock,reuseaddr,fork TCP:agent-tcp.xxxx.io:9299`.

> `socat` is a swiss army knife type tool, very powerful.

Then I thought that since the principle is so simple, it should be very simple to implement a forwarding tool with similar functionality. With the help of `io.Copy()` and the `net` package, I implemented the code in a short time, the total amount of code is less than 50 lines, but it's very practical and very fast to start up (even slightly faster than `socat` in real life).

**Remote Port Forwarding**
Later, I thought I could implement a tool similar to `frp` and `ngrok`, and the core code is not too far from a simple forwarding tool, so I started to write it.
The realization of the process did not draw too much code from other projects, many places are encountered problems and then go to check, so it is worthwhile to write a blog to record, there has been no idea to write, here to record the ideas and core code.
Code is the beginning of the year or the end of last year to write the first version, after changing some, now and the initial version compared to some complexity.

> The first version only realized `TCP` forwarding, and contained `Caddy` to do `Auto Subdomain https`, less than `1000` lines of code. It was optimized to support `TCP/UDP` protocols, so this article only covers `TCP/UDP` implementation (but most other protocols are similar).
> Incidentally, `GitHub` has many similar implementations, such as [ekzhang/bore](https://github.com/ekzhang/bore) and [rapiz1/rathole](https://github.com/rapiz1/rathole/) (`Tokio` is very powerful). Tokio` is so powerful that I couldn't resist rewriting it in `Rust` :P.

All the code is in [abcdlsj/pipe](https://github.com/abcdlsj/pipe/tree/484084da8b9edb99fb39e5d7561cc94d16d7031c) (version at the time of writing)

## How it works

Implementing **Remote Port Forwarding**

Suppose you have a Server and a Client. The IP of the Server can be accessed directly from the public network, while the IP of the Client cannot, and the Client can access the Server.

We would like to have a way to establish an association between the Server port and the Client port, and forward access to the Server port to the corresponding port on the Client, so that accessing the Server's port through the public network is equivalent to accessing the Client's port.

**How to implement this forwarding?**
First of all, the Server side should communicate with the Client side, for the inbound request from the Server side, the request and the Client side will be **binding**, and the Client will be **binding** to the target port and the communication connection of the Server side. **Bind** means `io.Copy()` for both connections.

Let's assume that our Server communication port is 8910, and we want to penetrate the Client's port 3000 to the Server's port 9000.
That's pretty much the final structure:

```d2
Flow: {
  server: {
    remoteport 9000
  }
  client: {
    localport 3000
  }
  client <-> server: 1. Create control connection, auth, send request forward...
  user -> server: 2. View remote port 9000
  server -> server: 3. io.Copy() control connection 8910 and user connection
  server -> client: 4. Send start proxy request
  client -> client: 5. io.Copy() control connection 8910 and local connection
}
```

Some things to think about are omitted here, such as:
1. how is the `Auth/Handshake` part of the connection established between Client and Server "authenticated"?
2. is the message length in the `Control` connection fixed? If not, how to deal with the "boundary" problem?
3. when the Client receives a `Proxy` request, which connection does `io.Copy()` and how does the Server handle it?

> For 1 and 2, you can see the detailed implementation below.
> For 3, if you haven't implemented a similar tool, you may not know why this problem exists, but you can see the detailed flow below (ignore the "Authentication" part).

1. first start the Server, "listening" on port 8910
2. start Client, Client and Server establish `Control` connection, and then send a `Forward` interface to tell Server to forward to port 9000. 3.
3. Server receives the `Forward` message from the `Control` connection and starts listening on port 9000, ready to receive requests from the client.
> Does Server want to `Copy' the user connection and the `Control' connection at this point?
> The answer is no, because the `Control` connection will also have "other" traffic from the Server or Client, such as `Close`, `Heartbeat` messages, etc., which can cause problems if it is `Copied` directly to the user connection. 4.
4. when a new user request arrives, the Server sends an `Exchange` message over the `Control` connection to tell the Client that there is a new user connection, and that it is ready to `Copy` the traffic. 5. the Client receives a `Copy` message over the `Control` connection, and the Client receives a `Close` message.
5. the Client receives the `Exchange` message, establishes a connection to port `Local 3000` and prepares to `Copy` the traffic.
> The Client can't directly `Copy` the `Control` connection and the `Local 3000` connection either, as in 3.

So we have the "connection multiplexing" problem, which is very common when dealing with multiple flows, and because of the direct `io.Copy` there is no way to differentiate between the flows.
There are many ways to solve this problem:
1. you can use connection multiplexing libraries such as [hashicorp/yamux](https://github.com/hashicorp/yamux), `frp` uses `yamux` by default.
2. differentiate messages at the application level, and process the `Copy` part as well (ps. `yamux` does the processing) 
3. the simplest method, and the one used by most intranet forwarding tools, is to create a new connection if a `Copy` is needed, simple and effective.
> The problem with method 3 is that the total number of connections to a port is limited, but normally it is enough (as long as the connection is properly `Close`, it is not a problem if there are not too many clients).

Method 1 and method 3 are the most suitable, and `yamux` access is not complicated, I choose method 3 to realize (will take time to add `yamux` support later)

After choosing method 3, because the Server side can't create a new communication connection, it needs to tell the Client to create a new connection, because the Client will `Copy` `Local 3000` traffic to this new connection, so for the Server in the `Main Branch`, it needs to determine whether it is a `Forward` or `Exchange` message, and then if it is an `Exchange` message, it needs to determine if it is a `Exchange` message. Then if it is an `Exchange`, it needs to **take out** the user connection `Copy` to this `Exchange` message connection.

So in step 3, Server needs to save the user request, create the corresponding `Connection UUID`, and then send the `Exchange` message to Client with it.
Step 5. Client receives the `Exchange` message, creates a new Server connection, and then first sends the `Exchange` message with the same `UUID` to Server, and then `Copy` the `Local 3000` traffic to this new Server connection.
Server receives the `Exchange` message, fetches the corresponding user connection by `UUID`, and then `Copy`s the user connection traffic to this connection.

At this point, the process of accessing the user's "one connection" on port 9000 on the Server side is complete.
The process is very simple, so next, I will write the code implementation of each flow

## Codes

### Structure
The structure is roughly like this
```
.
├── client
│   ├── cfg.toml
│   ├── cmd.go
│   ├── config.go
│   └── serve.go
├── cmd
│   └── cmd.go
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
A brief description:
- cmd: `cmd` entry, `cobra`.
- proxy: the part that `Copy`s the two connections
- proto: the message structure sent by `Control`, and the serialization wrapper.
- client: Client process
- server: Server process
- pio: Wrapper for `io.Reader` and `io.Writer` that implements the `Speed limit` functionality.

### Auth
Since we're going to bring up the main body of the code through `Message` sending, we won't write separate structures for `Client` and `Server`.

`Auth` uses a simple token validation, the message has a `Token` and a `Timestamp` field, and the received message will be verified by `md5(Token + Timestamp)`. (At first, my implementation of `Client` and `Server` would carry the validation field in every message sent and received, which has the advantage of reducing the time of sending the Auth once, but then I saw a lot of implementations that just set up the `Message` to bring out the main code. Later, I saw that many implementations only check when the connection is established, so I changed it to check when the connection is created.)

`MsgLogin` structure
[proto/msg.go#L56](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/proto/msg.go#L56)
```go
type MsgLogin struct {
	Token     string `json:"token"`
	Version   string `json:"client_version"`
	Timestamp int64  `json:"timestamp"`
}
```

**Client** The `Dial` creation and `Auth` part is more or less like this
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

There are other interesting implementations of the `Auth` part, such as `OpenID Connect (OIDC)`, which allows you to do things like `Sweep' authentication.
**Server part**
Server will `Listen` to port `8910` and wait for the Client connection to arrive (`8910` is used by default).
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

Server handle part checks `MsgLogin` and disconnects if it fails.
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
> At first, this was intended to return `MsgLoginResp`, but it turns out that there's no need to do that, and it's fine to just disconnect it.

Send` is used here, and the implementation of `message` is described next.
### Send' is used here, and the implementation of `message` is described next.
`TCP` is a `Stream` protocol, we don't know how many bytes we need to read, and the length of each message is variable, so we need to implement our own serialization rules.
#### format
The `format` of `Message` is specified as follows:

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

Now that the serialization and parsing of `Message` is complete, you don't need to worry about using `Msg` or adding new `Msg`s.

### Forward
The client part sends the `Forward` message and receives the returned `ForwardResp`.
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
After sending, if the check is successful, the Client receives the message from the Server in a `for` loop.

**Server-side processing of `Forward` messages**
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
  }
```

`handleForward` function
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
  case "tcp":
		uListener, err := net.Listen("tcp", fmt.Sprintf(":%d", uPort))
		if err != nil {
			failChan <- struct{}{}
			return
		}
		defer uListener.Close()
    
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
The approximate process is:
1. check port available
3. send `ForwardResp` message
4. create `uListener` and wait for user connection
5. receive user connection, create `uuid`, send `Exchange` message

### Exchange

The Client keeps getting messages from the `for` after sending a `Forward` message, and then if it's an `Exchange` message, the Client gets the message from the `Forward` message.
[client/serve.go#L124](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/client/serve.go#L124)
```go
	for {
		p, buf, err := proto.Read(rConn)
		if err != nil {
			f.logger.Errorf("Error reading msg from remote: %v", err)
			return
		}

		nlogger := f.logger.CloneAdd(p.String())
		switch p {
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
Here you can see that the logic is simple
1. after receiving the `Exchange` message
2. create a new Server connection
3. create a `Local` connection based on the `ProxyType`. 4. call `proxy.Stream` to `Copy` the traffic.
4. call `proxy.Stream` to `Copy` the traffic.

The Server side receives the `Exchange` message and it's a simple matter of pulling the corresponding connection out of the `tcpConnMap` and then doing the same `proxy.Stream` for the traffic `Copy`.

[server/serve.go#L254](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/server/serve.go#L254)
```go
func (s *Server) handleExchange(conn net.Conn, msg *proto.MsgExchange) {
	switch msg.ProxyType {
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
At this point, the implementation of `Pipe` is almost complete, and basically lists the complete process, next will write the `Feature` implemented by `Pipe`.

## Feat
## Auto subdomain https
The goal is to implement an automatic `Subdomain` assignment with `Https` support.
That is, add Server running on the `example.com` machine, and Client enables forwarding of `Local 3000` to `Server 9000` port.
Server generates a `Subdomain` of `xxx.example.com`, which can access Client `Local 3000` via `https://xxx.example.com`.
Since I don't want to go through the trouble of implementing `Https`, I'll use `Caddy` to do the `Https` part.
The code is in [server/caddy_service.go#L22](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/server/caddy_ service.go#L22)

Adding an `https` `route` with `Caddy`'s `API`.
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
Preparation:
1. To set up DNS resolution, set up two records, `A *.example.com <your server ip>` and `A example.com <your server ip>`.
2. Run `Caddy` (for `Cloudflare DNS`, you also need to compile your own version of `Caddy` that supports `Cloudflare DNS plugin`, and fill in `Cloudflare KEY` in the configuration, the specific procedure should be available if you need to look for it on the Internet).
3. Server side with parameters that support `Subdomain`, you can see the project `README.md`.

### Deploy at `fly.io`
**The important thing here is that only 1 Service can be deployed.**
> Can't I deploy more than one?
> No, because if you deploy more than one, `fly` will do `Load Balancing`, so some user requests can't be `Copied` because they're not in the `tcpConnMap` (`yamux` may be able to solve this problem).

Since `fly.io` supports `Dockerfile`, you can simply write a `Dockerfile`.
The key is `fly.toml`.
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

`Control` and `Admin` have a `tcp` `protocol` since they are both `TCP`, and then `Admin` wants to access it directly from <https://pipefly.fly.dev/>, so they need to add `handlers = ["http"]` as well as `https` `handlers = ["tls", "http"]`. handlers = ["tls", "http"]`

Then you need to specify the port for `Forward` in the configuration so that when you run Server and Client
After running Server and Client, you can access <https://pipefly.fly.dev:9000> to access Client `Local 3000`.

> The `UDP` configuration is also supported by `fly.io`, see the `fly` documentation, or see this example [AnimMouse/frp-flyapp](https://github.com/AnimMouse/frp-flyapp)


### `UDP`
UDP` support, since `UDP` does not have the concept of connection, only `Packet`, we can "encapsulate" the `UDP` traffic as `MsgUDPDatagram`, and then do a `Copy` of the traffic.
[proxy/udp.go#L1-L77](https://github.com/abcdlsj/pipe/blob/484084da8b9edb99fb39e5d7561cc94d16d7031c/proxy/udp.go#L1-L77)
```go
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
Equivalent to `proxy.Stream` in `TCP` forwarding instead.

### Speed limit
Thanks to the `io.Reader` and `io.Writer` interfaces, as well as the `rate` package, it's really easy to implement speed limits.

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
The principle is that if we want to limit the speed to `10k/s`, then initialize `rate.Limiter` with `burst=10k`.
When `Read`, call `WaitN`, since the capacity is `10k`, `WaitN` waits `1s` for every `10k byte` read.
This achieves the `10k/s` speed limit and is very simple to use, just initialize a `LimitStream` and you're done!

## Done
After writing about how to implement an intranet forwarding widget, there are still a lot of things that can be optimized in the code itself, such as
1. improve the "error handling" and "retry", for which errors need to retry, and for which errors to exit directly
2. support more forwarding protocols, such as `HTTP/Quic/WebSocket`, `Control` protocol can also support more, currently `TCP`, can support `UDP/KCP` and so on.
3. Improve the monitoring collection, this part can use `Prometheus`, but for small projects is too much trouble.
4. `Load Balancing` This part has been thinking about how to do, from the deployment of `fly.io` above, we can see that the `Server` side can only be accessed by a single machine.

After writing this small project, I realized that it is hard to maintain the project, and because of my personal limitations, it is hard to make a reasonable abstraction of the code at the beginning, which makes it difficult to make subsequent changes to the code. Before I wrote some small projects with hundreds of lines, I didn't feel it, but now after the amount of code in this project has increased, I feel that the code "structure" and "interface" are still not "clear" enough.

Thanks for reading!
## refs
<https://pandaychen.github.io/2020/01/01/MAGIC-GO-IO-PACKAGE/>
<https://github.com/ekzhang/bore>
<https://github.com/rapiz1/rathole>
<https://github.com/AnimMouse/frp-flyapp>