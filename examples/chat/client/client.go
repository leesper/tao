package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/chat"
)

func main() {
	tao.Register(chat.ChatMessage, chat.DeserializeMessage, nil)

	c, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		holmes.Fatal("%v", err)
	}

	onConnect := tao.OnConnectOption(func(c tao.WriteCloser) bool {
		holmes.Info("%s", "On connect")
		return true
	})

	onError := tao.OnErrorOption(func(c tao.WriteCloser) {
		holmes.Info("%s", "On error")
	})

	onClose := tao.OnCloseOption(func(c tao.WriteCloser) {
		holmes.Info("On close")
		os.Exit(0)
	})

	onMessage := tao.OnMessageOption(func(msg tao.Message, c tao.WriteCloser) {
		fmt.Print(msg.(chat.Message).Content)
	})

	conn := tao.NewClientConn(0, c, onConnect, onError, onClose, onMessage)
	defer conn.Close()

	conn.Start()
	for {
		reader := bufio.NewReader(os.Stdin)
		talk, _ := reader.ReadString('\n')
		if talk == "bye\n" {
			break
		} else {
			msg := chat.Message{
				Content: talk,
			}
			conn.Write(msg)
		}
	}
}
