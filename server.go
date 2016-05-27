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

func init() {
  netIdentifier = NewAtomicInt64(0)
  tlsWrapper = func(conn net.Conn) net.Conn {
    return conn
  }
}

var (
  netIdentifier *AtomicInt64
  tlsWrapper func(net.Conn) net.Conn
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
  finish *sync.WaitGroup
  address string
  closeServChan chan struct{}

  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc

  duration time.Duration
  onSchedule onScheduleFunc
}

func NewTCPServer(addr string) Server {
  return &TCPServer {
    isRunning: NewAtomicBoolean(true),
    connections: NewConcurrentMap(),
    timingWheel: NewTimingWheel(),
    workerPool: NewWorkerPool(WORKERS),
    finish: &sync.WaitGroup{},
    address: addr,
    closeServChan: make(chan struct{}),
  }
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
  server.finish.Add(1)
  go server.timeOutLoop()

  listener, err := net.Listen("tcp", server.address)
  if err != nil {
    log.Fatalln(err)
  }
  defer listener.Close()

  for server.IsRunning() {
    conn, err := listener.Accept()
    if err != nil {
      log.Println("Accept error - ", err)
      continue
    }

    conn = tlsWrapper(conn)  // wrap as a tls connection if configured

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

      // put tcpConn.Start() run in another WaitGroup-synchronized go-routine
      server.finish.Add(1)
      go func() {
        tcpConn.Start()
      }()

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

// wait until all connections closed
func (server *TCPServer) Close() {
  if server.isRunning.CompareAndSet(true, false) {
    for v := range server.GetAllConnections().IterValues() {
      c := v.(Connection)
      c.Close()
    }
    close(server.closeServChan)
    server.finish.Wait()
    server.GetTimingWheel().Stop()
    os.Exit(0)
  }
}

func (server *TCPServer)GetTimeOutChannel() chan *OnTimeOut {
  return server.timingWheel.GetTimeOutChannel()
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

  config, err := loadTLSConfig(server.certFile, server.keyFile)
  if err != nil {
    log.Fatalln(err)
  }

  setTLSWrapper(func(conn net.Conn) net.Conn{
    return tls.Server(conn, &config)
  })

  return server
}

func (server *TLSTCPServer)IsRunning() bool {
  return server.TCPServer.IsRunning()
}

func (server *TLSTCPServer)GetAllConnections() *ConcurrentMap {
  return server.TCPServer.GetAllConnections()
}

func (server *TLSTCPServer)GetTimingWheel() *TimingWheel {
  return server.TCPServer.GetTimingWheel()
}

func (server *TLSTCPServer)GetWorkerPool() *WorkerPool {
  return server.TCPServer.GetWorkerPool()
}

func (server *TLSTCPServer)GetServerAddress() string {
  return server.TCPServer.GetServerAddress()
}

func (server *TLSTCPServer)Start() {
  server.TCPServer.Start()
}

func (server *TLSTCPServer)Close() {
  server.TCPServer.Close()
}

func (server *TLSTCPServer)SetOnScheduleCallback(duration time.Duration, callback func(time.Time, interface{})) {
  server.TCPServer.SetOnScheduleCallback(duration, callback)
}

func (server *TLSTCPServer)GetOnScheduleCallback() (time.Duration, onScheduleFunc) {
  return server.TCPServer.GetOnScheduleCallback()
}

func (server *TLSTCPServer)SetOnConnectCallback(callback func(Connection)bool) {
  server.TCPServer.SetOnConnectCallback(callback)
}

func (server *TLSTCPServer)GetOnConnectCallback() onConnectFunc {
  return server.TCPServer.GetOnConnectCallback()
}

func (server *TLSTCPServer)SetOnMessageCallback(callback func(Message, Connection)) {
  server.TCPServer.SetOnMessageCallback(callback)
}

func (server *TLSTCPServer)GetOnMessageCallback() onMessageFunc {
  return server.TCPServer.GetOnMessageCallback()
}

func (server *TLSTCPServer)SetOnCloseCallback(callback func(Connection)) {
  server.TCPServer.SetOnCloseCallback(callback)
}

func (server *TLSTCPServer)GetOnCloseCallback() onCloseFunc {
  return server.TCPServer.GetOnCloseCallback()
}

func (server *TLSTCPServer)SetOnErrorCallback(callback func()) {
  server.TCPServer.SetOnErrorCallback(callback)
}

func (server *TLSTCPServer)GetOnErrorCallback() onErrorFunc {
  return server.TCPServer.GetOnErrorCallback()
}


/* Retrieve the extra data(i.e. net id), and then redispatch
timeout callbacks to corresponding client connection, this
prevents one client from running callbacks of other clients */

func (server *TCPServer) timeOutLoop() {
  defer server.finish.Done()

  for {
    select {
    case <-server.closeServChan:
      return

    case timeout := <-server.GetTimingWheel().GetTimeOutChannel():
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

func loadTLSConfig(certFile, keyFile string) (tls.Config, error) {
  var config tls.Config
  cert, err := tls.LoadX509KeyPair(certFile, keyFile)
  return config, err
  config = tls.Config{Certificates: []tls.Certificate{cert}}
  now := time.Now()
  config.Time = func() time.Time { return now }
  config.Rand = rand.Reader
  return config, nil
}

func setTLSWrapper(wrapper func(conn net.Conn) net.Conn) {
  tlsWrapper = wrapper
}
