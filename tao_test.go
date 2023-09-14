package tao

import "testing"

// example 1: tcp echo server
// echoServer, _ := tao.NewServer("tcp:8341").
// 	OnData(func(conn *tao.Conn, data *tao.Buffer) {
// 	    logger.Info("receiving %s from %s", data.String(), conn.RemoteAddr())
// 		conn.Write(data)
//  })
// echoServer.Serve()

// TcpServer
// - TODO: should return tcp server with proto:port specified
func TestShouldReturnTcpServerWithProtoPortSpecified(t *testing.T) {
	server, err := NewServer("tcp:1234")
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if port := server.Port(); port != 1234 {
		t.Fatalf("expected 1234, got %d", port)
	}
}

// - TODO: should return error if port not a number
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
	if server.Port() != 1234 {
		t.Fatalf("expected 8341, got %d", server.Port())
	}
}

// - TODO: should callback when data received
// - TODO: should wait and accept new connection when serve
// - TODO: should read data from client and echoed back
// - TODO: should shutdown server gracefully when ctrl+c pressed
// Conn
// - TODO: should write Bufferred bytes data to client
// - TODO: should return remote addr of client
// Buffer
// - TODO: should return data in string form
