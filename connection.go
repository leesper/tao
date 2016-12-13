package tao

import (
	"crypto/tls"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reechou/holmes"
)

const (
	NTYPE  = 4
	NLEN   = 4
	MAXLEN = 1 << 23 // 8M
)

type MessageHandler struct {
	message Message
	handler HandlerFunction
}

type Connection interface {
	SetNetId(netid int64)
	GetNetId() int64

	SetName(name string)
	GetName() string

	SetHeartBeat(beat int64)
	GetHeartBeat() int64

	SetExtraData(extra interface{})
	GetExtraData() interface{}

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
	IsRunning() bool
	Write(message Message) error

	RunAt(t time.Time, cb func(time.Time, interface{})) int64
	RunAfter(d time.Duration, cb func(time.Time, interface{})) int64
	RunEvery(i time.Duration, cb func(time.Time, interface{})) int64
	GetTimingWheel() *TimingWheel

	CancelTimer(timerId int64)

	GetRawConn() net.Conn
	GetMessageSendChannel() chan []byte
	GetMessageHandlerChannel() chan MessageHandler
	GetCloseChannel() chan struct{}
	GetTimeOutChannel() chan *OnTimeOut

	GetRemoteAddress() net.Addr
}

type ServerConnection struct {
	netid         int64
	name          string
	heartBeat     int64
	extraData     atomic.Value
	owner         *TCPServer
	running       *AtomicBoolean
	once          sync.Once
	pendingTimers []int64
	conn          net.Conn
	messageCodec  Codec
	finish        *sync.WaitGroup

	messageSendChan    chan []byte
	messageHandlerChan chan MessageHandler
	closeConnChan      chan struct{}
	timeOutChan        chan *OnTimeOut

	onConnect onConnectFunc
	onMessage onMessageFunc
	onClose   onCloseFunc
	onError   onErrorFunc
}

func NewServerConnection(netid int64, server *TCPServer, conn net.Conn) Connection {
	serverConn := &ServerConnection{
		netid:              netid,
		name:               conn.RemoteAddr().String(),
		heartBeat:          time.Now().UnixNano(),
		owner:              server,
		running:            NewAtomicBoolean(true),
		pendingTimers:      []int64{},
		conn:               conn,
		messageCodec:       TypeLengthValueCodec{},
		finish:             &sync.WaitGroup{},
		messageSendChan:    make(chan []byte, 1024),
		messageHandlerChan: make(chan MessageHandler, 1024),
		closeConnChan:      make(chan struct{}),
		timeOutChan:        make(chan *OnTimeOut),
	}
	serverConn.SetOnConnectCallback(server.onConnect)
	serverConn.SetOnMessageCallback(server.onMessage)
	serverConn.SetOnErrorCallback(server.onError)
	serverConn.SetOnCloseCallback(server.onClose)
	return serverConn
}

func (conn *ServerConnection) SetNetId(netid int64) {
	conn.netid = netid
}

func (conn *ServerConnection) GetNetId() int64 {
	return conn.netid
}

func (conn *ServerConnection) SetName(name string) {
	conn.name = name
}

func (conn *ServerConnection) GetName() string {
	return conn.name
}

func (conn *ServerConnection) SetHeartBeat(beat int64) {
	atomic.StoreInt64(&conn.heartBeat, beat)
}

func (conn *ServerConnection) GetHeartBeat() int64 {
	return atomic.LoadInt64(&conn.heartBeat)
}

func (conn *ServerConnection) SetExtraData(extra interface{}) {
	conn.extraData.Store(extra)
}

func (conn *ServerConnection) GetExtraData() interface{} {
	return conn.extraData.Load()
}

func (conn *ServerConnection) SetMessageCodec(codec Codec) {
	conn.messageCodec = codec
}

func (conn *ServerConnection) GetMessageCodec() Codec {
	return conn.messageCodec
}

func (conn *ServerConnection) GetOwner() *TCPServer {
	return conn.owner
}

