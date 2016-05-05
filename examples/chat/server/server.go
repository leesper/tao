package main

import (
  "runtime"
  "log"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/chat"
)

func init() {
  log.SetFlags(log.Lshortfile | log.LstdFlags)
}

type ChatServer struct {
  *tao.TcpServer
}

func NewChatServer(addr string) *ChatServer {
  return &ChatServer {
    tao.NewTcpServer(addr),
  }
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())

  tao.MessageMap.Register(chat.ChatMessage{}.MessageNumber(), tao.UnmarshalFunctionType(chat.UnmarshalChatMessage))
  tao.HandlerMap.Register(chat.ChatMessage{}.MessageNumber(), tao.NewHandlerFunctionType(chat.NewChatMessageHandler))

  chatServer := NewChatServer(":18341")

  chatServer.SetOnConnectCallback(func(client *tao.TcpConnection) bool {
    log.Printf("On connect\n")
    return true
  })

  chatServer.SetOnErrorCallback(func() {
    log.Printf("On error\n")
  })

  chatServer.SetOnCloseCallback(func(client *tao.TcpConnection) {
    log.Printf("Closing chat client\n")
  })

  chatServer.Start(true)
}
