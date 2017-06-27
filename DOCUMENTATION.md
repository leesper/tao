# Documentation

```go
import "github.com/leesper/tao"
```

## Overview
Package tao implements a light-weight TCP network programming framework.

Server represents a TCP server with various ServerOption supported.

1. Provides custom codec by CustomCodecOption;
2. Provides TLS server by TLSCredsOption;
3. Provides callback on connected by OnConnectOption;
4. Provides callback on meesage arrived by OnMessageOption;
5. Provides callback on closed by OnCloseOption;
6. Provides callback on error occurred by OnErrorOption;

ServerConn represents a connection on the server side.

ClientConn represents a connection connect to other servers. You can make it reconnectable by passing ReconnectOption when creating.

AtomicInt64, AtomicInt32 and AtomicBoolean are providing concurrent-safe atomic types in a Java-like style while ConnMap is a go-routine safe map for connection management.

Every handler function is defined as func(context.Context, WriteCloser). Usually a meesage and a net ID are shifted within the Context, developers can retrieve them by calling the following functions.

```go
func NewContextWithMessage(ctx context.Context, msg Message) context.Context
func MessageFromContext(ctx context.Context) Message
func NewContextWithNetID(ctx context.Context, netID int64) context.Context
func NetIDFromContext(ctx context.Context) int64
```

Programmers are free to define their own request-scoped data and put them in the context, but must be sure that the data is safe for multiple go-routines to access.

Every message must define according to the interface and a deserialization function:
```go
  type Message interface {
	 MessageNumber() int32
   Serialize() ([]byte, error)
  }

  func Deserialize(data []byte) (message Message, err error)
```
There is a TypeLengthValueCodec defined, but one can also define his/her own codec:
```go
  type Codec interface {
	  Decode(net.Conn) (Message, error)
	  Encode(Message) ([]byte, error)
  }
```
TimingWheel is a safe timer for running timed callbacks on connection.

WorkerPool is a go-routine pool for running message handlers, you can fetch one by calling func WorkerPoolInstance() \*WorkerPool.

## Constants
```go
const (
    // MessageTypeBytes is the length of type header.
    MessageTypeBytes = 4
    // MessageLenBytes is the length of length header.
    MessageLenBytes = 4
    // MessageMaxBytes is the maximum bytes allowed for application data.
    MessageMaxBytes = 1 << 23 // 8M
)

const (
    // MaxConnections is the maximum number of client connections allowed.
    MaxConnections = 1000
    BufferSize128  = 128
    BufferSize256  = 256
    BufferSize512  = 512
    BufferSize1024 = 1024
)

const (
    // HeartBeat is the default heart beat message number.
    HeartBeat = 0
)
```

## Variables
Error codes returned by failures dealing with server or connection.
```go
var (
    ErrParameter     = errors.New("parameter error")
    ErrNilKey        = errors.New("nil key")
    ErrNilValue      = errors.New("nil value")
    ErrWouldBlock    = errors.New("would block")
    ErrNotHashable   = errors.New("not hashable")
    ErrNilData       = errors.New("nil data")
    ErrBadData       = errors.New("more than 8M data")
    ErrNotRegistered = errors.New("handler not registered")
    ErrServerClosed  = errors.New("server has been closed")
)
```

## func HandleHeartBeat
```go
func HandleHeartBeat(ctx context.Context, c WriteCloser)
```
HandleHeartBeat updates connection heart beat timestamp.

## func LoadTLSConfig
```go
func LoadTLSConfig(certFile, keyFile string, isSkipVerify bool) (*tls.Config, error)
```
LoadTLSConfig returns a TLS configuration with the specified cert and key file.

## func MonitorOn
```go
func MonitorOn(port int)
```
MonitorOn starts up an HTTP monitor on port.

## func NetIDFromContext
```go
func NetIDFromContext(ctx context.Context) int64
```
NetIDFromContext returns a net ID from a Context.

## func NewContextWithMessage
```go
func NewContextWithMessage(ctx context.Context, msg Message) context.Context
```
NewContextWithMessage returns a new Context that carries message.

## func NewContextWithNetID
```go
func NewContextWithNetID(ctx context.Context, netID int64) context.Context
```
NewContextWithNetID returns a new Context that carries net ID.

## func Register
```go
func Register(msgType int32, unmarshaler func([]byte) (Message, error), handler func(context.Context, WriteCloser))
```
Register registers the unmarshal and handle functions for msgType. If no unmarshal function provided, the message will not be parsed. If no handler function provided, the message will not be handled unless you set a default one by calling SetOnMessageCallback. If Register being called twice on one msgType, it will panics.

