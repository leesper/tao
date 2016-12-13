package tao

import (
	"expvar"
	"fmt"
	"net/http"
	"strconv"

	"github.com/reechou/holmes"
)

var (
	handleExported *expvar.Int
	connExported   *expvar.Int
	timeExported   *expvar.Float
	qpsExported    *expvar.Float
)

func init() {
	handleExported = expvar.NewInt("TotalHandle")
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

func addTotalHandle() {
	handleExported.Add(1)
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

	totalHandle, err := strconv.ParseInt(handleExported.String(), 10, 64)
	if err != nil {
		holmes.Errorln(err)
		return
	}

	if float64(totalConn)*totalTime != 0 {
		// take the average time per worker go-routine
		qps := float64(totalHandle) / (float64(totalConn) * (totalTime / float64(WORKERS)))
		qpsExported.Set(qps)
	}
}
