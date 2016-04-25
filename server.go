package tao

import (
  "log"
  "net"
  "sync"
)

type OnConnectCallback func() bool
type OnMessageCallback func()
type OnCloseCallback func(*TcpConnection)
type OnErrorCallback func()

type TcpServer struct {
  running *AtomicBoolean
  connections map[int64]TcpConnection
  netids *AtomicInt64
  wg *sync.WaitGroup
  onConnect OnConnectCallback
  onMessage OnMessageCallback
  onClose OnCloseCallback
  onError OnErrorCallback
}

// todo: make it configurable
func NewTcpServer() *TcpServer {
  return &TcpServer {
    running: NewAtomicBoolean(true)
    connections: make(map[int64]TcpConnection) // todo: make it thread-safe
    netids: NewAtomicInt64(0)
    wg: &sync.WaitGroup{}
  }
}

func (server *TcpServer) SetOnConnectCallback(cb OnConnectCallback) {
  server.onConnect = cb
}

func (server *TcpServer) SetOnMessageCallback(cb OnMessageCallback) {
  server.onMessage = cb
}

func (server *TcpServer) SetOnErrorCallback(cb OnErrorCallback) {
  server.onError = cb
}

func (server *TcpServer) SetOnCloseCallback(cb OnCloseCallBack) {
  server.onClose = cb
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
    tcp.Conn.SetName(tcpConn.RemoteAddr())
    server.connections[netid] = tcpConn
    log.Printf("Accepting client %s\n", tcpConn)
    tcpConn.Do()
  }
}

fuc (server *TcpServer) Close() {
  server.running.CompareAndSet(true, false)
  for _, c := range server.connections {
    c.Close()
  }
}
