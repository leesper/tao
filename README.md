Tao --- 轻量级TCP异步框架，Go语言实现 1.5.5
===========================================

## Light-weight TCP Asynchronous gOlang framework

Announcing Tao 1.5 - Release Notes
--------
1. A Golang-style redesigning of the overall framework, a reduce about 500+ lines of codes;<br/>
按照Go语言风格重新设计的整体框架，精简500多行代码；
2. Providing new Server, ClientConn and ServerConn struct and a WriteCloser interface;<br/>
提供Server，ClientConn和ServerConn三种新结构和WriteCloser新接口；
3. Using standard context package to manage and spread request-scoped data acrossing go-routines;<br/>
使用标准库中的context包在多个Go线程中管理和传播与请求有关的数据；
4. Graceful stopping, all go-routines are related by context, and they will be noticed and exit when server stops or connection closes;<br/>
优雅停机，所有的Go线程都通过上下文进行关联，当服务器停机或连接关闭时它们都会收到通知并执行退出；
5. Providing new type HandlerFunc func(context.Context, WriteCloser) for defining message handlers;<br/>
提供新的HandlerFunc类型来定义消息处理器；
6. Developers can now use NewContextWithMessage() and MessageFromContext() to put and get message they are about to use in handler function's context, this also leads to a more clarified design;<br/>
开发者现在可以通过NewContextWithMessage()和MessageFromContext()函数来在上下文中存取他们将在处理器函数中使用的消息，这样的设计更简洁；
7. Go-routine functions readLoop(), writeLoop() and handleLoop() are all optimized to serve both ServerConn and ClientConn, serveral dead-lock bugs such as blocking on channels are fixed;<br/>
优化Go线程函数readLoop()，writeLoop()和handleLoop()使得它们能同时为ServerConn和ClientConn服务，修复了多个“通道阻塞”的死锁问题；
8. Reconnecting mechanism of ClientConn is redesigned and optimized;<br/>
重新设计和优化ClientConn的断线重连机制；

Announcing Tao 1.4 - Release Notes
--------
1. bugfix：TLS重连失败问题；<br/>
bugfix: failed to reconnect the TLS connection;
2. bugfix：ConnectionMap死锁问题；<br/>
bugfix: ConnectionMap dead-lock problem;
3. 优化TCP网络连接的关闭过程；<br/>
Optimize the closing process of TCP connection;
4. 优化服务器的关闭过程；<br/>
Optimize the closing process of server;
5. 更优雅的消息处理注册接口；<br/>
More elegant message handler register interface;

Announcing Tao 1.3 - Release Notes
--------
1. bugfix：修复断线重连状态不一致问题；<br/>
bugfix: fixed inconsistent status caused by reconnecting;
2. bugfix：修复ServerConnection和TimingWheel在连接关闭时并发访问导致崩溃问题；<br/>
bugfix: fixed a corruption caused by concurrent accessing between ServerConnection and TimingWheel during connection closing;
3. 无锁且线程安全的TimingWheel，优化CPU占用率；<br/>
Lock-free and thread-safe TimingWheel, optimized occupancy rate;<br/>
4. bugfix：修复TLS配置文件读取函数；<br/>
bugfix: Fixed errors when loading TLS config;
5. 添加消息相关的Context结构；简化消息注册机制，直接注册处理函数到HandlerMap；<br/>
A message-related Context struct added; Register handler functions in HandlerMap directly to simplify message registration mechanism;
6. 合并NewClientConnection()和NewTLSClientConnection()，提供一致的API；<br/>
Combine NewTLSConnection() into NewClientConnection(), providing a consistent API;
7. 工作者线程池改造成单例模式；<br/>
Make WorkerPool a singleton pattern;
8. 使用Holmes日志库代替glog；<br/>
Using Holmes logging package instead of glog;
9. 添加metrics.go：基于expvar标准包导出服务器关键信息；<br/>
Add metrics.go: exporting critical server information based on expvar standard pacakge;
10. 编写中文版框架设计原理文档，英文版正在翻译中；<br/>
A document about framework designing principles in Chinese, English version under developed;

