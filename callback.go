package tao

import (
  "time"
)

type onConnectCallbackType func(*TcpConnection) bool
type onMessageCallbackType func(Message, *TcpConnection)
type onCloseCallbackType func(*TcpConnection)
type onErrorCallbackType func()
type workerCallbackType func()

type onTimeOutCallbackType struct {
  identifier interface{}
  onTimeOut func(time.Time)
}

func newOnTimeOutCallbackType(id interface{}, cb func(time.Time)) *onTimeOutCallbackType {
  return &onTimeOutCallbackType{
    identifier: id,
    onTimeOut: cb,
  }
}
