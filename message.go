package tao

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/leesper/holmes"
)

const (
	// HeartBeat is the default heart beat message number.
	HeartBeat = 0
)

// Handler takes the responsibility to handle incoming messages.
type Handler interface {
	Handle(context.Context, interface{})
}

// HandlerFunc serves as an adapter to allow the use of ordinary functions as handlers.
type HandlerFunc func(context.Context, WriteCloser)

// Handle calls f(ctx, c)
func (f HandlerFunc) Handle(ctx context.Context, c WriteCloser) {
	f(ctx, c)
}

// UnmarshalFunc unmarshals bytes into Message.
type UnmarshalFunc func([]byte) (Message, error)

// handlerUnmarshaler is a combination of unmarshal and handle functions for message.
type handlerUnmarshaler struct {
	handler     HandlerFunc
	unmarshaler UnmarshalFunc
}

var (
	buf *bytes.Buffer
	// messageRegistry is the registry of all
	// message-related unmarshal and handle functions.
	messageRegistry map[int32]handlerUnmarshaler
)

func init() {
	messageRegistry = map[int32]handlerUnmarshaler{}
	buf = new(bytes.Buffer)
}

// Register registers the unmarshal and handle functions for msgType.
// If no unmarshal function provided, the message will not be parsed.
// If no handler function provided, the message will not be handled unless you
// set a default one by calling SetOnMessageCallback.
// If Register being called twice on one msgType, it will panics.
func Register(msgType int32, unmarshaler func([]byte) (Message, error), handler func(context.Context, WriteCloser)) {
	if _, ok := messageRegistry[msgType]; ok {
		panic(fmt.Sprintf("trying to register message %d twice", msgType))
	}

	messageRegistry[msgType] = handlerUnmarshaler{
		unmarshaler: unmarshaler,
		handler:     HandlerFunc(handler),
	}
}

// GetUnmarshalFunc returns the corresponding unmarshal function for msgType.
func GetUnmarshalFunc(msgType int32) UnmarshalFunc {
	entry, ok := messageRegistry[msgType]
	if !ok {
		return nil
	}
	return entry.unmarshaler
}

// GetHandlerFunc returns the corresponding handler function for msgType.
func GetHandlerFunc(msgType int32) HandlerFunc {
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

// HeartBeatMessage for application-level keeping alive.
type HeartBeatMessage struct {
	Timestamp int64
}

// Serialize serializes HeartBeatMessage into bytes.
func (hbm HeartBeatMessage) Serialize() ([]byte, error) {
	buf.Reset()
	err := binary.Write(buf, binary.LittleEndian, hbm.Timestamp)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MessageNumber returns message number.
func (hbm HeartBeatMessage) MessageNumber() int32 {
	return HeartBeat
}

// DeserializeHeartBeat deserializes bytes into Message.
func DeserializeHeartBeat(data []byte) (message Message, err error) {
	var timestamp int64
	if data == nil {
		return nil, ErrNilData
	}
	buf := bytes.NewReader(data)
	err = binary.Read(buf, binary.LittleEndian, &timestamp)
	if err != nil {
		return nil, err
	}
	return HeartBeatMessage{
		Timestamp: timestamp,
	}, nil
}

// HandleHeartBeat updates connection heart beat timestamp.
func HandleHeartBeat(ctx context.Context, c WriteCloser) {
	msg := MessageFromContext(ctx)
	switch c := c.(type) {
	case *ServerConn:
		c.SetHeartBeat(msg.(HeartBeatMessage).Timestamp)
	case *ClientConn:
		c.SetHeartBeat(msg.(HeartBeatMessage).Timestamp)
	}
}

// Codec is the interface for message coder and decoder.
// Application programmer can define a custom codec themselves.
type Codec interface {
	Decode(net.Conn) (Message, error)
	Encode(Message) ([]byte, error)
}

// TypeLengthValueCodec defines a special codec.
// Format: type-length-value |4 bytes|4 bytes|n bytes <= 8M|
type TypeLengthValueCodec struct{}

// Decode decodes the bytes data into Message
func (codec TypeLengthValueCodec) Decode(raw net.Conn) (Message, error) {
	byteChan := make(chan []byte)
	errorChan := make(chan error)

	go func(bc chan []byte, ec chan error) {
		typeData := make([]byte, MessageTypeBytes)
		_, err := io.ReadFull(raw, typeData)
		if err != nil {
			ec <- err
			close(bc)
			close(ec)
			holmes.Debugln("go-routine read message type exited")
			return
		}
		bc <- typeData
	}(byteChan, errorChan)

	var typeBytes []byte

	select {
	case err := <-errorChan:
		return nil, err

	case typeBytes = <-byteChan:
		if typeBytes == nil {
			holmes.Warnln("read type bytes nil")
			return nil, ErrBadData
		}
		typeBuf := bytes.NewReader(typeBytes)
		var msgType int32
		if err := binary.Read(typeBuf, binary.LittleEndian, &msgType); err != nil {
			return nil, err
		}

		lengthBytes := make([]byte, MessageLenBytes)
		_, err := io.ReadFull(raw, lengthBytes)
		if err != nil {
			return nil, err
		}
		lengthBuf := bytes.NewReader(lengthBytes)
		var msgLen uint32
		if err = binary.Read(lengthBuf, binary.LittleEndian, &msgLen); err != nil {
			return nil, err
		}
		if msgLen > MessageMaxBytes {
			holmes.Errorf("message(type %d) has bytes(%d) beyond max %d\n", msgType, msgLen, MessageMaxBytes)
			return nil, ErrBadData
		}

		// read application data
		msgBytes := make([]byte, msgLen)
		_, err = io.ReadFull(raw, msgBytes)
		if err != nil {
			return nil, err
		}

		// deserialize message from bytes
		unmarshaler := GetUnmarshalFunc(msgType)
		if unmarshaler == nil {
			return nil, ErrUndefined(msgType)
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

// ContextKey is the key type for putting context-related data.
type contextKey string

// Context keys for messge, server and net ID.
const (
	messageCtx contextKey = "message"
	serverCtx  contextKey = "server"
	netIDCtx   contextKey = "netid"
)

// NewContextWithMessage returns a new Context that carries message.
func NewContextWithMessage(ctx context.Context, msg Message) context.Context {
	return context.WithValue(ctx, messageCtx, msg)
}

// MessageFromContext extracts a message from a Context.
func MessageFromContext(ctx context.Context) Message {
	return ctx.Value(messageCtx).(Message)
}

// NewContextWithNetID returns a new Context that carries net ID.
func NewContextWithNetID(ctx context.Context, netID int64) context.Context {
	return context.WithValue(ctx, netIDCtx, netID)
}

// NetIDFromContext returns a net ID from a Context.
func NetIDFromContext(ctx context.Context) int64 {
	return ctx.Value(netIDCtx).(int64)
}
