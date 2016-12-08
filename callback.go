package tao

import (
	"time"
)

type onConnectFunc func(Connection) bool
type onMessageFunc func(Message, Connection)
type onCloseFunc func(Connection)
type onErrorFunc func()
type workerFunc func()
type onScheduleFunc func(time.Time, interface{})

type OnTimeOut struct {
	Callback  func(time.Time, interface{})
	ExtraData interface{}
}

func NewOnTimeOut(extra interface{}, cb func(time.Time, interface{})) *OnTimeOut {
	return &OnTimeOut{
		Callback:  cb,
		ExtraData: extra,
	}
}
