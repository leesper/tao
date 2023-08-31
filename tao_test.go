package tao

import (
	"strings"
	"testing"
)

// parse server type and address
func TestShouldParseAddressIntoProtocolAndPort(t *testing.T) {
	proto, port, err := parseAddress("tcp:8341")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proto != "tcp" {
		t.Fatalf("expected tcp, got %s", proto)
	}
	if port != 8341 {
		t.Fatalf("expected 8341, got %d", port)
	}
}

func TestShouldReturnErrorWhenPortNotNumber(t *testing.T) {
	_, _, err := parseAddress("tcp:abcd")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestShouldReturnErrorIfProtoNotTcpUdpUnix(t *testing.T) {
	_, _, err := parseAddress("abc:1234")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "abc") {
		t.Fatalf("unsupported protocol expected in error message")
	}
}

func TestShouldParseAddressWhenProtoMixCased(t *testing.T) {
	proto, port, err := parseAddress("TcP:8341")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proto != "tcp" {
		t.Fatalf("expected tcp, got %s", proto)
	}
	if port != 8341 {
		t.Fatalf("expected 8341, got %d", port)
	}
}

// TODO: NewServer()*
// 	TODO: KQueuePoller
// 	TODO: EventLoop
// 	TODO: non-blocking Listener by socket-bind-listen
//  TODO: register listener's accept logic in EventLoop
// 		TODO: fd = listener.accept()
//  	TODO: conn = newConn(fd)
//  	TODO: server.add(fd, conn)
// 		TODO: register OnDataCallback with conn in EventLoop
//  TODO: EventLoop.loop(): kevent() --> handle events
