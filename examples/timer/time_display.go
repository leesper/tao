package main

import (
  "fmt"
  "time"
  "sync"
  "github.com/leesper/tao"
)

func main() {
  wg := &sync.WaitGroup{}
  wheel := tao.NewTimingWheel()
  wheel.AddTimer(
    time.Now().Add(2 * time.Second),
    50 * time.Millisecond,
    func(t time.Time) { fmt.Printf("TIME OUT AT %s\n", t) })

  wg.Add(1)
  go func() {
    for i := 0; i < 1000; i++ {
      select {
      case cb := <-wheel.TimeOutChan:
        cb(time.Now())
      }
    }
    wheel.Stop()
    wg.Done()
  }()

  wg.Wait()
}
