package tao

import (
  "time"
)

type onConnectFunc func(*TcpConnection) bool
type onMessageFunc func(Message, *TcpConnection)
type onCloseFunc func(*TcpConnection)
type onErrorFunc func()
type workerFunc func()

type OnTimeOut struct {
  Callback func(time.Time)
  Id int64
  ExtraData interface{}
}

func NewOnTimeOut(extra interface{}, cb func(time.Time)) *OnTimeOut {
  return &OnTimeOut{
    Callback: cb,
    ExtraData: extra,
  }
}