## type AtomicBoolean
AtomicBoolean provides atomic boolean type.
```go
type AtomicBoolean int32
```

## func NewAtomicBoolean
```go
func NewAtomicBoolean(initialValue bool) *AtomicBoolean
```
NewAtomicBoolean returns an atomic boolean type.

## func (\*AtomicBoolean) CompareAndSet
```go
func (a *AtomicBoolean) CompareAndSet(oldValue, newValue bool) bool
```
CompareAndSet compares boolean with expected value, if equals as expected then sets the updated value, this operation performs atomically.

## func (\*AtomicBoolean) Get
```go
func (a *AtomicBoolean) Get() bool
```
Get returns the value of boolean atomically.

## func (\*AtomicBoolean) GetAndSet
```go
func (a *AtomicBoolean) GetAndSet(newValue bool) bool
```
GetAndSet sets new value and returns the old atomically.

#3 func (\*AtomicBoolean) Set
```go
func (a *AtomicBoolean) Set(newValue bool)
```
Set sets the value of boolean atomically.

## func (\*AtomicBoolean) String
```go
func (a *AtomicBoolean) String() string
```

##type AtomicInt32
AtomicInt32 provides atomic int32 type.
```go
type AtomicInt32 int32
```

## func NewAtomicInt32
```go
func NewAtomicInt32(initialValue int32) *AtomicInt32
```
NewAtomicInt32 returns an atomoic int32 type.

## func (\*AtomicInt32) AddAndGet
```go
func (a *AtomicInt32) AddAndGet(delta int32) int32
```
AddAndGet adds the value by delta and then gets the value, this operation performs atomically.

## func (\*AtomicInt32) CompareAndSet
```go
func (a *AtomicInt32) CompareAndSet(expect, update int32) bool
```
CompareAndSet compares int64 with expected value, if equals as expected then sets the updated value, this operation performs atomically.

## func (\*AtomicInt32) DecrementAndGet
```go
func (a *AtomicInt32) DecrementAndGet() int32
```
DecrementAndGet decrements the value by 1 and then gets the value, this operation performs atomically.

## func (\*AtomicInt32) Get
```go
func (a *AtomicInt32) Get() int32
```
Get returns the value of int32 atomically.

## func (\*AtomicInt32) GetAndAdd
```go
func (a *AtomicInt32) GetAndAdd(delta int32) int32
```
GetAndAdd gets the old value and then add by delta, this operation performs atomically.

## func (\*AtomicInt32) GetAndDecrement
```go
func (a *AtomicInt32) GetAndDecrement() int32
```
GetAndDecrement gets the old value and then decrement by 1, this operation performs atomically.

## func (\*AtomicInt32) GetAndIncrement
```go
func (a *AtomicInt32) GetAndIncrement() int32
```
GetAndIncrement gets the old value and then increment by 1, this operation performs atomically.

## func (\*AtomicInt32) GetAndSet
```go
func (a *AtomicInt32) GetAndSet(newValue int32) (oldValue int32)
```
GetAndSet sets new value and returns the old atomically.

## func (\*AtomicInt32) IncrementAndGet
```go
func (a *AtomicInt32) IncrementAndGet() int32
```
IncrementAndGet increments the value by 1 and then gets the value, this operation performs atomically.

## func (\*AtomicInt32) Set
```go
func (a *AtomicInt32) Set(newValue int32)
```
Set sets the value of int32 atomically.

## func (\*AtomicInt32) String
```go
func (a *AtomicInt32) String() string
```

## type AtomicInt64
AtomicInt64 provides atomic int64 type.
```go
type AtomicInt64 int64
```

## func NewAtomicInt64
```go
func NewAtomicInt64(initialValue int64) *AtomicInt64
```
NewAtomicInt64 returns an atomic int64 type.

## func (\*AtomicInt64) AddAndGet
```go
func (a *AtomicInt64) AddAndGet(delta int64) int64
```
AddAndGet adds the value by delta and then gets the value, this operation performs atomically.

## func (\*AtomicInt64) CompareAndSet
```go
func (a *AtomicInt64) CompareAndSet(expect, update int64) bool
```
CompareAndSet compares int64 with expected value, if equals as expected then sets the updated value, this operation performs atomically.

## func (\*AtomicInt64) DecrementAndGet
```go
func (a *AtomicInt64) DecrementAndGet() int64
```
DecrementAndGet decrements the value by 1 and then gets the value, this operation performs atomically.

