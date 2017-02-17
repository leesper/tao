package pingpong

import "github.com/leesper/tao"

const (
	// PingPontMessage defines message number.
	PingPontMessage int32 = 1
)

// Message defines message format.
type Message struct {
	Info string
}

// MessageNumber returns the message number.
func (pp Message) MessageNumber() int32 {
	return PingPontMessage
}

// Serialize serializes Message into bytes.
func (pp Message) Serialize() ([]byte, error) {
	return []byte(pp.Info), nil
}

// DeserializeMessage deserializes bytes into Message.
func DeserializeMessage(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, tao.ErrNilData
	}
	info := string(data)
	msg := Message{
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
