package main

import (
	"fmt"
	"net"
	"runtime"

	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/chat"
)

// ChatServer is the chatting server.
type ChatServer struct {
	*tao.Server
}

// NewChatServer returns a ChatServer.
func NewChatServer() *ChatServer {
	onConnectOption := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		holmes.Info("%s", "On connect")
		return true
	})
	onErrorOption := tao.OnErrorOption(func(conn tao.WriteCloser) {
		holmes.Info("%s", "On error")
	})
	onCloseOption := tao.OnCloseOption(func(conn tao.WriteCloser) {
		holmes.Info("%s", "Closing chat client")
	})
	return &ChatServer{
		tao.NewServer(onConnectOption, onErrorOption, onCloseOption),
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer holmes.Start().Stop()

	tao.Register(chat.ChatMessage, chat.DeserializeMessage, chat.ProcessMessage)

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "0.0.0.0", 12345))
	if err != nil {
		holmes.Fatal("listen error", err)
	}
	chatServer := NewChatServer()
	err = chatServer.Start(l)
	if err != nil {
		holmes.Fatal("start error %v", err)
	}
}
