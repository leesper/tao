package tao

import (
	"testing"
)

const (
	Port = 1234
	Addr = "tcp:1234"
)

// example 1: tcp echo server
// echoServer, _ := tao.NewServer("tcp:8341").
// 	OnData(func(conn *tao.Conn, data *tao.Buffer) {
// 	    logger.Info("receiving %s from %s", data.String(), conn.RemoteAddr())
// 		conn.Write(data)
//  })
// echoServer.Serve()

// TcpServer
func TestShouldReturnTcpServerWithProtoPortSpecified(t *testing.T) {
	server, err := NewServer(Addr)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if server.Port() != Port {
		t.Fatalf("expected %d, got %d", Port, server.Port())
	}
}

func TestShouldReturnErrorIfPortNotNumber(t *testing.T) {
	_, err := NewServer("tcp:abcd")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestShouldReturnTcpServerIfProtoOmitted(t *testing.T) {
	server, err := NewServer(":1234")
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if server.Port() != Port {
		t.Fatalf("expected %d, got %d", Port, server.Port())
	}
}

// - TODO: should call OnData callback when data received
// func TestShouldCallOnDataCallbackWhenDataReceived(t *testing.T) {
// 	server, err := NewServer(":1234")
// 	if err != nil {
// 		t.Fatalf("unexpected error %v", err)
// 	}

// 	called := false
// 	server.OnData(func(conn *Conn, data *Buffer) {
// 		called = true
// 		server.Close()
// 	})

// 	server.Serve()

// 	if !called {
// 		t.Fatalf("expected true, got %v", called)
// 	}
// }

// - TODO: should wait and accept new connection when serve
// - TODO: should read data from client and echoed back
// - TODO: should shutdown server gracefully when ctrl+c pressed
// Conn
// - TODO: should write Bufferred bytes data to client
// - TODO: should return remote addr of client
// Buffer
// - TODO: should return data in string form
