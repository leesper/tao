package tao

import (
  "hash"
  "hash/fnv"
  "reflect"
  "unsafe"
)

type Hashable interface {
	HashCode() int32
}

var h hash.Hash32

const intSize = unsafe.Sizeof(1)

func init() {
  h = fnv.New32a()
}

func hashCode(k interface{}) (uint32, error) {
  var code uint32
  var err error
  h.Reset()
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
    err = ErrorNotHashable
	}
	return code, err
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
