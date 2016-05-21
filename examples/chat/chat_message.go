package chat

import (
  "errors"
  "github.com/leesper/tao"
)

var ErrorNilData error = errors.New("Nil data")

type ChatMessage struct {
  Info string
}

func (cm ChatMessage) MessageNumber() int32 {
  return 1
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

type ChatMessageHandler struct {
  netid int64
  message tao.Message
}

func NewChatMessageHandler(net int64, msg tao.Message) tao.MessageHandler {
  return ChatMessageHandler{
    netid: net,
    message: msg,
  }
}

func (handler ChatMessageHandler) Process(conn tao.Connection) bool {
  if serverConn, ok := conn.(*tao.ServerConnection); ok {
    if serverConn.GetOwner() != nil {
      connections := serverConn.GetOwner().GetAllConnections()
      for v := range connections.IterValues() {
        c := v.(tao.Connection)
        c.Write(handler.message)
      }
      return true
    }
  }
  return false
}
