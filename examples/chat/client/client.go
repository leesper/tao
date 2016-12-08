package main

import (
	"bufio"
	"fmt"
	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/chat"
	"net"
	"os"
)

func main() {
	tao.MessageMap.Register(chat.ChatMessage{}.MessageNumber(), chat.DeserializeChatMessage)

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

	tcpConnection.SetOnMessageCallback(func(msg tao.Message, client tao.Connection) {
		fmt.Print(msg.(chat.ChatMessage).Info)
	})

	tcpConnection.Start()
	for {
		reader := bufio.NewReader(os.Stdin)
		talk, _ := reader.ReadString('\n')
		if talk == "bye\n" {
			break
		} else {
			msg := chat.ChatMessage{
				Info: talk,
			}
			tcpConnection.Write(msg)
		}
	}
	tcpConnection.Close()
}
