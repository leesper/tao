package tao

import (
	"time"
)

type onConnectFunc func(interface{}) bool
type onMessageFunc func(Message, interface{})
type onCloseFunc func(interface{})
type onErrorFunc func(interface{})

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
