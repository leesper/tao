package tao

import (
  "testing"
)

func TestNewConcurrentMap(t *testing.T) {
  cm := NewConcurrentMap()
  if cm == nil {
    t.Error("map is nil")
  }
  if !cm.IsEmpty() {
    t.Error("map not empty")
  }
  if cm.Size() != 0 {
    t.Error("map size != 0")
  }

}
