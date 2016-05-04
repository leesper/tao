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
)
