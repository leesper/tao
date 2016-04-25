package main

import (
  "runtime"
  "log"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/echo"
)

type EchoServer struct {
  *tao.TcpServer
}

func NewEchoServer() *EchoServer {
  return &EchoServer {
    tao.NewTcpServer()
  }
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  echoServer := NewEchoServer()
  defer echoServer.Close()
  echoServer.SetOnConnectCallback(func() bool {
    log.Printf("On connect\n")
    return true
  })

  echoServer.SetOnErrorCallback(func() {
    log.Printf("On error\n")
  })

  echoServer.SetOnCloseCallback(func(client *tao.TcpConnection) {
    log.Printf("Closing client\n")
  })

  echoServer.SetOnMessageCallback(func(msg *tao.Message, client *tao.TcpConnection) {
    echoMessage := msg.(echo.EchoMessage)
    log.Printf("Receving message %s\n", echoMessage.message)
    client.Write(msg)
  })

  echoServer.Start()
}
