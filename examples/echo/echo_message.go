package echo

import (
  "errors"
)

var ErrorNilData error = errors.New("Nil data")

type EchoMessage struct {
  Message string
}

func (em EchoMessage) MarshalBinary() ([]byte, error) {
  return []byte(em.Message), nil
}

func (em EchoMessage) UnmarshalBinary(data []byte) error {
  if data == nil {
    return ErrorNilData
  }
  msg := string(data)
  em.Message = msg
  return nil
}
