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

// WriteCloser is the interface that groups Write and Close methods.
type WriteCloser interface {
	Write(Message) error
	Close()
}

// ServerConn represents a server connection to a TCP server, it implments Conn.
type ServerConn struct {
	netid   int64
	belong  *Server
	rawConn net.Conn

	once      *sync.Once
	wg        *sync.WaitGroup
	sendCh    chan []byte
	handlerCh chan MessageHandler
	timerCh   chan *OnTimeOut

	mu      sync.Mutex // guards following
	name    string
	heart   int64
	pending []int64
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewServerConn returns a new server connection which has not started to
// serve requests yet.
func NewServerConn(id int64, s *Server, c net.Conn) *ServerConn {
	sc := &ServerConn{
		netid:     id,
		belong:    s,
		rawConn:   c,
		once:      &sync.Once{},
		wg:        &sync.WaitGroup{},
		sendCh:    make(chan []byte, s.opts.bufferSize),
		handlerCh: make(chan MessageHandler, s.opts.bufferSize),
		timerCh:   make(chan *OnTimeOut, s.opts.bufferSize),
		heart:     time.Now().UnixNano(),
	}
	sc.ctx, sc.cancel = context.WithCancel(context.WithValue(s.ctx, serverCtx, s))
	sc.name = c.RemoteAddr().String()
	sc.pending = []int64{}
	return sc
}

// ServerFromContext returns the server within the context.
func ServerFromContext(ctx context.Context) (*Server, bool) {
	server, ok := ctx.Value(serverCtx).(*Server)
	return server, ok
}

// NetID returns net ID of server connection.
func (sc *ServerConn) NetID() int64 {
	return sc.netid
}

// SetName sets name of server connection.
func (sc *ServerConn) SetName(name string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.name = name
}

// Name returns the name of server connection.
func (sc *ServerConn) Name() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	name := sc.name
	return name
}

// SetHeartBeat sets the heart beats of server connection.
func (sc *ServerConn) SetHeartBeat(heart int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.heart = heart
}

// HeartBeat returns the heart beats of server connection.
func (sc *ServerConn) HeartBeat() int64 {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	heart := sc.heart
	return heart
}

// SetContextValue sets extra data to server connection.
func (sc *ServerConn) SetContextValue(k, v interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.ctx = context.WithValue(sc.ctx, k, v)
}

// ContextValue gets extra data from server connection.
func (sc *ServerConn) ContextValue(k interface{}) interface{} {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.ctx.Value(k)
}

// Start starts the server connection, creating go-routines for reading,
// writing and handlng.
func (sc *ServerConn) Start() {
	holmes.Infof("conn start, <%v -> %v>\n", sc.rawConn.LocalAddr(), sc.rawConn.RemoteAddr())
	onConnect := sc.belong.opts.onConnect
	if onConnect != nil {
		onConnect(sc)
	}

	loopers := []func(WriteCloser, *sync.WaitGroup){readLoop, writeLoop, handleLoop}
	for _, l := range loopers {
		looper := l
		sc.wg.Add(1)
		go looper(sc, sc.wg)
	}
}

