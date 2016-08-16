package main

import (
  "os"
  "net"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/pingpong"
  "github.com/leesper/holmes"
)

var (
  rspChan = make(chan string)
)

func main() {
  defer holmes.Start().Stop()
  tao.MessageMap.Register(pingpong.PINGPONG_MESSAGE, pingpong.DeserializePingPongMessage)
  tao.HandlerMap.Register(pingpong.PINGPONG_MESSAGE, ProcessPingPongMessage)

  c, err := net.Dial("tcp", "127.0.0.1:18341")
  if err != nil {
    holmes.Fatal("%v", err)
  }

  tcpConnection := tao.NewClientConnection(0, false, c, nil)
  defer tcpConnection.Close()

  tcpConnection.SetOnConnectCallback(func(client tao.Connection) bool {
    holmes.Info("%s", "On connect")
    return true
  })

  tcpConnection.SetOnErrorCallback(func() {
    holmes.Info("%s", "On error")
  })

  tcpConnection.SetOnCloseCallback(func(client tao.Connection) {
    holmes.Info("On close")
    os.Exit(0)
  })

  tcpConnection.Start()
  req := pingpong.PingPongMessage{
    Info: "ping",
  }
  for {
      tcpConnection.Write(req)
      // holmes.Info(<-rspChan)
      <-rspChan
  }
  tcpConnection.Close()
}

func ProcessPingPongMessage(ctx tao.Context, conn tao.Connection) {
  rsp := ctx.Message().(pingpong.PingPongMessage)
  rspChan<- rsp.Info
}