func (conn *ServerConnection) Start() {
	if conn.GetOnConnectCallback() != nil {
		conn.GetOnConnectCallback()(conn)
	}

	conn.finish.Add(3)
	loopers := []func(Connection, *sync.WaitGroup){readLoop, writeLoop, handleServerLoop}
	for _, l := range loopers {
		looper := l // necessary
		go looper(conn, conn.finish)
	}
}

func (conn *ServerConnection) Close() {
	conn.once.Do(func() {
		if conn.running.CompareAndSet(true, false) {
			if conn.onClose != nil {
				conn.onClose(conn)
			}

			close(conn.messageSendChan)
			close(conn.messageHandlerChan)

			// clean up timed task
			close(conn.timeOutChan)
			for _, id := range conn.GetPendingTimers() {
				conn.CancelTimer(id)
			}

			// wait for go-routines holding readLoop, writeLoop and handleServerLoop to finish
			close(conn.closeConnChan)
			conn.finish.Wait()

			if tcp, ok := conn.GetRawConn().(*net.TCPConn); ok {
				// avoid time-wait state
				tcp.SetLinger(0)
			}
			conn.GetRawConn().Close()

			conn.GetOwner().connections.Remove(conn.GetNetId())
			addTotalConn(-1)
			holmes.Info("HOW MANY CONNECTIONS DO I HAVE: %d", conn.GetOwner().connections.Size())

			conn.GetOwner().finish.Done()
		}
	})
}

func (conn *ServerConnection) IsRunning() bool {
	return conn.running.Get()
}

func (conn *ServerConnection) SetPendingTimers(pending []int64) {
	conn.pendingTimers = pending
}

func (conn *ServerConnection) GetPendingTimers() []int64 {
	return conn.pendingTimers
}

func (conn *ServerConnection) Write(message Message) error {
	return asyncWrite(conn, message)
}

func (conn *ServerConnection) RunAt(timestamp time.Time, callback func(time.Time, interface{})) int64 {
	return runAt(conn, timestamp, callback)
}

func (conn *ServerConnection) RunAfter(duration time.Duration, callback func(time.Time, interface{})) int64 {
	return runAfter(conn, duration, callback)
}

func (conn *ServerConnection) RunEvery(interval time.Duration, callback func(time.Time, interface{})) int64 {
	return runEvery(conn, interval, callback)
}

func (conn *ServerConnection) GetTimingWheel() *TimingWheel {
	return conn.GetOwner().GetTimingWheel()
}

func (conn *ServerConnection) CancelTimer(timerId int64) {
	conn.GetTimingWheel().CancelTimer(timerId)
}

func (conn *ServerConnection) GetRawConn() net.Conn {
	return conn.conn
}

func (conn *ServerConnection) GetMessageSendChannel() chan []byte {
	return conn.messageSendChan
}

func (conn *ServerConnection) GetMessageHandlerChannel() chan MessageHandler {
	return conn.messageHandlerChan
}

func (conn *ServerConnection) GetCloseChannel() chan struct{} {
	return conn.closeConnChan
}

func (conn *ServerConnection) GetTimeOutChannel() chan *OnTimeOut {
	return conn.timeOutChan
}

func (conn *ServerConnection) GetRemoteAddress() net.Addr {
	return conn.conn.RemoteAddr()
}

func (conn *ServerConnection) SetOnConnectCallback(callback func(Connection) bool) {
	conn.onConnect = onConnectFunc(callback)
}

func (conn *ServerConnection) GetOnConnectCallback() onConnectFunc {
	return conn.onConnect
}

func (conn *ServerConnection) SetOnMessageCallback(callback func(Message, Connection)) {
	conn.onMessage = onMessageFunc(callback)
}

func (conn *ServerConnection) GetOnMessageCallback() onMessageFunc {
	return conn.onMessage
}

func (conn *ServerConnection) SetOnErrorCallback(callback func()) {
	conn.onError = onErrorFunc(callback)
}

func (conn *ServerConnection) GetOnErrorCallback() onErrorFunc {
	return conn.onError
}

