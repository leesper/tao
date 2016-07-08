package main

import (
  "fmt"
  "runtime"
  "time"
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

  tao.MessageMap.Register(tao.HeartBeatMessage{}.MessageNumber(), tao.DeserializeHeartBeatMessage)
  tao.HandlerMap.Register(tao.HeartBeatMessage{}.MessageNumber(), tao.ProcessHeartBeatMessage)

  tao.MessageMap.Register(chat.ChatMessage{}.MessageNumber(), chat.DeserializeChatMessage)
  tao.HandlerMap.Register(chat.ChatMessage{}.MessageNumber(), chat.ProcessChatMessage)

  chatServer := NewChatServer(fmt.Sprintf("%s:%d", "0.0.0.0", 18341))
  defer chatServer.Close()

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

  heartBeatDuration := 5 * time.Second
  chatServer.SetOnScheduleCallback(heartBeatDuration, func(now time.Time, data interface{}) {
    cli := data.(tao.Connection)
    holmes.Info("Checking client %d at %s", cli.GetNetId(), time.Now())
    last := cli.GetHeartBeat()
    period := heartBeatDuration.Nanoseconds()
    if last < now.UnixNano() - 2 * period {
      holmes.Warn("Client %s netid %d timeout, close it\n", cli.GetName(), cli.GetNetId())
      cli.Close()
    }
  })

  chatServer.Start()
}