## func (\*AtomicInt64) Get
```go
func (a *AtomicInt64) Get() int64
```
Get returns the value of int64 atomically.

## func (\*AtomicInt64) GetAndAdd
```go
func (a *AtomicInt64) GetAndAdd(delta int64) int64
```
GetAndAdd gets the old value and then add by delta, this operation performs atomically.

## func (\*AtomicInt64) GetAndDecrement
```go
func (a *AtomicInt64) GetAndDecrement() int64
```
GetAndDecrement gets the old value and then decrement by 1, this operation performs atomically.

## func (\*AtomicInt64) GetAndIncrement
```go
func (a *AtomicInt64) GetAndIncrement() int64
```
GetAndIncrement gets the old value and then increment by 1, this operation performs atomically.

## func (\*AtomicInt64) GetAndSet
```go
func (a *AtomicInt64) GetAndSet(newValue int64) int64
```
GetAndSet sets new value and returns the old atomically.

## func (\*AtomicInt64) IncrementAndGet
```go
func (a *AtomicInt64) IncrementAndGet() int64
```
IncrementAndGet increments the value by 1 and then gets the value, this operation performs atomically.

## func (\*AtomicInt64) Set
```go
func (a *AtomicInt64) Set(newValue int64)
```
Set sets the value of int64 atomically.

## func (\*AtomicInt64) String
```go
func (a *AtomicInt64) String() string
```

## type ClientConn
ClientConn represents a client connection to a TCP server.
```go
type ClientConn struct {
    // contains filtered or unexported fields
}
```

## func NewClientConn
```go
func NewClientConn(netid int64, c net.Conn, opt ...ServerOption) *ClientConn
```
NewClientConn returns a new client connection which has not started to serve requests yet.

## func (\*ClientConn) AddPendingTimer
```go
func (cc *ClientConn) AddPendingTimer(timerID int64)
```
AddPendingTimer adds a new timer ID to client connection.

## func (\*ClientConn) CancelTimer
```go
func (cc *ClientConn) CancelTimer(timerID int64)
```
CancelTimer cancels a timer with the specified ID.

## func (\*ClientConn) Close
```go
func (cc *ClientConn) Close()
```
Close gracefully closes the client connection. It blocked until all sub go-routines are completed and returned.

## func (\*ClientConn) GetContextValue
```go
func (cc *ClientConn) GetContextValue(k interface{}) interface{}
```
GetContextValue gets extra data from client connection.

## func (\*ClientConn) GetHeartBeat
```go
func (cc *ClientConn) GetHeartBeat() int64
```
GetHeartBeat gets the heart beats of client connection.

## func (\*ClientConn) GetName
```go
func (cc *ClientConn) GetName() string
```
GetName gets the name of client connection.

## func (\*ClientConn) GetNetID
```go
func (cc *ClientConn) GetNetID() int64
```
GetNetID returns the net ID of client connection.

## func (\*ClientConn) LocalAddr
```go
func (cc *ClientConn) LocalAddr() net.Addr
```
LocalAddr returns the local address of server connection.

## func (\*ClientConn) RemoteAddr
```go
func (cc *ClientConn) RemoteAddr() net.Addr
```
RemoteAddr returns the remote address of server connection.

## func (\*ClientConn) RunAfter
```go
func (cc *ClientConn) RunAfter(duration time.Duration, callback func(time.Time, WriteCloser)) int64
```
RunAfter runs a callback right after the specified duration ellapsed.

## func (\*ClientConn) RunAt
```
func (cc *ClientConn) RunAt(timestamp time.Time, callback func(time.Time, WriteCloser)) int64
```
RunAt runs a callback at the specified timestamp.

## func (\*ClientConn) RunEvery
```go
func (cc *ClientConn) RunEvery(interval time.Duration, callback func(time.Time, WriteCloser)) int64
```
RunEvery runs a callback on every interval time.

## func (\*ClientConn) SetContextValue
```go
func (cc *ClientConn) SetContextValue(k, v interface{})
```
SetContextValue sets extra data to client connection.

## func (\*ClientConn) SetHeartBeat
```go
func (cc *ClientConn) SetHeartBeat(heart int64)
```
SetHeartBeat sets the heart beats of client connection.

## func (\*ClientConn) SetName
```go
func (cc *ClientConn) SetName(name string)
```
SetName sets the name of client connection.

