package main

import (
	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/echo"
	"runtime"
)

type EchoServer struct {
	tao.Server
}

func NewEchoServer(addr string) *EchoServer {
	return &EchoServer{
		tao.NewTCPServer(addr),
	}
}

func main() {
	defer holmes.Start().Stop()

	runtime.GOMAXPROCS(runtime.NumCPU())

	tao.MessageMap.Register(echo.EchoMessage{}.MessageNumber(), echo.DeserializeEchoMessage)
	tao.HandlerMap.Register(echo.EchoMessage{}.MessageNumber(), echo.ProcessEchoMessage)

	echoServer := NewEchoServer(":18342")
	defer echoServer.Close()

	echoServer.SetOnConnectCallback(func(client tao.Connection) bool {
		holmes.Info("%v", "On connect")
		return true
	})

	echoServer.SetOnErrorCallback(func() {
		holmes.Info("%v", "On error")
	})

	echoServer.SetOnCloseCallback(func(client tao.Connection) {
		holmes.Info("%v", "Closing client")
	})

	echoServer.SetOnMessageCallback(func(msg tao.Message, client tao.Connection) {
		holmes.Info("%v", "Receving message")
	})

	echoServer.Start()
}
