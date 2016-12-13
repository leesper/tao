package tao

import (
	"crypto/rand"
	"crypto/tls"
	"net"
	"os"
	"sync"
	"time"

	"github.com/reechou/holmes"
)

func init() {
	netIdentifier = NewAtomicInt64(0)
	tlsWrapper = func(conn net.Conn) net.Conn {
		return conn
	}
}

var (
	netIdentifier *AtomicInt64
	tlsWrapper    func(net.Conn) net.Conn
)

type Server interface {
	IsRunning() bool
	GetConnections() []Connection
	GetConnectionMap() *ConnectionMap
	GetTimingWheel() *TimingWheel
	GetServerAddress() string
	Start()
	Close()

	SetOnScheduleCallback(time.Duration, func(time.Time, interface{}))
	GetOnScheduleCallback() (time.Duration, onScheduleFunc)
	SetOnConnectCallback(func(Connection) bool)
	GetOnConnectCallback() onConnectFunc
	SetOnMessageCallback(func(Message, Connection))
	GetOnMessageCallback() onMessageFunc
	SetOnCloseCallback(func(Connection))
	GetOnCloseCallback() onCloseFunc
	SetOnErrorCallback(func())
	GetOnErrorCallback() onErrorFunc
	SetCodec(codec Codec)
	GetCodec() Codec
}

type TCPServer struct {
	isRunning     *AtomicBoolean
	connections   *ConnectionMap
	timingWheel   *TimingWheel
	finish        *sync.WaitGroup
	address       string
	closeServChan chan struct{}

	onConnect onConnectFunc
	onMessage onMessageFunc
	onClose   onCloseFunc
	onError   onErrorFunc
	codec     Codec

	duration   time.Duration
	onSchedule onScheduleFunc
}

func NewTCPServer(addr string) Server {
	return &TCPServer{
		isRunning:     NewAtomicBoolean(true),
		connections:   NewConnectionMap(),
		timingWheel:   NewTimingWheel(),
		finish:        &sync.WaitGroup{},
		address:       addr,
		closeServChan: make(chan struct{}),
	}
}

func (server *TCPServer) IsRunning() bool {
	return server.isRunning.Get()
}

func (server *TCPServer) GetConnections() []Connection {
	conns := []Connection{}
	server.connections.RLock()
	for _, conn := range server.connections.m {
		conns = append(conns, conn)
	}
	server.connections.RUnlock()
	return conns
}

func (server *TCPServer) GetConnectionMap() *ConnectionMap {
	return server.connections
}

func (server *TCPServer) GetTimingWheel() *TimingWheel {
	return server.timingWheel
}

func (server *TCPServer) GetServerAddress() string {
	return server.address
}

func (server *TCPServer) Start() {
	server.finish.Add(1)
	go server.timeOutLoop()

	listener, err := net.Listen("tcp", server.address)
	if err != nil {
		holmes.Fatal("%v", err)
	}
	defer listener.Close()

	var tempDelay time.Duration
	for server.IsRunning() {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if tempDelay >= 1*time.Second {
					tempDelay = 1 * time.Second
				}
				holmes.Error("Accept error %v, retrying in %d", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		tempDelay = 0

		connSize := server.connections.Size()
		if server.connections.Size() >= MAX_CONNECTIONS {
			holmes.Error("Num of conns %d exceeding MAX %d, refuse", connSize, MAX_CONNECTIONS)
			conn.Close()
			continue
		}

		conn = tlsWrapper(conn) // wrap as a tls connection if configured

		/* Create a TCP connection upon accepting a new client, assign an net id
		   to it, then manage it in connections map, and start it */
		netid := netIdentifier.GetAndIncrement()
		tcpConn := NewServerConnection(netid, server, conn)
		tcpConn.SetName(tcpConn.GetRemoteAddress().String())
		duration, onSchedule := server.GetOnScheduleCallback()
		if onSchedule != nil {
			tcpConn.RunEvery(duration, onSchedule)
		}
		server.connections.Put(netid, tcpConn)
		addTotalConn(1)

		// put tcpConn.Start() run in another WaitGroup-synchronized go-routine
		server.finish.Add(1)
		go func() {
			tcpConn.Start()
		}()

		holmes.Info("Accepting client %s, net id %d, now %d\n", tcpConn.GetName(), netid, server.connections.Size())
		server.connections.RLock()
		for _, conn := range server.connections.m {
			holmes.Info("Client %s %t\n", conn.GetName(), conn.IsRunning())
		}
		server.connections.RUnlock()
	}
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

func (server *TCPServer) GetTimeOutChannel() chan *OnTimeOut {
	return server.timingWheel.GetTimeOutChannel()
}

func (server *TCPServer) SetOnScheduleCallback(duration time.Duration, callback func(time.Time, interface{})) {
	server.duration = duration
	server.onSchedule = onScheduleFunc(callback)
}

func (server *TCPServer) GetOnScheduleCallback() (time.Duration, onScheduleFunc) {
	return server.duration, server.onSchedule
}

func (server *TCPServer) SetOnConnectCallback(callback func(Connection) bool) {
	server.onConnect = onConnectFunc(callback)
}

func (server *TCPServer) GetOnConnectCallback() onConnectFunc {
	return server.onConnect
}

func (server *TCPServer) SetOnMessageCallback(callback func(Message, Connection)) {
	server.onMessage = onMessageFunc(callback)
}

func (server *TCPServer) GetOnMessageCallback() onMessageFunc {
	return server.onMessage
}

func (server *TCPServer) SetOnCloseCallback(callback func(Connection)) {
	server.onClose = onCloseFunc(callback)
}

func (server *TCPServer) GetOnCloseCallback() onCloseFunc {
	return server.onClose
}

func (server *TCPServer) SetOnErrorCallback(callback func()) {
	server.onError = onErrorFunc(callback)
}

func (server *TCPServer) GetOnErrorCallback() onErrorFunc {
	return server.onError
}

func (server *TCPServer) SetCodec(codec Codec) {
	server.codec = codec
}

func (server *TCPServer) GetCodec() Codec {
	return server.codec
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