## func (\*ClientConn) Start
```go
func (cc *ClientConn) Start()
```
Start starts the client connection, creating go-routines for reading, writing and handlng.

## func (\*ClientConn) Write
```go
func (cc *ClientConn) Write(message Message) error
```
Write writes a message to the client.

## type Codec
Codec is the interface for message coder and decoder. Application programmer can define a custom codec themselves.
```go
type Codec interface {
    Decode(net.Conn) (Message, error)
    Encode(Message) ([]byte, error)
}
```

## type ConnMap

ConnMap is a safe map for server connection management.

```go
type ConnMap struct {
    sync.RWMutex
    // contains filtered or unexported fields
}
```

## func NewConnMap
```go
func NewConnMap() *ConnMap
```
NewConnMap returns a new ConnMap.

## func (\*ConnMap) Clear
```go
func (cm *ConnMap) Clear()
```
Clear clears all elements in map.

## func (\*ConnMap) Get
```go
func (cm *ConnMap) Get(id int64) (*ServerConn, bool)
```
Get gets a server connection with specified net ID.

## func (\*ConnMap) IsEmpty
```go
func (cm *ConnMap) IsEmpty() bool
```
IsEmpty tells whether ConnMap is empty.

## func (\*ConnMap) Put
```go
func (cm *ConnMap) Put(id int64, sc *ServerConn)
```
Put puts a server connection with specified net ID in map.

## func (\*ConnMap) Remove
```go
func (cm *ConnMap) Remove(id int64)
```
Remove removes a server connection with specified net ID.

## func (\*ConnMap) Size
```go
func (cm *ConnMap) Size() int
```
Size returns map size.

## type ErrUndefined

ErrUndefined for undefined message type.
```go
type ErrUndefined int32
```

## func (ErrUndefined) Error
```go
func (e ErrUndefined) Error() string
```

## type Handler

Handler takes the responsibility to handle incoming messages.
```go
type Handler interface {
    Handle(context.Context, interface{})
}
```

## type HandlerFunc

HandlerFunc serves as an adapter to allow the use of ordinary functions as handlers.
```go
type HandlerFunc func(context.Context, WriteCloser)
```

## func GetHandlerFunc
```go
func GetHandlerFunc(msgType int32) HandlerFunc
```
GetHandlerFunc returns the corresponding handler function for msgType.

## func (HandlerFunc) Handle
```go
func (f HandlerFunc) Handle(ctx context.Context, c WriteCloser)
```
Handle calls f(ctx, c)

## type Hashable

Hashable is a interface for hashable object.
```go
type Hashable interface {
    HashCode() int32
}
```

## type HeartBeatMessage

HeartBeatMessage for application-level keeping alive.
```go
type HeartBeatMessage struct {
    Timestamp int64
}
```

## func (HeartBeatMessage) MessageNumber
```go
func (hbm HeartBeatMessage) MessageNumber() int32
```
MessageNumber returns message number.

## func (HeartBeatMessage) Serialize
```go
func (hbm HeartBeatMessage) Serialize() ([]byte, error)
```
Serialize serializes HeartBeatMessage into bytes.

## type Message

Message represents the structured data that can be handled.
```go
type Message interface {
    MessageNumber() int32
    Serialize() ([]byte, error)
}
```

## func DeserializeHeartBeat
```go
func DeserializeHeartBeat(data []byte) (message Message, err error)
```
DeserializeHeartBeat deserializes bytes into Message.

## func MessageFromContext
```go
func MessageFromContext(ctx context.Context) Message
```
MessageFromContext extracts a message from a Context.

## type MessageHandler

MessageHandler is a combination of message and its handler function.
```go
type MessageHandler struct {
    // contains filtered or unexported fields
}
```

## type OnTimeOut

OnTimeOut represents a timed task.
```go
type OnTimeOut struct {
    Callback func(time.Time, WriteCloser)
    Ctx      context.Context
}
```

## func NewOnTimeOut
```go
func NewOnTimeOut(ctx context.Context, cb func(time.Time, WriteCloser)) *OnTimeOut
```
NewOnTimeOut returns OnTimeOut.

## type Server

Server is a server to serve TCP requests.
```go
type Server struct {
    // contains filtered or unexported fields
}
```

## func NewServer
```go
func NewServer(opt ...ServerOption) *Server
```
NewServer returns a new TCP server which has not started to serve requests yet.

## func ServerFromContext
```go
func ServerFromContext(ctx context.Context) (*Server, bool)
```
ServerFromContext returns the server within the context.

