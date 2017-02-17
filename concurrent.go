/* Data types and structues for concurrent use:
AtomicInt32
AtomicInt64
AtomicBoolean
ConcurrentMap
*/

package tao

import (
	"fmt"
	"sync/atomic"
)

// AtomicInt64 provides atomic int64 type.
type AtomicInt64 int64

// NewAtomicInt64 returns an atomic int64 type.
func NewAtomicInt64(initialValue int64) *AtomicInt64 {
	a := AtomicInt64(initialValue)
	return &a
}

// Get returns the value of int64 atomically.
func (a *AtomicInt64) Get() int64 {
	return int64(*a)
}

// Set sets the value of int64 atomically.
func (a *AtomicInt64) Set(newValue int64) {
	atomic.StoreInt64((*int64)(a), newValue)
}

// GetAndSet sets new value and returns the old atomically.
func (a *AtomicInt64) GetAndSet(newValue int64) int64 {
	for {
		current := a.Get()
		if a.CompareAndSet(current, newValue) {
			return current
		}
	}
}

// CompareAndSet compares int64 with expected value, if equals as expected
// then sets the updated value, this operation performs atomically.
func (a *AtomicInt64) CompareAndSet(expect, update int64) bool {
	return atomic.CompareAndSwapInt64((*int64)(a), expect, update)
}

// GetAndIncrement gets the old value and then increment by 1, this operation
// performs atomically.
func (a *AtomicInt64) GetAndIncrement() int64 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}

}

// GetAndDecrement gets the old value and then decrement by 1, this operation
// performs atomically.
func (a *AtomicInt64) GetAndDecrement() int64 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// GetAndAdd gets the old value and then add by delta, this operation
// performs atomically.
func (a *AtomicInt64) GetAndAdd(delta int64) int64 {
	for {
		current := a.Get()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// IncrementAndGet increments the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicInt64) IncrementAndGet() int64 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// DecrementAndGet decrements the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicInt64) DecrementAndGet() int64 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// AddAndGet adds the value by delta and then gets the value, this operation
// performs atomically.
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

// AtomicInt32 provides atomic int32 type.
type AtomicInt32 int32

// NewAtomicInt32 returns an atomoic int32 type.
func NewAtomicInt32(initialValue int32) *AtomicInt32 {
	a := AtomicInt32(initialValue)
	return &a
}

// Get returns the value of int32 atomically.
func (a *AtomicInt32) Get() int32 {
	return int32(*a)
}

// Set sets the value of int32 atomically.
func (a *AtomicInt32) Set(newValue int32) {
	atomic.StoreInt32((*int32)(a), newValue)
}

// GetAndSet sets new value and returns the old atomically.
func (a *AtomicInt32) GetAndSet(newValue int32) (oldValue int32) {
	for {
		oldValue = a.Get()
		if a.CompareAndSet(oldValue, newValue) {
			return
		}
	}
}

// CompareAndSet compares int64 with expected value, if equals as expected
// then sets the updated value, this operation performs atomically.
func (a *AtomicInt32) CompareAndSet(expect, update int32) bool {
	return atomic.CompareAndSwapInt32((*int32)(a), expect, update)
}

// GetAndIncrement gets the old value and then increment by 1, this operation
// performs atomically.
func (a *AtomicInt32) GetAndIncrement() int32 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}

}

// GetAndDecrement gets the old value and then decrement by 1, this operation
// performs atomically.
func (a *AtomicInt32) GetAndDecrement() int32 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// GetAndAdd gets the old value and then add by delta, this operation
// performs atomically.
func (a *AtomicInt32) GetAndAdd(delta int32) int32 {
	for {
		current := a.Get()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// IncrementAndGet increments the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicInt32) IncrementAndGet() int32 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// DecrementAndGet decrements the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicInt32) DecrementAndGet() int32 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// AddAndGet adds the value by delta and then gets the value, this operation
// performs atomically.
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

// AtomicBoolean provides atomic boolean type.
type AtomicBoolean int32

// NewAtomicBoolean returns an atomic boolean type.
func NewAtomicBoolean(initialValue bool) *AtomicBoolean {
	var a AtomicBoolean
	if initialValue {
		a = AtomicBoolean(1)
	} else {
		a = AtomicBoolean(0)
	}
	return &a
}

// Get returns the value of boolean atomically.
func (a *AtomicBoolean) Get() bool {
	return atomic.LoadInt32((*int32)(a)) != 0
}

// Set sets the value of boolean atomically.
func (a *AtomicBoolean) Set(newValue bool) {
	if newValue {
		atomic.StoreInt32((*int32)(a), 1)
	} else {
		atomic.StoreInt32((*int32)(a), 0)
	}
}

// CompareAndSet compares boolean with expected value, if equals as expected
// then sets the updated value, this operation performs atomically.
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

// GetAndSet sets new value and returns the old atomically.
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
