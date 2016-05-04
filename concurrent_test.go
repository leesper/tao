package tao

import (
  "fmt"
  "sync"
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

func TestConcurrentMapInt(t *testing.T) {
  cm := NewConcurrentMap()
  cm.Put(1, 10)
  if cm.IsEmpty() {
    t.Error("map is empty")
  }
  if cm.Size() != 1 {
    t.Error("map size != 1")
  }

  var val interface{}
  var ok bool
  if val, ok = cm.Get(1); !ok {
    t.Error("map get error")
  }
  if val.(int) != 10 {
    t.Errorf("error value %d", val.(int))
  }

  cm.Put(1, 20)
  if val, ok = cm.Get(1); !ok || val.(int) != 20 {
    t.Errorf("map get error %d", val.(int))
  }
  if cm.IsEmpty() {
    t.Error("map is empty")
  }
  if cm.Size() != 1 {
    t.Error("map size != 1")
  }
}

func TestConcurrentMapString(t *testing.T) {
  cm := NewConcurrentMap()
  cm.Put("Lucy", "Product Manager")
  cm.Put("Lily", "C++")
  cm.Put("Kathy", "Python")
  cm.Put("Joana", "Golang")
  cm.Put("Belle", "Java")
  cm.PutIfAbsent("Joana", "Objective-C")
  cm.PutIfAbsent("Fiona", "Javascript")
  if cm.Size() != 6 {
    t.Error("map size != 6")
  }
  cm.Put("Lily", "Rust Programmer")
  fmt.Print("Keys: ")
  for key := range cm.IterKeys() {
    fmt.Print(key.(string), " ")
  }
  fmt.Println()
  fmt.Println("Items: ")
  for item := range cm.IterItems() {
    fmt.Printf("key %s value %s\n", item.Key.(string), item.Value.(string))
  }

  ok := cm.Remove("Lucy")
  if !ok {
    t.Error("Key Lucy not found")
  }

  if ok, _ = cm.ContainsKey("Lucy"); ok {
    t.Error("Key Lucy not removed")
  }

  if ok, _ = cm.ContainsKey("Joana"); !ok {
    t.Error("Key Joana not found")
  }


  fmt.Println()
  fmt.Print("Values: ")
  for val := range cm.IterValues() {
    fmt.Print(val.(string), " ")
  }
  fmt.Println()

  cm.Clear()
  if !cm.IsEmpty() || cm.Size() != 0 {
    t.Error("Map size error, not empty")
  }
}

func TestAtomicInt64(t *testing.T) {
  ai64 := NewAtomicInt64(0)
  wg := &sync.WaitGroup{}
  for i := 0; i < 3; i++ {
    wg.Add(1)
    go func() {
      ai64.GetAndIncrement()
      wg.Done()
    }()
  }
  wg.Wait()
  if ai64.Get() != 3 {
    t.Error("Get and increment error")
  }

  for i := 0; i < 3; i++ {
    wg.Add(1)
    go func() {
      ai64.GetAndDecrement()
      wg.Done()
    }()
  }
  wg.Wait()
  if ai64.Get() != 0 {
    t.Error("Get and decrement error")
  }
}
