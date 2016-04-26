package echo

import (
  "log"
  "errors"
  "github.com/leesper/tao"
)

var ErrorNilData error = errors.New("Nil data")

type EchoMessage struct {
  Message string
}

func (em EchoMessage) MarshalBinary() ([]byte, error) {
  return []byte(em.Message), nil
}

func (em EchoMessage) MessageNumber() int32 {
  return 1
}

type EchoMessageHandler struct {
  message tao.Message
}

func NewEchoMessageHandler(msg tao.Message) EchoMessageHandler {
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
