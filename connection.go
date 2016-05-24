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

  SetMessageCodec(codec Codec)
  GetMessageCodec() Codec

  SetPendingTimers(pending []int64)
  GetPendingTimers() []int64

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
  Write(message Message) error

  RunAt(t time.Time, cb func(time.Time, interface{})) int64
  RunAfter(d time.Duration, cb func(time.Time, interface{})) int64
  RunEvery(i time.Duration, cb func(time.Time, interface{})) int64
  GetTimingWheel() *TimingWheel

  CancelTimer(timerId int64)

  GetRawConn() net.Conn
  GetMessageSendChannel() chan []byte
  GetHandlerReceiveChannel() chan MessageHandler
  GetCloseChannel() chan struct{}
  GetTimeOutChannel() chan *OnTimeOut

  GetRemoteAddress() net.Addr
}

// type Reconnectable interface{
//   Reconnect()
// }

type ServerSide interface{
  GetOwner() *TCPServer
}

type ServerSideConnection interface{
  Connection
  ServerSide
}

type ClientSideConnection interface{
  Connection
  // Reconnectable
}

// implements ServerSideConnection
type ServerConnection struct{
  netid int64
  name string
  heartBeat int64
  extraData interface{}
  owner *TCPServer
  isClosed *AtomicBoolean
  once sync.Once
  pendingTimers []int64
  conn net.Conn
  messageCodec Codec
  finish *sync.WaitGroup

  messageSendChan chan []byte
  handlerRecvChan chan MessageHandler
  closeConnChan chan struct{}
  timeOutChan chan *OnTimeOut

  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc
}

func NewServerConnection(netid int64, server *TCPServer, conn net.Conn) Connection {
  serverConn := &ServerConnection{
    netid: netid,
    name: conn.RemoteAddr().String(),
    heartBeat: time.Now().UnixNano(),
    owner: server,
    isClosed: NewAtomicBoolean(false),
    pendingTimers: []int64{},
    conn: conn,
    messageCodec: TypeLengthValueCodec{},
    finish: &sync.WaitGroup{},
    messageSendChan: make(chan []byte, 1024),
    handlerRecvChan: make(chan MessageHandler, 1024),
    closeConnChan: make(chan struct{}),
    timeOutChan: make(chan *OnTimeOut),
  }
  serverConn.SetOnConnectCallback(server.onConnect)
  serverConn.SetOnMessageCallback(server.onMessage)
  serverConn.SetOnErrorCallback(server.onError)
  serverConn.SetOnCloseCallback(server.onClose)
  return serverConn
}

func (conn *ServerConnection)SetNetId(netid int64) {
  conn.netid = netid
}

func (conn *ServerConnection)GetNetId() int64 {
  return conn.netid
}

func (conn *ServerConnection)SetName(name string) {
  conn.name = name
}

func (conn *ServerConnection)GetName() string {
  return conn.name
}

func (conn *ServerConnection)SetHeartBeat(beat int64) {
  conn.heartBeat = beat
}

func (conn *ServerConnection)GetHeartBeat() int64 {
  return conn.heartBeat
}

func (conn *ServerConnection)SetExtraData(extra interface{}) {
  conn.extraData = extra
}

func (conn *ServerConnection)GetExtraData()interface{} {
  return conn.extraData
}

func (conn *ServerConnection)SetMessageCodec(codec Codec) {
  conn.messageCodec = codec
}

func (conn *ServerConnection)GetMessageCodec() Codec {
  return conn.messageCodec
}

func (conn *ServerConnection)GetOwner() *TCPServer{
  return conn.owner
}

func (conn *ServerConnection)Start() {
  if conn.GetOnConnectCallback() != nil {
    conn.GetOnConnectCallback()(conn)
  }

  conn.finish.Add(3)
  go func() {
    readLoop(conn, conn.finish)
  }()

  go func() {
    writeLoop(conn, conn.finish)
  }()

  go func() {
    handleServerLoop(conn, conn.finish)
  }()
}

