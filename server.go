package tao

import (
  "log"
  "net"
  "os"
  "time"
  "sync"
  "crypto/tls"
  "crypto/rand"
)

type TCPServer struct {
  running *AtomicBoolean
  connections *ConcurrentMap
  netids *AtomicInt64
  timing *TimingWheel
  workerPool *WorkerPool
  wg *sync.WaitGroup
  address string
  tlsMode bool
  certFile string
  keyFile string
  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc
  scheduleFunc func(time.Time, interface{})
  scheduleDuration time.Duration
}

func NewTCPServer(addr string) *TCPServer {
  server := &TCPServer {
    running: NewAtomicBoolean(true),
    connections: NewConcurrentMap(),
    netids: NewAtomicInt64(0),
    timing: NewTimingWheel(),
    workerPool: NewWorkerPool(WORKERS),
    wg: &sync.WaitGroup{},
    address: addr,
    tlsMode: false,
  }
  go server.timeOutLoop()
  return server
}

func NewTLSTCPServer(addr, cert, key string) *TCPServer {
  server := &TCPServer {
    running: NewAtomicBoolean(true),
    connections: NewConcurrentMap(),
    netids: NewAtomicInt64(0),
    timing: NewTimingWheel(),
    workerPool: NewWorkerPool(WORKERS),
    wg: &sync.WaitGroup{},
    address: addr,
    tlsMode: true,
    certFile: cert,
    keyFile: key,
  }
  go server.timeOutLoop()
  return server
}

func (server *TCPServer) GetWorkPool() *WorkerPool {
  return server.workerPool
}

/* Retrieve the extra data(i.e. net id), and then redispatch
timeout callbacks to corresponding client connection, this
prevents one client from running callbacks of other clients */
func (server *TCPServer) timeOutLoop() {
  for {
    select {
    case timeout := <-server.timing.TimeOutChan:
      netid := timeout.ExtraData.(int64)
      if conn, ok := server.connections.Get(netid); ok {
        tcpConn := conn.(*TCPConnection)
        if !tcpConn.closed.Get() {
          tcpConn.timeOutChan<- timeout
        }
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

func (server *TCPServer) SetOnScheduleCallback(d time.Duration, cb func(time.Time, interface{})) {
  server.scheduleDuration = d
  server.scheduleFunc = cb
}

func (server *TCPServer) loadTLSConfig() tls.Config {
  var config tls.Config
  cert, err := tls.LoadX509KeyPair(server.certFile, server.keyFile)
  if err != nil {
    log.Fatalln(err)
  }
  config = tls.Config{Certificates: []tls.Certificate{cert}}
  now := time.Now()
  config.Time = func() time.Time { return now }
  config.Rand = rand.Reader
  return config
}

func (server *TCPServer) Start() {
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
      conn = tls.Server(conn, &config)  // wrap as a tls connection
    }

    /* Create a TCP connection upon accepting a new client, assign an net id
    to it, then manage it in connections map, and start it */
    netid := server.netids.GetAndIncrement()
    tcpConn := ServerTCPConnection(netid, server, conn)
    tcpConn.SetName(tcpConn.RemoteAddr().String())
    if server.connections.Size() < MAX_CONNECTIONS {
      if server.scheduleFunc != nil {
        tcpConn.RunEvery(server.scheduleDuration, server.scheduleFunc)
      }
      server.connections.Put(netid, tcpConn)
      tcpConn.Do()
      log.Printf("Accepting client %s, net id %d, now %d\n", tcpConn, netid, server.connections.Size())
      for v := range server.connections.IterValues() {
        log.Printf("Client %s\n", v)
      }
    } else {
      log.Printf("WARN, MAX CONNS %d, refuse\n", MAX_CONNECTIONS)
      tcpConn.Close()
    }
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
