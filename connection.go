package tao

import (
  "log"
  "net"
  "sync"
  "time"
)

const (
  NTYPE = 4
  NLEN = 4
  MAXLEN = 1 << 23  // 8M
)

type Connection interface{
  SetNetId(netid int64)
  GetNetId() int64

  SetName(name string)
  GetName() string

  SetHeartBeat(beat int64)
  GetHeartBeat() int64

  SetExtraData(extra interface{})
  GetExtraData()interface{}

  SetMessageCodec(Codec codec)
  GetMessageCodec() Codec

  SetOnConnectCallback(callback func(Connection) bool)
  GetOnConnectCallback() onConnectFunc

  SetOnMessageCallback(callback func(Message, Connection))
  GetOnMessageCallback() onMessageFunc

  SetOnErrorCallback(callback func())
  GetOnErrorCallback() onErrorFunc

  SetOnCloseCallback(callback func(Connection))
  GetOnCloseCallback() onCloseFunc

  Start()
  Close()
  IsClosed() bool
  Write(message Message)

  RunAt(t time.Time, cb func(time.Time, interface{})) int64
  RunAfter(d time.Duration, cb func(time.Time, interface{})) int64
  RunEvery(i time.Duration, cb func(time.Time, interface{})) int64
  GetTimingWheel() *TimingWheel
  GetPendingTimers() []int64
  CancelTimer(timerId int64)

  GetRawConn() net.Conn
  GetMessageSendChannel() chan []byte
  GetHandlerReceiveChannel() chan MessageHandler
  GetCloseChannel() chan struct{}
  GetTimeOutChannel() chan *OnTimeOut

  GetRemoteAddress() string
  String() string
}

type Reconnectable interface{
  Reconnect()
}

type ServerSide interface{
  GetOwner() *TCPServer
}

type ServerSideConnection interface{
  Connection
  ServerSide
}

type ClientSideConnection interface{
  Connection
  Reconnectable
}

type ServerConnection struct{
  netid int64
  name string
  heartBeat int64
  extraData interface{}
  owner *TCPServer
  isClosed *AtomicBoolean
  once sync.Once
  pendingTimers []int64
  timingWheel *TimingWheel
  conn net.Conn

  messageSendChan chan []byte
  handlerRecvChan chan MessageHandler
  closeConnChan chan struct{}
  timeOutChan chan *OnTimeOut
}

func (server *ServerConnection)SetNetId(netid int64) {
  server.netid = netid
}

func (server *ServerConnection)GetNetId() int64 {
  return server.netid
}

func (server *ServerConnection)SetName(name string) {
  server.name = name
}

func (server *ServerConnection)GetName() string {
  return server.name
}

func (server *ServerConnection)SetHeartBeat(beat int64) {
  server.heartBeat = beat
}

func (server *ServerConnection)GetHeartBeat() int64 {
  return server.heartBeat
}

func (server *ServerConnection)SetExtraData(extra interface{}) {
  server.extraData = extra
}

func (server *ServerConnection)GetExtraData()interface{} {
  return server.extraData
}

func (server *ServerConnection)GetOwner() *TCPServer{
  return server.owner
}

func (server *ServerConnection)Start() {
  if server.GetOnConnectCallback() != nil {
    server.GetOnConnectCallback()(server)
  }
  server.GetOwner().wg.Add(3)
  loopers := []func(){readLoop, writeLoop, handleLoop}
  for _, looper := range loopers{
    go func() {
      looper()
      server.GetOwner().wg.Done()
    }()
  }
}

