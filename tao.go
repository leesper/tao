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
	sock, err := newSocket()
	if err != nil {
		panic(err)
	}

	err = sock.setNonblock()
	if err != nil {
		panic(err)
	}

	err = sock.setSockOpt(syscall.SO_REUSEADDR, syscall.SO_REUSEPORT)
	if err != nil {
		panic(err)
	}

	err = sock.bind(s.port)
	if err != nil {
		panic(err)
	}

	err = sock.listen()
	if err != nil {
		panic(err)
	}

	// acceptor := newAcceptor(sock)
	// poller := newKqueuePoller()
	// poller.addHandler(acceptor)
	//   map[socket]Handler
	//   interface Handler handle() error
	// handlers := poller.poll()
	// for _, handler := range handlers {
	//   handler.handle()
	// }

	kfd, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}

	events := []syscall.Kevent_t{
		{
			Ident:  uint64(sock),
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