// Close gracefully closes the server connection. It blocked until all sub
// go-routines are completed and returned.
func (sc *ServerConn) Close() {
	sc.once.Do(func() {
		holmes.Infof("conn close gracefully, <%v -> %v>\n", sc.rawConn.LocalAddr(), sc.rawConn.RemoteAddr())

		// callback on close
		onClose := sc.belong.opts.onClose
		if onClose != nil {
			onClose(sc)
		}

		// remove connection from server
		sc.belong.conns.Remove(sc.netid)
		addTotalConn(-1)

		// close net.Conn, any blocked read or write operation will be unblocked and
		// return errors.
		if tc, ok := sc.rawConn.(*net.TCPConn); ok {
			// avoid time-wait state
			tc.SetLinger(0)
		}
		sc.rawConn.Close()

		// cancel readLoop, writeLoop and handleLoop go-routines.
		sc.mu.Lock()
		sc.cancel()
		pending := sc.pending
		sc.pending = nil
		sc.mu.Unlock()

		// clean up pending timers
		for _, id := range pending {
			sc.CancelTimer(id)
		}

		// wait until all go-routines exited.
		sc.wg.Wait()

		// close all channels and block until all go-routines exited.
		close(sc.sendCh)
		close(sc.handlerCh)
		close(sc.timerCh)

		// tell server I'm done :( .
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
func (sc *ServerConn) RunAt(timestamp time.Time, callback func(time.Time, WriteCloser)) int64 {
	id := runAt(sc.ctx, sc.netid, sc.belong.timing, timestamp, callback)
	if id >= 0 {
		sc.AddPendingTimer(id)
	}
	return id
}

// RunAfter runs a callback right after the specified duration ellapsed.
func (sc *ServerConn) RunAfter(duration time.Duration, callback func(time.Time, WriteCloser)) int64 {
	id := runAfter(sc.ctx, sc.netid, sc.belong.timing, duration, callback)
	if id >= 0 {
		sc.AddPendingTimer(id)
	}
	return id
}

// RunEvery runs a callback on every interval time.
func (sc *ServerConn) RunEvery(interval time.Duration, callback func(time.Time, WriteCloser)) int64 {
	id := runEvery(sc.ctx, sc.netid, sc.belong.timing, interval, callback)
	if id >= 0 {
		sc.AddPendingTimer(id)
	}
	return id
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

// RemoteAddr returns the remote address of server connection.
func (sc *ServerConn) RemoteAddr() net.Addr {
	return sc.rawConn.RemoteAddr()
}

// LocalAddr returns the local address of server connection.
func (sc *ServerConn) LocalAddr() net.Addr {
	return sc.rawConn.LocalAddr()
}

// ClientConn represents a client connection to a TCP server.
type ClientConn struct {
	addr      string
	opts      options
	netid     int64
	rawConn   net.Conn
	once      *sync.Once
	wg        *sync.WaitGroup
	sendCh    chan []byte
	handlerCh chan MessageHandler
	timing    *TimingWheel
	mu        sync.Mutex // guards following
	name      string
	heart     int64
	pending   []int64
	ctx       context.Context
	cancel    context.CancelFunc
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
	if opts.bufferSize <= 0 {
		opts.bufferSize = BufferSize256
	}
	return newClientConnWithOptions(netid, c, opts)
}

func newClientConnWithOptions(netid int64, c net.Conn, opts options) *ClientConn {
	cc := &ClientConn{
		addr:      c.RemoteAddr().String(),
		opts:      opts,
		netid:     netid,
		rawConn:   c,
		once:      &sync.Once{},
		wg:        &sync.WaitGroup{},
		sendCh:    make(chan []byte, opts.bufferSize),
		handlerCh: make(chan MessageHandler, opts.bufferSize),
		heart:     time.Now().UnixNano(),
	}
	cc.ctx, cc.cancel = context.WithCancel(context.Background())
	cc.timing = NewTimingWheel(cc.ctx)
	cc.name = c.RemoteAddr().String()
	cc.pending = []int64{}
	return cc
}

// NetID returns the net ID of client connection.
func (cc *ClientConn) NetID() int64 {
	return cc.netid
}

// SetName sets the name of client connection.
func (cc *ClientConn) SetName(name string) {
	cc.mu.Lock()
	cc.name = name
	cc.mu.Unlock()
}

// Name gets the name of client connection.
func (cc *ClientConn) Name() string {
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

// HeartBeat gets the heart beats of client connection.
func (cc *ClientConn) HeartBeat() int64 {
	cc.mu.Lock()
	heart := cc.heart
	cc.mu.Unlock()
	return heart
}

// SetContextValue sets extra data to client connection.
func (cc *ClientConn) SetContextValue(k, v interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.ctx = context.WithValue(cc.ctx, k, v)
}

// ContextValue gets extra data from client connection.
func (cc *ClientConn) ContextValue(k interface{}) interface{} {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.ctx.Value(k)
}

// Start starts the client connection, creating go-routines for reading,
// writing and handlng.
func (cc *ClientConn) Start() {
	holmes.Infof("conn start, <%v -> %v>\n", cc.rawConn.LocalAddr(), cc.rawConn.RemoteAddr())
	onConnect := cc.opts.onConnect
	if onConnect != nil {
		onConnect(cc)
	}

	loopers := []func(WriteCloser, *sync.WaitGroup){readLoop, writeLoop, handleLoop}
	for _, l := range loopers {
		looper := l
		cc.wg.Add(1)
		go looper(cc, cc.wg)
	}
}

// Close gracefully closes the client connection. It blocked until all sub
// go-routines are completed and returned.
func (cc *ClientConn) Close() {
	cc.once.Do(func() {
		holmes.Infof("conn close gracefully, <%v -> %v>\n", cc.rawConn.LocalAddr(), cc.rawConn.RemoteAddr())

		// callback on close
		onClose := cc.opts.onClose
		if onClose != nil {
			onClose(cc)
		}

		// close net.Conn, any blocked read or write operation will be unblocked and
		// return errors.
		cc.rawConn.Close()

		// cancel readLoop, writeLoop and handleLoop go-routines.
		cc.mu.Lock()
		cc.cancel()
		cc.pending = nil
		cc.mu.Unlock()

		// stop timer
		cc.timing.Stop()

		// wait until all go-routines exited.
		cc.wg.Wait()

		// close all channels.
		close(cc.sendCh)
		close(cc.handlerCh)

		// cc.once is a *sync.Once. After reconnect() returned, cc.once will point
		// to a newly-allocated one while other go-routines such as readLoop,
		// writeLoop and handleLoop blocking on the old *sync.Once continue to
		// execute Close() (and of course do nothing because of sync.Once).
		// NOTE that it will cause an "unlock of unlocked mutex" error if cc.once is
		// a sync.Once struct, because "defer o.m.Unlock()" in sync.Once.Do() will
		// be performed on an unlocked mutex(the newly-allocated one noticed above)
		if cc.opts.reconnect {
			cc.reconnect()
		}
	})
}

// reconnect reconnects and returns a new *ClientConn.
func (cc *ClientConn) reconnect() {
	var c net.Conn
	var err error
	if cc.opts.tlsCfg != nil {
		c, err = tls.Dial("tcp", cc.addr, cc.opts.tlsCfg)
		if err != nil {
			holmes.Fatalln("tls dial error", err)
		}
	} else {
		c, err = net.Dial("tcp", cc.addr)
		if err != nil {
			holmes.Fatalln("net dial error", err)
		}
	}
	// copy the newly-created *ClientConn to cc, so after
	// reconnect returned cc will be updated to new one.
	*cc = *newClientConnWithOptions(cc.netid, c, cc.opts)
	cc.Start()
}

// Write writes a message to the client.
func (cc *ClientConn) Write(message Message) error {
	return asyncWrite(cc, message)
}

// RunAt runs a callback at the specified timestamp.
func (cc *ClientConn) RunAt(timestamp time.Time, callback func(time.Time, WriteCloser)) int64 {
	id := runAt(cc.ctx, cc.netid, cc.timing, timestamp, callback)
	if id >= 0 {
		cc.AddPendingTimer(id)
	}
	return id
}

// RunAfter runs a callback right after the specified duration ellapsed.
func (cc *ClientConn) RunAfter(duration time.Duration, callback func(time.Time, WriteCloser)) int64 {
	id := runAfter(cc.ctx, cc.netid, cc.timing, duration, callback)
	if id >= 0 {
		cc.AddPendingTimer(id)
	}
	return id
}

// RunEvery runs a callback on every interval time.
func (cc *ClientConn) RunEvery(interval time.Duration, callback func(time.Time, WriteCloser)) int64 {
	id := runEvery(cc.ctx, cc.netid, cc.timing, interval, callback)
	if id >= 0 {
		cc.AddPendingTimer(id)
	}
	return id
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

// RemoteAddr returns the remote address of server connection.
func (cc *ClientConn) RemoteAddr() net.Addr {
	return cc.rawConn.RemoteAddr()
}

// LocalAddr returns the local address of server connection.
func (cc *ClientConn) LocalAddr() net.Addr {
	return cc.rawConn.LocalAddr()
}

func runAt(ctx context.Context, netID int64, timing *TimingWheel, ts time.Time, cb func(time.Time, WriteCloser)) int64 {
	timeout := NewOnTimeOut(NewContextWithNetID(ctx, netID), cb)
	return timing.AddTimer(ts, 0, timeout)
}

func runAfter(ctx context.Context, netID int64, timing *TimingWheel, d time.Duration, cb func(time.Time, WriteCloser)) int64 {
	delay := time.Now().Add(d)
	return runAt(ctx, netID, timing, delay, cb)
}

func runEvery(ctx context.Context, netID int64, timing *TimingWheel, d time.Duration, cb func(time.Time, WriteCloser)) int64 {
	delay := time.Now().Add(d)
	timeout := NewOnTimeOut(NewContextWithNetID(ctx, netID), cb)
	return timing.AddTimer(delay, d, timeout)
}

func asyncWrite(c interface{}, m Message) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = ErrServerClosed
		}
	}()

	var (
		pkt    []byte
		sendCh chan []byte
	)
	switch c := c.(type) {
	case *ServerConn:
		pkt, err = c.belong.opts.codec.Encode(m)
		sendCh = c.sendCh

	case *ClientConn:
		pkt, err = c.opts.codec.Encode(m)
		sendCh = c.sendCh
	}

	if err != nil {
		holmes.Errorf("asyncWrite error %v\n", err)
		return
	}

	select {
	case sendCh <- pkt:
		err = nil
	default:
		err = ErrWouldBlock
	}
	return
}

/* readLoop() blocking read from connection, deserialize bytes into message,
then find corresponding handler, put it into channel */
func readLoop(c WriteCloser, wg *sync.WaitGroup) {
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
		handlerCh = c.handlerCh
	case *ClientConn:
		rawConn = c.rawConn
		codec = c.opts.codec
		cDone = c.ctx.Done()
		sDone = nil
		setHeartBeatFunc = c.SetHeartBeat
		onMessage = c.opts.onMessage
		handlerCh = c.handlerCh
	}

	defer func() {
		if p := recover(); p != nil {
			holmes.Errorf("panics: %v\n", p)
		}
		wg.Done()
		holmes.Debugln("readLoop go-routine exited")
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			holmes.Debugln("receiving cancel signal from conn")
			return
		case <-sDone: // server closed
			holmes.Debugln("receiving cancel signal from server")
			return
		default:
			msg, err = codec.Decode(rawConn)
			if err != nil {
				holmes.Errorf("error decoding message %v\n", err)
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
					holmes.Infof("message %d call onMessage()\n", msg.MessageNumber())
					onMessage(msg, c.(WriteCloser))
				} else {
					holmes.Warnf("no handler or onMessage() found for message %d\n", msg.MessageNumber())
				}
				continue
			}
			handlerCh <- MessageHandler{msg, handler}
		}
	}
}

