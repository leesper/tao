/*
Package tao implements a light-weight TCP network programming framework.

Server represents a TCP server with various ServerOption supported.

1. Provides custom codec by CustomCodecOption;
2. Provides TLS server by TLSCredsOption;
3. Provides callback on connected by OnConnectOption;
4. Provides callback on meesage arrived by OnMessageOption;
5. Provides callback on closed by OnCloseOption;
6. Provides callback on error occurred by OnErrorOption;

ServerConn represents a connection on the server side.

ClientConn represents a connection connect to other servers. You can make it
reconnectable by passing ReconnectOption when creating.

AtomicInt64, AtomicInt32 and AtomicBoolean are providing concurrent-safe atomic
types in a Java-like style while ConnMap is a go-routine safe map for connection
management.

Every handler function is defined as func(context.Context, WriteCloser). Usually
a meesage and a net ID are shifted within the Context, developers can retrieve
them by calling the following functions.

  func NewContextWithMessage(ctx context.Context, msg Message) context.Context
  func MessageFromContext(ctx context.Context) Message
  func NewContextWithNetID(ctx context.Context, netID int64) context.Context
  func NetIDFromContext(ctx context.Context) int64

Programmers are free to define their own request-scoped data and put them in the
context, but must be sure that the data is safe for multiple go-routines to
access.

Every message must define according to the interface and a deserialization
function:

  type Message interface {
	 MessageNumber() int32
   Serialize() ([]byte, error)
  }

  func Deserialize(data []byte) (message Message, err error)

There is a TypeLengthValueCodec defined, but one can also define his/her own
codec:

  type Codec interface {
	  Decode(net.Conn) (Message, error)
	  Encode(Message) ([]byte, error)
  }

TimingWheel is a safe timer for running timed callbacks on connection.

WorkerPool is a go-routine pool for running message handlers, you can fetch one
by calling func WorkerPoolInstance() *WorkerPool.
*/
package tao