func (conn *ServerConnection) SetOnCloseCallback(callback func(Connection)) {
	conn.onClose = onCloseFunc(callback)
}

func (conn *ServerConnection) GetOnCloseCallback() onCloseFunc {
	return conn.onClose
}

// implements Connection
type ClientConnection struct {
	netid         int64
	name          string
	address       string
	heartBeat     int64
	extraData     atomic.Value
	running       *AtomicBoolean
	once          *sync.Once
	pendingTimers []int64
	timingWheel   *TimingWheel
	conn          net.Conn
	messageCodec  Codec
	finish        *sync.WaitGroup
	reconnectable bool
	tconfig       *tls.Config

	messageSendChan    chan []byte
	messageHandlerChan chan MessageHandler
	closeConnChan      chan struct{}

	onConnect onConnectFunc
	onMessage onMessageFunc
	onClose   onCloseFunc
	onError   onErrorFunc
}

func NewClientConnection(netid int64, reconnectable bool, c net.Conn, tconf *tls.Config) Connection {
	return &ClientConnection{
		netid:              netid,
		name:               c.RemoteAddr().String(),
		address:            c.RemoteAddr().String(),
		heartBeat:          time.Now().UnixNano(),
		running:            NewAtomicBoolean(true),
		once:               &sync.Once{},
		pendingTimers:      []int64{},
		timingWheel:        NewTimingWheel(),
		conn:               c,
		messageCodec:       TypeLengthValueCodec{},
		finish:             &sync.WaitGroup{},
		reconnectable:      reconnectable,
		tconfig:            tconf,
		messageSendChan:    make(chan []byte, 1024),
		messageHandlerChan: make(chan MessageHandler, 1024),
		closeConnChan:      make(chan struct{}),
	}
}

func (client *ClientConnection) SetNetId(netid int64) {
	client.netid = netid
}

func (client *ClientConnection) GetNetId() int64 {
	return client.netid
}

func (client *ClientConnection) SetName(name string) {
	client.name = name
}

func (client *ClientConnection) GetName() string {
	return client.name
}

func (client *ClientConnection) SetHeartBeat(beat int64) {
	atomic.StoreInt64(&client.heartBeat, beat)
}

func (client *ClientConnection) GetHeartBeat() int64 {
	return atomic.LoadInt64(&client.heartBeat)
}

func (client *ClientConnection) SetExtraData(extra interface{}) {
	client.extraData.Store(extra)
}

func (client *ClientConnection) GetExtraData() interface{} {
	return client.extraData.Load()
}

func (client *ClientConnection) SetMessageCodec(codec Codec) {
	client.messageCodec = codec
}

func (client *ClientConnection) GetMessageCodec() Codec {
	return client.messageCodec
}

func (client *ClientConnection) SetOnConnectCallback(callback func(Connection) bool) {
	client.onConnect = onConnectFunc(callback)
}

func (client *ClientConnection) GetOnConnectCallback() onConnectFunc {
	return client.onConnect
}

func (client *ClientConnection) SetOnMessageCallback(callback func(Message, Connection)) {
	client.onMessage = onMessageFunc(callback)
}

func (client *ClientConnection) GetOnMessageCallback() onMessageFunc {
	return client.onMessage
}

func (client *ClientConnection) SetOnErrorCallback(callback func()) {
	client.onError = onErrorFunc(callback)
}

func (client *ClientConnection) GetOnErrorCallback() onErrorFunc {
	return client.onError
}

func (client *ClientConnection) SetOnCloseCallback(callback func(Connection)) {
	client.onClose = onCloseFunc(callback)
}

func (client *ClientConnection) GetOnCloseCallback() onCloseFunc {
	return client.onClose
}

func (client *ClientConnection) Start() {
	if client.GetOnConnectCallback() != nil {
		client.GetOnConnectCallback()(client)
	}

	client.finish.Add(3)
	loopers := []func(Connection, *sync.WaitGroup){readLoop, writeLoop, handleClientLoop}
	for _, l := range loopers {
		looper := l // necessary
		go looper(client, client.finish)
	}
}

