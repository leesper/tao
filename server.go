package tao

import (
  "log"
  "net"
  "os"
)

type TcpServer struct {
  running *AtomicBoolean
  connections *ConcurrentMap
  netids *AtomicInt64
  timing *TimingWheel
  workerPool *WorkerPool
  address string
  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc
}

// todo: make it configurable
func NewTcpServer(addr string) *TcpServer {
  server := &TcpServer {
    running: NewAtomicBoolean(true),
    connections: NewConcurrentMap(),
    netids: NewAtomicInt64(0),
    timing: NewTimingWheel(),
    workerPool: NewWorkerPool(10),
    address: addr,
  }
  go server.timeOutLoop()
  return server
}

func (server *TcpServer) timeOutLoop() {
  for {
    select {
    case timeout := <-server.timing.TimeOutChan:
      netid := timeout.ExtraData.(int64)
      if conn, ok := server.connections.Get(netid); ok {
        tcpConn := conn.(*TcpConnection)
        tcpConn.timeOutChan<- timeout
      } else {
        log.Printf("Invalid client %d\n", netid)
      }
    }
  }
}

func (server *TcpServer) SetOnConnectCallback(cb func(*TcpConnection) bool) {
  server.onConnect = onConnectFunc(cb)
}

func (server *TcpServer) SetOnMessageCallback(cb func(Message, *TcpConnection)) {
  server.onMessage = onMessageFunc(cb)
}

func (server *TcpServer) SetOnErrorCallback(cb func()) {
  server.onError = onErrorFunc(cb)
}

func (server *TcpServer) SetOnCloseCallback(cb func(*TcpConnection)) {
  server.onClose = onCloseFunc(cb)
}

func (server *TcpServer) Start(keepAlive bool) {
  listener, err := net.Listen("tcp", server.address)
  if err != nil {
    log.Fatalln(err)
  }
  defer listener.Close()

  for server.running.Get() {
    conn, err := listener.Accept()
    if err != nil {
      log.Fatalln(err)
    }
    netid := server.netids.GetAndIncrement()
    tcpConn := ServerTCPConnection(netid, server, conn, keepAlive)
    tcpConn.SetName(tcpConn.RemoteAddr().String())
    server.connections.Put(netid, tcpConn)
    log.Printf("Accepting client %s, net id %d\n", tcpConn, netid)
    tcpConn.Do()
  }
}

func (server *TcpServer) Close() {
  server.running.CompareAndSet(true, false)
  for v := range server.connections.IterValues() {
    c := v.(*TcpConnection)
    c.Close()
    c.wg.Wait()
  }
  os.Exit(0)
}

func (server *TcpServer) GetAllConnections() *ConcurrentMap {
  return server.connections
}
