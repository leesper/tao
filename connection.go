package tao

import (
  "bytes"
  "io"
  "log"
  "net"
  "time"
  "encoding/binary"
  "errors"
  "sync"
)

const (
  NTYPE = 3
  NLEN = 4
  MAXLEN = 1 << 23  // 8M
)

var ErrorWouldBlock error = errors.New("Would block")

type TcpConnection struct {
  owner *TcpServer
  conn *net.TCPConn
  name string
  closeOnce sync.Once
  msgSendChan chan *Message
  msgRecvChan chan *Message
}

func NewTcpConnection(s *TcpServer, c *net.TCPConn) *TcpConnection {
  return &TcpConnection {
    owner: s,
    conn: c,
    msgSendChan: make(chan *Message, 1024), // todo: make it configurable
    msgRecvChan: make(chan *Message, 1024), // todo: make it configurable
  }
}

func (client *TcpConnection) RemoteAddr() net.Addr {
  return client.conn.RemoteAddr()
}

func (client *TcpConnection) SetName(n string) {
  client.name = n
}

func (client *TcpConnection) String() string {
  return client.name
}

func (client *TcpConnection) Close() {
  client.closeOnce.Do(func() {
    client.conn.Close()
    if (client.owner.onClose != nil) {
      client.owner.onClose(client)
    }
  })
}

func (client *TcpConnection) Write(msg *Message) (err error) {
  select {
  case client.msgSendChan<- msg:
    return nil
  default:
    return ErrorWouldBlock
  }
}

func (client *TcpConnection) Do() {
  if client.owner.onConnect != nil && !client.owner.onConnect() {
    log.Fatalln("on connect callback failed\n")
  }

  // start read, write and handle loop
  client.startLoop(client.readLoop)
  client.startLoop(client.writeLoop)
  client.startLoop(client.handleLoop)
}

func (client *TcpConnection) startLoop(looper func()) {
  client.owner.wg.Add(1)
  go func() {
    looper()
    client.owner.wg.Done()
  }()
}

// use type-length-value format: |3 bytes|4 bytes|n bytes <= 8M|
func (client *TcpConnection) readLoop() {
  typeBytes := make([]byte, NTYPE)
  lengthBytes := make([]byte, NLEN)

  for client.owner.running.Get() {
    // read type info
    _, err := io.ReadFull(client.conn, typeBytes)
    if err == io.EOF {
      time.Sleep(2 * time.Millisecond)
    } else {
      log.Printf("Error reading message type\n")
      // todo: do sth to handle error
      continue
    }
    typeBuf := bytes.NewReader(typeBytes)
    var msgType int32
    err = binary.Read(typeBuf, binary.BigEndian, &msgType)
    if err != nil {
      log.Fatalln(err)
    }

    // read length info
    _, err = io.ReadFull(client.conn, lengthBytes)
    if err != nil {
      log.Printf("Error reading message length\n")
      // todo: do sth to handle error
      continue
    }
    lengthBuf := bytes.NewReader(lengthBytes)
    var msgLen uint32
    err = binary.Read(lengthBuf, binary.BigEndian, &msgLen)
    if err != nil {
      log.Fatalln(err)
    }
    if msgLen > MAXLEN {
      log.Printf("error: more than 8M data\n")
      // todo: do something to handle it
      continue
    }

    // read message info
    log.Printf("type %d len %d\n", msgType, msgLen)
    msgBytes := make([]byte, msgLen)
    _, err = io.ReadFull(client.conn, msgBytes)
    if err != nil {
      log.Printf("Error reading message value\n")
      // todo: do sth to handle error
      continue
    }

    // deserialize message from bytes
    msg := MessageMapper.Get(msgType)
    if msg == nil {
      log.Printf("Error undefined message %d\n", msgType)
      continue
    }
    if err = (*msg).UnmarshalBinary(msgBytes); err != nil {
      log.Printf("Error unmarshal message %d\n", msgType)
      continue
    }

    // send message to handleLoop
    client.msgRecvChan<- msg
  }
}

func (client *TcpConnection) writeLoop() {
  for client.owner.running.Get() {
    select {
    case msg := <-client.msgSendChan:
      data, err := (*msg).MarshalBinary();
      if err != nil {
        log.Printf("Error serializing data\n")
        continue
      }
      if _, err = client.conn.Write(data); err != nil {
        log.Printf("Error writing data %s\n", err)
      }
    }
  }
}

func (client *TcpConnection) handleLoop() {
  for client.owner.running.Get() {
    select {
    case msg := <-client.msgRecvChan:
      handlerFactory := HandlerMapper.Get(msgType)
      if handlerFactory == nil {
        log.Printf("Error undefined handler for message %d\n", msgType)
        continue
      }
      handler := handlerFactory(msg)
      // todo: hash handler into worker threads
      handler.Process(client)
    }
  }
}
