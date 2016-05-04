package tao

import (
  "log"
)

type WorkerPool struct {
  workers []*worker
  closeChan chan struct{}
}

func NewWorkerPool(vol int) *WorkerPool {
  if vol <= 0 {
    vol = 10
  }

  pool := &WorkerPool{
    workers: make([]*worker, vol),
    closeChan: make(chan struct{}),
  }

  for i, _ := range pool.workers {
    pool.workers[i] = newWorker(i, 1024, pool.closeChan)
    if pool.workers[i] == nil {
      panic("worker nil")
    }
  }

  return pool
}

func (wp *WorkerPool) Put(k interface{}, cb func()) error {
  var code uint32
  var err error
  if code, err = hashCode(k); err != nil {
    return err
  }
  return wp.workers[code & uint32(len(wp.workers) - 1)].put(workerCallbackType(cb))
}

func (wp *WorkerPool) Close() {
  close(wp.closeChan)
}

type worker struct {
  index int
  callbackChan chan workerCallbackType
  closeChan chan struct{}
}

func newWorker(i int, c int, closeChan chan struct{}) *worker {
  w := &worker{
    index: i,
    callbackChan: make(chan workerCallbackType, c),
    closeChan: closeChan,
  }
  go w.start()
  return w
}

func (w *worker) start() {
  log.Printf("worker %d start\n", w.index)
  for {
    select {
    case <-w.closeChan:
      break
    case cb := <-w.callbackChan:
      cb()
    }
  }
  close(w.callbackChan)
}

func (w *worker) put(cb workerCallbackType) error {
  select {
  case w.callbackChan<- cb:
    return nil
  default:
    return ErrorWouldBlock
  }
}
