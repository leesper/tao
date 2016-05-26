package tao

import (
  "log"
  "time"
  "sync"
  "container/heap"
)

var timerIds *AtomicInt64
func init() {
  log.SetFlags(log.Lshortfile | log.LstdFlags)
  timerIds = NewAtomicInt64(0)
}

// timerQueue is a thread-safe priority queue
type timerQueue struct {
  queue []*timerType
  sync.RWMutex
}

func newTimerQueue() *timerQueue {
  return &timerQueue{
    queue: make([]*timerType, 0),
  }
}

func (tq timerQueue) IterValues() chan *timerType {
  vch := make(chan *timerType)
  go func() {
    for _, t := range tq.queue {
      tq.RLock()
      defer tq.RUnlock()
      vch <- t
    }
  }()
  return vch
}

func (tq timerQueue) Len() int {
  tq.RLock()
  defer tq.RUnlock()
  return len(tq.queue)
}

func (tq timerQueue) Less(i, j int) bool {
  tq.RLock()
  defer tq.RUnlock()
  return tq.queue[i].expiration.UnixNano() < tq.queue[j].expiration.UnixNano()
}

func (tq timerQueue) Swap(i, j int) {
  tq.Lock()
  defer tq.Unlock()
  tq.queue[i], tq.queue[j] = tq.queue[j], tq.queue[i]
  tq.queue[i].index = i
  tq.queue[j].index = j
}

func (tq *timerQueue) Push(x interface{}) {
  tq.Lock()
  defer tq.Unlock()
  n := len(tq.queue)
  timer := x.(*timerType)
  timer.index = n
  tq.queue = append(tq.queue, timer)
}

func (tq *timerQueue) Pop() interface{} {
  tq.Lock()
  defer tq.Unlock()
  old := tq.queue
  n := len(old)
  timer := old[n-1]
  timer.index = -1
  tq.queue = old[0 : n-1]
  return timer
}

/* 'expiration' is the time when timer time out, if 'interval' > 0
the timer will time out periodically, 'timeout' contains the callback
to be called when times out */
type timerType struct {
  id int64
  expiration time.Time
  interval time.Duration
  timeout *OnTimeOut
  index int  // for container/heap
}

func newTimer(when time.Time, interv time.Duration, to *OnTimeOut) *timerType {
  return &timerType{
    id: timerIds.GetAndIncrement(),
    expiration: when,
    interval: interv,
    timeout: to,
  }
}

func (t *timerType) isRepeat() bool {
  return int64(t.interval) > 0
}

type TimingWheel struct {
  TimeOutChan chan *OnTimeOut
  timers *timerQueue
  ticker *time.Ticker
  finish *sync.WaitGroup
  quit chan struct{}
}

func NewTimingWheel() *TimingWheel {
  timingWheel := &TimingWheel{
    TimeOutChan: make(chan *OnTimeOut, 1024),
    timers: newTimerQueue(),
    ticker: time.NewTicker(time.Millisecond),
    finish: &sync.WaitGroup{},
    quit: make(chan struct{}),
  }
  heap.Init(timingWheel.timers)
  timingWheel.finish.Add(1)
  go func() {
    timingWheel.start()
    timingWheel.finish.Done()
  }()
  return timingWheel
}

func (tw *TimingWheel) AddTimer(when time.Time, interv time.Duration, to *OnTimeOut) int64 {
  if to == nil {
    return int64(-1)
  }
  timer := newTimer(when, interv, to)
  heap.Push(tw.timers, timer)
  return timer.id
}

func (tw *TimingWheel) Size() int {
  return tw.timers.Len()
}

func (tw *TimingWheel) CancelTimer(timerId int64) {
  index := -1
  for t := range tw.timers.IterValues() {
    if t.id == timerId {
      index = t.index
      break
    }
  }
  if index >= 0 { // found
    heap.Remove(tw.timers, index)
  }
}

func (tw *TimingWheel) Stop() {
  close(tw.quit)
  tw.finish.Wait()
}

func (tw *TimingWheel) getExpired() []*timerType {
  expired := make([]*timerType, 0)
  now := time.Now()
  for tw.timers.Len() > 0 {
    timer := heap.Pop(tw.timers).(*timerType)
    delta := float64(timer.expiration.UnixNano() - now.UnixNano()) / 1e9
    if -1.0 < delta && delta <= 0.0 {
      expired = append(expired, timer)
      continue
    } else {
      heap.Push(tw.timers, timer)
      break
    }
    if delta <= -1.0 {
      log.Println("DELTA ", delta)
    }
  }
  return expired
}

func (tw *TimingWheel) update(timers []*timerType) {
  if timers != nil {
    for _, t := range timers {
      if t.isRepeat() {
        t.expiration = t.expiration.Add(t.interval)
        heap.Push(tw.timers, t)
      }
    }
  }
}

func (tw *TimingWheel) start() {
  for {
    select {
    case <-tw.quit:
      tw.ticker.Stop()
      return

    case <-tw.ticker.C:
      timers := tw.getExpired()
      for _, t := range timers {
        tw.TimeOutChan<- t.timeout
      }
      tw.update(timers)

    }
  }
}
