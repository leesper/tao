package echo

import (
  "log"
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

type EchoMessageHandler struct {
  netid int64
  message tao.Message
}

func NewEchoMessageHandler(net int64, msg tao.Message) tao.MessageHandler {
  return EchoMessageHandler{
    netid: net,
    message: msg,
  }
}

func (handler EchoMessageHandler) Process(client *tao.TCPConnection) bool {
  echoMessage := handler.message.(EchoMessage)
  log.Printf("Receving message %s\n", echoMessage.Message)
  client.Write(handler.message)
  return true
}
