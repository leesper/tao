package chat

import (
	"errors"
	"github.com/leesper/tao"
)

const (
	CHAT_MESSAGE int32 = 1
)

var ErrorNilData error = errors.New("Nil data")

type ChatMessage struct {
	Info string
}

func (cm ChatMessage) MessageNumber() int32 {
	return CHAT_MESSAGE
}

func (cm ChatMessage) Serialize() ([]byte, error) {
	return []byte(cm.Info), nil
}

func DeserializeChatMessage(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, ErrorNilData
	}
	info := string(data)
	msg := ChatMessage{
		Info: info,
	}
	return msg, nil
}

func ProcessChatMessage(ctx tao.Context, conn tao.Connection) {
	if serverConn, ok := conn.(*tao.ServerConnection); ok {
		if serverConn.GetOwner() != nil {
			connections := serverConn.GetOwner().GetConnections()
			for _, c := range connections {
				c.Write(ctx.Message())
			}
		}
	}
}
