package tao

import (
	"hash/fnv"
	"os"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

type ConnectionMap struct {
	sync.RWMutex
	m map[int64]Connection
}

func NewConnectionMap() *ConnectionMap {
	return &ConnectionMap{
		m: make(map[int64]Connection),
	}
}

func (cm *ConnectionMap) Clear() {
	cm.Lock()
	cm.m = make(map[int64]Connection)
	cm.Unlock()
}

func (cm *ConnectionMap) Get(k int64) (Connection, bool) {
	cm.RLock()
	conn, ok := cm.m[k]
	cm.RUnlock()
	return conn, ok
}

func (cm *ConnectionMap) Put(k int64, v Connection) {
	cm.Lock()
	cm.m[k] = v
	cm.Unlock()
}

func (cm *ConnectionMap) Remove(k int64) {
	cm.Lock()
	delete(cm.m, k)
	cm.Unlock()
}

func (cm *ConnectionMap) Size() int {
	cm.RLock()
	size := len(cm.m)
	cm.RUnlock()
	return size
}

func (cm *ConnectionMap) IsEmpty() bool {
	return cm.Size() <= 0
}

type Hashable interface {
	HashCode() int32
}

const intSize = unsafe.Sizeof(1)

func hashCode(k interface{}) uint32 {
	var code uint32
	h := fnv.New32a()
	switch v := k.(type) {
	case bool:
		h.Write((*((*[1]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int:
		h.Write((*((*[intSize]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int8:
		h.Write((*((*[1]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int16:
		h.Write((*((*[2]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int32:
		h.Write((*((*[4]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int64:
		h.Write((*((*[8]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint:
		h.Write((*((*[intSize]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint8:
		h.Write((*((*[1]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint16:
		h.Write((*((*[2]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint32:
		h.Write((*((*[4]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint64:
		h.Write((*((*[8]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case string:
		h.Write([]byte(v))
		code = h.Sum32()
	case Hashable:
		c := v.HashCode()
		h.Write((*((*[4]byte)(unsafe.Pointer(&c))))[:])
		code = h.Sum32()
	default:
		panic("key not hashable")
	}
	return code
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	kd := rv.Type().Kind()
	switch kd {
	case reflect.Ptr, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func printStack() {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	os.Stderr.Write(buf[:n])
}
