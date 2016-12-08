package pingpong

import (
	"errors"
	"github.com/leesper/tao"
)

const (
	PINGPONG_MESSAGE int32 = 1
)

var ErrorNilData error = errors.New("Nil data")

type PingPongMessage struct {
	Info string
}

func (pp PingPongMessage) MessageNumber() int32 {
	return PINGPONG_MESSAGE
}

func (pp PingPongMessage) Serialize() ([]byte, error) {
	return []byte(pp.Info), nil
}

func DeserializePingPongMessage(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, ErrorNilData
	}
	info := string(data)
	msg := PingPongMessage{
		Info: info,
	}
	return msg, nil
}

// func ProcessPingPongMessage(ctx tao.Context, conn tao.Connection) {
//   if serverConn, ok := conn.(*tao.ServerConnection); ok {
//     if serverConn.GetOwner() != nil {
//       connections := serverConn.GetOwner().GetAllConnections()
//       for v := range connections.IterValues() {
//         c := v.(tao.Connection)
//         c.Write(ctx.Message())
//       }
//     }
//   }
// }
