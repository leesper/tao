package tao

import (
  "log"
  "net"
  "sync"
)

type onConnectCallbackType func() bool
type onMessageCallbackType func(*Message, *TcpConnection)
type onCloseCallbackType func(*TcpConnection)
type onErrorCallbackType func()

type TcpServer struct {
  running *AtomicBoolean
  connections map[int64]TcpConnection
  netids *AtomicInt64
  wg *sync.WaitGroup
  onConnect onConnectCallbackType
  onMessage onMessageCallbackType
  onClose onCloseCallbackType
  onError onErrorCallbackType
}

// todo: make it configurable
func NewTcpServer() *TcpServer {
  return &TcpServer {
    running: NewAtomicBoolean(true),
    connections: make(map[int64]TcpConnection),  // todo: make it thread-safe
    netids: NewAtomicInt64(0),
    wg: &sync.WaitGroup{},
  }
}

func (server *TcpServer) SetOnConnectCallback(cb func() bool) {
  server.onConnect = onConnectCallbackType(cb)
}

func (server *TcpServer) SetOnMessageCallback(cb func()) {
  server.onMessage = onMessageCallbackType(cb)
}

func (server *TcpServer) SetOnErrorCallback(cb func()) {
  server.onError = onErrorCallbackType(cb)
}

func (server *TcpServer) SetOnCloseCallback(cb func(*TcpConnection)) {
  server.onClose = onCloseCallbackType(cb)
}

func (server *TcpServer) Start() {
  listener, err := net.Listen("tcp", ":8341")
  if err != nil {
    log.Fatalln(err)
  }
  defer listener.Close()

  for server.running.Get() {
    rawConn, err := listener.Accept()
    if err != nil {
      log.Fatalln(err)
    }
    netid := server.netids.GetAndIncrement()
    tcpConn := NewTcpConnection(server, rawConn)
    tcpConn.SetName(tcpConn.RemoteAddr())
    server.connections[netid] = tcpConn
    log.Printf("Accepting client %s\n", tcpConn)
    tcpConn.Do()
  }
}

func (server *TcpServer) Close() {
  server.running.CompareAndSet(true, false)
  for _, c := range server.connections {
    c.Close()
  }
}