/* writeLoop() receive message from channel, serialize it into bytes,
then blocking write into connection */
func writeLoop(c WriteCloser, wg *sync.WaitGroup) {
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
		sendCh = c.sendCh
		cDone = c.ctx.Done()
		sDone = c.belong.ctx.Done()
	case *ClientConn:
		rawConn = c.rawConn
		sendCh = c.sendCh
		cDone = c.ctx.Done()
		sDone = nil
	}

	defer func() {
		if p := recover(); p != nil {
			holmes.Errorf("panics: %v\n", p)
		}
		// drain all pending messages before exit
	OuterFor:
		for {
			select {
			case pkt = <-sendCh:
				if pkt != nil {
					if _, err = rawConn.Write(pkt); err != nil {
						holmes.Errorf("error writing data %v\n", err)
					}
				}
			default:
				break OuterFor
			}
		}
		wg.Done()
		holmes.Debugln("writeLoop go-routine exited")
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			holmes.Debugln("receiving cancel signal from conn")
			return
		case <-sDone: // server closed
			holmes.Debugln("receiving cancel signal from server")
			return
		case pkt = <-sendCh:
			if pkt != nil {
				if _, err = rawConn.Write(pkt); err != nil {
					holmes.Errorf("error writing data %v\n", err)
					return
				}
			}
		}
	}
}

