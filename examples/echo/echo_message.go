package echo

import (
  "encoding/binary"
  "bytes"
  "errors"
  "github.com/leesper/tao"
)

var ErrorNilData error = errors.New("Nil data")

type EchoMessage {
  message string
}

func (em *EchoMessage) MarshalBinary() ([]byte, error) {
  return []bytes(em.message), nil
}

func (em *EchoMessage) UnmarshalBinary(data []byte) error {
  if data == nil {
    return ErrorNilData
  }
  msg := string(data)
  em.message = msg
  return nil
}
