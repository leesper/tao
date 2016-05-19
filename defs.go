package tao

import (
  "errors"
)

var (
  ErrorParameter error = errors.New("Parameter error")
  ErrorNilKey error = errors.New("Nil key")
  ErrorNilValue error = errors.New("Nil value")
  ErrorWouldBlock error = errors.New("Would block")
  ErrorNotHashable error = errors.New("Not hashable")
  ErrorNilData error = errors.New("Nil data")
  ErrorIllegalData error = errors.New("More than 8M data")
  ErrorUndefined error = errors.New("Undefined message")
  ErrorNotImplemented error = errors.New("Not implemented")
)

const (
  WORKERS = 10
  MAX_CONNECTIONS = 1000
)
