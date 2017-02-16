package tao

import (
	"errors"
	"fmt"
)

// ErrUndefined for undefined message type.
type ErrUndefined int32

func (e ErrUndefined) Error() string {
	return fmt.Sprintf("undefined message type %d", e)
}

var (
	// ErrParameter for parameter error.
	ErrParameter = errors.New("parameter error")
	// ErrNilKey for nil key.
	ErrNilKey = errors.New("nil key")
	// ErrNilValue for nil value.
	ErrNilValue = errors.New("nil value")
	// ErrWouldBlock for opertion may be blocked.
	ErrWouldBlock = errors.New("would block")
	// ErrNotHashable for type not hashable.
	ErrNotHashable = errors.New("not hashable")
	// ErrNilData for nil data.
	ErrNilData = errors.New("nil data")
	// ErrBadData for more than 8M data.
	ErrBadData = errors.New("more than 8M data")
	// ErrNotRegistered for message handler not registered.
	ErrNotRegistered = errors.New("handler not registered")
	// ErrServerClose for connection closed by server.
	ErrServerClosed = errors.New("server has been closed")
)

const (
	// WorkersNum is the number of worker go-routines.
	WorkersNum = 20
	// MaxConnections is the maximum number of client connections allowed.
	MaxConnections = 1000
)
