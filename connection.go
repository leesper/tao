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
  conn *net.TCPConn
  name string
  closeOnce sync.Once
  wg *sync.WaitGroup
  messageSendChan chan Message
  handlerRecvChan chan MessageHandler
  closeConnChan chan struct{}
  timing *TimingWheel
  heartBeat int64
  onConnect onConnectCallbackType
  onMessage onMessageCallbackType
  onClose onCloseCallbackType
  onError onErrorCallbackType
}

func NewTcpConnection(id int64, s *TcpServer, c *net.TCPConn, t *TimingWheel, keepAlive bool) *TcpConnection {
  tcpConn := &TcpConnection {
    netid: id,
    Owner: s,
    conn: c,
    wg: &sync.WaitGroup{},
    messageSendChan: make(chan Message, 1024),
    handlerRecvChan: make(chan MessageHandler, 1024),
    closeConnChan: make(chan struct{}),
    timing: t,
    heartBeat: time.Now().UnixNano(),
  }
  if s != nil {
    tcpConn.SetOnConnectCallback(s.onConnect)
    tcpConn.SetOnMessageCallback(s.onMessage)
    tcpConn.SetOnErrorCallback(s.onError)
    tcpConn.SetOnCloseCallback(s.onClose)
  }
  if keepAlive {
    tcpConn.activateKeepAlive()
  }
  return tcpConn
}

func (client *TcpConnection)activateKeepAlive() {
  if client.Owner != nil { // server mode, check
    MessageMap.Register(HeartBeatMessage{}.MessageNumber(), UnmarshalFunctionType(UnmarshalHeartBeatMessage))
    HandlerMap.Register(HeartBeatMessage{}.MessageNumber(), NewHandlerFunctionType(NewHeartBeatMessageHandler))
    var timerId int
    timerId = client.RunEvery(HEART_BEAT_PERIOD, func(now time.Time) {
      log.Printf("Checking client %s at %s", client, time.Now())
      last := client.heartBeat
      period := HEART_BEAT_PERIOD.Nanoseconds()
      if last < now.UnixNano() - 2 * period {
        log.Printf("Client %s netid %d timeout, close it\n", client, client.netid)
        client.Close()
        client.timing.CancelTimer(timerId)
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
    client.onConnect = onConnectCallbackType(cb)
  }
}

func (client *TcpConnection)SetOnMessageCallback(cb func(Message, *TcpConnection)) {
  if cb != nil {
    client.onMessage = onMessageCallbackType(cb)
  }
}

func (client *TcpConnection)SetOnErrorCallback(cb func()) {
  if cb != nil {
    client.onError = onErrorCallbackType(cb)
  }
}

func (client *TcpConnection)Wait() {
  client.wg.Wait()
}

func (client *TcpConnection)SetOnCloseCallback(cb func(*TcpConnection)) {
  if cb != nil {
    client.onClose = onCloseCallbackType(cb)
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
    client.conn.Close()
    if (client.onClose != nil) {
      client.onClose(client)
    }
    if client.isServerMode() {
      client.Owner.connections.Remove(client.netid)
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

func (client *TcpConnection)RunAt(t time.Time, cb func(t time.Time)) int {
  if client.timing != nil {
    return client.timing.AddTimer(t, 0, cb)
  }
  return -1
}

func (client *TcpConnection)RunAfter(d time.Duration, cb func(t time.Time)) int {
  delay := time.Now().Add(d)
  if client.timing != nil {
    return client.RunAt(delay, cb)
  }
  return -1
}

func (client *TcpConnection)RunEvery(i time.Duration, cb func(t time.Time)) int {
  delay := time.Now().Add(i)
  if client.timing != nil {
    return client.timing.AddTimer(delay, i, cb)
  }
  return -1
}

func (client *TcpConnection)CancelTimer(timerId int) {
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
  defer func() {
    recover()
    client.Close()
  }()

  for {
    if client.timing == nil {
      select {
        case <-client.closeConnChan:
          return

        case handler := <-client.handlerRecvChan:
          if handler != nil {
            if client.isServerMode() {
              client.Owner.workerPool.Put(client.netid, func() {
                handler.Process(client)
              })
            } else {
              handler.Process(client)
            }
            // update heart beat timestamp
            client.heartBeat = time.Now().UnixNano()
          }

      }
    } else {
      select {
        case <-client.closeConnChan:
          return

        case handler := <-client.handlerRecvChan:
          if handler != nil {
            if client.isServerMode() {
              client.Owner.workerPool.Put(client.netid, func() {
                handler.Process(client)
              })
            } else {
              handler.Process(client)
            }
            // update heart beat timestamp
            client.heartBeat = time.Now().UnixNano()
          }

        case timeoutcb := <-client.timing.TimeOutChan:
          // put callback into workers
          if timeoutcb != nil {
            if client.isServerMode() {
              client.Owner.workerPool.Put(client.netid, func() {
                timeoutcb(time.Now())
              })
            } else {
              timeoutcb(time.Now())
            }
          }
      }
    }
  }
}
