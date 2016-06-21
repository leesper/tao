package main

import (
  "fmt"
  "bufio"
  "os"
  "time"
  "github.com/golang/glog"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/chat"
)

func main() {
  tao.MessageMap.Register(chat.ChatMessage{}.MessageNumber(), chat.DeserializeChatMessage)

  tcpConnection := tao.NewClientConnection(0, "127.0.0.1:18341", false)
  defer tcpConnection.Close()

  tcpConnection.SetOnConnectCallback(func(client tao.Connection) bool {
    glog.Infoln("On connect")
    return true
  })

  tcpConnection.SetOnErrorCallback(func() {
    glog.Infoln("On error")
  })

  tcpConnection.SetOnCloseCallback(func(client tao.Connection) {
    glog.Infoln("On close")
    os.Exit(0)
  })

  tcpConnection.SetOnMessageCallback(func(msg tao.Message, client tao.Connection) {
    fmt.Print(msg.(chat.ChatMessage).Info)
  })

  heartBeatDuration := 5 * time.Second
  tcpConnection.RunEvery(heartBeatDuration, func(now time.Time, data interface{}) {
    cli := data.(tao.Connection)
    msg := tao.DefaultHeartBeatMessage {
      Timestamp: now.UnixNano(),
    }
    glog.Infof("Sending heart beat at %s, timestamp %d\n", now, msg.Timestamp)
    cli.Write(msg)
  })

  tcpConnection.Start()
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
