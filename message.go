package tao

import (
  "encoding"
)

type Message interface {
  MessageNumber() int32
  encoding.BinaryMarshaler
}

type ProtocolHandler interface {
  Process(client *TcpConnection) bool
}

type NewHandlerFunctionType func(m Message) ProtocolHandler
type UnmarshalFunctionType func(data []byte) (message Message, err error)
type MessageMapType map[int32]UnmarshalFunctionType
type HandlerMapType map[int32]NewHandlerFunctionType

func (mm *MessageMapType) Register(msgType int32, unmarshaler UnmarshalFunctionType) {
  (*mm)[msgType] = unmarshaler
}

func (mm *MessageMapType) get(msgType int32) UnmarshalFunctionType {
  if unmarshaler, ok := (*mm)[msgType]; ok {
    return unmarshaler
  }
  return nil
}

var MessageMap MessageMapType
var HandlerMap HandlerMapType

func init() {
  MessageMap = make(MessageMapType)
  HandlerMap = make(HandlerMapType)
}

func (hm *HandlerMapType) Register(msgType int32, fn NewHandlerFunctionType) {
  (*hm)[msgType] = fn
}

func (hm *HandlerMapType) get(msgType int32) NewHandlerFunctionType {
  if fn, ok := (*hm)[msgType]; ok {
    return fn
  }
  return nil
}