func (client *ClientConnection) Close() {
	done := false
	client.once.Do(func() {
		if client.running.CompareAndSet(true, false) {
			if client.GetOnCloseCallback() != nil {
				client.GetOnCloseCallback()(client)
			}

			close(client.GetCloseChannel())
			close(client.GetMessageSendChannel())
			close(client.GetMessageHandlerChannel())
			client.GetTimingWheel().Stop()
			close(client.GetTimeOutChannel())

			// wait for all loops to finish
			client.finish.Wait()
			client.GetRawConn().Close()
			done = true
		}
	})

	if done && client.reconnectable {
		client.reconnect()
	}
}

func (client *ClientConnection) reconnect() {
	var c net.Conn
	var err error
	if client.tconfig == nil {
		c, err = net.Dial("tcp", client.address)
		if err != nil {
			holmes.Fatal("net Dial error %v", err)
		}
	} else {
		c, err = tls.Dial("tcp", client.address, client.tconfig)
		if err != nil {
			holmes.Fatal("tls Dial error %v", err)
		}
	}

	client.name = c.RemoteAddr().String()
	client.heartBeat = time.Now().UnixNano()
	client.extraData.Store(struct{}{})
	client.once = &sync.Once{}
	client.pendingTimers = []int64{}
	client.timingWheel = NewTimingWheel()
	client.conn = c
	client.messageSendChan = make(chan []byte, 1024)
	client.messageHandlerChan = make(chan MessageHandler, 1024)
	client.closeConnChan = make(chan struct{})
	client.Start()
	client.running.CompareAndSet(false, true)
}

func (client *ClientConnection) IsRunning() bool {
	return client.running.Get()
}

func (client *ClientConnection) Write(message Message) error {
	return asyncWrite(client, message)
}

func (client *ClientConnection) RunAt(timestamp time.Time, callback func(time.Time, interface{})) int64 {
	return runAt(client, timestamp, callback)
}

func (client *ClientConnection) RunAfter(duration time.Duration, callback func(time.Time, interface{})) int64 {
	return runAfter(client, duration, callback)
}

func (client *ClientConnection) RunEvery(interval time.Duration, callback func(time.Time, interface{})) int64 {
	return runEvery(client, interval, callback)
}

func (client *ClientConnection) GetTimingWheel() *TimingWheel {
	return client.timingWheel
}

func (client *ClientConnection) SetPendingTimers(pending []int64) {
	client.pendingTimers = pending
}

func (client *ClientConnection) GetPendingTimers() []int64 {
	return client.pendingTimers
}

func (client *ClientConnection) CancelTimer(timerId int64) {
	client.GetTimingWheel().CancelTimer(timerId)
}

func (client *ClientConnection) GetRawConn() net.Conn {
	return client.conn
}

func (client *ClientConnection) GetMessageSendChannel() chan []byte {
	return client.messageSendChan
}

func (client *ClientConnection) GetMessageHandlerChannel() chan MessageHandler {
	return client.messageHandlerChan
}

func (client *ClientConnection) GetCloseChannel() chan struct{} {
	return client.closeConnChan
}

func (client *ClientConnection) GetTimeOutChannel() chan *OnTimeOut {
	return client.timingWheel.GetTimeOutChannel()
}

func (client *ClientConnection) GetRemoteAddress() net.Addr {
	return client.conn.RemoteAddr()
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
	defer func() error {
		if p := recover(); p != nil {
			return ErrorConnClosed
		}
		return nil
	}()

	packet, err := conn.GetMessageCodec().Encode(message)
	if err != nil {
		holmes.Error("asyncWrite error %v", err)
		return err
	}

	select {
	case conn.GetMessageSendChannel() <- packet:
		return nil
	default:
		return ErrorWouldBlock
	}
}

