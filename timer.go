package tao

import (
  "log"
  "time"
  "container/heap"
)

func init() {
  log.SetFlags(log.Lshortfile | log.LstdFlags)
}

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

func newTimer(when time.Time, interv time.Duration, cb func(time.Time)) *timerType {
  return &timerType{
    expiration: when,
    interval: interv,
    onTimeOut: onTimeOutCallbackType(cb),
  }
}

func (t *timerType) isRepeat() bool {
  return int64(t.interval) > 0
}

type TimingWheel struct {
  TimeOutChan chan onTimeOutCallbackType
  timers timerQueueType
  ticker *time.Ticker
  quit chan struct{}
}

func NewTimingWheel() *TimingWheel {
  timingWheel := &TimingWheel{
    TimeOutChan: make(chan onTimeOutCallbackType, 1024),
    timers: make(timerQueueType, 0),
    ticker: time.NewTicker(time.Millisecond),
    quit: make(chan struct{}),
  }
  heap.Init(&timingWheel.timers)
  go timingWheel.start()
  return timingWheel
}

func (tw *TimingWheel) AddTimer(when time.Time, interv time.Duration, cb func(time.Time)) {
  heap.Push(&tw.timers, newTimer(when, interv, cb))
}

func (tw *TimingWheel) Stop() {
  close(tw.quit)
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
        tw.TimeOutChan<- t.onTimeOut  // fixme: may block if channel full
      }
      tw.update(timers)
    }
  }
}