func (server *ServerConnection)Close() {
  server.once.Do(func(){
    if server.isClosed.CompareAndSet(false, true) {
      close(server.GetCloseChannel())
      close(server.GetMessageSendChannel())
      close(server.GetHandlerReceiveChannel())
      close(server.GetTimeOutChannel())
      server.GetRawConn().Close()

      if (server.GetOnCloseCallback() != nil) {
        server.GetOnCloseCallback()(client)
      }

      server.GetOwner().connections.Remove(server.GetNetId())
      for _, id := range server.GetPendingTimers() {
        server.CancelTimer(id)
      }

      // if server.isServerMode() {
      //   if !server.GetOwner().connections.Remove(server.GetNetId()) {
      //     log.Println("Remove error!")
      //   }
      //   for _, id := range server.GetPendingTimers() {
      //     server.CancelTimer(id)
      //   }
      // } else {
      //   server.Reconnect()
      // }
      // log.Println("Connection closed, now ", server.GetOwner().connections.Size())
  })
}

func (server *ServerConnection)IsClosed() bool {
  return server.isClosed.Get()
}

func (server *ServerConnection)GetPendingTimers() []int64 {
  return server.pendingTimers
}

func (server *ServerConnection)Write(message Message) error {
  return asyncWrite(server, message)
}

func (server *ServerConnection)RunAt(timestamp time.Time, callback func(time.Time, interface{})) int64 {
  return runAt(server, timestamp, callback)
}

func (server *ServerConnection)RunAfter(duration time.Duration, callback func(time.Time, interface{})) int64 {
  return runAfter(server, duration, callback)
}

func (server *ServerConnection)RunEvery(interval time.Duration, callback func(time.Time, interface{})) int64 {
  return runEvery(server, interval, callback)
}

func (server *ServerConnection)GetTimingWheel() *TimingWheel {
  return server.timingWheel
}

func (server *ServerConnection)GetPendingTimers() []int64 {
  return server.pendingTimers
}

func (server *ServerConnection)CancelTimer(timerId int64) {
  client.GetTimingWheel().CancelTimer(timerId)
}

func (server *ServerConnection)GetRawConn() net.Conn {
  return server.conn
}

func (server *ServerConnection)GetMessageSendChannel() chan []byte {
  return server.messageSendChan
}

func (server *ServerConnection)GetHandlerReceiveChannel() chan MessageHandler {
  return server.handlerRecvChan
}

func (server *ServerConnection)GetCloseChannel() chan struct{} {
  return server.closeChan
}

func (server *ServerConnection)GetTimeOutChannel() chan *OnTimeOut {
  return server.timeOutChan
}

func runAt(conn Connection, timestamp time.Time, callback func(time.Time, interface{})) {
  timeout := NewOnTimeOut(conn.GetNetId(), callback)
  var id int64 = -1
  if conn.GetTimingWheel() != nil {
    id = conn.GetTimingWheel().AddTimer(timestamp, 0, timeout)
    if id >= 0 {
      conn.GetPendingTimers() = append(conn.GetPendingTimers(), id)
    }
  }
  return id
}

func runAfter(conn Connection, duration time.Duration, callback func(time.Time, interface{})) int64 {
  delay := time.Now().Add(duration)
  var id int64 = -1
  if conn.GetTimingWheel() != nil {
    id = conn.RunAt(delay, callback)
  }
  return id
}

func runEvery(conn Connection, interval time.Duration, callback func(time.Time, interface{})) int64 {
  delay := time.Now().Add(interval)
  timeout := NewOnTimeOut(conn.GetNetId(), callback)
  var id int64 = -1
  if conn.GetTimingWheel() != nil {
    id = client.GetTimingWheel().AddTimer(delay, interval, timeout)
    if id >= 0 {
      client.GetPendingTimers() = append(client.GetPendingTimers(), id)
    }
  }
  return id
}

func asyncWrite(conn Connection, message Message) error {
  packet, err := conn.GetMessageCodec().Encode(message)
  if err != nil {
    return err
  }

  select {
  case conn.GetMessageSendChannel()<- packet:
    return nil
  default:
    return ErrorWouldBlock
  }
}

/* readLoop() blocking read from connection, deserialize bytes into message,
then find corresponding handler, put it into channel */
func readLoop(conn Connection) {
  defer func() {
    recover()
    conn.Close()
  }()

  for {
    select {
    case <-conn.GetCloseChannel():
      log.Println("readLoop get close")
      return

    default:
    }

    msg, err := conn.GetMessageCodec().Decode(conn)
    if err != nil {
      log.Printf("Error decoding message - %s", err)
      if err == ErrorUndefined {
        // update heart beat timestamp
        conn.SetHeartBeat(time.Now().UnixNano())
        continue
      }
      return
    }

    // update heart beat timestamp
    conn.SetHeartBeat(time.Now().UnixNano())
    handlerFactory := HandlerMap.Get(msg.MessageNumber())
    if handlerFactory == nil {
      if client.GetOnMessageCallback() != nil {
        log.Printf("Message %d call onMessage()\n", msg.MessageNumber())
        conn.GetOnMessageCallback()(msg, conn)
      } else {
        log.Printf("No handler or onMessage() found for message %d", msg.MessageNumber())
      }
      continue
    }

    // send handler to handleLoop
    handler := handlerFactory(conn.NetId(), msg)
    conn.GetHandlerReceiveChannel()<- handler
  }
}

/* writeLoop() receive message from channel, serialize it into bytes,
then blocking write into connection */
func writeLoop(conn Connection) {
  defer func() {
    recover()
    for packet := range conn.GetMessageSendChannel() {
      if packet != nil {
        if _, err := conn.GetRawConn().Write(packet); err != nil {
          log.Printf("Error writing data - %s\n", err)
        }
      }
    }
    conn.Close()
  }()

  for {
    select {
    case <-conn.GetCloseChannel():
      log.Println("writeLoop get close")
      return

    case packet := <-conn.GetMessageSendChannel():
      if packet != nil {
        if _, err := conn.GetRawConn.Write(packet); err != nil {
          log.Printf("Error writing data - %s\n", err)
        }
      }
    }
  }
}

// handleServerLoop() - put handler or timeout callback into worker go-routines
func handleServerLoop(conn ServerSideConnection) {
  defer func() {
    recover()
    conn.Close()
  }()

  for {
    select {
    case <-conn.GetCloseChannel():
      log.Println("handleServerMode get close")
      return

    case handler := <-conn.GetHandlerReceiveChannel():
      if !isNil(handler) {
        conn.GetOwner().workerPool.Put(conn.netid, func() {
          handler.Process(conn)
        })
      }

    case timeout := <-conn.GetTimeOutChannel():
      if timeout != nil {
        extraData := timeout.ExtraData.(int64)
        if extraData != conn.GetNetId() {
          log.Printf("[Warn] time out of %d running on client %d", extraData, conn.GetNetId())
        }
        conn.GetOwner().workerPool.Put(conn.GetNetId(), func() {
          timeout.Callback(time.Now(), conn)
        })
      }
    }
  }
}

// handleClientLoop() - run handler or timeout callback in handleLoop() go-routine
func handleClientLoop(conn Connection) {
  defer func() {
    recover()
    conn.Close()
  }()

  for {
    select {
    case <-conn.GetCloseChannel():
      return

    case handler := <-conn.GetHandlerReceiveChannel():
      if !isNil(handler) {
        handler.Process(conn)
      }

    case timeout := <-conn.GetTimeOutChannel():
      if timeout != nil {
        extraData := timeout.ExtraData.(int64)
        if extraData != conn.GetNetId() {
          log.Printf("[Warn] time out of %d running on client %d", extraData, conn.GetNetId())
        }
        timeout.Callback(time.Now(), conn)
      }
    }
  }
}



// type TCPConnection struct {
//   netid int64
//   Owner *TCPServer
//   conn net.Conn
//   address string
//   name string
//   closeOnce sync.Once
//   wg *sync.WaitGroup
//   timing *TimingWheel
//   messageSendChan chan []byte
//   handlerRecvChan chan MessageHandler
//   closeConnChan chan struct{}
//   timeOutChan chan *OnTimeOut
//   HeartBeat int64
//   pendingTimers []int64
//   reconnect bool
//   closed *AtomicBoolean
//   messageCodec Codec
//   extraData interface{}
//   onConnect onConnectFunc
//   onMessage onMessageFunc
//   onClose onCloseFunc
//   onError onErrorFunc
// }
//
// func ClientTCPConnection(id int64, addr string, t *TimingWheel, reconn bool) *TCPConnection {
//   c, err := net.Dial("tcp", addr)
//   if err != nil {
//     log.Fatalln(err)
//   }
//   return &TCPConnection {
//     netid: id,
//     conn: c,
//     address: addr,
//     wg: &sync.WaitGroup{},
//     timing: t,
//     messageSendChan: make(chan []byte, 1024),
//     handlerRecvChan: make(chan MessageHandler, 1024),
//     closeConnChan: make(chan struct{}),
//     timeOutChan: make(chan *OnTimeOut),
//     HeartBeat: time.Now().UnixNano(),
//     pendingTimers: []int64{},
//     reconnect: reconn,
//     closed: NewAtomicBoolean(false),
//     messageCodec: TypeLengthValueCodec{},
//   }
// }
//
// func ServerTCPConnection(id int64, s *TCPServer, c net.Conn) *TCPConnection {
//   tcpConn := &TCPConnection {
//     netid: id,
//     Owner: s,
//     conn: c,
//     wg: &sync.WaitGroup{},
//     timing: s.timing,
//     messageSendChan: make(chan []byte, 1024),
//     handlerRecvChan: make(chan MessageHandler, 1024),
//     closeConnChan: make(chan struct{}),
//     timeOutChan: make(chan *OnTimeOut),
//     HeartBeat: time.Now().UnixNano(),
//     pendingTimers: []int64{},
//     reconnect: false,
//     closed: NewAtomicBoolean(false),
//     messageCodec: TypeLengthValueCodec{},
//   }
//   tcpConn.SetOnConnectCallback(s.onConnect)
//   tcpConn.SetOnMessageCallback(s.onMessage)
//   tcpConn.SetOnErrorCallback(s.onError)
//   tcpConn.SetOnCloseCallback(s.onClose)
//   return tcpConn
// }
//
// func (client *TCPConnection)SetExtraData(data interface{}) {
//   client.extraData = data
// }
//
// func (client *TCPConnection)GetExtraData() interface{} {
//   return client.extraData
// }
//
// func (client *TCPConnection)NetId() int64 {
//   return client.netid
// }
//
// func (client *TCPConnection)Reconnect() {
//   netid := client.netid
//   address := client.address
//   timing := client.timing
//   reconnect := client.reconnect
//   client = ClientTCPConnection(netid, address, timing, reconnect)
//   client.Do()
// }
//
// func (client *TCPConnection)SetCodec(cdc Codec) {
//   client.messageCodec = cdc
// }
//
// func (client *TCPConnection)isServerMode() bool {
//   return client.Owner != nil
// }
//
// func (client *TCPConnection)SetOnConnectCallback(cb func(*TCPConnection) bool) {
//   if cb != nil {
//     client.onConnect = onConnectFunc(cb)
//   }
// }
//
// func (client *TCPConnection)SetOnMessageCallback(cb func(Message, *TCPConnection)) {
//   if cb != nil {
//     client.onMessage = onMessageFunc(cb)
//   }
// }
//
// func (client *TCPConnection)SetOnErrorCallback(cb func()) {
//   if cb != nil {
//     client.onError = onErrorFunc(cb)
//   }
// }
//
// func (client *TCPConnection)Wait() {
//   client.wg.Wait()
// }
//
// func (client *TCPConnection)SetOnCloseCallback(cb func(*TCPConnection)) {
//   if cb != nil {
//     client.onClose = onCloseFunc(cb)
//   }
// }
//
// func (client *TCPConnection)RemoteAddr() net.Addr {
//   return client.conn.RemoteAddr()
// }
//
// func (client *TCPConnection)SetName(n string) {
//   client.name = n
// }
//
// func (client *TCPConnection)String() string {
//   return client.name
// }
//
// func (client *TCPConnection)IsClosed() bool {
//   return client.closed.Get()
// }
//

// }
//



//

//


//
// func (client *TCPConnection)startLoop(looper func()) {
//   log.Println("WaitGroup ADD")
//   client.wg.Add(1)
//   go func() {
//     looper()
//     log.Println("WaitGroup Done")
//     client.wg.Done()
//   }()
// }
//
// func (client *TCPConnection) RawConn() net.Conn {
//   return client.conn
// }
//
