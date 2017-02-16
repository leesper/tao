package tao

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"flag"
	"net"
	"os"
	"sync"
	"time"

	"github.com/leesper/holmes"
)

func init() {
	flag.Parse()
	netIdentifier = NewAtomicInt64(0)
	tlsWrapper = func(conn net.Conn) net.Conn {
		return conn
	}
}

var (
	netIdentifier *AtomicInt64
	tlsWrapper    func(net.Conn) net.Conn
)

type options struct {
	tlsCfg *tls.Config
	codec  Codec
}

// ServerOption sets server options.
type ServerOption func(*options)

// CustomCodec returns a ServerOption that will apply a custom Codec.
func CustomCodec(codec Codec) ServerOption {
	return func(o *options) {
		o.codec = codec
	}
}

// TLSCreds returns a ServerOption taht will set TLS credentials for server connections.
func TLSCreds(config *tls.Config) ServerOption {
	return func(o *options) {
		o.tlsCfg = config
	}
}

// TCPServer is a server to serve TCP requests.
type TCPServer struct {
	opts   options
	ctx    context.Context
	cancel context.CancelFunc

	conns  *ConnMap
	timing *TimingWheel
	wg     *sync.WaitGroup

	// guards following
	mu        sync.Mutex
	lis       map[net.Listener]bool
	onConnect onConnectFunc
	onMessage onMessageFunc
	onClose   onCloseFunc
	onError   onErrorFunc
	// for periodically running function every duration.
	interv time.Duration
	sched  onScheduleFunc
}

// NewTCPServer returns a new TCP server which has not started
// to server requests yet.
func NewTCPServer(opt ...ServerOption) *TCPServer {
	var opts options
	for _, o := range opt {
		o(&opts)
	}
	if opts.codec == nil {
		opts.codec = TypeLengthValueCodec{}
	}

	s := &TCPServer{
		addr:   addr,
		opts:   opts,
		conns:  NewConnMap(),
		timing: NewTimingWheel(),
		wg:     &sync.WaitGroup{},
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

// SetOnSchedule sets a callback to call every duration.
func (s *TCPServer) SetOnSchedule(duration time.Duration, callback func(time.Time, interface{})) {
	s.mu.Lock()
	s.interv = duration
	s.sched = onScheduleFunc(callback)
	s.mu.Unlock()
}

// SetOnConnect sets a callback to call on client connected.
func (s *TCPServer) SetOnConnect(callback func(Connection) bool) {
	s.mu.Lock()
	s.onConnect = onConnectFunc(callback)
	s.mu.Unlock()
}

// SetOnMessage sets a callback to call on message arrived.
func (s *TCPServer) SetOnMessage(callback func(Message, Connection)) {
	s.mu.Lock()
	s.onMessage = onMessageFunc(callback)
	s.mu.Unlock()
}

// SetOnClose sets a callback to call on client closed.
func (s *TCPServer) SetOnClose(callback func(Connection)) {
	s.mu.Lock()
	s.onClose = onCloseFunc(callback)
	s.mu.Unlock()
}

// SetOnError sets a callback to call on error occurs.
func (s *TCPServer) SetOnError(callback func()) {
	s.mu.Lock()
	s.onError = onErrorFunc(callback)
	s.mu.Unlock()
}

// Start starts the TCP server, accepting new clients and creating service
// go-routine for each. The service go-routines read messages and then call
// the registered handlers to handle them. Start returns when failed with fatal
// errors, the listener willl be closed when returned.
func (s *TCPServer) Start(l net.Listener) error {
	s.mu.Lock()
	if s.lis == nil {
		s.mu.Unlock()
		l.Close()
		return ErrServerClosed
	}
	s.lis[l] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if s.lis != nil && s.lis[l] {
			l.Close()
			delete(s.lis, l)
		}
		s.mu.Unlock()
	}()

	holmes.Info("server start, net %s addr %s", l.Addr().Network(), l.Addr().String())

	s.wg.Add(1)
	go s.timeOutLoop()

	var tempDelay time.Duration
	for {
		rawConn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay >= max {
					tempDelay = max
				}
				holmes.Error("accept error %v, retrying in %d", err, tempDelay)
				select {
				case <-time.After(tempDelay):
				case <-s.ctx.Done():
				}
				continue
			}
			return err
		}
		tempDelay = 0

		// how many connections do we have ?
		sz := s.conns.Size()
		if sz >= MaxConnections {
			holmes.Warn("max connections size %d, refuse", sz)
			rawConn.Close()
			continue
		}

		if s.opts.tlsCfg != nil {
			rawConn = tls.Server(rawConn, s.opts.tlsCfg)
		}

		netid := netIdentifier.GetAndIncrement()
		sc := NewServerConn(netid, s, rawConn)
		sc.SetName(sc.GetRemoteAddress().String())

		s.mu.Lock()
		if s.sched != nil {
			sc.RunEvery(s.interv, s.sched)
		}
		s.mu.Unlock()

		s.conns.Put(netid, sc)
		addTotalConn(1)

		s.wg.Add(1)
		go func() {
			sc.Start()
		}()

		holmes.Info("accepted client %s, id %d, total %d\n", sc.GetName(), netid, s.conns.Size())
		s.conns.RLock()
		for _, c := range s.conns.m {
			holmes.Info("client %s %t\n", c.GetName(), c.IsRunning())
		}
		s.conns.RUnlock()
	} // for loop
}