// handleLoop() - put handler or timeout callback into worker go-routines
func handleLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		cDone        <-chan struct{}
		sDone        <-chan struct{}
		timerCh      chan *OnTimeOut
		handlerCh    chan MessageHandler
		netID        int64
		ctx          context.Context
		askForWorker bool
		err          error
	)

	switch c := c.(type) {
	case *ServerConn:
		cDone = c.ctx.Done()
		sDone = c.belong.ctx.Done()
		timerCh = c.timerCh
		handlerCh = c.handlerCh
		netID = c.netid
		ctx = c.ctx
		askForWorker = true
	case *ClientConn:
		cDone = c.ctx.Done()
		sDone = nil
		timerCh = c.timing.timeOutChan
		handlerCh = c.handlerCh
		netID = c.netid
		ctx = c.ctx
	}

	defer func() {
		if p := recover(); p != nil {
			holmes.Errorf("panics: %v\n", p)
		}
		wg.Done()
		holmes.Debugln("handleLoop go-routine exited")
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connectin closed
			holmes.Debugln("receiving cancel signal from conn")
			return
		case <-sDone: // server closed
			holmes.Debugln("receiving cancel signal from server")
			return
		case msgHandler := <-handlerCh:
			msg, handler := msgHandler.message, msgHandler.handler
			if handler != nil {
				if askForWorker {
					err = WorkerPoolInstance().Put(netID, func() {
						handler(NewContextWithNetID(NewContextWithMessage(ctx, msg), netID), c)
					})
					if err != nil {
						holmes.Errorln(err)
					}
					addTotalHandle()
				} else {
					handler(NewContextWithNetID(NewContextWithMessage(ctx, msg), netID), c)
				}
			}
		case timeout := <-timerCh:
			if timeout != nil {
				timeoutNetID := NetIDFromContext(timeout.Ctx)
				if timeoutNetID != netID {
					holmes.Errorf("timeout net %d, conn net %d, mismatched!\n", timeoutNetID, netID)
				}
				if askForWorker {
					err = WorkerPoolInstance().Put(netID, func() {
						timeout.Callback(time.Now(), c.(WriteCloser))
					})
					if err != nil {
						holmes.Errorln(err)
					}
				} else {
					timeout.Callback(time.Now(), c.(WriteCloser))
				}
			}
		}
	}
}
