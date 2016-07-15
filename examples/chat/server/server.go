package main

import (
  "fmt"
  "runtime"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/chat"
  "github.com/leesper/holmes"
)

type ChatServer struct {
  tao.Server
}

func NewChatServer(addr string) *ChatServer {
  return &ChatServer {
    tao.NewTCPServer(addr),
  }
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  defer holmes.Start().Stop()

  tao.MessageMap.Register(chat.CHAT_MESSAGE, chat.DeserializeChatMessage)
  tao.HandlerMap.Register(chat.CHAT_MESSAGE, chat.ProcessChatMessage)

  chatServer := NewChatServer(fmt.Sprintf("%s:%d", "0.0.0.0", 18341))

  chatServer.SetOnConnectCallback(func(conn tao.Connection) bool {
    holmes.Info("%s", "On connect")
    return true
  })

  chatServer.SetOnErrorCallback(func() {
    holmes.Info("%s", "On error")
  })

  chatServer.SetOnCloseCallback(func(conn tao.Connection) {
    holmes.Info("%s", "Closing chat client")
  })

  chatServer.Start()
}
