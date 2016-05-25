Tao --- 轻量级TCP异步框架，Go语言实现
===========================================

## Light-weight TCP Asynchronous gOlang framework

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

### Chat Server Example

    package main

    import (
      "runtime"
      "log"
      "fmt"
      "github.com/leesper/tao"
      "github.com/leesper/tao/examples/chat"
    )

    func init() {
      log.SetFlags(log.Lshortfile | log.LstdFlags)
    }

    type ChatServer struct {
      *tao.TCPServer
    }

    func NewChatServer(addr string) *ChatServer {
      return &ChatServer {
        tao.NewTCPServer(addr),
      }
    }

    func main() {
      runtime.GOMAXPROCS(runtime.NumCPU())

      tao.MessageMap.Register(tao.DefaultHeartBeatMessage{}.MessageNumber(), tao.UnmarshalDefaultHeartBeatMessage)
      tao.HandlerMap.Register(tao.DefaultHeartBeatMessage{}.MessageNumber(), tao.NewDefaultHeartBeatMessageHandler)
      tao.MessageMap.Register(chat.ChatMessage{}.MessageNumber(), chat.UnmarshalChatMessage)
      tao.HandlerMap.Register(chat.ChatMessage{}.MessageNumber(), chat.NewChatMessageHandler)

      chatServer := NewChatServer(fmt.Sprintf("%s:%d", "0.0.0.0", 18341))
      defer chatServer.Close()

      chatServer.SetOnConnectCallback(func(client *tao.TCPConnection) bool {
        log.Printf("On connect\n")
        return true
      })

      chatServer.SetOnErrorCallback(func() {
        log.Printf("On error\n")
      })

      chatServer.SetOnCloseCallback(func(client *tao.TCPConnection) {
        log.Printf("Closing chat client\n")
      })

      heartBeatDuration := 5 * time.Second
      chatServer.SetOnScheduleCallback(heartBeatDuration, func(now time.Time, cli *tao.TCPConnection) {
        log.Printf("Checking client %d at %s", cli.NetId(), time.Now())
        last := cli.HeartBeat
        period := heartBeatDuration.Nanoseconds()
        if last < now.UnixNano() - 2 * period {
          log.Printf("Client %s netid %d timeout, close it\n", cli, cli.NetId())
          cli.Close()
        }
      })

      chatServer.Start()
    }



### TODO list:   
1.  [ ] Support Google flatbuffers;  
2.  [ ] Add more use-case examples;    
3.  [ ] Add logger support;