func (conn *ServerConnection)Close() {
  conn.once.Do(func(){
    if conn.isClosed.CompareAndSet(false, true) {
      if (conn.GetOnCloseCallback() != nil) {
        conn.GetOnCloseCallback()(conn)
      }

      close(conn.GetCloseChannel())
      close(conn.GetMessageSendChannel())
      close(conn.GetHandlerReceiveChannel())
      close(conn.GetTimeOutChannel())

      // wait for all loops to finish
      conn.finish.Wait()

      conn.GetRawConn().Close()
      conn.GetOwner().connections.Remove(conn.GetNetId())
      for _, id := range conn.GetPendingTimers() {
        conn.CancelTimer(id)
      }
      conn.GetOwner().finish.Done()
    }
  })
}

func (conn *ServerConnection)IsClosed() bool {
  return conn.isClosed.Get()
}

func (conn *ServerConnection)SetPendingTimers(pending []int64) {
  conn.pendingTimers = pending
}

func (conn *ServerConnection)GetPendingTimers() []int64 {
  return conn.pendingTimers
}

func (conn *ServerConnection)Write(message Message) error {
  return asyncWrite(conn, message)
}

func (conn *ServerConnection)RunAt(timestamp time.Time, callback func(time.Time, interface{})) int64 {
  return runAt(conn, timestamp, callback)
}

func (conn *ServerConnection)RunAfter(duration time.Duration, callback func(time.Time, interface{})) int64 {
  return runAfter(conn, duration, callback)
}

func (conn *ServerConnection)RunEvery(interval time.Duration, callback func(time.Time, interface{})) int64 {
  return runEvery(conn, interval, callback)
}

func (conn *ServerConnection)GetTimingWheel() *TimingWheel {
  return conn.GetOwner().GetTimingWheel()
}

func (conn *ServerConnection)CancelTimer(timerId int64) {
  conn.GetTimingWheel().CancelTimer(timerId)
}

func (conn *ServerConnection)GetRawConn() net.Conn {
  return conn.conn
}

func (conn *ServerConnection)GetMessageSendChannel() chan []byte {
  return conn.messageSendChan
}

func (conn *ServerConnection)GetHandlerReceiveChannel() chan MessageHandler {
  return conn.handlerRecvChan
}

func (conn *ServerConnection)GetCloseChannel() chan struct{} {
  return conn.closeConnChan
}

func (conn *ServerConnection)GetTimeOutChannel() chan *OnTimeOut {
  return conn.timeOutChan
}

func (conn *ServerConnection)GetRemoteAddress() net.Addr {
  return conn.conn.RemoteAddr()
}

func (conn *ServerConnection)SetOnConnectCallback(callback func(Connection) bool) {
  conn.onConnect = onConnectFunc(callback)
}

func (conn *ServerConnection)GetOnConnectCallback() onConnectFunc {
  return conn.onConnect
}

func (server *ServerConnection)SetOnMessageCallback(callback func(Message, Connection)) {
  server.onMessage = onMessageFunc(callback)
}

func (conn *ServerConnection)GetOnMessageCallback() onMessageFunc {
  return conn.onMessage
}

func (conn *ServerConnection)SetOnErrorCallback(callback func()) {
  conn.onError = onErrorFunc(callback)
}

func (conn *ServerConnection)GetOnErrorCallback() onErrorFunc {
  return conn.onError
}

func (conn *ServerConnection)SetOnCloseCallback(callback func(Connection)) {
  conn.onClose = onCloseFunc(callback)
}

func (conn *ServerConnection)GetOnCloseCallback() onCloseFunc {
  return conn.onClose
}

