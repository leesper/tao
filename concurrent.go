/* Data types and structues for concurrent use:
AtomicInt32
AtomicInt64
AtomicBoolean
ConcurrentMap
*/

package tao

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type AtomicInt64 int64

func NewAtomicInt64(initialValue int64) *AtomicInt64 {
	a := AtomicInt64(initialValue)
	return &a
}

func (a *AtomicInt64) Get() int64 {
	return int64(*a)
}

func (a *AtomicInt64) Set(newValue int64) {
	atomic.StoreInt64((*int64)(a), newValue)
}

func (a *AtomicInt64) GetAndSet(newValue int64) int64 {
	for {
		current := a.Get()
		if a.CompareAndSet(current, newValue) {
			return current
		}
	}
}

func (a *AtomicInt64) CompareAndSet(expect, update int64) bool {
	return atomic.CompareAndSwapInt64((*int64)(a), expect, update)
}

func (a *AtomicInt64) GetAndIncrement() int64 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}

}

func (a *AtomicInt64) GetAndDecrement() int64 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

func (a *AtomicInt64) GetAndAdd(delta int64) int64 {
	for {
		current := a.Get()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

func (a *AtomicInt64) IncrementAndGet() int64 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt64) DecrementAndGet() int64 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt64) AddAndGet(delta int64) int64 {
	for {
		current := a.Get()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt64) String() string {
	return fmt.Sprintf("%d", a.Get())
}

type AtomicInt32 int32

func NewAtomicInt32(initialValue int32) *AtomicInt32 {
	a := AtomicInt32(initialValue)
	return &a
}

func (a *AtomicInt32) Get() int32 {
	return int32(*a)
}

func (a *AtomicInt32) Set(newValue int32) {
	atomic.StoreInt32((*int32)(a), newValue)
}

func (a *AtomicInt32) GetAndSet(newValue int32) (oldValue int32) {
	for {
		oldValue = a.Get()
		if a.CompareAndSet(oldValue, newValue) {
			return
		}
	}
}

func (a *AtomicInt32) CompareAndSet(expect, update int32) bool {
	return atomic.CompareAndSwapInt32((*int32)(a), expect, update)
}

func (a *AtomicInt32) GetAndIncrement() int32 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}

}

func (a *AtomicInt32) GetAndDecrement() int32 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

func (a *AtomicInt32) GetAndAdd(delta int32) int32 {
	for {
		current := a.Get()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

func (a *AtomicInt32) IncrementAndGet() int32 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt32) DecrementAndGet() int32 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt32) AddAndGet(delta int32) int32 {
	for {
		current := a.Get()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt32) String() string {
	return fmt.Sprintf("%d", a.Get())
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
	return atomic.LoadInt32((*int32)(a)) != 0
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

func (a *AtomicBoolean) GetAndSet(newValue bool) bool {
	for {
		current := a.Get()
		if a.CompareAndSet(current, newValue) {
			return current
		}
	}
}

func (a *AtomicBoolean) String() string {
	return fmt.Sprintf("%t", a.Get())
}

const INITIAL_SHARD_SIZE = 16

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

func (cm *ConcurrentMap) Put(k, v interface{}) error {
	if isNil(k) {
		return ErrorNilKey
	}
	if isNil(v) {
		return ErrorNilValue
	}
	cm.shardFor(k).put(k, v)
	return nil
}

func (cm *ConcurrentMap) PutIfAbsent(k, v interface{}) error {
	if isNil(k) {
		return ErrorNilKey
	}
	if isNil(v) {
		return ErrorNilValue
	}
	shard := cm.shardFor(k)
	if _, ok := shard.get(k); !ok {
		shard.put(k, v)
	}
	return nil
}

func (cm *ConcurrentMap) Get(k interface{}) (interface{}, bool) {
	return cm.shardFor(k).get(k)
}

func (cm *ConcurrentMap) ContainsKey(k interface{}) bool {
	if isNil(k) {
		return false
	}
	_, ok := cm.shardFor(k).get(k)
	return ok
}

func (cm *ConcurrentMap) Remove(k interface{}) error {
	if isNil(k) {
		return ErrorNilKey
	}
	cm.shardFor(k).remove(k)
	return nil
}

func (cm *ConcurrentMap) IsEmpty() bool {
	return cm.Size() <= 0
}

func (cm *ConcurrentMap) Clear() {
	for _, s := range cm.shards {
		s.clear()
	}
}

func (cm *ConcurrentMap) Size() int {
	var size int = 0
	for _, s := range cm.shards {
		size += s.size()
	}
	return size
}

func (cm *ConcurrentMap) shardFor(k interface{}) *syncMap {
	return cm.shards[hashCode(k)&uint32(INITIAL_SHARD_SIZE-1)]
}

func (cm *ConcurrentMap) IterKeys() <-chan interface{} {
	kch := make(chan interface{})
	go func() {
		for _, s := range cm.shards {
			s.RLock()
			for k, _ := range s.shard {
				kch <- k
			}
			s.RUnlock()
		}
		close(kch)
	}()
	return kch
}

func (cm *ConcurrentMap) IterValues() <-chan interface{} {
	vch := make(chan interface{})
	go func() {
		for _, s := range cm.shards {
			s.RLock()
			for _, v := range s.shard {
				vch <- v
			}
			s.RUnlock()
		}
		close(vch)
	}()
	return vch
}

func (cm *ConcurrentMap) IterItems() <-chan Item {
	ich := make(chan Item)
	go func() {
		for _, s := range cm.shards {
			s.RLock()
			for k, v := range s.shard {
				ich <- Item{k, v}
			}
			s.RUnlock()
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

func newSyncMap() *syncMap {
	return &syncMap{
		shard: make(map[interface{}]interface{}, 1024),
	}
}

func (sm *syncMap) put(k, v interface{}) {
	sm.Lock()
	sm.shard[k] = v
	sm.Unlock()
}

func (sm *syncMap) get(k interface{}) (interface{}, bool) {
	sm.RLock()
	v, ok := sm.shard[k]
	sm.RUnlock()
	return v, ok
}

func (sm *syncMap) size() int {
	var ret int
	sm.RLock()
	ret = len(sm.shard)
	sm.RUnlock()
	return ret
}

func (sm *syncMap) remove(k interface{}) {
	sm.Lock()
	delete(sm.shard, k)
	sm.Unlock()
}

func (sm *syncMap) clear() {
	sm.Lock()
	sm.shard = make(map[interface{}]interface{})
	sm.Unlock()
}
