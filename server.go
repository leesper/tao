package tao

import (
  "log"
  "net"
  "os"
  "time"
  "sync"
  // "crypto/tls"
  // "crypto/rand"
)

func init() {
  netIdentifier = NewAtomicInt64(0)
}

var (
  netIdentifier *AtomicInt64
)

type Server interface{
  IsRunning() bool
  GetAllConnections() *ConcurrentMap
  GetTimingWheel() *TimingWheel
  GetWorkerPool() *WorkerPool
  GetServerAddress() string
  Start()
  Close()

  SetOnScheduleCallback(time.Duration, func(time.Time, interface{}))
  GetOnScheduleCallback() (time.Duration, onScheduleFunc)
  SetOnConnectCallback(func(Connection)bool)
  GetOnConnectCallback() onConnectFunc
  SetOnMessageCallback(func(Message, Connection))
  GetOnMessageCallback() onMessageFunc
  SetOnCloseCallback(func(Connection))
  GetOnCloseCallback() onCloseFunc
  SetOnErrorCallback(func())
  GetOnErrorCallback() onErrorFunc
}

type TCPServer struct{
  isRunning *AtomicBoolean
  connections *ConcurrentMap
  timingWheel *TimingWheel
  workerPool *WorkerPool
  wg *sync.WaitGroup
  address string
  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc
  duration time.Duration
  onSchedule onScheduleFunc
}

func NewTCPServer(addr string) Server {
  server := &TCPServer {
    isRunning: NewAtomicBoolean(true),
    connections: NewConcurrentMap(),
    timingWheel: NewTimingWheel(),
    workerPool: NewWorkerPool(WORKERS),
    wg: &sync.WaitGroup{},
    address: addr,
  }
  go server.timeOutLoop()
  return server
}

func (server *TCPServer)IsRunning() bool {
  return server.isRunning.Get()
}

func (server *TCPServer)GetAllConnections() *ConcurrentMap {
  return server.connections
}

func (server *TCPServer)GetTimingWheel() *TimingWheel {
  return server.timingWheel
}

func (server *TCPServer)GetWorkerPool() *WorkerPool {
  return server.workerPool
}

func (server *TCPServer)GetServerAddress() string {
  return server.address
}

func (server *TCPServer) Start() {
  listener, err := net.Listen("tcp", server.address)
  if err != nil {
    log.Fatalln(err)
  }
  defer listener.Close()

  // var config tls.Config
  // if server.tlsMode {
  //   config = server.loadTLSConfig()
  // }

  for server.IsRunning() {
    conn, err := listener.Accept()
    if err != nil {
      log.Fatalln(err)
    }
    // if server.tlsMode {
    //   conn = tls.Server(conn, &config)  // wrap as a tls connection
    // }

    /* Create a TCP connection upon accepting a new client, assign an net id
    to it, then manage it in connections map, and start it */
    netid := netIdentifier.GetAndIncrement()
    tcpConn := NewServerConnection(netid, server, conn)
    tcpConn.SetName(tcpConn.GetRemoteAddress().String())
    if server.connections.Size() < MAX_CONNECTIONS {
      duration, onSchedule := server.GetOnScheduleCallback()
      if onSchedule != nil {
        tcpConn.RunEvery(duration, onSchedule)
      }
      server.connections.Put(netid, tcpConn)
      // FIXME: put tcpConn.Start() run in another WaitGroup-controlled go-routine
      tcpConn.Start()
      log.Printf("Accepting client %s, net id %d, now %d\n", tcpConn.GetName(), netid, server.connections.Size())
      for v := range server.connections.IterValues() {
        log.Printf("Client %s\n", v.(Connection).GetName())
      }
    } else {
      log.Printf("WARN, MAX CONNS %d, refuse\n", MAX_CONNECTIONS)
      tcpConn.Close()
    }
  }
}

// FIXME: wait until all go-routines finished
func (server *TCPServer) Close() {
  if server.isRunning.CompareAndSet(true, false) {
    for v := range server.connections.IterValues() {
      c := v.(Connection)
      c.Close()
    }
    os.Exit(0)
  }
}

func (server *TCPServer)GetTimeOutChannel() chan *OnTimeOut {
  return server.timingWheel.TimeOutChan
}

func (server *TCPServer)SetOnScheduleCallback(duration time.Duration, callback func(time.Time, interface{})) {
  server.duration = duration
  server.onSchedule = onScheduleFunc(callback)
}

func (server *TCPServer)GetOnScheduleCallback() (time.Duration, onScheduleFunc) {
  return server.duration, server.onSchedule
}

func (server *TCPServer)SetOnConnectCallback(callback func(Connection)bool) {
  server.onConnect = onConnectFunc(callback)
}

func (server *TCPServer)GetOnConnectCallback() onConnectFunc {
  return server.onConnect
}

func (server *TCPServer)SetOnMessageCallback(callback func(Message, Connection)) {
  server.onMessage = onMessageFunc(callback)
}

func (server *TCPServer)GetOnMessageCallback() onMessageFunc {
  return server.onMessage
}

func (server *TCPServer)SetOnCloseCallback(callback func(Connection)) {
  server.onClose = onCloseFunc(callback)
}

func (server *TCPServer)GetOnCloseCallback() onCloseFunc {
  return server.onClose
}

func (server *TCPServer)SetOnErrorCallback(callback func()) {
  server.onError = onErrorFunc(callback)
}

func (server *TCPServer)GetOnErrorCallback() onErrorFunc {
  return server.onError
}

// FIXME: redesign TCPServer, programming by interface
type TLSTCPServer struct{
  certFile string
  keyFile string
  *TCPServer
}

func NewTLSTCPServer(addr, cert, key string) Server {
  server := &TLSTCPServer {
    certFile: cert,
    keyFile: key,
    TCPServer: NewTCPServer(addr).(*TCPServer),
  }
  return server
}


/* Retrieve the extra data(i.e. net id), and then redispatch
timeout callbacks to corresponding client connection, this
prevents one client from running callbacks of other clients */
// FIXME: make it controlled by sync.WaitGroup
func (server *TCPServer) timeOutLoop() {
  for {
    select {
    case timeout := <-server.GetTimingWheel().TimeOutChan:
      netid := timeout.ExtraData.(int64)
      if conn, ok := server.connections.Get(netid); ok {
        tcpConn := conn.(Connection)
        if !tcpConn.IsClosed() {
          tcpConn.GetTimeOutChannel()<- timeout
        }
      } else {
        log.Printf("Invalid client %d\n", netid)
      }
    }
  }
}

// func (server *TCPServer) loadTLSConfig() tls.Config {
//   var config tls.Config
//   cert, err := tls.LoadX509KeyPair(server.certFile, server.keyFile)
//   if err != nil {
//     log.Fatalln(err)
//   }
//   config = tls.Config{Certificates: []tls.Certificate{cert}}
//   now := time.Now()
//   config.Time = func() time.Time { return now }
//   config.Rand = rand.Reader
//   return config
// }