// wait until all connections closed
func (server *TCPServer) Close() {
	if server.isRunning.CompareAndSet(true, false) {
		server.GetTimingWheel().Stop()

		var conns []Connection
		server.connections.RLock()
		for _, c := range server.connections.m {
			conns = append(conns, c)
		}
		server.connections.RUnlock()

		for _, conn := range conns {
			conn.Close()
		}

		close(server.closeServChan)
		server.finish.Wait()

		os.Exit(0)
	}
}

type TLSTCPServer struct {
	certFile string
	keyFile  string
	*TCPServer
}

func NewTLSTCPServer(addr, cert, key string) Server {
	server := &TLSTCPServer{
		certFile:  cert,
		keyFile:   key,
		TCPServer: NewTCPServer(addr).(*TCPServer),
	}

	config, err := LoadTLSConfig(server.certFile, server.keyFile, false)
	if err != nil {
		holmes.Fatal("loading %s %s: %v", server.certFile, server.keyFile, err)
	}

	setTLSWrapper(func(conn net.Conn) net.Conn {
		return tls.Server(conn, &config)
	})

	return server
}

func (server *TLSTCPServer) IsRunning() bool {
	return server.TCPServer.IsRunning()
}

func (server *TLSTCPServer) GetConnections() []Connection {
	return server.TCPServer.GetConnections()
}

func (server *TLSTCPServer) GetConnectionMap() *ConnectionMap {
	return server.TCPServer.GetConnectionMap()
}

func (server *TLSTCPServer) GetTimingWheel() *TimingWheel {
	return server.TCPServer.GetTimingWheel()
}

func (server *TLSTCPServer) GetServerAddress() string {
	return server.TCPServer.GetServerAddress()
}

func (server *TLSTCPServer) Start() {
	server.TCPServer.Start()
}

func (server *TLSTCPServer) Close() {
	server.TCPServer.Close()
}

func (server *TLSTCPServer) SetOnScheduleCallback(duration time.Duration, callback func(time.Time, interface{})) {
	server.TCPServer.SetOnScheduleCallback(duration, callback)
}

func (server *TLSTCPServer) GetOnScheduleCallback() (time.Duration, onScheduleFunc) {
	return server.TCPServer.GetOnScheduleCallback()
}

func (server *TLSTCPServer) SetOnConnectCallback(callback func(Connection) bool) {
	server.TCPServer.SetOnConnectCallback(callback)
}

func (server *TLSTCPServer) GetOnConnectCallback() onConnectFunc {
	return server.TCPServer.GetOnConnectCallback()
}

func (server *TLSTCPServer) SetOnMessageCallback(callback func(Message, Connection)) {
	server.TCPServer.SetOnMessageCallback(callback)
}

func (server *TLSTCPServer) GetOnMessageCallback() onMessageFunc {
	return server.TCPServer.GetOnMessageCallback()
}

func (server *TLSTCPServer) SetOnCloseCallback(callback func(Connection)) {
	server.TCPServer.SetOnCloseCallback(callback)
}

func (server *TLSTCPServer) GetOnCloseCallback() onCloseFunc {
	return server.TCPServer.GetOnCloseCallback()
}

func (server *TLSTCPServer) SetOnErrorCallback(callback func()) {
	server.TCPServer.SetOnErrorCallback(callback)
}

func (server *TLSTCPServer) GetOnErrorCallback() onErrorFunc {
	return server.TCPServer.GetOnErrorCallback()
}

/* Retrieve the extra data(i.e. net id), and then redispatch
timeout callbacks to corresponding client connection, this
prevents one client from running callbacks of other clients */
func (server *TCPServer) timeOutLoop() {
	defer server.finish.Done()

	for {
		select {
		case <-server.closeServChan:
			return

		case timeout := <-server.GetTimingWheel().GetTimeOutChannel():
			netid := timeout.ExtraData.(int64)
			if conn, ok := server.connections.Get(netid); ok {
				tcpConn := conn.(Connection)
				if tcpConn.IsRunning() {
					tcpConn.GetTimeOutChannel() <- timeout
				}
			} else {
				holmes.Warn("Invalid client %d", netid)
			}
		}
	}
}

func LoadTLSConfig(certFile, keyFile string, isSkipVerify bool) (tls.Config, error) {
	var config tls.Config
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return config, err
	}
	config = tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: isSkipVerify,
		CipherSuites: []uint16{
			tls.TLS_RSA_WITH_RC4_128_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
	}
	now := time.Now()
	config.Time = func() time.Time { return now }
	config.Rand = rand.Reader
	return config, nil
}

func setTLSWrapper(wrapper func(conn net.Conn) net.Conn) {
	tlsWrapper = wrapper
}
