package tao

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewConcurrentMap(t *testing.T) {
	cm := NewConcurrentMap()
	if cm == nil {
		t.Error("NewConcurrentMap() = nil, want non-nil")
	}
	if !cm.IsEmpty() {
		t.Error("ConcurrentMap::IsEmpty() = false")
	}
	if size := cm.Size(); size != 0 {
		t.Errorf("ConcurrentMap::Size() = %d, want 0", size)
	}
}

func TestConcurrentMapPutAndRemove(t *testing.T) {
	cm := NewConcurrentMap()
	var i int64
	for i = 0; i < 10000; i++ {
		cm.Put(i, fmt.Sprintf("%d", i))
	}

	if size := cm.Size(); size != 10000 {
		t.Errorf("ConcurrentMap::Size() = %d, want 10000", size)
	}

	for i = 0; i < 10000; i++ {
		cm.Remove(i)
	}

	if !cm.IsEmpty() {
		t.Errorf("ConcurrentMap::IsEmpty() = false, size %d", cm.Size())
	}
}

func TestConcurrentMapInt(t *testing.T) {
	key, val := 1, 10
	cm := NewConcurrentMap()
	cm.Put(key, val)
	if size := cm.Size(); size != 1 {
		t.Errorf("ConcurrentMap::Size() = %d, want 1", size)
	}

	var ret interface{}
	var ok bool
	if ret, ok = cm.Get(key); !ok {
		t.Errorf("ConcurrentMap::Get(%d) not ok", key)
	}
	if ret.(int) != val {
		t.Errorf("ConcurrentMap::Get(%d) = %d, want %d", key, ret.(int), val)
	}
}

func TestConcurrentMapString(t *testing.T) {
	keysMap := map[string]bool{
		"Lucy":  false,
		"Lily":  false,
		"Kathy": false,
		"Joana": false,
		"Belle": false,
		"Fiona": false,
	}

	valuesMap := map[string]bool{
		"Product Manager": false,
		"Rust Programmer": false,
		"Python":          false,
		"Golang":          false,
		"Java":            false,
		"Javascript":      false,
	}

	keyValueMap := map[string]string{
		"Lucy":  "Product Manager",
		"Lily":  "Rust Programmer",
		"Kathy": "Python",
		"Joana": "Golang",
		"Belle": "Java",
		"Fiona": "Javascript",
	}

	cm := NewConcurrentMap()
	cm.Put("Lucy", "Product Manager")
	cm.Put("Lily", "C++")
	cm.Put("Kathy", "Python")
	cm.Put("Joana", "Golang")
	cm.Put("Belle", "Java")
	cm.PutIfAbsent("Joana", "Objective-C")
	cm.PutIfAbsent("Fiona", "Javascript")

	if size := cm.Size(); size != 6 {
		t.Errorf("ConcurrentMap::Size() = %d, want 6", size)
	}
	cm.Put("Lily", "Rust Programmer")

	for key := range cm.IterKeys() {
		keysMap[key.(string)] = true
	}

	for k, v := range keysMap {
		if !v {
			t.Errorf("Key %s not in ConcurrentMap", k)
		}
	}

	for value := range cm.IterValues() {
		valuesMap[value.(string)] = true
	}

	for k, v := range valuesMap {
		if !v {
			t.Errorf("Value %s not in ConcurrentMap", k)
		}
	}

	for item := range cm.IterItems() {
		mapKey := item.Key.(string)
		mapVal := item.Value.(string)
		if keyValueMap[mapKey] != mapVal {
			t.Errorf("ConcurrentMap[%s] != %s", mapKey, keyValueMap[mapKey])
		}
	}

	cm.Remove("Lucy")

	if ok := cm.ContainsKey("Lucy"); ok {
		t.Error(`ConcurrentMap::ContainsKey("Lucy") = true`)
	}

	if ok := cm.ContainsKey("Joana"); !ok {
		t.Error(`ConcurrentMap::ContainsKey("Joana") = false`)
	}

	cm.Clear()
	if !cm.IsEmpty() {
		t.Errorf("ConcurrentMap::IsEmpty() = false, size %d", cm.Size())
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

	if cnt := ai64.Get(); cnt != 3 {
		t.Errorf("AtomicInt64::Get() = %d, want 3", cnt)
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			ai64.GetAndDecrement()
			wg.Done()
		}()
	}
	wg.Wait()

	if cnt := ai64.Get(); cnt != 0 {
		t.Errorf("AtomicInt64::Get() = %d, want 0", cnt)
	}
}
