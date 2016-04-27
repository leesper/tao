package tao

import (
  "time"
  "math"
  "container/heap"
)

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

type timerType struct {
  expiration time.Time
  interval time.Duration
  onTimeOut onTimeOutCallbackType
  index int  // for container/heap
}

func newTimer(cb func(time.Time), when time.Time, interv time.Duration) *timerType {
  return &timerType{
    expiration: when,
    interval: interv,
    onTimeOut: onTimeOutCallbackType(cb ),
  }
}

func (t *timerType) isRepeat() bool {
  return t.interval > 0
}

type TimingWheel struct {
  accuracy time.Duration
  timeOutChan chan onTimeOutCallbackType
  timers timerQueueType
  ticker *time.Ticker
  quit chan struct{}
}

func NewTimingWheel(unit time.Duration) *TimingWheel {
  timingWheel := &TimingWheel{
    accuracy: unit,
    timeOutChan: make(chan onTimeOutCallbackType, 1024),
    timers: make(timerQueueType, 0),
    ticker: time.NewTicker(unit),
    quit: make(chan struct{}),
  }
  heap.Init(&timingWheel.timers)
  go timingWheel.start()
  return timingWheel
}

func (tw *TimingWheel) AddTimer(cb func(time.Time), when time.Time, interv time.Duration) {
  heap.Push(&tw.timers, newTimer(cb, when, interv))
}

func (tw *TimingWheel) getExpired() []*timerType {
  expired := make([]*timerType, 0)
  now := time.Now()
  for tw.timers.Len() > 0 {
    timer := heap.Pop(&tw.timers).(*timerType)
    delta := math.Abs(float64(now.UnixNano() - timer.expiration.UnixNano()))
    if 0.0 <= delta && delta < 1.0 {
      expired = append(expired, timer)
    } else {
      heap.Push(&tw.timers, timer)
      break
    }
  }
  return expired
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
        tw.timeOutChan<- t.onTimeOut
      }
    }
  }
}
