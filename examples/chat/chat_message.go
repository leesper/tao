package chat

import (
	"context"

	"github.com/leesper/holmes"
	"github.com/leesper/tao"
)

const (
	// ChatMessage is the message number of chat message.
	ChatMessage int32 = 1
)

// Message defines the chat message.
type Message struct {
	Content string
}

// MessageNumber returns the message number.
func (cm Message) MessageNumber() int32 {
	return ChatMessage
}

// Serialize Serializes Message into bytes.
func (cm Message) Serialize() ([]byte, error) {
	return []byte(cm.Content), nil
}

// DeserializeMessage deserializes bytes into Message.
func DeserializeMessage(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, tao.ErrNilData
	}
	content := string(data)
	msg := Message{
		Content: content,
	}
	return msg, nil
}

// ProcessMessage handles the Message logic.
func ProcessMessage(ctx context.Context, conn tao.WriteCloser) {
	holmes.Info("ProcessMessage")
	s, ok := ctx.Value(tao.ServerCtx).(*tao.Server)
	if ok {
		msg, ok := ctx.Value(tao.MessageCtx).(tao.Message)
		if ok {
			s.Broadcast(msg)
		}
	}
}