// implements Connection
type ClientConnection struct{
  netid int64
  name string
  address string
  heartBeat int64
  extraData interface{}
  isClosed *AtomicBoolean
  once sync.Once
  pendingTimers []int64
  timingWheel *TimingWheel
  conn net.Conn
  messageCodec Codec
  finish *sync.WaitGroup
  reconnectable bool

  messageSendChan chan []byte
  handlerRecvChan chan MessageHandler
  closeConnChan chan struct{}

  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc
}

func NewClientConnection(netid int64, address string, reconnectable bool) Connection {
  c, err := net.Dial("tcp", address)
  if err != nil {
    log.Fatalln(err)
  }
  return &ClientConnection {
    netid: netid,
    name: c.RemoteAddr().String(),
    address: address,
    heartBeat: time.Now().UnixNano(),
    isClosed: NewAtomicBoolean(false),
    pendingTimers: []int64{},
    timingWheel: NewTimingWheel(),
    conn: c,
    messageCodec: TypeLengthValueCodec{},
    finish: &sync.WaitGroup{},
    reconnectable: reconnectable,
    messageSendChan: make(chan []byte, 1024),
    handlerRecvChan: make(chan MessageHandler, 1024),
    closeConnChan: make(chan struct{}),
  }
}

func (conn *ClientConnection)SetNetId(netid int64) {
  conn.netid = netid
}

func (conn *ClientConnection)GetNetId() int64 {
  return conn.netid
}

func (conn *ClientConnection)SetName(name string) {
  conn.name = name
}

func (conn *ClientConnection)GetName() string {
  return conn.name
}

func (conn *ClientConnection)SetHeartBeat(beat int64) {
  conn.heartBeat = beat
}

func (conn *ClientConnection)GetHeartBeat() int64 {
  return conn.heartBeat
}

func (conn *ClientConnection)SetExtraData(extra interface{}) {
  conn.extraData = extra
}

func (conn *ClientConnection)GetExtraData()interface{} {
  return conn.extraData
}

func (conn *ClientConnection)SetMessageCodec(codec Codec) {
  conn.messageCodec = codec
}

func (conn *ClientConnection)GetMessageCodec() Codec {
  return conn.messageCodec
}

func (conn *ClientConnection)SetOnConnectCallback(callback func(Connection) bool) {
  conn.onConnect = onConnectFunc(callback)
}

func (conn *ClientConnection)GetOnConnectCallback() onConnectFunc {
  return conn.onConnect
}

func (conn *ClientConnection)SetOnMessageCallback(callback func(Message, Connection)) {
  conn.onMessage = onMessageFunc(callback)
}

func (conn *ClientConnection)GetOnMessageCallback() onMessageFunc {
  return conn.onMessage
}

func (conn *ClientConnection)SetOnErrorCallback(callback func()) {
  conn.onError = onErrorFunc(callback)
}

func (conn *ClientConnection)GetOnErrorCallback() onErrorFunc {
  return conn.onError
}

func (conn *ClientConnection)SetOnCloseCallback(callback func(Connection)) {
  conn.onClose = onCloseFunc(callback)
}

func (conn *ClientConnection)GetOnCloseCallback() onCloseFunc {
  return conn.onClose
}

func (conn *ClientConnection)Start() {
  if conn.GetOnConnectCallback() != nil {
    conn.GetOnConnectCallback()(conn)
  }

  conn.finish.Add(3)

  go func() {
    readLoop(conn, conn.finish)
  }()

  go func() {
    writeLoop(conn, conn.finish)
  }()

  go func() {
    handleClientLoop(conn, conn.finish)
  }()
}

func (conn *ClientConnection)Close() {
  conn.once.Do(func() {
    if conn.isClosed.CompareAndSet(false, true) {
      if conn.GetOnCloseCallback() != nil {
        conn.GetOnCloseCallback()(conn)
      }

      close(conn.GetCloseChannel())
      close(conn.GetMessageSendChannel())
      close(conn.GetHandlerReceiveChannel())

      // wait for all loops to finish
      conn.finish.Wait()
      conn.GetRawConn().Close()
      if conn.reconnectable {
        /* don't stop and close the timing wheel if reconnectable, just leave it alone,
        passing this wheel to the newly-created client when reconnect */
        conn.reconnect()
      } else {
        conn.GetTimingWheel().Stop()
        close(conn.GetTimeOutChannel())
      }
    }
  })
}

