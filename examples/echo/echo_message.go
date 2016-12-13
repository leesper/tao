package echo

import (
	"github.com/reechou/holmes"
	"github.com/leesper/tao"
)

type EchoMessage struct {
	Message string
}

func (em EchoMessage) Serialize() ([]byte, error) {
	return []byte(em.Message), nil
}

func (em EchoMessage) MessageNumber() int32 {
	return 1
}

func DeserializeEchoMessage(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, tao.ErrorNilData
	}
	msg := string(data)
	echo := EchoMessage{
		Message: msg,
	}
	return echo, nil
}

func ProcessEchoMessage(ctx tao.Context, conn tao.Connection) {
	echoMessage := ctx.Message().(EchoMessage)
	holmes.Info("Receving message %s\n", echoMessage.Message)
	conn.Write(ctx.Message())
}
