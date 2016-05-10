package main

import (
  "log"
  "net"
  "fmt"
  "bufio"
  "os"
  "github.com/leesper/tao"
  "github.com/leesper/tao/examples/chat"
)

func init() {
  log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
  tao.MessageMap.Register(chat.ChatMessage{}.MessageNumber(), tao.UnmarshalFunctionType(chat.UnmarshalChatMessage))

  serverAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:18341")
  if err != nil {
    log.Fatalln(err)
  }

  tcpConn, err := net.DialTCP("tcp", nil, serverAddr)

  if err != nil {
    log.Fatalln(err)
  }

  tcpConnection := tao.ClientTCPConnection(0, tcpConn, tao.NewTimingWheel(), true)
  defer tcpConnection.Close()

  tcpConnection.SetOnConnectCallback(func(client *tao.TCPConnection) bool {
    log.Printf("On connect\n")
    return true
  })

  tcpConnection.SetOnErrorCallback(func() {
    log.Printf("On error\n")
  })

  tcpConnection.SetOnCloseCallback(func(client *tao.TCPConnection) {
    log.Printf("On close\n")
    os.Exit(0)
  })

  tcpConnection.SetOnMessageCallback(func(msg tao.Message, client *tao.TCPConnection) {
    fmt.Print(msg.(chat.ChatMessage).Info)
  })

  tcpConnection.Do()

  for {
    reader := bufio.NewReader(os.Stdin)
    talk, _ := reader.ReadString('\n')
    if talk == "bye\n" {
      break
    } else {
      msg := chat.ChatMessage{
        Info: talk,
      }
      tcpConnection.Write(msg)
    }
  }
  tcpConnection.Close()
}
