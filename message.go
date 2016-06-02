package tao

import (
  "bytes"
  "log"
  "io"
  "encoding/binary"
)

func init() {
  MessageMap = make(MessageMapType)
  HandlerMap = make(HandlerMapType)
  buf = new(bytes.Buffer)
}

var (
  MessageMap MessageMapType
  HandlerMap HandlerMapType
  buf *bytes.Buffer
)

type NewHandlerFunctionType func(int64, Message) MessageHandler
type UnmarshalFunctionType func([]byte) (Message, error)

type Message interface {
  MessageNumber() int32
  Serialize() ([]byte, error)
}

type MessageHandler interface {
  Process(conn Connection) bool
}

type MessageMapType map[int32]UnmarshalFunctionType

func (mm *MessageMapType) Register(msgType int32, unmarshaler func([]byte) (Message, error)) {
  (*mm)[msgType] = UnmarshalFunctionType(unmarshaler)
}

func (mm *MessageMapType) Get(msgType int32) UnmarshalFunctionType {
  if unmarshaler, ok := (*mm)[msgType]; ok {
    return unmarshaler
  }
  return nil
}

type HandlerMapType map[int32]NewHandlerFunctionType

func (hm *HandlerMapType) Register(msgType int32, factory func(int64, Message) MessageHandler) {
  (*hm)[msgType] = NewHandlerFunctionType(factory)
}

func (hm *HandlerMapType) Get(msgType int32) NewHandlerFunctionType {
  if fn, ok := (*hm)[msgType]; ok {
    return fn
  }
  return nil
}

/* Message number 0 is the preserved message
for long-term connection keeping alive */
type DefaultHeartBeatMessage struct {
  Timestamp int64
}

func (dhbm DefaultHeartBeatMessage) Serialize() ([]byte, error) {
  buf.Reset()
  err := binary.Write(buf, binary.BigEndian, dhbm.Timestamp)
  if err != nil {
    return nil, err
  }
  return buf.Bytes(), nil
}

func (dhbm DefaultHeartBeatMessage) MessageNumber() int32 {
  return 0
}

func DeserializeDefaultHeartBeatMessage(data []byte) (message Message, err error) {
  var timestamp int64
  if data == nil {
    return nil, ErrorNilData
  }
  buf := bytes.NewReader(data)
  err = binary.Read(buf, binary.BigEndian, &timestamp)
  if err != nil {
    return nil, err
  }
  return DefaultHeartBeatMessage{
    Timestamp: timestamp,
  }, nil
}

type DefaultHeartBeatMessageHandler struct {
  netid int64
  message Message
}

func NewDefaultHeartBeatMessageHandler(net int64, msg Message) MessageHandler {
  return DefaultHeartBeatMessageHandler{
    netid: net,
    message: msg,
  }
}

func (handler DefaultHeartBeatMessageHandler) Process(client Connection) bool {
  heartBeatMessage := handler.message.(DefaultHeartBeatMessage)
  log.Printf("Receiving heart beat at %d, updating\n", heartBeatMessage.Timestamp)
  client.SetHeartBeat(heartBeatMessage.Timestamp)
  return true
}

/* Application programmer can define a custom codec themselves */
type Codec interface {
  Decode(Connection) (Message, error)
  Encode(Message) ([]byte, error)
}

// use type-length-value format: |4 bytes|4 bytes|n bytes <= 8M|
type TypeLengthValueCodec struct {}

func (codec TypeLengthValueCodec)Decode(c Connection) (Message, error) {
  byteChan := make(chan []byte)
  errorChan := make(chan error)
  var err error

  go func(bc chan []byte, ec chan error) {
    typeData := make([]byte, NTYPE)
    _, err = io.ReadFull(c.GetRawConn(), typeData)
    if err != nil {
      ec<- err
      return
    }
    bc<- typeData
  }(byteChan, errorChan)

  var typeBytes []byte

  select {
  case <-c.GetCloseChannel():
    return nil, ErrorConnClosed

  case err = <-errorChan:
    return nil, err

  case typeBytes = <-byteChan:
    typeBuf := bytes.NewReader(typeBytes)
    var msgType int32
    if err = binary.Read(typeBuf, binary.LittleEndian, &msgType); err != nil {
      return nil, err
    }
    lengthBytes := make([]byte, NLEN)
    _, err = io.ReadFull(c.GetRawConn(), lengthBytes)
    if err != nil {
      return nil, err
    }
    lengthBuf := bytes.NewReader(lengthBytes)
    var msgLen uint32
    if err = binary.Read(lengthBuf, binary.LittleEndian, &msgLen); err != nil {
      return nil, err
    }
    if msgLen > MAXLEN {
      log.Printf("len %d, type %d\n", msgLen, msgType)
      return nil, ErrorIllegalData
    }
    // read real application message
    msgBytes := make([]byte, msgLen)
    _, err = io.ReadFull(c.GetRawConn(), msgBytes)
    if err != nil {
      return nil, err
    }

    // deserialize message from bytes
    unmarshaler := MessageMap.Get(msgType)
    if unmarshaler == nil {
      return nil, Undefined(msgType)
    }
    return unmarshaler(msgBytes)
  }

}

func (codec TypeLengthValueCodec) Encode(msg Message) ([]byte, error) {
  data, err := msg.Serialize()
  if err != nil {
    return nil, err
  }
  buf := new(bytes.Buffer)
  binary.Write(buf, binary.LittleEndian, msg.MessageNumber())
  binary.Write(buf, binary.LittleEndian, int32(len(data)))
  buf.Write(data)
  // if len(data) > 0 {
  //   binary.Write(buf, binary.BigEndian, data)
  // }
  packet := buf.Bytes()
  return packet, nil
}
