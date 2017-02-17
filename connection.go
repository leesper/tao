package tao

import (
	"context"
	"crypto/tls"
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
	sc.ctx, sc.cancel = context.WithCancel(s.ctx)
	sc.name = c.RemoteAddr().String()
	sc.pending = []int64{}
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

	loopers := []func(interface{}, *sync.WaitGroup){readLoop, writeLoop, handleLoop}
	for _, l := range loopers {
		looper := l
		sc.wg.Add(1)
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

		// wait for go-routines holding readLoop, writeLoop and handleServerLoop to finish
		sc.mu.Lock()
		sc.cancel()
		pending := sc.pending
		sc.pending = nil
		sc.mu.Unlock()
		sc.wg.Wait()

		for _, id := range pending {
			sc.CancelTimer(id)
		}

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
	defer sc.mu.Unlock()
	if sc.pending != nil {
		sc.pending = append(sc.pending, timerID)
	}
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

// CancelTimer cancels a timer with the specified ID.
func (sc *ServerConn) CancelTimer(timerID int64) {
	cancelTimer(sc.belong.timing, timerID)
}

func cancelTimer(timing *TimingWheel, timerID int64) {
	if timing != nil {
		timing.CancelTimer(timerID)
	}
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
	return newClientConnWithOptions(netid, c, opts)
}

func newClientConnWithOptions(netid int64, c net.Conn, opts options) *ClientConn {
	cc := &ClientConn{
		addr:        c.RemoteAddr().String(),
		opts:        opts,
		netid:       netid,
		rawConn:     c,
		wg:          &sync.WaitGroup{},
		sendChan:    make(chan []byte, 1024),
		handlerChan: make(chan MessageHandler, 1024),
		heart:       time.Now().UnixNano(),
	}
	cc.ctx, cc.cancel = context.WithCancel(context.Background())
	cc.timing = NewTimingWheel(cc.ctx)
	cc.name = c.RemoteAddr().String()
	cc.pending = []int64{}
	return cc
}

// GetNetID returns the net ID of client connection.
func (cc *ClientConn) GetNetID() int64 {
	return cc.netid
}

// SetName sets the name of client connection.
func (cc *ClientConn) SetName(name string) {
	cc.mu.Lock()
	cc.name = name
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

	loopers := []func(interface{}, *sync.WaitGroup){readLoop, writeLoop, handleLoop}
	for _, l := range loopers {
		looper := l
		cc.wg.Add(1)
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
func (cc *ClientConn) reconnect() {
	var c net.Conn
	var err error
	if cc.opts.tlsCfg != nil {
		c, err = tls.Dial("tcp", cc.addr, cc.opts.tlsCfg)
		if err != nil {
			holmes.Fatal("tls dial error", err)
		}
	} else {
		c, err = net.Dial("tcp", cc.addr)
		if err != nil {
			holmes.Fatal("net dial error", err)
		}
	}
	cc = newClientConnWithOptions(cc.netid, c, cc.opts)
	cc.Start()
}

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
	defer cc.mu.Unlock()
	if cc.pending != nil {
		cc.pending = append(cc.pending, timerID)
	}
}

// CancelTimer cancels a timer with the specified ID.
func (cc *ClientConn) CancelTimer(timerID int64) {
	cancelTimer(cc.timing, timerID)
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
		cDone            <-chan struct{}
		sDone            <-chan struct{}
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
		cDone = c.ctx.Done()
		sDone = c.belong.ctx.Done()
		setHeartBeatFunc = c.SetHeartBeat
		onMessage = c.belong.opts.onMessage
	case *ClientConn:
		rawConn = c.rawConn
		codec = c.opts.codec
		cDone = c.ctx.Done()
		sDone = nil
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
		case <-cDone: // connection closed
			return
		case <-sDone: // server closed
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
func writeLoop(c interface{}, wg *sync.WaitGroup) {
	var (
		rawConn net.Conn
		sendCh  chan []byte
		cDone   <-chan struct{}
		sDone   <-chan struct{}
		pkt     []byte
		err     error
	)

	switch c := c.(type) {
	case *ServerConn:
		rawConn = c.rawConn
		sendCh = c.sendChan
		cDone = c.ctx.Done()
		sDone = c.belong.ctx.Done()
	case *ClientConn:
		rawConn = c.rawConn
		sendCh = c.sendChan
		cDone = c.ctx.Done()
		sDone = nil
	}

	defer func() {
		if p := recover(); p != nil {
			holmes.Error("panics: %v", p)
		}
		// drain all pending messages before exit
		for pkt = range sendCh {
			if pkt != nil {
				if _, err = rawConn.Write(pkt); err != nil {
					holmes.Error("error writing data %v", err)
				}
			}
		}
		wg.Done()
		rawConn.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			return
		case <-sDone: // server closed
			return
		case pkt = <-sendCh:
			if pkt != nil {
				if _, err = rawConn.Write(pkt); err != nil {
					holmes.Error("error writing data %v", err)
					return
				}
			}
		}
	}
}

// handleLoop() - put handler or timeout callback into worker go-routines
func handleLoop(c interface{}, wg *sync.WaitGroup) {
	var (
		rawConn      net.Conn
		cDone        <-chan struct{}
		sDone        <-chan struct{}
		timerChan    chan *OnTimeOut
		handlerChan  chan MessageHandler
		netID        int64
		ctx          context.Context
		askForWorker bool
	)

	switch c := c.(type) {
	case *ServerConn:
		rawConn = c.rawConn
		cDone = c.ctx.Done()
		sDone = c.belong.ctx.Done()
		timerChan = c.belong.timing.GetTimeOutChannel()
		handlerChan = c.handlerChan
		netID = c.netid
		ctx = c.ctx
		askForWorker = true
	case *ClientConn:
		rawConn = c.rawConn
		cDone = c.ctx.Done()
		sDone = nil
		timerChan = c.timing.timeOutChan
		handlerChan = c.handlerChan
		netID = c.netid
		ctx = c.ctx
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
		case <-cDone: // connectin closed
			return
		case <-sDone: // server closed
			return
		case msgHandler := <-handlerChan:
			msg, handler := msgHandler.message, msgHandler.handler
			if handler != nil {
				if askForWorker {
					WorkerPoolInstance().Put(netID, func() {
						handler(NewContextWithMessage(ctx, msg), c)
					})
					addTotalHandle()
				} else {
					handler(NewContextWithMessage(ctx, msg), c)
				}
			}
		case timeout := <-timerChan:
			if timeout != nil {
				extraData := timeout.ExtraData.(int64)
				if extraData != netID {
					holmes.Error("timeout net %d, conn net %d, mismatched!", extraData, netID)
				}
				if askForWorker {
					WorkerPoolInstance().Put(netID, func() {
						timeout.Callback(time.Now(), c)
					})
				} else {
					timeout.Callback(time.Now(), c)
				}
			}
		}
	}
}
