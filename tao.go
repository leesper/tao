package tao

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	supportedProtos = map[string]bool{
		"tcp":  true,
		"udp":  true,
		"unix": true,
	}
)

func isSupportedProto(proto string) bool {
	_, ok := supportedProtos[proto]
	return ok
}

func parseAddress(address string) (proto string, port int, err error) {
	parts := strings.Split(address, ":")
	proto = strings.ToLower(parts[0])
	if !isSupportedProto(proto) {
		return proto, port, newErrUnsupportedProto(proto)
	}
	port, err = strconv.Atoi(parts[1])
	return proto, port, err
}

type ErrUnsupportedProto struct {
	proto string
}

func (e ErrUnsupportedProto) Error() string {
	return fmt.Sprintf("unsupported protocol %s", e.proto)
}

func newErrUnsupportedProto(proto string) ErrUnsupportedProto {
	return ErrUnsupportedProto{proto: proto}
}
