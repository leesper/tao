package main

import (
  "log"
  "fmt"
  "bufio"
  "os"
  "time"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/chat"
)

func init() {
  log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
  tao.MessageMap.Register(chat.ChatMessage{}.MessageNumber(), tao.UnmarshalFunctionType(chat.UnmarshalChatMessage))

  tcpConnection := tao.ClientTCPConnection(0, "127.0.0.1:18341", tao.NewTimingWheel())
  defer tcpConnection.Close()

  tcpConnection.SetOnConnectCallback(func(client *tao.TCPConnection) bool {
    log.Printf("On connect\n")
    return true
  })

  tcpConnection.SetOnErrorCallback(func() {
    log.Printf("On error\n")
  })

  tcpConnection.SetOnCloseCallback(func(client *tao.TCPConnection) {
    log.Printf("On close\n")
    os.Exit(0)
  })

  tcpConnection.SetOnMessageCallback(func(msg tao.Message, client *tao.TCPConnection) {
    fmt.Print(msg.(chat.ChatMessage).Info)
  })

  heartBeatDuration := 5 * time.Second
  tcpConnection.RunEvery(heartBeatDuration, func(now time.Time, data interface{}) {
    cli := data.(*tao.TCPConnection)
    msg := tao.DefaultHeartBeatMessage {
      Timestamp: now.UnixNano(),
    }
    log.Printf("Sending heart beat at %s, timestamp %d\n", now, msg.Timestamp)
    cli.Write(msg)
  })

  tcpConnection.Do()

  for {
    reader := bufio.NewReader(os.Stdin)
    talk, _ := reader.ReadString('\n')
    if talk == "bye\n" {
      break
    } else {
      msg := chat.ChatMessage{
        Info: talk,
      }
      tcpConnection.Write(msg)
    }
  }
  tcpConnection.Close()
}
