package tao

import (
  "bytes"
  "encoding"
  "log"
  "encoding/binary"
)

type Message interface {
  MessageNumber() int32
  encoding.BinaryMarshaler
}

type MessageHandler interface {
  Process(client *TCPConnection) bool
}

type NewHandlerFunctionType func(m Message) MessageHandler
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

var (
  MessageMap MessageMapType
  HandlerMap HandlerMapType
  buf *bytes.Buffer
)

func init() {
  MessageMap = make(MessageMapType)
  HandlerMap = make(HandlerMapType)
  buf = new(bytes.Buffer)
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

type HeartBeatMessage struct {
  Timestamp int64
}

func (hbm HeartBeatMessage) MarshalBinary() ([]byte, error) {
  buf.Reset()
  err := binary.Write(buf, binary.BigEndian, hbm.Timestamp)
  if err != nil {
    return nil, err
  }
  return buf.Bytes(), nil
}

func (hbm HeartBeatMessage) MessageNumber() int32 {
  return 0
}

func UnmarshalHeartBeatMessage(data []byte) (message Message, err error) {
  var timestamp int64
  if data == nil {
    return nil, ErrorNilData
  }
  buf := bytes.NewReader(data)
  err = binary.Read(buf, binary.BigEndian, &timestamp)
  if err != nil {
    return nil, err
  }
  return HeartBeatMessage{
    Timestamp: timestamp,
  }, nil
}

type HeartBeatMessageHandler struct {
  message Message
}

func NewHeartBeatMessageHandler(msg Message) MessageHandler {
  return HeartBeatMessageHandler{
    message: msg,
  }
}

func (handler HeartBeatMessageHandler) Process(client *TCPConnection) bool {
  heartBeatMessage := handler.message.(HeartBeatMessage)
  log.Printf("Receiving heart beat at %d, updating\n", heartBeatMessage.Timestamp)
  client.heartBeat = heartBeatMessage.Timestamp
  return true
}
