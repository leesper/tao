package tao

import (
  "encoding/json"
  "log"
)

type ServerConfig struct {
  IP string
  Port int
  Workers int
  Heartbeat int
  Certfile string
  Keyfile string
}

var ServerConf ServerConfig

func init() {
  err := json.Unmarshal(blob, &ServerConf)
  if err != nil {
    log.Fatalln(err)
  }
}

var blob = []byte(
`{
  "IP": "0.0.0.0",
  "Port": 18341,
  "Workers": 10,
  "Heartbeat": 5,
  "Certfile": "",
  "Keyfile": ""
}`)
