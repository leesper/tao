package tao

import (
  "expvar"
  "fmt"
  "net/http"
  "strconv"

  "github.com/leesper/holmes"
)

var (
  procExported *expvar.Int
  connExported *expvar.Int
  timeExported *expvar.Float
  qpsExported *expvar.Float
)

func init() {
  procExported = expvar.NewInt("TotalProcess")
  connExported = expvar.NewInt("TotalConn")
  timeExported = expvar.NewFloat("TotalTime")
  qpsExported = expvar.NewFloat("QPS")
}

func MonitorOn(port int) {
  go func() {
    if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
      holmes.Errorln(err)
      return
    }
  }()
}

func addTotalConn(delta int64) {
  connExported.Add(delta)
  calculateQps()
}

func addTotalProcess() {
  procExported.Add(1)
  calculateQps()
}

func addTotalTime(seconds float64) {
  timeExported.Add(seconds)
  calculateQps()
}

func calculateQps() {
  totalConn, err := strconv.ParseInt(connExported.String(), 10, 64)
  if err != nil {
    holmes.Errorln(err)
    return
  }

  totalTime, err := strconv.ParseFloat(timeExported.String(), 64)
  if err != nil {
    holmes.Errorln(err)
    return
  }

  totalProcess, err := strconv.ParseInt(procExported.String(), 10, 64)
  if err != nil {
    holmes.Errorln(err)
    return
  }

  if float64(totalConn) * totalTime != 0 {
    qps := float64(totalProcess) / (float64(totalConn) * totalTime)
    qpsExported.Set(qps)
  }
}
