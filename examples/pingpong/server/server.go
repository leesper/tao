package main

import (
  "fmt"
  "runtime"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/pingpong"
  "github.com/leesper/holmes"
)

type PingPongServer struct {
  tao.Server
}

func NewPingPongServer(addr string) *PingPongServer {
  return &PingPongServer {
    tao.NewTCPServer(addr),
  }
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  defer holmes.Start().Stop()
  tao.MonitorOn(12345)

  tao.MessageMap.Register(pingpong.PINGPONG_MESSAGE, pingpong.DeserializePingPongMessage)
  tao.HandlerMap.Register(pingpong.PINGPONG_MESSAGE, ProcessPingPongMessage)

  server := NewPingPongServer(fmt.Sprintf("%s:%d", "0.0.0.0", 18341))

  server.SetOnConnectCallback(func(conn tao.Connection) bool {
    holmes.Info("%s", "On connect")
    return true
  })

  server.SetOnErrorCallback(func() {
    holmes.Info("%s", "On error")
  })

  server.SetOnCloseCallback(func(conn tao.Connection) {
    holmes.Info("%s", "Closing chat client")
  })

  server.Start()
}

func ProcessPingPongMessage(ctx tao.Context, conn tao.Connection) {
  // req := ctx.Message().(pingpong.PingPongMessage)
  // holmes.Infoln(req.Info)
  rsp := pingpong.PingPongMessage{
    Info: "pong",
  }
  conn.Write(rsp)
}
