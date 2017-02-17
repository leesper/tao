package main

import (
	"context"
	"net"
	"runtime"

	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/pingpong"
)

// PingPongServer defines pingpong server.
type PingPongServer struct {
	*tao.Server
}

// NewPingPongServer returns PingPongServer.
func NewPingPongServer() *PingPongServer {
	onConnect := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		holmes.Info("%s", "On connect")
		return true
	})

	onError := tao.OnErrorOption(func(conn tao.WriteCloser) {
		holmes.Info("%s", "On error")
	})

	onClose := tao.OnCloseOption(func(conn tao.WriteCloser) {
		holmes.Info("%s", "Closing pingpong client")
	})

	return &PingPongServer{
		tao.NewServer(onConnect, onError, onClose),
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer holmes.Start().Stop()
	tao.MonitorOn(12345)
	tao.Register(pingpong.PingPontMessage, pingpong.DeserializeMessage, ProcessPingPongMessage)

	l, err := net.Listen("tcp", ":12346")
	if err != nil {
		holmes.Fatal("listen error", err)
	}

	server := NewPingPongServer()

	server.Start(l)
}

// ProcessPingPongMessage handles business logic.
func ProcessPingPongMessage(ctx context.Context, conn tao.WriteCloser) {
	ping := ctx.Value(tao.MessageCtx).(pingpong.Message)
	holmes.Infoln(ping.Info)
	rsp := pingpong.Message{
		Info: "pong",
	}
	conn.Write(rsp)
}
