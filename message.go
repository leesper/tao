package tao

import (
  "encoding"
)

type Message interface {
  encoding.BinaryMarshaler
  encoding.BinaryUnmarshaler
}

type MessageMap map[int32]*Message

func (mm *MessageMap) put(msgType int32, msg *Message) {
  mm[msgType] = msg
}

func (mm *MessageMap) get(msgType int32) *Message {
  if msg, ok := mm[msgType]; ok {
    return msg
  }
  return nil
}

var messageMap MessageMap
var handlerMap HandlerMap
func init() {
  messageMap = make(MessageMap)
  handlerMap = make(HandlerMap)
}

type ProtocolHandler interface {
  Process(client *TcpConnection) bool
}

type NewHandlerFunc func(m *Message)*ProtocolHandler
type HandlerMap map[int32]NewHandlerFunc

func (hm *HandlerMap) put(msgType int32, fn NewHandlerFunc) {
  hm[msgType] = fn
}

func (hm *HandlerMap) get(msgType int32) NewHandlerFunc {
  if fn, ok := hm[msgType]; ok {
    return fn
  }
  return nil
}