func (conn *ClientConnection)reconnect() {
  netid := conn.GetNetId()
  name := conn.GetName()
  address := conn.address
  pendingTimers := conn.GetPendingTimers()
  timingWheel := conn.GetTimingWheel()
  reconnectable := conn.reconnectable
  onConnect := conn.GetOnConnectCallback()
  onMessage := conn.GetOnMessageCallback()
  onClose := conn.GetOnCloseCallback()
  onError := conn.GetOnErrorCallback()

  c, err := net.Dial("tcp", address)
  if err != nil {
    log.Fatalln(err)
  }

  conn = &ClientConnection {
    netid: netid,
    name: name,
    address: address,
    heartBeat: time.Now().UnixNano(),
    isClosed: NewAtomicBoolean(false),
    pendingTimers: pendingTimers,
    timingWheel: timingWheel,
    conn: c,
    messageCodec: TypeLengthValueCodec{},
    finish: &sync.WaitGroup{},
    reconnectable: reconnectable,

    messageSendChan: make(chan []byte, 1024),
    handlerRecvChan: make(chan MessageHandler, 1024),
    closeConnChan: make(chan struct{}),

    onConnect: onConnect,
    onMessage: onMessage,
    onClose: onClose,
    onError: onError,
  }

  conn.Start()
}

func (conn *ClientConnection)IsClosed() bool {
  return conn.isClosed.Get()
}

func (conn *ClientConnection)Write(message Message) error {
  return asyncWrite(conn, message)
}

func (conn *ClientConnection)RunAt(timestamp time.Time, callback func(time.Time, interface{})) int64 {
  return runAt(conn, timestamp, callback)
}

func (conn *ClientConnection)RunAfter(duration time.Duration, callback func(time.Time, interface{})) int64 {
  return runAfter(conn, duration, callback)
}

func (conn *ClientConnection)RunEvery(interval time.Duration, callback func(time.Time, interface{})) int64 {
  return runEvery(conn, interval, callback)
}

func (conn *ClientConnection)GetTimingWheel() *TimingWheel {
  return conn.timingWheel
}

func (conn *ClientConnection)SetPendingTimers(pending []int64) {
  conn.pendingTimers = pending
}

func (conn *ClientConnection)GetPendingTimers() []int64 {
  return conn.pendingTimers
}

func (conn *ClientConnection)CancelTimer(timerId int64) {
  conn.GetTimingWheel().CancelTimer(timerId)
}

func (conn *ClientConnection)GetRawConn() net.Conn {
  return conn.conn
}

func (conn *ClientConnection)GetMessageSendChannel() chan []byte {
  return conn.messageSendChan
}

func (conn *ClientConnection)GetHandlerReceiveChannel() chan MessageHandler {
  return conn.handlerRecvChan
}

func (conn *ClientConnection)GetCloseChannel() chan struct{} {
  return conn.closeConnChan
}

func (conn *ClientConnection)GetTimeOutChannel() chan *OnTimeOut {
  return conn.timingWheel.TimeOutChan
}

func (conn *ClientConnection)GetRemoteAddress() net.Addr {
  return conn.conn.RemoteAddr()
}

