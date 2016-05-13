package main

import (
  "runtime"
  "log"
  "fmt"
  "time"
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
  chatServer.SetOnScheduleCallback(heartBeatDuration, func(now time.Time, data interface{}) {
    cli := data.(*tao.TCPConnection)
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
