package tao

import (
  "log"
  "net"
  "os"
  "time"
  "crypto/tls"
  "crypto/rand"
)

type TCPServer struct {
  running *AtomicBoolean
  connections *ConcurrentMap
  netids *AtomicInt64
  timing *TimingWheel
  workerPool *WorkerPool
  address string
  tlsMode bool
  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc
}

// todo: make it configurable
func NewTCPServer(addr string, tls bool) *TCPServer {
  server := &TCPServer {
    running: NewAtomicBoolean(true),
    connections: NewConcurrentMap(),
    netids: NewAtomicInt64(0),
    timing: NewTimingWheel(),
    workerPool: NewWorkerPool(ServerConf.Workers),
    address: addr,
    tlsMode: tls,
  }
  go server.timeOutLoop()
  return server
}

func (server *TCPServer) timeOutLoop() {
  for {
    select {
    case timeout := <-server.timing.TimeOutChan:
      netid := timeout.ExtraData.(int64)
      if conn, ok := server.connections.Get(netid); ok {
        tcpConn := conn.(*TCPConnection)
        tcpConn.timeOutChan<- timeout
      } else {
        log.Printf("Invalid client %d\n", netid)
      }
    }
  }
}

func (server *TCPServer) SetOnConnectCallback(cb func(*TCPConnection) bool) {
  server.onConnect = onConnectFunc(cb)
}

func (server *TCPServer) SetOnMessageCallback(cb func(Message, *TCPConnection)) {
  server.onMessage = onMessageFunc(cb)
}

func (server *TCPServer) SetOnErrorCallback(cb func()) {
  server.onError = onErrorFunc(cb)
}

func (server *TCPServer) SetOnCloseCallback(cb func(*TCPConnection)) {
  server.onClose = onCloseFunc(cb)
}

func (server *TCPServer) loadTLSConfig() tls.Config {
  var config tls.Config
  cert, err := tls.LoadX509KeyPair(ServerConf.Certfile, ServerConf.Keyfile)
  if err != nil {
    log.Fatalln(err)
  }
  config = tls.Config{Certificates: []tls.Certificate{cert}}
  now := time.Now()
  config.Time = func() time.Time { return now }
  config.Rand = rand.Reader
  return config
}

func (server *TCPServer) Start(keepAlive bool) {
  listener, err := net.Listen("tcp", server.address)
  if err != nil {
    log.Fatalln(err)
  }
  defer listener.Close()

  var config tls.Config
  if server.tlsMode {
    config = server.loadTLSConfig()
  }

  for server.running.Get() {
    conn, err := listener.Accept()
    if err != nil {
      log.Fatalln(err)
    }
    if server.tlsMode {
      conn = tls.Server(conn, &config)
    }
    netid := server.netids.GetAndIncrement()
    tcpConn := ServerTCPConnection(netid, server, conn, keepAlive)
    tcpConn.SetName(tcpConn.RemoteAddr().String())
    server.connections.Put(netid, tcpConn)
    log.Printf("Accepting client %s, net id %d\n", tcpConn, netid)
    tcpConn.Do()
  }
}

func (server *TCPServer) Close() {
  server.running.CompareAndSet(true, false)
  for v := range server.connections.IterValues() {
    c := v.(*TCPConnection)
    c.Close()
    c.wg.Wait()
  }
  os.Exit(0)
}

func (server *TCPServer) GetAllConnections() *ConcurrentMap {
  return server.connections
}