Announcing Tao 1.2 - Release Notes
--------
1. 更优雅的消息注册接口；<br/>
More elegant message register interface;
2. TCPConnection的断线重连机制；<br/>
TCPConnection reconnecting upon closing;
3. bugfix：协议未注册时不关闭客户端连接；<br/>
bugfix: Don't close client when messages not registered;
4. bugfix：在readLoop()协程中处理心跳时间戳更新；<br/>
bugfix: Updating heart-beat timestamp in readLoop() go-routine;
5. bugfix：Message接口使用Serialize()替代之前的MarshalBinary()，以免框架使用者使用gob.Encoder/Decoder的时候栈溢出；<br/>
bugfix: Use Serialize() instead of MarshalBinary() in Message interface, preventing stack overflows when framework users use gob.Encoder/Decoder;
6. bugfix：当应用层数据长度大于0时才对其进行序列化；<br/>
bugfix: Serialize application data when its length greater than 0;
7. 新API：SetCodec()，允许TCPConnection自定义编解码器；<br/>
New API: SetCodec() allowing TCPConnection defines its own codec;
8. 新API：SetDBInitializer()，允许框架使用者定义数据访问接口；<br/>
New API: SetDBInitializer() allowing framework users define data access interface;
9. 允许框架使用者在TCPConnection上设置自定义数据；<br/>
Allowing framework users setting custom data on TCPConnection;
10. 为新客户端连接的启动单独开辟一个对应的go协程；<br/>
Allocating a corresponding go-routine for newly-connected clients respectively;
11. bugfix：写事件循环在连接关闭时将信道中的数据全部发送出去；<br/>
bugfix: writeLoop() flushes all packets left in channel when performing closing;
12. bugfix：服务器和客户端连接等待所有go协程关闭后再退出；<br/>
bugfix: Servers and client connections wait for the exits of all go-routines before shutting down;
13. 重构Server和Connection，采用针对接口编程的设计；<br/>
Refactoring Server and Connection, adopting a programming-by-interface design;
14. 设置500毫秒读超时，防止readLoop()发生阻塞；<br/>
Setting 500ms read-timeout prevents readLoop() from blocking;

Announcing Tao 1.1 - Release Notes
--------
1. 添加注释，提高代码可读性；<br/>
Add comments, make it more readable;
2. 限制服务器的最大并发连接数（默认1000）；<br/>
Server max connections limit (default to 1000);
3. 新API：NewTLSTCPServer() 创建传输层安全的TCP服务器；<br/>
New API: NewTLSTCPServer() for creating TLS-supported TCP server;
4. 新特性：SetOnScheduleCallback() 由框架使用者来定义计划任务（比如心跳）；<br/>
New Feature: SetOnScheduleCallback() make scheduled task managed by framwork users(such as heart beat);
5. 新特性：支持默认的消息编解码器TypeLengthValueCodec，并允许框架使用者开发自定义编解码器； <br/>
Support TypeLengthValueCodec by default, while allowing framework users develop  their own codecs;

Announcing Tao 1.0 - Release Notes
--------
1. 完全异步的读，写以及消息处理；<br/>
Completely asynchronous reading, writing and message handling;
2. 工作者协程池；<br/>
Worker go-routine pool;
3. 并发数据结构和原子数据类型；<br/>
Concurrent data structure and atomic data types;
4. 毫秒精度的定时器功能；<br/>
Millisecond-precision timer function;
5. 传输层安全支持；<br/>
Transport layer security support;
6. 应用层心跳协议；<br/>
Application-level heart-beating protocol;

### Documentation
1. [Tao - Go语言实现的TCP网络编程框架](http://www.jianshu.com/p/c322edca985f)
2. [Tao Framework API Reference](https://github.com/leesper/tao/blob/master/DOCUMENTATION.md)
3. English(TBD)

### Chat Server Example

```go
package main

import (
	"fmt"
	"net"

	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/chat"
)

// ChatServer is the chatting server.
type ChatServer struct {
	*tao.Server
}

// NewChatServer returns a ChatServer.
func NewChatServer() *ChatServer {
	onConnectOption := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		holmes.Infoln("on connect")
		return true
	})
	onErrorOption := tao.OnErrorOption(func(conn tao.WriteCloser) {
		holmes.Infoln("on error")
	})
	onCloseOption := tao.OnCloseOption(func(conn tao.WriteCloser) {
		holmes.Infoln("close chat client")
	})
	return &ChatServer{
		tao.NewServer(onConnectOption, onErrorOption, onCloseOption),
	}
}

func main() {
	defer holmes.Start().Stop()

	tao.Register(chat.ChatMessage, chat.DeserializeMessage, chat.ProcessMessage)

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "0.0.0.0", 12345))
	if err != nil {
		holmes.Fatalln("listen error", err)
	}
	chatServer := NewChatServer()
	err = chatServer.Start(l)
	if err != nil {
		holmes.Fatalln("start error", err)
	}
}
```
