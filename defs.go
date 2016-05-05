package tao

import (
  "errors"
  "time"
)

var (
  ErrorParameter error = errors.New("Parameter error")
  ErrorNilKey error = errors.New("Nil key")
  ErrorNilValue error = errors.New("Nil value")
  ErrorWouldBlock error = errors.New("Would block")
  ErrorNotHashable error = errors.New("Not hashable")
  ErrorNilData error = errors.New("Nil data")
)

const (
  HEART_BEAT_PERIOD = 10 * time.Second
)
