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
  *tao.TCPServer
}

func NewEchoServer(addr string) *EchoServer {
  return &EchoServer {
    tao.NewTCPServer(addr),
  }
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())

  tao.MessageMap.Register(echo.EchoMessage{}.MessageNumber(), tao.UnmarshalFunctionType(echo.UnmarshalEchoMessage))
  tao.HandlerMap.Register(echo.EchoMessage{}.MessageNumber(), tao.NewHandlerFunctionType(echo.NewEchoMessageHandler))

  echoServer := NewEchoServer(":18342")
  defer echoServer.Close()

  echoServer.SetOnConnectCallback(func(client *tao.TCPConnection) bool {
    log.Printf("On connect\n")
    return true
  })

  echoServer.SetOnErrorCallback(func() {
    log.Printf("On error\n")
  })

  echoServer.SetOnCloseCallback(func(client *tao.TCPConnection) {
    log.Printf("Closing client\n")
  })

  echoServer.SetOnMessageCallback(func(msg tao.Message, client *tao.TCPConnection) {
    log.Printf("Receving message\n")
  })


  echoServer.Start(false)

}
