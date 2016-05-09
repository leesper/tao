package main

import (
  "log"
  "net"
  "time"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/echo"
)

func init() {
  log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
  tao.MessageMap.Register(echo.EchoMessage{}.MessageNumber(), tao.UnmarshalFunctionType(echo.UnmarshalEchoMessage))

  serverAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:18342")
  if err != nil {
    log.Fatalln(err)
  }

  tcpConn, err := net.DialTCP("tcp", nil, serverAddr)
  if err != nil {
    log.Fatalln(err)
  }

  tcpConnection := tao.ClientTCPConnection(0, tcpConn, tao.NewTimingWheel(), false)

  tcpConnection.SetOnConnectCallback(func(client *tao.TCPConnection) bool {
    log.Printf("On connect\n")
    return true
  })

  tcpConnection.SetOnErrorCallback(func() {
    log.Printf("On error\n")
  })

  tcpConnection.SetOnCloseCallback(func(client *tao.TCPConnection) {
    log.Printf("On close\n")
  })

  tcpConnection.SetOnMessageCallback(func(msg tao.Message, client *tao.TCPConnection) {
    echoMessage := msg.(echo.EchoMessage)
    log.Printf("%s\n", echoMessage.Message)
  })

  echoMessage := echo.EchoMessage{
    Message: "hello, world",
  }

  tcpConnection.RunAt(time.Now().Add(time.Second * 2), func(now time.Time) {
    log.Println("Closing after 2 seconds")
    tcpConnection.Close()
  })

  tcpConnection.Do()

  for i := 0; i < 3; i++ {
    err = tcpConnection.Write(echoMessage)
    if err != nil {
      log.Println(err)
    }
  }
  tcpConnection.Wait()

}
