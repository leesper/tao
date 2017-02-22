package main

import (
	"fmt"
	"net"
	"time"

	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/echo"
)

func main() {
	tao.Register(echo.Message{}.MessageNumber(), echo.DeserializeMessage, nil)

	c, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		holmes.Fatalln(err)
	}

	onConnect := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		holmes.Infoln("on connect")
		return true
	})

	onError := tao.OnErrorOption(func(conn tao.WriteCloser) {
		holmes.Infoln("on error")
	})

	onClose := tao.OnCloseOption(func(conn tao.WriteCloser) {
		holmes.Infoln("on close")
	})

	onMessage := tao.OnMessageOption(func(msg tao.Message, conn tao.WriteCloser) {
		echo := msg.(echo.Message)
		fmt.Printf("%s\n", echo.Content)
	})

	conn := tao.NewClientConn(0, c, onConnect, onError, onClose, onMessage)

	echo := echo.Message{
		Content: "hello, world",
	}

	conn.Start()

	for i := 0; i < 10; i++ {
		time.Sleep(60 * time.Millisecond)
		err := conn.Write(echo)
		if err != nil {
			holmes.Errorln(err)
		}
	}
	holmes.Debugln("hello")
	conn.Close()
}
