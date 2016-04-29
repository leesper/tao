package tao

import (
  "log"
  "net"
  "os"
)

type TcpServer struct {
  running *AtomicBoolean
  connections map[int64]*TcpConnection
  netids *AtomicInt64
  timing *TimingWheel
  onConnect onConnectCallbackType
  onMessage onMessageCallbackType
  onClose onCloseCallbackType
  onError onErrorCallbackType
}

// todo: make it configurable
func NewTcpServer() *TcpServer {
  return &TcpServer {
    running: NewAtomicBoolean(true),
    connections: make(map[int64]*TcpConnection),  // todo: make it thread-safe
    netids: NewAtomicInt64(0),
    timing: NewTimingWheel(),
  }
}

func (server *TcpServer) SetOnConnectCallback(cb func(*TcpConnection) bool) {
  server.onConnect = onConnectCallbackType(cb)
}

func (server *TcpServer) SetOnMessageCallback(cb func(Message, *TcpConnection)) {
  server.onMessage = onMessageCallbackType(cb)
}

func (server *TcpServer) SetOnErrorCallback(cb func()) {
  server.onError = onErrorCallbackType(cb)
}

func (server *TcpServer) SetOnCloseCallback(cb func(*TcpConnection)) {
  server.onClose = onCloseCallbackType(cb)
}

func (server *TcpServer) Start() {
  tcpAddr, err := net.ResolveTCPAddr("tcp", ":18341")
  if err != nil {
    log.Fatalln(err)
  }

  listener, err := net.ListenTCP("tcp", tcpAddr)
  if err != nil {
    log.Fatalln(err)
  }
  defer listener.Close()

  for server.running.Get() {
    rawConn, err := listener.AcceptTCP()
    if err != nil {
      log.Fatalln(err)
    }
    netid := server.netids.GetAndIncrement()
    tcpConn := NewTcpConnection(netid, server, rawConn, server.timing)
    tcpConn.SetName(tcpConn.RemoteAddr().String())
    server.connections[netid] = tcpConn
    log.Printf("Accepting client %s\n, net id %d", tcpConn, netid)
    tcpConn.Do()
  }
}

func (server *TcpServer) Close() {
  server.running.CompareAndSet(true, false)
  for _, c := range server.connections {
    c.Close()
    c.wg.Wait()
  }
  os.Exit(0)
}

func (server *TcpServer) GetAllConnections() map[int64]*TcpConnection {
  return server.connections
}
