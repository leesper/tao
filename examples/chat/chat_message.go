package chat

import (
  "errors"
  "github.com/leesper/tao"
)

var ErrorNilData error = errors.New("Nil data")

type ChatMessage struct {
  Info string
}

func (cm ChatMessage) MarshalBinary() ([]byte, error) {
  return []byte(cm.Info), nil
}

func (cm ChatMessage) MessageNumber() int32 {
  return 1
}

func UnmarshalChatMessage(data []byte) (message tao.Message, err error) {
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
  message tao.Message
}

func NewChatMessageHandler(msg tao.Message) tao.ProtocolHandler {
  return ChatMessageHandler{
    message: msg,
  }
}

func (handler ChatMessageHandler) Process(client *tao.TcpConnection) bool {
  if client.Owner != nil {
    connections := client.Owner.GetAllConnections()
    for _, c := range connections {
      c.Write(handler.message)
    }
    return true
  }
  return false
}
