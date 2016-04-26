package main

import (
  "runtime"
  "log"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/echo"
)

func init() {
  log.SetFlags(log.Lshortfile | log.LstdFlags)
}

type EchoServer struct {
  *tao.TcpServer
}

func NewEchoServer() *EchoServer {
  return &EchoServer {
    tao.NewTcpServer(),
  }
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())

  tao.MessageMap.Register(echo.EchoMessage{}.MessageNumber(), tao.UnmarshalFunctionType(echo.UnmarshalEchoMessage))
  tao.HandlerMap.Register(echo.EchoMessage{}.MessageNumber(), tao.NewHandlerFunctionType(echo.NewEchoMessageHandler))

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

  echoServer.SetOnMessageCallback(func(msg tao.Message, client *tao.TcpConnection) {
    log.Printf("Receving message\n")
  })

  echoServer.Start()
}
