package main

import (
	"fmt"
	"github.com/leesper/tao"
	"sync"
	"time"
)

func main() {
	wg := &sync.WaitGroup{}
	wheel := tao.NewTimingWheel()
	timerId := wheel.AddTimer(
		time.Now().Add(2*time.Second),
		1*time.Second,
		tao.NewOnTimeOut(nil, func(t time.Time, d interface{}) { fmt.Printf("TIME OUT AT %s\n", t) }))
	fmt.Printf("Add timer %d\n", timerId)

	wg.Add(1)
	go func() {
		for i := 0; i < 20; i++ {
			select {
			case timeout := <-wheel.GetTimeOutChannel():
				timeout.Callback(time.Now(), nil)
			}
		}
		wg.Done()
	}()
	wg.Wait()

	wheel.Stop()
}
