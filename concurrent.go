package tao

import (
  "errors"
  "sync"
  "sync/atomic"
)

type AtomicInt64 int64

func NewAtomicInt64(initialValue int64) *AtomicInt64 {
  a := AtomicInt64(initialValue)
  return &a
}

func (a *AtomicInt64) GetAndIncrement() int64 {
  value := int64(*a)
  atomic.AddInt64((*int64)(a), 1)
  return value
}

type AtomicBoolean int32

func NewAtomicBoolean(initialValue bool) *AtomicBoolean {
  var a AtomicBoolean
  if initialValue {
    a = AtomicBoolean(1)
  } else {
    a = AtomicBoolean(0)
  }
  return &a
}

func (a *AtomicBoolean) Get() bool {
  return int32(*a) != 0
}

func (a *AtomicBoolean) Set(newValue bool) {
  if newValue {
    atomic.StoreInt32((*int32)(a), 1)
  } else {
    atomic.StoreInt32((*int32)(a), 0)
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
  return atomic.CompareAndSwapInt32((*int32)(a), o, n)
}

const INITIAL_SHARD_SIZE = 16
var ErrorNilKey = errors.New("Nil key")
var ErrorNilValue = errors.New("Nil value")

type ConcurrentMap struct {
  shards []*syncMap
}

func NewConcurrentMap() *ConcurrentMap {
  cm := &ConcurrentMap{
    shards: make([]*syncMap, INITIAL_SHARD_SIZE),
  }
  for i, _ := range cm.shards {
    cm.shards[i] = newSyncMap()
  }
  return cm
}

func (cm *ConcurrentMap)Put(k, v interface{}) error {
  if isNil(k) {
    return ErrorNilKey
  }
  if isNil(v) {
    return ErrorNilValue
  }
  if shard, err := cm.shardFor(k); err != nil {
    return err
  } else {
    shard.put(k, v)
  }
  return nil
}

func (cm *ConcurrentMap)Get(k interface{}) (interface{}, bool) {
  if isNil(k) {
    return nil, false
  }
  if shard, err := cm.shardFor(k); err != nil {
    return nil, false
  } else {
    return shard.get(k)
  }

}

func (cm *ConcurrentMap)Remove(k interface{}) bool {
  if isNil(k) {
    return false
  }
  if shard, err := cm.shardFor(k); err != nil {
    return false
  } else {
    return shard.remove(k)
  }
}

func (cm *ConcurrentMap)IsEmpty() bool {
  return cm.Size() <= 0
}

func (cm *ConcurrentMap)Clear() {
  for _, s := range cm.shards {
    s.clear()
  }
}

func (cm *ConcurrentMap)Size() int {
  var size int32 = 0
  for _, s := range cm.shards {
    size += s.size()
  }
  return int(size)
}

func (cm *ConcurrentMap)shardFor(k interface{}) (*syncMap, error) {
  if code, err := hashCode(k); err != nil {
    return nil, err
  } else {
    return cm.shards[code&uint32(INITIAL_SHARD_SIZE-1)], nil
  }
}

func (cm *ConcurrentMap)IterKeys() <-chan interface{} {
  kch := make(chan interface{})
  go func() {
    for _, s := range cm.shards {
      s.RLock()
      defer s.RUnlock()
      for k, _ := range s.shard {
        kch <- k
      }
    }
    close(kch)
  }()
  return kch
}

func (cm *ConcurrentMap)IterValues() <-chan interface{} {
  vch := make(chan interface{})
  go func() {
    for _, s := range cm.shards {
      s.RLock()
      defer s.RUnlock()
      for _, v := range s.shard {
        vch <- v
      }
    }
    close(vch)
  }()
  return vch
}

func (cm *ConcurrentMap)IterItems() <-chan Item {
  ich := make(chan Item)
  go func() {
    for _, s := range cm.shards {
      s.RLock()
      defer s.RUnlock()
      for k, v := range s.shard {
        ich <- Item{k, v}
      }
    }
    close(ich)
  }()
  return ich
}

type Item struct {
  Key, Value interface{}
}

type syncMap struct {
  shard map[interface{}]interface{}
  sync.RWMutex
}

func newSyncMap()*syncMap {
  return &syncMap{
    shard: make(map[interface{}]interface{}),
  }
}

func (sm *syncMap) put(k, v interface{}) {
  sm.Lock()
  defer sm.Unlock()
  sm.shard[k] = v
}

func (sm *syncMap) get(k interface{}) (interface{}, bool) {
  sm.RLock()
  defer sm.RUnlock()
  v, ok := sm.shard[k]
  return v, ok
}

func (sm *syncMap) size() int32 {
  sm.RLock()
  defer sm.RUnlock()
  return int32(len(sm.shard))
}

func (sm *syncMap) remove(k interface{}) bool {
  sm.Lock()
  defer sm.Unlock()
  _, ok := sm.shard[k]
  delete(sm.shard, k)
  return ok
}

func (sm *syncMap) clear() {
  sm.Lock()
  defer sm.Unlock()
  sm.shard = make(map[interface{}]interface{})
}
