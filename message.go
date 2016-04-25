package tao

import (
  "encoding"
)

type Message interface {
  encoding.BinaryMarshaler
  encoding.BinaryUnmarshaler
}

type MessageMapType map[int32]*Message

func (mm *MessageMapType) Put(msgType int32, msg *Message) {
  mm[msgType] = msg
}

func (mm *MessageMapType) Get(msgType int32) *Message {
  if msg, ok := mm[msgType]; ok {
    return msg
  }
  return nil
}

var MessageMapper MessageMapType
var HandlerMapper HandlerMapType
func init() {
  messageMap = make(MessageMapType)
  handlerMap = make(HandlerMapType)
}

type ProtocolHandler interface {
  Process(client *TcpConnection) bool
}

type NewHandlerFunctionType func(m *Message)*ProtocolHandler
type HandlerMapType map[int32]NewHandlerFunctionType

func (hm *HandlerMapType) Put(msgType int32, fn NewHandlerFunctionType) {
  hm[msgType] = fn
}

func (hm *HandlerMapType) Get(msgType int32) NewHandlerFunctionType {
  if fn, ok := hm[msgType]; ok {
    return fn
  }
  return nil
}
