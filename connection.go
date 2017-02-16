package tao

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/leesper/holmes"
)

const (
	// MessageTypeBytes is the length of type header.
	MessageTypeBytes = 4
	// MessageLenBytes is the length of length header.
	MessageLenBytes = 4
	// MessageMaxBytes is the maximum bytes allowed for application data.
	MessageMaxBytes = 1 << 23 // 8M
)

// MessageHandler is a combination of message and its handler function.
type MessageHandler struct {
	message Message
	handler HandlerFunc
}

// ServerConn represents a server connection to a TCP server, it implments Conn.
type ServerConn struct {
	netid   int64
	belong  *TCPServer
	rawConn net.Conn

	once        sync.Once
	wg          *sync.WaitGroup
	sendChan    chan []byte
	handlerChan chan MessageHandler
	timerChan   chan *OnTimeOut

	mu      sync.Mutex // guards following
	name    string
	heart   int64
	pending []int64
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewServerConn returns a new server connection which has not started to
// serve requests yet.
func NewServerConn(id int64, s *TCPServer, c net.Conn) *ServerConn {
	sc := &ServerConn{
		netid:       id,
		belong:      s,
		rawConn:     c,
		wg:          &sync.WaitGroup{},
		sendChan:    make(chan []byte, 1024),
		handlerChan: make(chan MessageHandler, 1024),
		timerChan:   make(chan *OnTimeOut),
		heart:       time.Now().UnixNano(),
	}
	sc.ctx, sc.cancel = context.WithCancel(context.Background())
	sc.name.Store(c.RemoteAddr().String())
	sc.pending.Store([]int64{})
	return sc
}

// GetNetID returns net ID of server connection.
func (sc *ServerConn) GetNetID() int64 {
	return sc.netid
}

// SetName sets name of server connection.
func (sc *ServerConn) SetName(name string) {
	sc.mu.Lock()
	sc.name = name
	sc.mu.Unlock()
}

// GetName returns the name of server connection.
func (sc *ServerConn) GetName() string {
	sc.mu.Lock()
	name := sc.name
	sc.mu.Unlock()
	return name
}

// SetHeartBeat sets the heart beats of server connection.
func (sc *ServerConn) SetHeartBeat(heart int64) {
	sc.mu.Lock()
	sc.heart = heart
	sc.mu.Unlock()
}

// GetHeartBeat returns the heart beats of server connection.
func (sc *ServerConn) GetHeartBeat() int64 {
	sc.mu.Lock()
	heart := sc.heart
	sc.mu.Unlock()
	return heart
}

// SetExtraData sets extra data to server connection.
func (sc *ServerConn) SetExtraData(extra interface{}) {
	sc.mu.Lock()
	sc.ctx = context.WithValue(sc.ctx, srvCtxKey, extra)
	sc.mu.Unlock()
}

// GetExtraData gets extra data from server connection.
func (sc *ServerConn) GetExtraData() interface{} {
	sc.mu.Lock()
	extra := sc.ctx.Value(srvCtxKey)
	sc.mu.Unlock()
	return extra
}

// Start starts the server connection, creating go-routines for reading,
// writing and handlng.
func (sc *ServerConn) Start() {
	onConnect := sc.belong.opts.onConnect
	if onConnect != nil {
		onConnect(sc)
	}

	sc.wg.Add(3)
	loopers := []func(Connection, *sync.WaitGroup){readLoop, writeLoop, handleServerLoop}
	for _, l := range loopers {
		looper := l
		go looper(sc, sc.wg)
	}
}

// Stop gracefully closes the server connection. It blocked until all sub
// go-routines are completed and returned.
func (sc *ServerConn) Stop() {
	sc.once.Do(func() {
		onClose := sc.belong.opts.onClose
		if onClose != nil {
			onClose(sc)
		}

		close(sc.sendChan)
		close(sc.handlerChan)
		// TODO(lkj) will this cause a panic ?
		close(sc.timerChan)

		for _, id := range sc.pending.Load().([]int64) {
			sc.CancelTimer(id)
		}

		// wait for go-routines holding readLoop, writeLoop and handleServerLoop to finish
		sc.mu.Lock()
		sc.cancel()
		sc.mu.Unlock()
		sc.wg.Wait()

		if tc, ok := sc.rawConn.(*net.TCPConn); ok {
			// avoid time-wait state
			tc.SetLinger(0)
		}
		sc.rawConn.Close()

		sc.belong.conns.Remove(sc.netid)
		addTotalConn(-1)
		holmes.Info("HOW MANY CONNECTIONS DO I HAVE: %d", sc.belong.conns.Size())

		sc.belong.wg.Done()
	})
}

// AddPendingTimer adds a timer ID to server Connection.
func (sc *ServerConn) AddPendingTimer(timerID int64) {
	sc.mu.Lock()
	sc.pending = append(sc.pending, timerID)
	sc.mu.Unlock()
}

// GetMessageCodec returns the underlying codec.
func (sc *ServerConn) GetMessageCodec() Codec {
	return sc.belong.opts.codec
}

// Write writes a message to the client.
func (sc *ServerConn) Write(message Message) error {
	return asyncWrite(sc, message)
}

// RunAt runs a callback at the specified timestamp.
func (sc *ServerConn) RunAt(timestamp time.Time, callback func(time.Time, interface{})) int64 {
	return runAt(sc, timestamp, callback)
}

// RunAfter runs a callback right after the specified duration ellapsed.
func (sc *ServerConn) RunAfter(duration time.Duration, callback func(time.Time, interface{})) int64 {
	return runAfter(sc, duration, callback)
}

// RunEvery runs a callback on every interval time.
func (sc *ServerConn) RunEvery(interval time.Duration, callback func(time.Time, interface{})) int64 {
	return runEvery(sc, interval, callback)
}

// GetTimingWheel gets the timer controller from the belonged server.
func (sc *ServerConn) GetTimingWheel() *TimingWheel {
	return sc.belong.timing
}

// CancelTimer cancels a timer with the specified ID.
func (sc *ServerConn) CancelTimer(timerID int64) {
	conn.GetTimingWheel().CancelTimer(timerID)
}

// GetRemoteAddr returns the peer address of server connection.
func (sc *ServerConn) GetRemoteAddr() net.Addr {
	return sc.rawConn.RemoteAddr()
}

// ClientConn represents a client connection to a TCP server.
type ClientConn struct {
	addr        string
	opts        options
	netid       int64
	rawConn     net.Conn
	once        sync.Once
	wg          *sync.WaitGroup
	sendChan    chan []byte
	handlerChan chan MessageHandler
	timing      *TimingWheel
	mu          sync.Mutex // guards following
	name        string
	heart       int64
	pending     []int64
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewClientConn returns a new client connection which has not started to
// serve requests yet.
func NewClientConn(netid int64, c net.Conn, opt ...ServerOption) *ClientConn {
	var opts options
	for _, o := range opt {
		o(&opts)
	}
	if opts.codec == nil {
		opts.codec = TypeLengthValueCodec{}
	}

	cc := &ClientConn{
		opts:        opts,
		netid:       netid,
		rawConn:     c,
		wg:          &sync.WaitGroup{},
		sendChan:    make(chan []byte, 1024),
		handlerChan: make(chan MessageHandler, 1024),
		heart:       time.Now().UnixNano(),
		address:     c.RemoteAddr().String(),
		timing:      NewTimingWheel(),
	}
	cc.ctx, cc.cancel = context.WithCancel(context.Background())
	cc.name.Store(c.RemoteAddr().String())
	cc.pending.Store([]int64{})
	return cc
}

// GetNetID returns the net ID of client connection.
func (cc *ClientConn) GetNetID() int64 {
	return cc.netid
}

// SetName sets the name of client connection.
func (cc *ClientConn) SetName(name string) {
	cc.mu.Lock()
	cc.name.Store(name)
	cc.mu.Unlock()
}

// GetName gets the name of client connection.
func (cc *ClientConn) GetName() string {
	cc.mu.Lock()
	name := cc.name
	cc.mu.Unlock()
	return name
}

// SetHeartBeat sets the heart beats of client connection.
func (cc *ClientConn) SetHeartBeat(heart int64) {
	cc.mu.Lock()
	cc.heart = heart
	cc.mu.Unlock()
}

// GetHeartBeat gets the heart beats of client connection.
func (cc *ClientConn) GetHeartBeat() int64 {
	cc.mu.Lock()
	heart := cc.heart
	cc.mu.Unlock()
	return heart
}

// GetMessageCodec returns the underlying codec.
func (cc *ClientConn) GetMessageCodec() Codec {
	return cc.opts.codec
}

// SetExtraData sets extra data to client connection.
func (cc *ClientConn) SetExtraData(extra interface{}) {
	cc.mu.Lock()
	cc.ctx = context.WithValue(cc.ctx, cliCtxKey, extra)
	cc.mu.Unlock()
}

// GetExtraData gets extra data from client connection.
func (cc *ClientConn) GetExtraData() interface{} {
	cc.mu.Lock()
	extra := cc.ctx.Value(cliCtxKey)
	cc.mu.Unlock()
	return extra
}

// Start starts the client connection, creating go-routines for reading,
// writing and handlng.
func (cc *ClientConn) Start() {
	onConnect := cc.opts.onConnect
	if onConnect != nil {
		onConnect(cc)
	}

	cc.wg.Add(3)
	loopers := []func(Connection, *sync.WaitGroup){readLoop, writeLoop, handleClientLoop}
	for _, l := range loopers {
		looper := l
		go looper(cc, cc.wg)
	}
}

// Stop gracefully closes the client connection. It blocked until all sub
// go-routines are completed and returned.
func (cc *ClientConn) Stop() {
	cc.once.Do(func() {
		onClose := cc.opts.onClose
		if onClose != nil {
			onClose(cc)
		}

		close(cc.sendChan)
		close(cc.handlerChan)
		cc.timing.Stop()

		cc.mu.Lock()
		cc.cancel()
		cc.mu.Unlock()

		// wait for all loops to finish
		cc.wg.Wait()
		cc.rawConn.Close()
	})

}

// deprecated
// func (client *ClientConnection) reconnect() {
// 	var c net.Conn
// 	var err error
// 	if client.tconfig == nil {
// 		c, err = net.Dial("tcp", client.address)
// 		if err != nil {
// 			holmes.Fatal("net Dial error %v", err)
// 		}
// 	} else {
// 		c, err = tls.Dial("tcp", client.address, client.tconfig)
// 		if err != nil {
// 			holmes.Fatal("tls Dial error %v", err)
// 		}
// 	}
//
// 	client.name = c.RemoteAddr().String()
// 	client.heartBeat = time.Now().UnixNano()
// 	client.extraData.Store(struct{}{})
// 	client.once = &sync.Once{}
// 	client.pendingTimers = []int64{}
// 	client.timingWheel = NewTimingWheel()
// 	client.conn = c
// 	client.messageSendChan = make(chan []byte, 1024)
// 	client.messageHandlerChan = make(chan MessageHandler, 1024)
// 	client.closeConnChan = make(chan struct{})
// 	client.Start()
// 	client.running.CompareAndSet(false, true)
// }

// Write writes a message to the client.
func (cc *ClientConn) Write(message Message) error {
	return asyncWrite(cc, message)
}

// RunAt runs a callback at the specified timestamp.
func (cc *ClientConn) RunAt(timestamp time.Time, callback func(time.Time, interface{})) int64 {
	return runAt(cc, timestamp, callback)
}

// RunAfter runs a callback right after the specified duration ellapsed.
func (cc *ClientConn) RunAfter(duration time.Duration, callback func(time.Time, interface{})) int64 {
	return runAfter(cc, duration, callback)
}

// RunEvery runs a callback on every interval time.
func (cc *ClientConn) RunEvery(interval time.Duration, callback func(time.Time, interface{})) int64 {
	return runEvery(cc, interval, callback)
}

// AddPendingTimer adds a new timer ID to client connection.
func (cc *ClientConn) AddPendingTimer(timerID int64) {
	cc.mu.Lock()
	cc.pending = append(cc.pending, timerID)
	cc.mu.Unlock()
}

// CancelTimer cancels a timer with the specified ID.
func (cc *ClientConn) CancelTimer(timerID int64) {
	cc.timing.CancelTimer(timerID)
}

// GetRemoteAddr returns the peer address of server connection.
func (cc *ClientConn) GetRemoteAddr() net.Addr {
	return cc.rawConn.RemoteAddr()
}

func runAt(c interface{}, ts time.Time, cb func(time.Time, interface{})) int64 {
	var id int64 = -1
	switch c := c.(type) {
	case *ServerConn:
		timeout := NewOnTimeOut(c.netid, cb)
		id = c.belong.timing.AddTimer(ts, 0, timeout)
		if id >= 0 {
			c.AddPendingTimer(id)
		}
	case *ClientConn:
		timeout := NewOnTimeOut(c.netid, cb)
		id = c.timing.AddTimer(ts, 0, timeout)
		if id >= 0 {
			c.AddPendingTimer(id)
		}
	}

	return id
}

func runAfter(c interface{}, d time.Duration, cb func(time.Time, interface{})) int64 {
	delay := time.Now().Add(d)
	return runAt(c, delay, cb)
}

func runEvery(c interface{}, d time.Duration, cb func(time.Time, interface{})) int64 {
	var id int64 = -1
	delay := time.Now().Add(d)
	switch c := c.(type) {
	case *ServerConn:
		timeout := NewOnTimeOut(c.netid, cb)
		id = c.belong.timing.AddTimer(delay, d, timeout)
		if id >= 0 {
			c.AddPendingTimer(id)
		}
	case *ClientConn:
		timeout := NewOnTimeOut(c.netid, cb)
		id = c.timing.AddTimer(delay, d, timeout)
		if id >= 0 {
			c.AddPendingTimer(id)
		}
	}
	return id
}

func asyncWrite(c interface{}, m Message) error {
	defer func() error {
		if p := recover(); p != nil {
			return ErrServerClosed
		}
		return nil
	}()

	var (
		pkt    []byte
		err    error
		sendCh chan []byte
	)
	switch c := c.(type) {
	case *ServerConn:
		pkt, err = c.belong.opts.codec.Encode(m)
		sendCh = c.sendChan

	case *ClientConn:
		pkt, err = c.opts.codec.Encode(m)
		sendCh = c.sendChan
	}

	if err != nil {
		holmes.Error("asyncWrite error %v", err)
		return err
	}

	select {
	case sendCh <- pkt:
		return nil
	default:
		return ErrWouldBlock
	}
}

/* readLoop() blocking read from connection, deserialize bytes into message,
then find corresponding handler, put it into channel */
func readLoop(c interface{}, wg *sync.WaitGroup) {
	var (
		rawConn          net.Conn
		codec            Codec
		done             chan struct{}
		setHeartBeatFunc func(int64)
		onMessage        onMessageFunc
		handlerCh        chan MessageHandler
		msg              Message
		err              error
	)

	switch c := c.(type) {
	case *ServerConn:
		rawConn = c.rawConn
		codec = c.belong.opts.codec
		done = c.ctx.Done()
		setHeartBeatFunc = c.SetHeartBeat
		onMessage = c.belong.opts.onMessage
	case *ClientConn:
		rawConn = c.rawConn
		codec = c.opts.codec
		done = c.ctx.Done()
		setHeartBeatFunc = c.SetHeartBeat
		onMessage = c.opts.onMessage
	}

	defer func() {
		if p := recover(); p != nil {
			holmes.Error("panics: %v", p)
		}
		wg.Done()
		rawConn.Close()
	}()

	for {
		select {
		case <-done:
			return
		default:
			msg, err = codec.Decode(rawConn)
			if err != nil {
				holmes.Error("error decoding message %v", err)
				if _, ok := err.(ErrUndefined); ok {
					// update heart beats
					setHeartBeatFunc(time.Now().UnixNano())
					continue
				}
				return
			}
			setHeartBeatFunc(time.Now().UnixNano())
			handler := GetHandlerFunc(msg.MessageNumber())
			if handler == nil {
				if onMessage != nil {
					holmes.Info("message %d call onMessage()", msg.MessageNumber())
					onMessage(msg, c)
				} else {
					holmes.Warn("no handler or onMessage() found for message %d", msg.MessageNumber())
				}
				continue
			}
			handlerCh <- MessageHandler{msg, handler}
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
					handler(NewContextWithMessage(context.Background(), msg), conn)
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
				handler(NewContextWithMessage(context.Background(), msg), conn)
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
