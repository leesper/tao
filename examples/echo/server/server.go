package main

import (
  "runtime"
  "github.com/golang/glog"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/echo"
)

type EchoServer struct {
  tao.Server
}

func NewEchoServer(addr string) *EchoServer {
  return &EchoServer {
    tao.NewTCPServer(addr),
  }
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())

  tao.MessageMap.Register(echo.EchoMessage{}.MessageNumber(), echo.DeserializeEchoMessage)
  tao.HandlerMap.Register(echo.EchoMessage{}.MessageNumber(), echo.NewEchoMessageHandler)

  echoServer := NewEchoServer(":18342")
  defer echoServer.Close()

  echoServer.SetOnConnectCallback(func(client tao.Connection) bool {
    glog.Infoln("On connect")
    return true
  })

  echoServer.SetOnErrorCallback(func() {
    glog.Infoln("On error")
  })

  echoServer.SetOnCloseCallback(func(client tao.Connection) {
    glog.Infoln("Closing client")
  })

  echoServer.SetOnMessageCallback(func(msg tao.Message, client tao.Connection) {
    glog.Infoln("Receving message")
  })

  echoServer.Start()
}
