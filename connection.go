package tao

import (
  "bytes"
  "log"
  "net"
  "encoding/binary"
  "sync"
  "time"
  "io"
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
  name string
  closeOnce sync.Once
  wg *sync.WaitGroup
  timing *TimingWheel
  messageSendChan chan Message
  handlerRecvChan chan MessageHandler
  closeConnChan chan struct{}
  timeOutChan chan *OnTimeOut
  heartBeat int64
  pendingTimers []int64
  onConnect onConnectFunc
  onMessage onMessageFunc
  onClose onCloseFunc
  onError onErrorFunc
}

func ClientTCPConnection(id int64, c net.Conn, t *TimingWheel, keepAlive bool) *TCPConnection {
  tcpConn := &TCPConnection {
    netid: id,
    conn: c,
    wg: &sync.WaitGroup{},
    timing: t,
    messageSendChan: make(chan Message, 1024),
    handlerRecvChan: make(chan MessageHandler, 1024),
    closeConnChan: make(chan struct{}),
    timeOutChan: make(chan *OnTimeOut),
    heartBeat: time.Now().UnixNano(),
    pendingTimers: []int64{},
  }
  if keepAlive {
    tcpConn.activateKeepAlive()
  }
  return tcpConn
}

func ServerTCPConnection(id int64, s *TCPServer, c net.Conn, keepAlive bool) *TCPConnection {
  tcpConn := &TCPConnection {
    netid: id,
    Owner: s,
    conn: c,
    wg: &sync.WaitGroup{},
    timing: s.timing,
    messageSendChan: make(chan Message, 1024),
    handlerRecvChan: make(chan MessageHandler, 1024),
    closeConnChan: make(chan struct{}),
    timeOutChan: make(chan *OnTimeOut),
    heartBeat: time.Now().UnixNano(),
    pendingTimers: []int64{},
  }
  tcpConn.SetOnConnectCallback(s.onConnect)
  tcpConn.SetOnMessageCallback(s.onMessage)
  tcpConn.SetOnErrorCallback(s.onError)
  tcpConn.SetOnCloseCallback(s.onClose)

  if keepAlive {
    tcpConn.activateKeepAlive()
  }
  return tcpConn
}

func (client *TCPConnection)activateKeepAlive() {
  if client.isServerMode() { // server mode, check
    MessageMap.Register(HeartBeatMessage{}.MessageNumber(), UnmarshalFunctionType(UnmarshalHeartBeatMessage))
    HandlerMap.Register(HeartBeatMessage{}.MessageNumber(), NewHandlerFunctionType(NewHeartBeatMessageHandler))
    timerId := client.RunEvery(HEART_BEAT_PERIOD, func(now time.Time) {
      log.Printf("Checking client %d at %s", client.netid, time.Now())
      last := client.heartBeat
      period := HEART_BEAT_PERIOD.Nanoseconds()
      if last < now.UnixNano() - 2 * period {
        log.Printf("Client %s netid %d timeout, close it\n", client, client.netid)
        client.Close()
      }
    })
    client.pendingTimers = append(client.pendingTimers, timerId)
  } else { // client mode, send
    client.RunEvery(HEART_BEAT_PERIOD, func(now time.Time) {
      msg := HeartBeatMessage {
        Timestamp: now.UnixNano(),
      }
      log.Printf("Sending heart beat at %s, timestamp %d\n", now, msg.Timestamp)
      client.Write(msg)
    })
  }
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
    }

    for _, id := range client.pendingTimers {
      client.CancelTimer(id)
    }
  })
}

