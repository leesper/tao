package tao

import "syscall"

type socket int

func newSocket() (socket, error) {
	sfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return socket(-1), err
	}
	return socket(sfd), nil
}

func (sock socket) setNonblock() error {
	return syscall.SetNonblock(int(sock), true)
}

func (sock socket) setSockOpt(opts ...int) error {
	for _, opt := range opts {
		err := syscall.SetsockoptInt(int(sock), syscall.SOL_SOCKET, opt, 1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sock socket) bind(port int) error {
	serverAddr := &syscall.SockaddrInet4{
		Port: port,
		Addr: [4]byte{},
	}
	err := syscall.Bind(int(sock), serverAddr)
	return err
}

func (sock socket) listen() error {
	return syscall.Listen(int(sock), syscall.SOMAXCONN)
}
