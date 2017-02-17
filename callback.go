package tao

import (
	"context"
	"time"
)

type onConnectFunc func(WriteCloser) bool
type onMessageFunc func(Message, WriteCloser)
type onCloseFunc func(WriteCloser)
type onErrorFunc func(WriteCloser)

type workerFunc func()
type onScheduleFunc func(time.Time, WriteCloser)

type OnTimeOut struct {
	Callback func(time.Time, WriteCloser)
	Ctx      context.Context
}

func NewOnTimeOut(ctx context.Context, cb func(time.Time, WriteCloser)) *OnTimeOut {
	return &OnTimeOut{
		Callback: cb,
		Ctx:      ctx,
	}
}
