package tao

import (
  "bytes"
  "io"
  "encoding/binary"
  "github.com/golang/glog"
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

const (
  DEFAULT_HEART_BEAT = 0
)

type HandlerFunction func(Context, Connection)
type UnmarshalFunction func([]byte) (Message, error)

type Message interface {
  MessageNumber() int32
  Serialize() ([]byte, error)
}

// User context info
type Context struct{
  message Message
  netid int64
}

func NewContext(msg Message) Context {
  return Context{
    message: msg,
    netid: -1,
  }
}

func NewContextWithId(msg Message, id int64) Context {
  return Context{
    message: msg,
    netid: id,
  }
}

func (ctx Context)Message() Message {
  return ctx.message
}

func (ctx Context)Id() int64 {
  return ctx.netid
}

type MessageMapType map[int32]UnmarshalFunction

func (mm *MessageMapType) Register(msgType int32, unmarshaler func([]byte) (Message, error)) {
  (*mm)[msgType] = UnmarshalFunction(unmarshaler)
}

func (mm *MessageMapType) Get(msgType int32) UnmarshalFunction {
  if unmarshaler, ok := (*mm)[msgType]; ok {
    return unmarshaler
  }
  return nil
}

type HandlerMapType map[int32]HandlerFunction

func (hm *HandlerMapType) Register(msgType int32, handler func(Context, Connection)) {
  (*hm)[msgType] = HandlerFunction(handler)
}

func (hm *HandlerMapType) Get(msgType int32) HandlerFunction {
  if fn, ok := (*hm)[msgType]; ok {
    return fn
  }
  return nil
}

/* Message number 0 is the preserved message
for long-term connection keeping alive */
type HeartBeatMessage struct {
  Timestamp int64
}

func (hbm HeartBeatMessage) Serialize() ([]byte, error) {
  buf.Reset()
  err := binary.Write(buf, binary.BigEndian, hbm.Timestamp)
  if err != nil {
    return nil, err
  }
  return buf.Bytes(), nil
}

func (hbm HeartBeatMessage) MessageNumber() int32 {
  return DEFAULT_HEART_BEAT
}

func DeserializeHeartBeatMessage(data []byte) (message Message, err error) {
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

func ProcessHeartBeatMessage(ctx Context, conn Connection) {
  heartBeatMsg := ctx.Message().(HeartBeatMessage)
  conn.SetHeartBeat(heartBeatMsg.Timestamp)
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
      glog.Errorf("len %d, type %d\n", msgLen, msgType)
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
  packet := buf.Bytes()
  return packet, nil
}
