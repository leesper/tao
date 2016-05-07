Tao --- 轻量级TCP异步框架，Go语言实现
===========================================

## Light-weight TCP Asynchronous gOlang framework

Features
--------
1. 完全异步的读，写以及消息处理 Completely asynchronous reading, writing and message handling;
2. 负载均衡的工作者协程池 Load-balanced worker go-routine pool;
3. 并发和原子数据结构 Concurrent data structure and atomic data types;
4. 毫秒精度的定时器功能 Millisecond-precision timer function;
5. 传输层安全支持 Transport layer security support;
6. 应用层心跳协议 Application-level heart-beating protocol;

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
        tao.NewTCPServer(addr, false),
      }
    }

    func main() {
      runtime.GOMAXPROCS(runtime.NumCPU())

      tao.MessageMap.Register(chat.ChatMessage{}.MessageNumber(), tao.UnmarshalFunctionType(chat.UnmarshalChatMessage))
      tao.HandlerMap.Register(chat.ChatMessage{}.MessageNumber(), tao.NewHandlerFunctionType(chat.NewChatMessageHandler))

      chatServer := NewChatServer(fmt.Sprintf("%s:%d", tao.ServerConf.IP, tao.ServerConf.Port))

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

      chatServer.Start(true)
    }

### Chat Message Example

    package chat
    import (
      "errors"
      "github.com/leesper/tao"
    )

    var ErrorNilData error = errors.New("Nil data")

    type ChatMessage struct {
      Info string
    }

    func (cm ChatMessage) MarshalBinary() ([]byte, error) {
      return []byte(cm.Info), nil
    }

    func (cm ChatMessage) MessageNumber() int32 {
      return 1
    }

    func UnmarshalChatMessage(data []byte) (message tao.Message, err error) {
      if data == nil {
        return nil, ErrorNilData
      }
      info := string(data)
      msg := ChatMessage{
        Info: info,
      }
      return msg, nil
    }

    type ChatMessageHandler struct {
      message tao.Message
    }

    func NewChatMessageHandler(msg tao.Message) tao.MessageHandler {
      return ChatMessageHandler{
        message: msg,
      }
    }

    func (handler ChatMessageHandler) Process(client *tao.TCPConnection) bool {
      if client.Owner != nil {
        connections := client.Owner.GetAllConnections()
        for v := range connections.IterValues() {
          c := v.(*tao.TCPConnection)
          c.Write(handler.message)
        }
        return true
      }
      return false
    }


### TODO list:  
1.  [x] Make it configurable by JSON;    
2.  [ ] Support Google flatbuffers;  
3.  [ ] Add more use-case examples;     
