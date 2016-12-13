package tao

import (
	"container/heap"
	"sync"
	"time"

	"github.com/reechou/holmes"
)

var timerIds *AtomicInt64

func init() {
	timerIds = NewAtomicInt64(0)
}

// timerHeap is a heap-based priority queue
type timerHeapType []*timerType

func (heap timerHeapType) getIndexById(id int64) int {
	for _, t := range heap {
		if t.id == id {
			return t.index
		}
	}
	return -1
}

func (heap timerHeapType) Len() int {
	return len(heap)
}

func (heap timerHeapType) Less(i, j int) bool {
	return heap[i].expiration.UnixNano() < heap[j].expiration.UnixNano()
}

func (heap timerHeapType) Swap(i, j int) {
	heap[i], heap[j] = heap[j], heap[i]
	heap[i].index = i
	heap[j].index = j
}

func (heap *timerHeapType) Push(x interface{}) {
	n := len(*heap)
	timer := x.(*timerType)
	timer.index = n
	*heap = append(*heap, timer)
}

func (heap *timerHeapType) Pop() interface{} {
	old := *heap
	n := len(old)
	timer := old[n-1]
	timer.index = -1
	*heap = old[0 : n-1]
	return timer
}

/* 'expiration' is the time when timer time out, if 'interval' > 0
the timer will time out periodically, 'timeout' contains the callback
to be called when times out */
type timerType struct {
	id         int64
	expiration time.Time
	interval   time.Duration
	timeout    *OnTimeOut
	index      int // for container/heap
}

func newTimer(when time.Time, interv time.Duration, to *OnTimeOut) *timerType {
	return &timerType{
		id:         timerIds.GetAndIncrement(),
		expiration: when,
		interval:   interv,
		timeout:    to,
	}
}

func (t *timerType) isRepeat() bool {
	return int64(t.interval) > 0
}

type TimingWheel struct {
	timeOutChan chan *OnTimeOut
	timers      timerHeapType
	ticker      *time.Ticker
	finish      *sync.WaitGroup
	addChan     chan *timerType // add timer in loop
	cancelChan  chan int64      // cancel timer in loop
	sizeChan    chan int        // get size in loop
	quitChan    chan struct{}
}

func NewTimingWheel() *TimingWheel {
	timingWheel := &TimingWheel{
		timeOutChan: make(chan *OnTimeOut, 1024),
		timers:      make(timerHeapType, 0),
		ticker:      time.NewTicker(500 * time.Millisecond),
		finish:      &sync.WaitGroup{},
		addChan:     make(chan *timerType, 1024),
		cancelChan:  make(chan int64, 1024),
		sizeChan:    make(chan int),
		quitChan:    make(chan struct{}),
	}
	heap.Init(&timingWheel.timers)
	timingWheel.finish.Add(1)
	go func() {
		timingWheel.start()
		timingWheel.finish.Done()
	}()
	return timingWheel
}

func (tw *TimingWheel) GetTimeOutChannel() chan *OnTimeOut {
	return tw.timeOutChan
}

func (tw *TimingWheel) AddTimer(when time.Time, interv time.Duration, to *OnTimeOut) int64 {
	if to == nil {
		return int64(-1)
	}
	timer := newTimer(when, interv, to)
	tw.addChan <- timer
	return timer.id
}

func (tw *TimingWheel) Size() int {
	return <-tw.sizeChan
}

func (tw *TimingWheel) CancelTimer(timerId int64) {
	tw.cancelChan <- timerId
}

func (tw *TimingWheel) Stop() {
	close(tw.quitChan)
	tw.finish.Wait()
}

func (tw *TimingWheel) getExpired() []*timerType {
	expired := make([]*timerType, 0)
	now := time.Now()
	for tw.timers.Len() > 0 {
		timer := heap.Pop(&tw.timers).(*timerType)
		delta := float64(timer.expiration.UnixNano()-now.UnixNano()) / 1e9
		if -1.0 < delta && delta <= 0.0 {
			expired = append(expired, timer)
			continue
		} else {
			heap.Push(&tw.timers, timer)
			break
		}
		if delta <= -1.0 {
			holmes.Warn("Delta %f", delta)
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
		case timerId := <-tw.cancelChan:
			index := tw.timers.getIndexById(timerId)
			if index >= 0 {
				heap.Remove(&tw.timers, index)
			}

		case tw.sizeChan <- tw.timers.Len():

		case <-tw.quitChan:
			tw.ticker.Stop()
			return

		case timer := <-tw.addChan:
			heap.Push(&tw.timers, timer)

		case <-tw.ticker.C:
			timers := tw.getExpired()
			for _, t := range timers {
				tw.GetTimeOutChannel() <- t.timeout
			}
			tw.update(timers)
		}
	}
}
