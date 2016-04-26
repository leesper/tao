package main

import (
  "log"
  "net"
  "fmt"
  "github.com/leesper/tao/examples/echo"
)

func main() {
  serverAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:18341")
  if err != nil {
    log.Fatalln(err)
  }

  tcpConn, err := net.DialTCP("tcp", nil, serverAddr)
  if err != nil {
    log.Fatalln(err)
  }

  echoMessage := echo.EchoMessage{
    Message: "hello, world",
  }

  for i := 0; i < 3; i++ {
    _, err = tcpConn.Write(echoMessage)
    if err != nil {
      log.Println(err)
    }


  }
}
