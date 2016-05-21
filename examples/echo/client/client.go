package main

import (
  "log"
  "time"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/echo"
)

func init() {
  log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
  tao.MessageMap.Register(echo.EchoMessage{}.MessageNumber(), echo.DeserializeEchoMessage)

  tcpConnection := tao.NewClientConnection(0, "127.0.0.1:18342", false)

  tcpConnection.SetOnConnectCallback(func(client tao.Connection) bool {
    log.Printf("On connect\n")
    return true
  })

  tcpConnection.SetOnErrorCallback(func() {
    log.Printf("On error\n")
  })

  tcpConnection.SetOnCloseCallback(func(client tao.Connection) {
    log.Printf("On close\n")
  })

  tcpConnection.SetOnMessageCallback(func(msg tao.Message, c tao.Connection) {
    echoMessage := msg.(echo.EchoMessage)
    log.Printf("%s\n", echoMessage.Message)
  })

  echoMessage := echo.EchoMessage{
    Message: "hello, world",
  }

  tcpConnection.RunAt(time.Now().Add(time.Second * 2), func(now time.Time, data interface{}) {
    cli := data.(tao.Connection)
    log.Println("Closing after 2 seconds")
    cli.Close()
  })

  tcpConnection.Start()

  for i := 0; i < 3; i++ {
    err := tcpConnection.Write(echoMessage)
    if err != nil {
      log.Println(err)
    }
  }
  time.Sleep(time.Second)
  tcpConnection.Close()
}
