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

type TCPConnection struct {
  netid int64
  Owner *TCPServer
  conn net.Conn
  address string
  name string
  closeOnce sync.Once
  wg *sync.WaitGroup
  timing *TimingWheel
  messageSendChan chan []byte
  handlerRecvChan chan MessageHandler
  closeConnChan chan struct{}
  timeOutChan chan *OnTimeOut
  HeartBeat int64
  pendingTimers []int64
  reconnect bool
  closed *AtomicBoolean
  messageCodec Codec
  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc
}

func ClientTCPConnection(id int64, addr string, t *TimingWheel, reconn bool) *TCPConnection {
  c, err := net.Dial("tcp", addr)
  if err != nil {
    log.Fatalln(err)
  }
  return &TCPConnection {
    netid: id,
    conn: c,
    address: addr,
    wg: &sync.WaitGroup{},
    timing: t,
    messageSendChan: make(chan []byte, 1024),
    handlerRecvChan: make(chan MessageHandler, 1024),
    closeConnChan: make(chan struct{}),
    timeOutChan: make(chan *OnTimeOut),
    HeartBeat: time.Now().UnixNano(),
    pendingTimers: []int64{},
    reconnect: reconn,
    closed: NewAtomicBoolean(false),
    messageCodec: TypeLengthValueCodec{},
  }
}

func ServerTCPConnection(id int64, s *TCPServer, c net.Conn) *TCPConnection {
  tcpConn := &TCPConnection {
    netid: id,
    Owner: s,
    conn: c,
    wg: &sync.WaitGroup{},
    timing: s.timing,
    messageSendChan: make(chan []byte, 1024),
    handlerRecvChan: make(chan MessageHandler, 1024),
    closeConnChan: make(chan struct{}),
    timeOutChan: make(chan *OnTimeOut),
    HeartBeat: time.Now().UnixNano(),
    pendingTimers: []int64{},
    reconnect: false,
    closed: NewAtomicBoolean(false),
    messageCodec: TypeLengthValueCodec{},
  }
  tcpConn.SetOnConnectCallback(s.onConnect)
  tcpConn.SetOnMessageCallback(s.onMessage)
  tcpConn.SetOnErrorCallback(s.onError)
  tcpConn.SetOnCloseCallback(s.onClose)
  return tcpConn
}

func (client *TCPConnection)NetId() int64 {
  return client.netid
}

func (client *TCPConnection)Reconnect() {
  netid := client.netid
  address := client.address
  timing := client.timing
  reconnect := client.reconnect
  client = ClientTCPConnection(netid, address, timing, reconnect)
  client.Do()
}

func (client *TCPConnection)SetCodec(cdc Codec) {
  client.messageCodec = cdc
}

func (client *TCPConnection)isServerMode() bool {
  return client.Owner != nil
}

func (client *TCPConnection)SetOnConnectCallback(cb func(*TCPConnection) bool) {
  if cb != nil {
    client.onConnect = onConnectFunc(cb)
  }
}

func (client *TCPConnection)SetOnMessageCallback(cb func(Message, *TCPConnection)) {
  if cb != nil {
    client.onMessage = onMessageFunc(cb)
  }
}

func (client *TCPConnection)SetOnErrorCallback(cb func()) {
  if cb != nil {
    client.onError = onErrorFunc(cb)
  }
}

func (client *TCPConnection)Wait() {
  client.wg.Wait()
}

func (client *TCPConnection)SetOnCloseCallback(cb func(*TCPConnection)) {
  if cb != nil {
    client.onClose = onCloseFunc(cb)
  }
}

func (client *TCPConnection)RemoteAddr() net.Addr {
  return client.conn.RemoteAddr()
}

func (client *TCPConnection)SetName(n string) {
  client.name = n
}

func (client *TCPConnection)String() string {
  return client.name
}

func (client *TCPConnection)Close() {
  client.closeOnce.Do(func() {
    if client.closed.CompareAndSet(false, true) {
      close(client.closeConnChan)
      close(client.messageSendChan)
      close(client.handlerRecvChan)
      close(client.timeOutChan)
      client.conn.Close()

      if (client.onClose != nil) {
        client.onClose(client)
      }

      if client.isServerMode() {
        client.Owner.connections.Remove(client.netid)
        for _, id := range client.pendingTimers {
          client.CancelTimer(id)
        }
      } else {
        client.Reconnect()
      }
    }
  })
}

func (client *TCPConnection)Write(msg Message) error {
  packet, err := client.messageCodec.Encode(msg)
  if err != nil {
    return err
  }

  select {
  case client.messageSendChan<- packet:
    return nil
  default:
    return ErrorWouldBlock
  }
}

/* If onConnect() returns true, start three go-routines for each client:
readLoop(), writeLoop() and handleLoop() */
func (client *TCPConnection)Do() {
  if client.onConnect != nil && !client.onConnect(client) {
    log.Fatalln("Error onConnect()\n")
  }

  // start read, write and handle loop
  client.startLoop(client.readLoop)
  client.startLoop(client.writeLoop)
  client.startLoop(client.handleLoop)
}