## func (\*Server) Broadcast
```go
func (s *Server) Broadcast(msg Message)
```
Broadcast broadcasts message to all server connections managed.

## func (\*Server) ConnsMap
```go
func (s *Server) ConnsMap() *ConnMap
```
ConnsMap returns connections managed.

## func (\*Server) GetConn
```go
func (s *Server) GetConn(id int64) (*ServerConn, bool)
```
GetConn returns a server connection with specified ID.

## func (\*Server) Sched
```go
func (s *Server) Sched(dur time.Duration, sched func(time.Time, WriteCloser))
```
Sched sets a callback to invoke every duration.

## func (\*Server) Start
```go
func (s *Server) Start(l net.Listener) error
```
Start starts the TCP server, accepting new clients and creating service go-routine for each. The service go-routines read messages and then call the registered handlers to handle them. Start returns when failed with fatal errors, the listener willl be closed when returned.

## func (\*Server) Stop
```go
func (s *Server) Stop()
```
Stop gracefully closes the server, it blocked until all connections are closed and all go-routines are exited.

## func (\*Server) Unicast
```go
func (s *Server) Unicast(id int64, msg Message) error
```
Unicast unicasts message to a specified conn.

## type ServerConn

ServerConn represents a server connection to a TCP server, it implments Conn.
```go
type ServerConn struct {
    // contains filtered or unexported fields
}
```

## func NewServerConn
```go
func NewServerConn(id int64, s *Server, c net.Conn) *ServerConn
```
NewServerConn returns a new server connection which has not started to serve requests yet.

## func (\*ServerConn) AddPendingTimer
```go
func (sc *ServerConn) AddPendingTimer(timerID int64)
```
AddPendingTimer adds a timer ID to server Connection.

## func (\*ServerConn) CancelTimer
```go
func (sc *ServerConn) CancelTimer(timerID int64)
```
CancelTimer cancels a timer with the specified ID.

## func (\*ServerConn) Close
```go
func (sc *ServerConn) Close()
```
Close gracefully closes the server connection. It blocked until all sub go-routines are completed and returned.

## func (\*ServerConn) GetContextValue
```go
func (sc *ServerConn) GetContextValue(k interface{}) interface{}
```
GetContextValue gets extra data from server connection.

## func (\*ServerConn) GetHeartBeat
```go
func (sc *ServerConn) GetHeartBeat() int64
GetHeartBeat returns the heart beats of server connection.
```

## func (\*ServerConn) GetName
```go
func (sc *ServerConn) GetName() string
GetName returns the name of server connection.
```

## func (\*ServerConn) GetNetID
```go
func (sc *ServerConn) GetNetID() int64
```
GetNetID returns net ID of server connection.

## func (\*ServerConn) LocalAddr
```go
func (sc *ServerConn) LocalAddr() net.Addr
```
LocalAddr returns the local address of server connection.

## func (\*ServerConn) RemoteAddr
```go
func (sc *ServerConn) RemoteAddr() net.Addr
```
RemoteAddr returns the remote address of server connection.

## func (\*ServerConn) RunAfter
```go
func (sc *ServerConn) RunAfter(duration time.Duration, callback func(time.Time, WriteCloser)) int64
```
RunAfter runs a callback right after the specified duration ellapsed.

## func (\*ServerConn) RunAt
```go
func (sc *ServerConn) RunAt(timestamp time.Time, callback func(time.Time, WriteCloser)) int64
```
RunAt runs a callback at the specified timestamp.

## func (\*ServerConn) RunEvery
```go
func (sc *ServerConn) RunEvery(interval time.Duration, callback func(time.Time, WriteCloser)) int64
```
RunEvery runs a callback on every interval time.

## func (\*ServerConn) SetContextValue
```go
func (sc *ServerConn) SetContextValue(k, v interface{})
```
SetContextValue sets extra data to server connection.

## func (\*ServerConn) SetHeartBeat
```go
func (sc *ServerConn) SetHeartBeat(heart int64)
```
SetHeartBeat sets the heart beats of server connection.

## func (\*ServerConn) SetName
```go
func (sc *ServerConn) SetName(name string)
```
SetName sets name of server connection.

## func (\*ServerConn) Start
```go
func (sc *ServerConn) Start()
```
Start starts the server connection, creating go-routines for reading, writing and handlng.

## func (\*ServerConn) Write
```go
func (sc *ServerConn) Write(message Message) error
```
Write writes a message to the client.

## type ServerOption

ServerOption sets server options.
```go
type ServerOption func(*options)
```

