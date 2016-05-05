package echo

import (
  "log"
  "github.com/leesper/tao"
)

type EchoMessage struct {
  Message string
}

func (em EchoMessage) MarshalBinary() ([]byte, error) {
  return []byte(em.Message), nil
}

func (em EchoMessage) MessageNumber() int32 {
  return 1
}

func UnmarshalEchoMessage(data []byte) (message tao.Message, err error) {
  if data == nil {
    return nil, tao.ErrorNilData
  }
  msg := string(data)
  echo := EchoMessage{
    Message: msg,
  }
  return echo, nil
}

type EchoMessageHandler struct {
  message tao.Message
}

func NewEchoMessageHandler(msg tao.Message) tao.MessageHandler {
  return EchoMessageHandler{
    message: msg,
  }
}

func (handler EchoMessageHandler) Process(client *tao.TcpConnection) bool {
  echoMessage := handler.message.(EchoMessage)
  log.Printf("Receving message %s\n", echoMessage.Message)
  client.Write(handler.message)
  return true
}
