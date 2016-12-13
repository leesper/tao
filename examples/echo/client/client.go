package main

import (
	"fmt"
	"github.com/reechou/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/echo"
	"net"
	"time"
)

func main() {
	tao.MessageMap.Register(echo.EchoMessage{}.MessageNumber(), echo.DeserializeEchoMessage)
	c, err := net.Dial("tcp", "127.0.0.1:18342")
	if err != nil {
		holmes.Fatal("%v", err)
	}

	tcpConnection := tao.NewClientConnection(0, false, c, nil)

	tcpConnection.SetOnConnectCallback(func(client tao.Connection) bool {
		holmes.Info("%v", "On connect")
		return true
	})

	tcpConnection.SetOnErrorCallback(func() {
		holmes.Info("%v", "On error")
	})

	tcpConnection.SetOnCloseCallback(func(client tao.Connection) {
		holmes.Info("%v", "On close")
	})

	tcpConnection.SetOnMessageCallback(func(msg tao.Message, c tao.Connection) {
		echoMessage := msg.(echo.EchoMessage)
		fmt.Printf("%s\n", echoMessage.Message)
	})

	echoMessage := echo.EchoMessage{
		Message: "hello, world",
	}

	tcpConnection.Start()

	for i := 0; i < 100; i++ {
		time.Sleep(600 * time.Millisecond)
		err := tcpConnection.Write(echoMessage)
		if err != nil {
			holmes.Error("%v", err)
		}
	}
	time.Sleep(time.Second)
	tcpConnection.Close()
}
