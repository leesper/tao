package tao

import (
  "bytes"
  "encoding/binary"
  "fmt"
  "io"

  "github.com/leesper/holmes"
)

// 0 is the preserved heart beat message number, you can define your own.
const (
  HEART_BEAT = 0
)

// handlerFunc handles the business logic for some type of Message.
type handlerFunc func(Context, Connection)

// unmarshalFunc parses bytes into Message.
type unmarshalFunc func([]byte) (Message, error)

// messageFunc is the collection of unmarshal and handle functions about Message.
type messageFunc struct {
  handler handlerFunc
  unmarshaler unmarshalFunc
}

// messageRegistry is the inner registry of all message related unmarshal and handle functions.
var (
  buf *bytes.Buffer
  messageRegistry map[int32]messageFunc
)

func init() {
  messageRegistry = map[int32]messageFunc{}
  buf = new(bytes.Buffer)
}

// Register registers the unmarshal and handle functions for msgType.
// If no unmarshal function provided, the message will not be parsed.
// If no handler function provided, the message will not be handled unless you
// set a default one by calling SetOnMessageCallback.
// If Register being called twice on one msgType, it will panics.
func Register(msgType int32, unmarshaler func([]byte) (Message, error), handler func(Context, Connection)) {
  if _, ok := messageRegistry[msgType]; ok {
    panic(fmt.Sprintf("trying to register message %d twice", msgType))
  }

  messageRegistry[msgType] = messageFunc{
    unmarshaler: unmarshaler,
    handler: handler,
  }
}

// GetUnmarshaler returns the corresponding unmarshal function for msgType.
func GetUnmarshaler(msgType int32) unmarshalFunc {
  entry, ok := messageRegistry[msgType]
  if !ok {
    return nil
  }
  return entry.unmarshaler
}

// GetHandler returns the corresponding handler function for msgType.
func GetHandler(msgType int32) handlerFunc {
  entry, ok := messageRegistry[msgType]
  if !ok {
    return nil
  }
  return entry.handler
}

// Message represents the structured data that can be handled.
type Message interface {
  MessageNumber() int32
  Serialize() ([]byte, error)
}

// Context is the context info for every handler function.
// Handler function handles the business logic about message.
// We can find the client connection who sent this message by netid and send back responses.
type Context struct{
  message Message
  netid int64
}

func NewContext(msg Message, id int64) Context {
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

// HeartBeatMessage for long-term connection keeping alive.
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
  return HEART_BEAT
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

// Codec is the interface for message coder and decoder.
// Application programmer can define a custom codec themselves.
type Codec interface {
  Decode(Connection) (Message, error)
  Encode(Message) ([]byte, error)
}

// Format: type-length-value |4 bytes|4 bytes|n bytes <= 8M|
type TypeLengthValueCodec struct {}

// Decode decodes the bytes data into Message
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
      holmes.Error("len %d, type %d", msgLen, msgType)
      return nil, ErrorIllegalData
    }
    // read real application message
    msgBytes := make([]byte, msgLen)
    _, err = io.ReadFull(c.GetRawConn(), msgBytes)
    if err != nil {
      return nil, err
    }

    // deserialize message from bytes
    unmarshaler := GetUnmarshaler(msgType)
    if unmarshaler == nil {
      return nil, Undefined(msgType)
    }
    return unmarshaler(msgBytes)
  }
}

// Encode encodes the message into bytes data.
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
