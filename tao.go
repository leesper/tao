package tao

import (
	"strconv"
	"strings"
)

type TcpServer struct {
	port int
}

func (s *TcpServer) Port() int {
	return s.port
}

func NewServer(addr string) (*TcpServer, error) {
	parts := strings.Split(addr, ":")
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	return &TcpServer{port}, nil
}
