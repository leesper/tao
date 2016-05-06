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

type TcpConnection struct {
  netid int64
  Owner *TcpServer
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

func ClientTCPConnection(id int64, c net.Conn, t *TimingWheel, keepAlive bool) *TcpConnection {
  tcpConn := &TcpConnection {
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

func ServerTCPConnection(id int64, s *TcpServer, c net.Conn, keepAlive bool) *TcpConnection {
  tcpConn := &TcpConnection {
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

func (client *TcpConnection)activateKeepAlive() {
  if client.isServerMode() { // server mode, check
    MessageMap.Register(HeartBeatMessage{}.MessageNumber(), UnmarshalFunctionType(UnmarshalHeartBeatMessage))
    HandlerMap.Register(HeartBeatMessage{}.MessageNumber(), NewHandlerFunctionType(NewHeartBeatMessageHandler))
    client.RunEvery(HEART_BEAT_PERIOD, func(now time.Time) {
      log.Printf("Checking client %d at %s", client.netid, time.Now())
      last := client.heartBeat
      period := HEART_BEAT_PERIOD.Nanoseconds()
      if last < now.UnixNano() - 2 * period {
        log.Printf("Client %s netid %d timeout, close it\n", client, client.netid)
        client.Close()
      }
    })
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

func (client *TcpConnection)isServerMode() bool {
  return client.Owner != nil
}

func (client *TcpConnection)SetOnConnectCallback(cb func(*TcpConnection) bool) {
  if cb != nil {
    client.onConnect = onConnectFunc(cb)
  }
}

func (client *TcpConnection)SetOnMessageCallback(cb func(Message, *TcpConnection)) {
  if cb != nil {
    client.onMessage = onMessageFunc(cb)
  }
}

func (client *TcpConnection)SetOnErrorCallback(cb func()) {
  if cb != nil {
    client.onError = onErrorFunc(cb)
  }
}

func (client *TcpConnection)Wait() {
  client.wg.Wait()
}

func (client *TcpConnection)SetOnCloseCallback(cb func(*TcpConnection)) {
  if cb != nil {
    client.onClose = onCloseFunc(cb)
  }
}

func (client *TcpConnection)RemoteAddr() net.Addr {
  return client.conn.RemoteAddr()
}

func (client *TcpConnection)SetName(n string) {
  client.name = n
}

func (client *TcpConnection)String() string {
  return client.name
}

func (client *TcpConnection)Close() {
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

    log.Println(client.pendingTimers)
    for _, id := range client.pendingTimers {
      client.CancelTimer(id)
    }
  })
}

func (client *TcpConnection)Write(msg Message) (err error) {
  select {
  case client.messageSendChan<- msg:
    return nil
  default:
    return ErrorWouldBlock
  }
}

func (client *TcpConnection)Do() {
  if client.onConnect != nil && !client.onConnect(client) {
    log.Fatalln("Error onConnect()\n")
  }

  // start read, write and handle loop
  client.startLoop(client.readLoop)
  client.startLoop(client.writeLoop)
  client.startLoop(client.handleLoop)
}

func (client *TcpConnection)RunAt(t time.Time, cb func(t time.Time)) {
  timeout := NewOnTimeOut(client.netid, cb)
  if client.timing != nil {
    id := client.timing.AddTimer(t, 0, timeout)
    client.pendingTimers = append(client.pendingTimers, id)
  }
}

func (client *TcpConnection)RunAfter(d time.Duration, cb func(t time.Time)) {
  delay := time.Now().Add(d)
  if client.timing != nil {
    client.RunAt(delay, cb)
  }
}

func (client *TcpConnection)RunEvery(i time.Duration, cb func(t time.Time)) {
  log.Println("netid ", client.netid)
  delay := time.Now().Add(i)
  timeout := NewOnTimeOut(client.netid, cb)
  if client.timing != nil {
    id := client.timing.AddTimer(delay, i, timeout)
    log.Println("RunEvery timer ", id)
    client.pendingTimers = append(client.pendingTimers, id)
  }
}

func (client *TcpConnection)CancelTimer(timerId int64) {
  client.timing.CancelTimer(timerId)
}

func (client *TcpConnection)startLoop(looper func()) {
  client.wg.Add(1)
  go func() {
    looper()
    client.wg.Done()
  }()
}

// use type-length-value format: |4 bytes|4 bytes|n bytes <= 8M|
// todo: maybe a special codec?
func (client *TcpConnection)readLoop() {
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

func (client *TcpConnection)writeLoop() {
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

func (client *TcpConnection)handleLoop() {
  if client.isServerMode() {
    client.handleServerMode()
  } else {
    client.handleClientMode()
  }
}

func (client *TcpConnection)handleServerMode() {
  defer func() {
    recover()
    client.Close()
  }()

  for {
    select {
    case <-client.closeConnChan:
      return

    case handler := <-client.handlerRecvChan:
      if handler != nil {
        client.Owner.workerPool.Put(client.netid, func() {
          handler.Process(client)
        })
        // update heart beat timestamp
        client.heartBeat = time.Now().UnixNano()
      }

    case timeoutcb := <-client.timeOutChan:
      if timeoutcb != nil {
        client.Owner.workerPool.Put(client.netid, func() {
          timeoutcb.Callback(time.Now())
        })
      }
    }
  }
}

func (client *TcpConnection)handleClientMode() {
  defer func() {
    recover()
    client.Close()
  }()

  for {
    select {
    case <-client.closeConnChan:
      return

    case handler := <-client.handlerRecvChan:
      if handler != nil {
        handler.Process(client)
        // update heart beat timestamp
        client.heartBeat = time.Now().UnixNano()
      }

    case timeoutcb := <-client.timing.TimeOutChan:
      // put callback into workers
      timeoutcb.Callback(time.Now())
    }
  }
}
