package tao

type onConnectCallbackType func() bool
type onMessageCallbackType func(Message, *TcpConnection)
type onCloseCallbackType func(*TcpConnection)
type onErrorCallbackType func()