/* readLoop() blocking read from connection, deserialize bytes into message,
then find corresponding handler, put it into channel */
func readLoop(conn Connection, finish *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			holmes.Error("panics: %v", p)
		}
		finish.Done()
		conn.Close()
	}()

	for conn.IsRunning() {
		select {
		case <-conn.GetCloseChannel():
			return

		default:
			msg, err := conn.GetMessageCodec().Decode(conn)
			if err != nil {
				holmes.Error("Error decoding message %v", err)
				if _, ok := err.(ErrorUndefined); ok {
					// update heart beat timestamp
					conn.SetHeartBeat(time.Now().UnixNano())
					continue
				}
				return
			}

			// update heart beat timestamp
			conn.SetHeartBeat(time.Now().UnixNano())
			handler := HandlerMap.Get(msg.MessageNumber())
			if handler == nil {
				if conn.GetOnMessageCallback() != nil {
					holmes.Info("Message %d call onMessage()", msg.MessageNumber())
					conn.GetOnMessageCallback()(msg, conn)
				} else {
					holmes.Warn("No handler or onMessage() found for message %d", msg.MessageNumber())
				}
				continue
			}

			// send handler to handleLoop
			conn.GetMessageHandlerChannel() <- MessageHandler{msg, handler}
		}
	}
}

/* writeLoop() receive message from channel, serialize it into bytes,
then blocking write into connection */
func writeLoop(conn Connection, finish *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			holmes.Error("panics: %v", p)
		}
		// write all pending messages before close
		for packet := range conn.GetMessageSendChannel() {
			if packet != nil {
				if _, err := conn.GetRawConn().Write(packet); err != nil {
					holmes.Error("Error writing data %v", err)
				}
			}
		}
		finish.Done()
		conn.Close()
	}()

	for conn.IsRunning() {
		select {
		case <-conn.GetCloseChannel():
			return

		case packet := <-conn.GetMessageSendChannel():
			if packet != nil {
				if _, err := conn.GetRawConn().Write(packet); err != nil {
					holmes.Error("Error writing data %v", err)
					return
				}
			}
		}
	}
}

// handleServerLoop() - put handler or timeout callback into worker go-routines
func handleServerLoop(conn Connection, finish *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			holmes.Error("panics: %v", p)
		}
		finish.Done()
		conn.Close()
	}()

	for conn.IsRunning() {
		select {
		case <-conn.GetCloseChannel():
			return

		case msgHandler := <-conn.GetMessageHandlerChannel():
			msg := msgHandler.message
			handler := msgHandler.handler
			if !isNil(handler) {
				WorkerPoolInstance().Put(conn.GetNetId(), func() {
					handler(NewContext(msg, conn.GetNetId()), conn)
				})
				addTotalHandle()
			}

		case timeout := <-conn.GetTimeOutChannel():
			if timeout != nil {
				extraData := timeout.ExtraData.(int64)
				if extraData != conn.GetNetId() {
					holmes.Error("time out of %d running on client %d", extraData, conn.GetNetId())
				}
				WorkerPoolInstance().Put(conn.GetNetId(), func() {
					timeout.Callback(time.Now(), conn)
				})
			}
		}
	}
}

// handleClientLoop() - run handler or timeout callback in handleLoop() go-routine
func handleClientLoop(conn Connection, finish *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			holmes.Error("panics: %v", p)
		}
		finish.Done()
		conn.Close()
	}()

	for conn.IsRunning() {
		select {
		case <-conn.GetCloseChannel():
			return

		case msgHandler := <-conn.GetMessageHandlerChannel():
			msg := msgHandler.message
			handler := msgHandler.handler
			if !isNil(handler) {
				handler(NewContext(msg, conn.GetNetId()), conn)
			}

		case timeout := <-conn.GetTimeOutChannel():
			if timeout != nil {
				extraData := timeout.ExtraData.(int64)
				if extraData != conn.GetNetId() {
					holmes.Error("time out of %d running on client %d", extraData, conn.GetNetId())
				}
				timeout.Callback(time.Now(), conn)
			}
		}
	}
}
