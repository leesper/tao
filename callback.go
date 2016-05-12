package tao

import (
  "time"
)

type onConnectFunc func(*TCPConnection) bool
type onMessageFunc func(Message, *TCPConnection)
type onCloseFunc func(*TCPConnection)
type onErrorFunc func()
type workerFunc func()

type OnTimeOut struct {
  Callback func(time.Time, interface{})
  ExtraData interface{}
}

func NewOnTimeOut(extra interface{}, cb func(time.Time, interface{})) *OnTimeOut {
  return &OnTimeOut{
    Callback: cb,
    ExtraData: extra,
  }
}
