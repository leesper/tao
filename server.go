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
}

var (
	netIdentifier *AtomicInt64
	tlsWrapper    func(net.Conn) net.Conn
)

type options struct {
	tlsCfg    *tls.Config
	codec     Codec
	onConnect onConnectFunc
	onMessage onMessageFunc
	onClose   onCloseFunc
	onError   onErrorFunc
}

// ServerOption sets server options.
type ServerOption func(*options)

// CustomCodec returns a ServerOption that will apply a custom Codec.
func CustomCodec(codec Codec) ServerOption {
	return func(o *options) {
		o.codec = codec
	}
}

// TLSCreds returns a ServerOption that will set TLS credentials for server
// connections.
func TLSCreds(config *tls.Config) ServerOption {
	return func(o *options) {
		o.tlsCfg = config
	}
}

// OnConnect returns a ServerOption that will set callback to call when new
// client connected.
func OnConnect(cb func(interface{}) bool) ServerOption {
	return func(o *options) {
		o.onConnect = cb
	}
}

// OnMessage returns a ServerOption that will set callback to call when new
// message arrived.
func OnMessage(cb func(Message, interface{})) ServerOption {
	return func(o *options) {
		o.onMessage = cb
	}
}

// OnClose returns a ServerOption that will set callback to call when client
// closed.
func OnClose(cb func(interface{})) ServerOption {
	return func(o *options) {
		o.onClose = cb
	}
}

// OnError returns a ServerOption that will set callback to call when error
// occurs.
func OnError(cb func(interface{})) ServerOption {
	return func(o *options) {
		o.onError = cb
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
	mu     sync.Mutex // guards following
	lis    map[net.Listener]bool

	// for periodically running function every duration.
	interv time.Duration
	sched  onScheduleFunc
}

// NewTCPServer returns a new TCP server which has not started
// to serve requests yet.
func NewTCPServer(opt ...ServerOption) *TCPServer {
	var opts options
	for _, o := range opt {
		o(&opts)
	}
	if opts.codec == nil {
		opts.codec = TypeLengthValueCodec{}
	}

	s := &TCPServer{
		opts:   opts,
		conns:  NewConnMap(),
		timing: NewTimingWheel(),
		wg:     &sync.WaitGroup{},
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
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

// Stop gracefully closes the server, it blocked until all connections
// are closed and all go-routines are exited.
func (s *TCPServer) Stop() {
	// immediately stop accepting new clients
	s.mu.Lock()
	listeners := s.lis
	s.lis = nil
	s.mu.Unlock()

	for l := range listeners {
		l.Close()
		holmes.Info("stop accepting at address %s", l.Addr().String())
	}

	// close all connections
	conns := map[int64]Connection{}
	s.conns.RLock()
	for k, v := range s.conns.m {
		conns[k] = v
	}
	s.conns.RUnlock()

	for _, c := range conns {
		c.Close()
		holmes.Info("close client %s", c.GetName())
	}

	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()

	holmes.Info("server stopped gracefully, good bye my friend.")
	os.Exit(0)
}

// Retrieve the extra data(i.e. net id), and then redispatch timeout callbacks
// to corresponding client connection, this prevents one client from running
// callbacks of other clients
func (s *TCPServer) timeOutLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return

		case timeout := <-s.timing.GetTimeOutChannel():
			netid := timeout.ExtraData.(int64)
			if conn, ok := s.conns.Get(netid); ok {
				sc := conn.(Connection)
				if sc.IsRunning() {
					sc.GetTimeOutChannel() <- timeout
				}
			} else {
				holmes.Warn("invalid client %d", netid)
			}
		}
	}
}

// LoadTLSConfig returns a TLS configuration with the specified cert and key file.
func LoadTLSConfig(certFile, keyFile string, isSkipVerify bool) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
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
