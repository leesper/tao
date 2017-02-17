package tao

import (
	"hash/fnv"
	"os"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

// ConnMap is a safe map for server connection management.
type ConnMap struct {
	sync.RWMutex
	m map[int64]*ServerConn
}

// NewConnMap returns a new ConnMap.
func NewConnMap() *ConnMap {
	return &ConnMap{
		m: make(map[int64]*ServerConn),
	}
}

// Clear clears all elements in map.
func (cm *ConnMap) Clear() {
	cm.Lock()
	cm.m = make(map[int64]*ServerConn)
	cm.Unlock()
}

// Get gets a server connection with specified net ID.
func (cm *ConnMap) Get(id int64) (*ServerConn, bool) {
	cm.RLock()
	sc, ok := cm.m[id]
	cm.RUnlock()
	return sc, ok
}

// Put puts a server connection with specified net ID in map.
func (cm *ConnMap) Put(id int64, sc *ServerConn) {
	cm.Lock()
	cm.m[id] = sc
	cm.Unlock()
}

// Remove removes a server connection with specified net ID.
func (cm *ConnMap) Remove(id int64) {
	cm.Lock()
	delete(cm.m, id)
	cm.Unlock()
}

// Size returns map size.
func (cm *ConnMap) Size() int {
	cm.RLock()
	size := len(cm.m)
	cm.RUnlock()
	return size
}

// IsEmpty tells whether ConnMap is empty.
func (cm *ConnMap) IsEmpty() bool {
	return cm.Size() <= 0
}

// Hashable is a interface for hashable object.
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
