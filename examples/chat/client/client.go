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
	defer holmes.Start().Stop()

	tao.Register(chat.ChatMessage, chat.DeserializeMessage, nil)

	c, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		holmes.Fatalln(err)
	}

	onConnect := tao.OnConnectOption(func(c tao.WriteCloser) bool {
		holmes.Infoln("on connect")
		return true
	})

	onError := tao.OnErrorOption(func(c tao.WriteCloser) {
		holmes.Infoln("on error")
	})

	onClose := tao.OnCloseOption(func(c tao.WriteCloser) {
		holmes.Infoln("on close")
	})

	onMessage := tao.OnMessageOption(func(msg tao.Message, c tao.WriteCloser) {
		fmt.Print(msg.(chat.Message).Content)
	})

	options := []tao.ServerOption{
		onConnect,
		onError,
		onClose,
		onMessage,
		tao.ReconnectOption(),
	}

	conn := tao.NewClientConn(0, c, options...)
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
			if err := conn.Write(msg); err != nil {
				holmes.Infoln("error", err)
			}
		}
	}
	fmt.Println("goodbye")
}
