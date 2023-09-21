package tao

import (
	"strconv"
	"strings"
	"syscall"
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

func (s *TcpServer) Serve() {
	serverSock, err := newServerSocket(s.port)
	if err != nil {
		panic(err)
	}

	kfd, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}

	events := []syscall.Kevent_t{
		{
			Ident:  uint64(serverSock.fd),
			Filter: syscall.EVFILT_READ,
			Flags:  syscall.EV_ADD,
		},
	}

	_, err = syscall.Kevent(kfd, events, nil, nil)
	if err != nil {
		panic(err)
	}

	// wait forever
	_, err = syscall.Kevent(kfd, nil, events, nil)
	if err != nil {
		panic(err)
	}

	for _, event := range events {
		if event.Filter == syscall.EVFILT_READ {
			_, _, err = syscall.Accept(int(event.Ident))
			if err != nil {
				panic(err)
			}
		}
	}
}
