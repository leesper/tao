package main

import (
	"github.com/leesper/tao"
)

func main() {
	echoServer := tao.NewServer("tcp:8341").
		OnData(func(conn *tao.Conn, data *tao.Buffer) {
			conn.Write(data)
		})
	echoServer.Run()
}
