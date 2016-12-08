package tao

import (
	"errors"
	"fmt"
)

var (
	ErrorParameter      error = errors.New("Parameter error")
	ErrorNilKey         error = errors.New("Nil key")
	ErrorNilValue       error = errors.New("Nil value")
	ErrorWouldBlock     error = errors.New("Would block")
	ErrorNotHashable    error = errors.New("Not hashable")
	ErrorNilData        error = errors.New("Nil data")
	ErrorIllegalData    error = errors.New("More than 8M data")
	ErrorNotImplemented error = errors.New("Not implemented")
	ErrorConnClosed     error = errors.New("Connection closed")
)

const (
	WORKERS         = 20
	MAX_CONNECTIONS = 1000
)

func Undefined(msgType int32) error {
	return ErrorUndefined{
		msgType: msgType,
	}
}

type ErrorUndefined struct {
	msgType int32
}

func (eu ErrorUndefined) Error() string {
	return fmt.Sprintf("Undefined message %d", eu.msgType)
}