func (client *TCPConnection)RunAt(t time.Time, cb func(time.Time, interface{})) int64 {
  timeout := NewOnTimeOut(client.netid, cb)
  var id int64 = -1
  if client.timing != nil {
    id = client.timing.AddTimer(t, 0, timeout)
    if id >= 0 {
      client.pendingTimers = append(client.pendingTimers, id)
      log.Println("Pending timers ", client.pendingTimers)
    }
  }
  return id
}

func (client *TCPConnection)RunAfter(d time.Duration, cb func(time.Time, interface{})) int64 {
  delay := time.Now().Add(d)
  var id int64 = -1
  if client.timing != nil {
    id = client.RunAt(delay, cb)
  }
  return id
}

func (client *TCPConnection)RunEvery(i time.Duration, cb func(time.Time, interface{})) int64 {
  delay := time.Now().Add(i)
  timeout := NewOnTimeOut(client.netid, cb)
  var id int64 = -1
  if client.timing != nil {
    id = client.timing.AddTimer(delay, i, timeout)
    if id >= 0 {
      client.pendingTimers = append(client.pendingTimers, id)
    }
  }
  return id
}

func (client *TCPConnection)CancelTimer(timerId int64) {
  client.timing.CancelTimer(timerId)
}

func (client *TCPConnection)startLoop(looper func()) {
  client.wg.Add(1)
  go func() {
    looper()
    client.wg.Done()
  }()
}

func (client *TCPConnection) RawConn() net.Conn {
  return client.conn
}

/* readLoop() blocking read from connection, deserialize bytes into message,
then find corresponding handler, put it into channel */
func (client *TCPConnection)readLoop() {
  defer func() {
    recover()
    client.Close()
  }()

  for {
    select {
    case <-client.closeConnChan:
      return

    default:
    }

    msg, err := client.messageCodec.Decode(client)
    if err != nil {
      log.Printf("Error decoding message - %s", err)
      if err == ErrorUndefined {
        // update heart beat timestamp
        client.HeartBeat = time.Now().UnixNano()
        continue
      }
      return
    }

    // update heart beat timestamp
    client.HeartBeat = time.Now().UnixNano()
    handlerFactory := HandlerMap.get(msg.MessageNumber())
    if handlerFactory == nil {
      if client.onMessage != nil {
        log.Printf("Message %d call onMessage()\n", msg.MessageNumber())
        client.onMessage(msg, client)
      } else {
        log.Printf("No handler or onMessage() found for message %d", msg.MessageNumber())
      }
      continue
    }

    // send handler to handleLoop
    handler := handlerFactory(msg)
    client.handlerRecvChan<- handler
  }
}

/* writeLoop() receive message from channel, serialize it into bytes,
then blocking write into connection */
func (client *TCPConnection)writeLoop() {
  defer func() {
    recover()
    client.Close()
  }()

  for {
    select {
    case <-client.closeConnChan:
      return

    case packet := <-client.messageSendChan:
      if _, err := client.conn.Write(packet); err != nil {
        log.Printf("Error writing data - %s\n", err)
      }
    }
  }
}

/* handleLoop() handles business logic in server or client mode:
(1) server mode - put handler or timeout callback into worker go-routines
(2) client mode - run handler or timeout callback in handleLoop() go-routine */
func (client *TCPConnection)handleLoop() {
  if client.isServerMode() {
    client.handleServerMode()
  } else {
    client.handleClientMode()
  }
}

func (client *TCPConnection)handleServerMode() {
  defer func() {
    recover()
    client.Close()
  }()

  for {
    select {
    case <-client.closeConnChan:
      return

    case handler := <-client.handlerRecvChan:
      if !isNil(handler) {
        client.Owner.workerPool.Put(client.netid, func() {
          handler.Process(client)
        })
      }

    case timeout := <-client.timeOutChan:
      if timeout != nil {
        extraData := timeout.ExtraData.(int64)
        if extraData != client.netid {
          log.Printf("[Warn] time out of %d running on client %d", extraData, client.netid)
        }
        client.Owner.workerPool.Put(client.netid, func() {
          timeout.Callback(time.Now(), client)
        })
      }
    }
  }
}

func (client *TCPConnection)handleClientMode() {
  defer func() {
    recover()
    client.Close()
  }()

  for {
    select {
    case <-client.closeConnChan:
      return

    case handler := <-client.handlerRecvChan:
      if !isNil(handler) {
        handler.Process(client)
      }

    case timeout := <-client.timing.TimeOutChan:
      if timeout != nil {
        extraData := timeout.ExtraData.(int64)
        if extraData != client.netid {
          log.Printf("[Warn] time out of %d running on client %d", extraData, client.netid)
        }
        timeout.Callback(time.Now(), client)
      }
    }
  }
}
