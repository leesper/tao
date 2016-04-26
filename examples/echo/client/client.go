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

  serverAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:18341")
  if err != nil {
    log.Fatalln(err)
  }

  tcpConn, err := net.DialTCP("tcp", nil, serverAddr)
  if err != nil {
    log.Fatalln(err)
  }

  tcpConnection := tao.NewTcpConnection(nil, tcpConn)

  tcpConnection.SetOnConnectCallback(func() bool {
    log.Printf("On connect\n")
    return true
  })

  tcpConnection.SetOnErrorCallback(func() {
    log.Printf("On error\n")
  })

  tcpConnection.SetOnCloseCallback(func(client *tao.TcpConnection) {
    log.Printf("On close\n")
  })

  tcpConnection.SetOnMessageCallback(func(msg tao.Message, client *tao.TcpConnection) {
    echoMessage := msg.(echo.EchoMessage)
    log.Printf("%s\n", echoMessage.Message)
  })

  echoMessage := echo.EchoMessage{
    Message: "hello, world",
  }

  tcpConnection.Do()

  for i := 0; i < 3; i++ {
    err = tcpConnection.Write(echoMessage)
    if err != nil {
      log.Println(err)
    }
  }
  time.Sleep(5 * time.Second)
}
