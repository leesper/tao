package tao

import (
	"strconv"
	"strings"
	"syscall"
)

var (
	addrAny = [4]byte{}
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

type ioEvent int32

const (
	eventRead ioEvent = iota
	eventWrite
)

type acceptor struct {
	sock *socket
}

type handler interface {
	handle(e ioEvent) error
}

func newAcceptor(port int) *acceptor {
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

	err = sock.bind(port)
	if err != nil {
		panic(err)
	}

	err = sock.listen()
	if err != nil {
		panic(err)
	}

	return &acceptor{sock}
}

func (a *acceptor) handle(e ioEvent) error {
	if e == eventRead {
		_, _, err := syscall.Accept(a.sock.fd)
		if err != nil {
			return err
		}
	}
	return nil
}

type kqueuePoller struct {
	kfd      int
	handlers map[socket]handler
}

func newKqueuePoller() (*kqueuePoller, error) {
	kfd, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	handlers := make(map[socket]handler)
	return &kqueuePoller{kfd, handlers}, nil
}

func (p *kqueuePoller) addHandler(sock socket, h handler) {
	p.handlers[sock] = h
}

func (p *kqueuePoller) handler(sock socket) handler {
	return p.handlers[sock]
}

// func (p *kqueuePoller) poll() []event {
// 	events := []event{}

// }

type event struct {
	ident socket
	typ   ioEvent
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

	// sock.enableRead()
	// acceptor := newAcceptor(sock)
	// poller := newKqueuePoller()
	// poller.addHandler(acceptor->sock, acceptor)
	//   map[socket]Handler
	//   interface Handler handle() error
	// events := poller.poll()
	// for _, event := range events {
	//   poller.handler(event.sock).handle(event.value)
	// }

	kfd, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}

	events := []syscall.Kevent_t{
		{
			Ident:  uint64(sock.fd),
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
