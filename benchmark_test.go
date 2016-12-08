package tao

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

var (
	listN  int
	number int
	list   [][]interface{}
	cMap   *ConcurrentMap
	lMap   *lockMap
)

func init() {
	MAXPROCS := runtime.NumCPU()
	runtime.GOMAXPROCS(MAXPROCS)
	listN = MAXPROCS + 1
	number = 100000
	fmt.Println("MAXPROCS is ", MAXPROCS, ", listN is", listN, ", n is ", number, "\n")

	list = make([][]interface{}, listN, listN)
	for i := 0; i < listN; i++ {
		list1 := make([]interface{}, 0, number)
		for j := 0; j < number; j++ {
			list1 = append(list1, j+i*number/10)
		}
		list[i] = list1
	}

	cMap = NewConcurrentMap()
	lMap = newLockMap()
	for i := range list[0] {
		cMap.Put(i, i)
		lMap.put(i, i)
	}
}

type lockMap struct {
	m map[interface{}]interface{}
	sync.RWMutex
}

func (t *lockMap) put(k interface{}, v interface{}) {
	t.Lock()
	t.m[k] = v
	t.Unlock()
}

func (t *lockMap) putIfNotExist(k interface{}, v interface{}) (ok bool) {
	t.Lock()
	if _, ok = t.m[k]; !ok {
		t.m[k] = v
	}
	t.Unlock()
	return
}

func (t *lockMap) get(k interface{}) (v interface{}, ok bool) {
	t.RLock()
	v, ok = t.m[k]
	t.RUnlock()
	return
}

func (t *lockMap) len() int {
	t.RLock()
	return len(t.m)
	t.RUnlock()
}

func newLockMap() *lockMap {
	return &lockMap{
		m: make(map[interface{}]interface{}),
	}
}

func newLockMap1(initCap int) *lockMap {
	return &lockMap{
		m: make(map[interface{}]interface{}, initCap),
	}
}

func BenchmarkLockMapPut(b *testing.B) {
	cm := newLockMap()
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			k := i
			go func() {
				for _, j := range list[k] {
					cm.put(j, j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkConcurrentMapPut(b *testing.B) {
	cm := NewConcurrentMap()
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			k := i
			go func() {
				for _, j := range list[k] {
					cm.Put(j, j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkLockMapPutNoGrow(b *testing.B) {
	cm := newLockMap1(listN * number)
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			k := i
			go func() {
				for _, j := range list[k] {
					cm.put(j, j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkConcurrentMapPutNoGrow(b *testing.B) {
	cm := NewConcurrentMap()
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			k := i
			go func() {
				for _, j := range list[k] {
					cm.Put(j, j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkLockMapPut2(b *testing.B) {
	cm := newLockMap()
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			k := i
			go func() {
				for _, j := range list[k] {
					cm.put(strconv.Itoa(j.(int)), j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkConcurrentMapPut2(b *testing.B) {
	cm := NewConcurrentMap()
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			k := i
			go func() {
				for _, j := range list[k] {
					cm.Put(strconv.Itoa(j.(int)), j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkLockMapGet(b *testing.B) {
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			go func() {
				for k := range list[0] {
					_, _ = lMap.get(k)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkConcurrentMapGet(b *testing.B) {
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			go func() {
				for k := range list[0] {
					_, _ = cMap.Get(k)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkLockMapPutAndGet(b *testing.B) {
	cm := newLockMap()
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			k := i
			go func() {
				for _, j := range list[k] {
					cm.put(j, j)
					_, _ = cm.get(j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkConcurrentMapPutAndGet(b *testing.B) {
	cm := NewConcurrentMap()
	for n := 0; n < b.N; n++ {
		wg := new(sync.WaitGroup)
		wg.Add(listN)
		for i := 0; i < listN; i++ {
			k := i
			go func() {
				for _, j := range list[k] {
					cm.Put(j, j)
					_, _ = cm.Get(j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}
