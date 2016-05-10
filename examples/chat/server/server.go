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

  chatServer.Start(false)
}
