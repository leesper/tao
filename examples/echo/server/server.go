package main

import (
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"

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
		holmes.Infoln("on connect")
		return true
	})

	onClose := tao.OnCloseOption(func(conn tao.WriteCloser) {
		holmes.Infoln("closing client")
	})

	onError := tao.OnErrorOption(func(conn tao.WriteCloser) {
		holmes.Infoln("on error")
	})

	onMessage := tao.OnMessageOption(func(msg tao.Message, conn tao.WriteCloser) {
		holmes.Infoln("receving message")
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
		holmes.Fatalf("listen error %v", err)
	}
	echoServer := NewEchoServer()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		echoServer.Stop()
	}()

	echoServer.Start(l)
}