func runAt(conn Connection, timestamp time.Time, callback func(time.Time, interface{})) int64 {
  timeout := NewOnTimeOut(conn.GetNetId(), callback)
  var id int64 = -1
  if conn.GetTimingWheel() != nil {
    id = conn.GetTimingWheel().AddTimer(timestamp, 0, timeout)
    if id >= 0 {
      pending := conn.GetPendingTimers()
      pending = append(pending, id)
      conn.SetPendingTimers(pending)
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
    id = conn.GetTimingWheel().AddTimer(delay, interval, timeout)
    if id >= 0 {
      pending := conn.GetPendingTimers()
      pending = append(pending, id)
      conn.SetPendingTimers(pending)
    }
  }
  return id
}

func asyncWrite(conn Connection, message Message) error {
  if conn.IsClosed() {
    return ErrorConnClosed
  }

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
func readLoop(conn Connection, finish *sync.WaitGroup) {
  defer func() {
    recover()
    finish.Done()
    conn.Close()
  }()

  for {
    select {
    case <-conn.GetCloseChannel():
      return

    default:
    }

    msg, err := conn.GetMessageCodec().Decode(conn)
    if err != nil {
      if err == ErrorUndefined{
        // update heart beat timestamp
        conn.SetHeartBeat(time.Now().UnixNano())
        continue
      }

      netError, ok := err.(net.Error)
      if ok && netError.Timeout(){
        // time out
        continue
      }

      log.Printf("Error decoding message - %s\n", err)
      return
    }

    // update heart beat timestamp
    conn.SetHeartBeat(time.Now().UnixNano())
    handlerFactory := HandlerMap.Get(msg.MessageNumber())
    if handlerFactory == nil {
      if conn.GetOnMessageCallback() != nil {
        log.Printf("Message %d call onMessage()\n", msg.MessageNumber())
        conn.GetOnMessageCallback()(msg, conn)
      } else {
        log.Printf("No handler or onMessage() found for message %d", msg.MessageNumber())
      }
      continue
    }

    // send handler to handleLoop
    handler := handlerFactory(conn.GetNetId(), msg)
    if !conn.IsClosed() {
      conn.GetHandlerReceiveChannel()<- handler
    }
  }
}

/* writeLoop() receive message from channel, serialize it into bytes,
then blocking write into connection */
func writeLoop(conn Connection, finish *sync.WaitGroup) {
  defer func() {
    recover()
    for packet := range conn.GetMessageSendChannel() {
      if packet != nil {
        if _, err := conn.GetRawConn().Write(packet); err != nil {
          log.Printf("Error writing data - %s\n", err)
        }
      }
    }
    finish.Done()
    conn.Close()
  }()

  for {
    select {
    case <-conn.GetCloseChannel():
      return

    case packet := <-conn.GetMessageSendChannel():
      if packet != nil {
        if _, err := conn.GetRawConn().Write(packet); err != nil {
          log.Printf("Error writing data - %s\n", err)
        }
      }
    }
  }
}

// handleServerLoop() - put handler or timeout callback into worker go-routines
func handleServerLoop(conn Connection, finish *sync.WaitGroup) {
  defer func() {
    recover()
    finish.Done()
    conn.Close()
  }()

  for {
    select {
    case <-conn.GetCloseChannel():
      return

    case handler := <-conn.GetHandlerReceiveChannel():
      if !isNil(handler) {
        serverConn, ok := conn.(*ServerConnection)
        if ok {
          serverConn.GetOwner().workerPool.Put(conn.GetNetId(), func() {
            handler.Process(conn)
          })
        }
      }

    case timeout := <-conn.GetTimeOutChannel():
      if timeout != nil {
        extraData := timeout.ExtraData.(int64)
        if extraData != conn.GetNetId() {
          log.Printf("[Warn] time out of %d running on client %d", extraData, conn.GetNetId())
        }
        serverConn, ok := conn.(*ServerConnection)
        if ok {
          serverConn.GetOwner().workerPool.Put(conn.GetNetId(), func() {
            timeout.Callback(time.Now(), conn)
          })
        } else {
          log.Printf("[Fatal] conn %s is not of type *ServerConnection\n", conn.GetName())
        }
      }
    }
  }
}

// handleClientLoop() - run handler or timeout callback in handleLoop() go-routine
func handleClientLoop(conn Connection, finish *sync.WaitGroup) {
  defer func() {
    recover()
    finish.Done()
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
