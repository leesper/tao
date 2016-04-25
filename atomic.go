package tao

import (
  "sync/atomic"
)

type AtomicInt64 int64

func NewAtomicInt64(initialValue int64) *AtomicInt64 {
  return &AtomicInt64(initialValue)
}

func (a *AtomicInt64) GetAndIncrement() int64 {
  value := int64(a)
  atomic.AddInt64(*int64(a), 1)
  return value
}

type AtomicBoolean int32

func NewAtomicBoolean(initialValue bool) *AtomicBoolean {
  if initialValue {
    return &AtomicBoolean(1)
  }
  return &AtomicBoolean(0)
}

func (a *AtomicBoolean) Get() bool {
  return a != 0
}

func (a *AtomicBoolean) Set(newValue bool) {
  if newValue {
    atomic.StoreInt32(*int32(a), 1)
  } else {
    atomic.StoreInt32(*int32(a), 0)
  }
}

func (a *AtomicBoolean) CompareAndSet(oldValue, newValue bool) bool {
  var o int32
  var n int32
  if oldValue {
    o = 1
  } else {
    o = 0
  }
  if newValue {
    n = 1
  } else {
    n = 0
  }
  return atomic.CompareAndSwapInt32(*int32(a), o, n)
}
