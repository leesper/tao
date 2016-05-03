package tao

import (
  "hash/fnv"
  "errors"
  "reflect"
  "bytes"
  "encoding/binary"
)

type Hashable interface {
	HashCode() int32
}

var ErrorNotHashable = errors.New("Not hashable")
var buf *bytes.Buffer = new(bytes.Buffer)
func hashCode(k interface{}) (uint32, error) {
  h := fnv.New32a()
  var code uint32
  var err error
  buf.Reset()
  switch v := k.(type) {
  case bool:
    if v {
      binary.Write(buf, binary.BigEndian, uint8(1))
      h.Write(buf.Bytes())
    } else {
      binary.Write(buf, binary.BigEndian, uint8(0))
      h.Write(buf.Bytes())
    }
    code = h.Sum32()
  case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
    binary.Write(buf, binary.BigEndian, v)
    h.Write(buf.Bytes())
    code = h.Sum32()
  case string:
    h.Write([]byte(v))
    code = h.Sum32()
  case Hashable:
    binary.Write(buf, binary.BigEndian, v.HashCode())
    h.Write(buf.Bytes())
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
