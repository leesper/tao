package tao

import "syscall"

type socket struct {
	fd       int
	readable bool
}

func newSocket() (*socket, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return nil, err
	}
	return &socket{fd, false}, nil
}

func (sock *socket) setNonblock() error {
	return syscall.SetNonblock(sock.fd, true)
}

func (sock *socket) setSockOpt(opts ...int) error {
	for _, opt := range opts {
		err := syscall.SetsockoptInt(sock.fd, syscall.SOL_SOCKET, opt, 1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sock *socket) bind(port int) error {
	serverAddr := &syscall.SockaddrInet4{
		Port: port,
		Addr: [4]byte{},
	}
	err := syscall.Bind(sock.fd, serverAddr)
	return err
}

func (sock *socket) listen() error {
	return syscall.Listen(sock.fd, syscall.SOMAXCONN)
}