func (client *TCPConnection)Write(msg Message) (err error) {
  select {
  case client.messageSendChan<- msg:
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

func (client *TCPConnection)RunAt(t time.Time, cb func(t time.Time)) int64 {
  timeout := NewOnTimeOut(client.netid, cb)
  var id int64
  if client.timing != nil {
    id = client.timing.AddTimer(t, 0, timeout)
  }
  return id
}

func (client *TCPConnection)RunAfter(d time.Duration, cb func(t time.Time)) int64 {
  delay := time.Now().Add(d)
  var id int64
  if client.timing != nil {
    id = client.RunAt(delay, cb)
  }
  return id
}

func (client *TCPConnection)RunEvery(i time.Duration, cb func(t time.Time)) int64 {
  delay := time.Now().Add(i)
  timeout := NewOnTimeOut(client.netid, cb)
  var id int64
  if client.timing != nil {
    id = client.timing.AddTimer(delay, i, timeout)
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

/* readLoop() blocking read from connection, deserialize bytes into message,
then find corresponding handler, put it into channel */
func (client *TCPConnection)readLoop() {
  defer func() {
    recover()
    client.Close()
  }()

  typeBytes := make([]byte, NTYPE)
  lengthBytes := make([]byte, NLEN)
  for {
    select {
    case <-client.closeConnChan:
      return

    default:
    }

    // use type-length-value format: |4 bytes|4 bytes|n bytes <= 8M|
    _, err := io.ReadFull(client.conn, typeBytes)
    if err != nil {
      log.Printf("Error: failed to read message type - %s", err)
      return
    }
    typeBuf := bytes.NewReader(typeBytes)
    var msgType int32
    if err := binary.Read(typeBuf, binary.BigEndian, &msgType); err != nil {
      log.Fatalln(err)
    }

    _, err = io.ReadFull(client.conn, lengthBytes)
    if err != nil {
      log.Printf("Error: failed to read message length - %s", err)
      return
    }
    lengthBuf := bytes.NewReader(lengthBytes)
    var msgLen uint32
    if err := binary.Read(lengthBuf, binary.BigEndian, &msgLen); err != nil {
      log.Fatalln(err)
    }
    if msgLen > MAXLEN {
      log.Printf("Error: more than 8M data: %d\n", msgLen)
      return
    }

    // read real application message
    msgBytes := make([]byte, msgLen)
    _, err = io.ReadFull(client.conn, msgBytes)
    if err != nil {
      log.Printf("Error: failed to read message value - %s", err)
      return
    }

    // deserialize message from bytes
    unmarshaler := MessageMap.get(msgType)
    if unmarshaler == nil {
      log.Printf("Error: undefined message %d\n", msgType)
      continue
    }
    var msg Message
    if msg, err = unmarshaler(msgBytes); err != nil {
      log.Printf("Error: unmarshal message %d - %s\n", msgType, err)
      continue
    }

    handlerFactory := HandlerMap.get(msgType)
    if handlerFactory == nil {
      if client.onMessage != nil {
        log.Printf("Message %d call onMessage()\n", msgType)
        client.onMessage(msg, client)
      } else {
        log.Printf("No handler or onMessage() found for message %d", msgType)
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

    case msg := <-client.messageSendChan:
      data, err := msg.MarshalBinary();
      if err != nil {
        log.Printf("Error serializing data\n")
        continue
      }
      buf := new(bytes.Buffer)
      binary.Write(buf, binary.BigEndian, msg.MessageNumber())
      binary.Write(buf, binary.BigEndian, int32(len(data)))
      binary.Write(buf, binary.BigEndian, data)
      packet := buf.Bytes()
      if _, err = client.conn.Write(packet); err != nil {
        log.Printf("Error writing data %s\n", err)
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
        // update heart beat timestamp
        client.heartBeat = time.Now().UnixNano()
      }

    case timeout := <-client.timeOutChan:
      if timeout != nil {
        client.Owner.workerPool.Put(client.netid, func() {
          timeout.Callback(time.Now())
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
        // update heart beat timestamp
        client.heartBeat = time.Now().UnixNano()
        handler.Process(client)
      }

    case timeout := <-client.timing.TimeOutChan:
      // put callback into workers
      timeout.Callback(time.Now())
    }
  }
}
