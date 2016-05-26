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

// timerQueueType is a priority queue based on container/heap
type timerQueueType []*timerType

func (tq timerQueueType) Len() int {
  return len(tq)
}

func (tq timerQueueType) Less(i, j int) bool {
  return tq[i].expiration.UnixNano() < tq[j].expiration.UnixNano()
}

func (tq timerQueueType) Swap(i, j int) {
  tq[i], tq[j] = tq[j], tq[i]
  tq[i].index = i
  tq[j].index = j
}

func (tq *timerQueueType) Push(x interface{}) {
  n := len(*tq)
  timer := x.(*timerType)
  timer.index = n
  *tq = append(*tq, timer)
}

func (tq *timerQueueType) Pop() interface{} {
  old := *tq
  n := len(old)
  timer := old[n-1]
  timer.index = -1
  *tq = old[0 : n-1]
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
  timers timerQueueType
  ticker *time.Ticker
  finish *sync.WaitGroup
  quit chan struct{}
}

func NewTimingWheel() *TimingWheel {
  timingWheel := &TimingWheel{
    TimeOutChan: make(chan *OnTimeOut, 1024),
    timers: make(timerQueueType, 0),
    ticker: time.NewTicker(time.Millisecond),
    finish: &sync.WaitGroup{},
    quit: make(chan struct{}),
  }
  heap.Init(&timingWheel.timers)
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
  heap.Push(&tw.timers, timer)
  return timer.id
}

func (tw *TimingWheel) Size() int {
  return tw.timers.Len()
}

func (tw *TimingWheel) CancelTimer(timerId int64) {
  index := -1
  for _, t := range tw.timers {
    if t.id == timerId {
      index = t.index
      break
    }
  }
  if index >= 0 { // found
    heap.Remove(&tw.timers, index)
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
    timer := heap.Pop(&tw.timers).(*timerType)
    delta := float64(timer.expiration.UnixNano() - now.UnixNano()) / 1e9
    if -1.0 < delta && delta <= 0.0 {
      expired = append(expired, timer)
      continue
    } else {
      heap.Push(&tw.timers, timer)
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
        heap.Push(&tw.timers, t)
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
