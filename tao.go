package tao

import (
	"strconv"
	"strings"
)

type TcpServer struct {
	port int
}

func NewServer(addr string) (*TcpServer, error) {
	parts := strings.Split(addr, ":")
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	return &TcpServer{port}, nil
}

func (s *TcpServer) Port() int {
	return s.port
}

func (s *TcpServer) Close() {

}

func (s *TcpServer) Serve() {}

func (s *TcpServer) OnData(func(c *Conn, d *Buffer)) {}

type Conn struct{}
type Buffer struct{}