## func BufferSizeOption
```go
func BufferSizeOption(indicator int) ServerOption
```
BufferSizeOption returns a ServerOption that is the size of buffered channel, for example an indicator of BufferSize256 means a size of 256.

## func CustomCodecOption
```go
func CustomCodecOption(codec Codec) ServerOption
```
CustomCodecOption returns a ServerOption that will apply a custom Codec.

## func OnCloseOption
```go
func OnCloseOption(cb func(WriteCloser)) ServerOption
```
OnCloseOption returns a ServerOption that will set callback to call when client closed.

## func OnConnectOption
```go
func OnConnectOption(cb func(WriteCloser) bool) ServerOption
```
OnConnectOption returns a ServerOption that will set callback to call when new client connected.

## func OnErrorOption
```go
func OnErrorOption(cb func(WriteCloser)) ServerOption
```
OnErrorOption returns a ServerOption that will set callback to call when error occurs.

## func OnMessageOption
```go
func OnMessageOption(cb func(Message, WriteCloser)) ServerOption
```
OnMessageOption returns a ServerOption that will set callback to call when new message arrived.

## func ReconnectOption
```go
func ReconnectOption() ServerOption
```
ReconnectOption returns a ServerOption that will make ClientConn reconnectable.

## func TLSCredsOption
```go
func TLSCredsOption(config *tls.Config) ServerOption
```
TLSCredsOption returns a ServerOption that will set TLS credentials for server connections.

## func WorkerSizeOption
```go
func WorkerSizeOption(workerSz int) ServerOption
```
WorkerSizeOption returns a ServerOption that will set the number of go-routines in WorkerPool.

## type TimingWheel

TimingWheel manages all the timed task.
```go
type TimingWheel struct {
    // contains filtered or unexported fields
}
```

## func NewTimingWheel
```go
func NewTimingWheel(ctx context.Context) *TimingWheel
```
NewTimingWheel returns a \*TimingWheel ready for use.

## func (\*TimingWheel) AddTimer

```go
func (tw *TimingWheel) AddTimer(when time.Time, interv time.Duration, to *OnTimeOut) int64
AddTimer adds new timed task.
```

## func (\*TimingWheel) CancelTimer
```go
func (tw *TimingWheel) CancelTimer(timerID int64)
```
CancelTimer cancels a timed task with specified timer ID.

## func (\*TimingWheel) GetTimeOutChannel
```go
func (tw *TimingWheel) GetTimeOutChannel() chan *OnTimeOut
```
GetTimeOutChannel returns the timeout channel.

## func (\*TimingWheel) Size
```go
func (tw *TimingWheel) Size() int
```
Size returns the number of timed tasks.

## func (\*TimingWheel) Stop
```go
func (tw *TimingWheel) Stop()
```
Stop stops the TimingWheel.

## type TypeLengthValueCodec

TypeLengthValueCodec defines a special codec. Format: type-length-value |4 bytes|4 bytes|n bytes <= 8M|
```go
type TypeLengthValueCodec struct{}
```

## func (TypeLengthValueCodec) Decode
```go
func (codec TypeLengthValueCodec) Decode(raw net.Conn) (Message, error)
```
Decode decodes the bytes data into Message

## func (TypeLengthValueCodec) Encode
```go
func (codec TypeLengthValueCodec) Encode(msg Message) ([]byte, error)
```
Encode encodes the message into bytes data.

## type UnmarshalFunc

UnmarshalFunc unmarshals bytes into Message.
```go
type UnmarshalFunc func([]byte) (Message, error)
```

## func GetUnmarshalFunc
```go
func GetUnmarshalFunc(msgType int32) UnmarshalFunc
```
GetUnmarshalFunc returns the corresponding unmarshal function for msgType.

## type WorkerPool

WorkerPool is a pool of go-routines running functions.
```go
type WorkerPool struct {
    // contains filtered or unexported fields
}
```

## func WorkerPoolInstance
```go
func WorkerPoolInstance() *WorkerPool
```
WorkerPoolInstance returns the global pool.

## func (\*WorkerPool) Close
```go
func (wp *WorkerPool) Close()
```
Close closes the pool, stopping it from executing functions.

## func (\*WorkerPool) Put
```go
func (wp *WorkerPool) Put(k interface{}, cb func()) error
```
Put appends a function to some worker's channel.

## type WriteCloser

WriteCloser is the interface that groups Write and Close methods.
```go
type WriteCloser interface {
    Write(Message) error
    Close()
}
```
