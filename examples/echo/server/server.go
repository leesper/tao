package main

import (
	"net"
	"runtime"

	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/echo"
)

// EchoServer represents the echo server.
type EchoServer struct {
	*tao.Server
}

// NewEchoServer returns an EchoServer.
func NewEchoServer() *EchoServer {
	onConnect := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		holmes.Info("%v", "On connect")
		return true
	})

	onClose := tao.OnCloseOption(func(conn tao.WriteCloser) {
		holmes.Info("%v", "Closing client")
	})

	onError := tao.OnErrorOption(func(conn tao.WriteCloser) {
		holmes.Info("%v", "On error")
	})

	onMessage := tao.OnMessageOption(func(msg tao.Message, conn tao.WriteCloser) {
		holmes.Info("%v", "Receving message")
	})

	return &EchoServer{
		tao.NewServer(onConnect, onClose, onError, onMessage),
	}
}

func main() {
	defer holmes.Start().Stop()

	runtime.GOMAXPROCS(runtime.NumCPU())

	tao.Register(echo.Message{}.MessageNumber(), echo.DeserializeMessage, echo.ProcessMessage)

	l, err := net.Listen("tcp", ":12345")
	if err != nil {
		holmes.Fatal("listen error %v", err)
	}
	echoServer := NewEchoServer()
	defer echoServer.Stop()

	echoServer.Start(l)
}
