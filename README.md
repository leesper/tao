Tao --- 轻量级TCP异步框架，Go语言实现
===========================================

## Light-weight TCP Asynchronous gOlang framework

1.1 Announcing Tao 1.1 - Release Notes
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

1.0 Announcing Tao 1.0 - Release Notes
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
