package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

var seq int64
var seqMutex sync.Mutex

func Seq() int64 {
	seqMutex.Lock()
	seq++
	seqMutex.Unlock()
	return seq
}

type WorkerContext struct {
	WorkerID       int64
	BatchSize      int64
	TargetLocation string
	Method         string
}

func createTransport() *http.Transport {
	// See https://cloud.tencent.com/developer/article/1684426 for option detail
	var HTTPTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 25 * time.Second,
		}).DialContext,
		// prevent pool, see
		// https://stackoverflow.com/questions/57683132/turning-off-connection-pool-for-go-http-client
		// http://tleyden.github.io/blog/2016/11/21/tuning-the-go-http-client-library-for-load-testing/
		DisableKeepAlives: true,
	}
	return HTTPTransport
}

var trans = createTransport()

func worker(wc *WorkerContext, agp ArgProvider) {
	fmt.Printf("worker %d started\n", wc.WorkerID)
	cli := &http.Client{Transport: trans, Timeout: 25 * time.Second}
	var batchID int64 = 0
	for ; batchID < wc.BatchSize; batchID++ {
		sendRequest(batchID, wc, agp, cli)
	}
	fmt.Printf("worker %d stopped\n", wc.WorkerID)
}

func sendRequest(batchID int64, wc *WorkerContext, agp ArgProvider, cli *http.Client) {
	reqCtx := &RequestContext{
		WorkerID:   wc.WorkerID,
		BatchID:    batchID,
		RequestSeq: Seq(),
	}
	req, err := http.NewRequest(wc.Method, wc.TargetLocation,
		bytes.NewBuffer(agp.GetBody(reqCtx)),
	)
	for k, v := range agp.GetHeaders(reqCtx) {
		req.Header.Add(k, v)
	}
	panicErr(err)
	resp, err := cli.Do(req)
	if err != nil {
		Stat.ReportFailed(err.Error())
		return
	}
	//
	if !(resp.StatusCode <= 200 && resp.StatusCode < 300) {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			fmt.Println(string(body))
			Stat.ReportFailed(string(body))
		} else {
			Stat.ReportFailed(fmt.Sprintf("Status code %d", resp.StatusCode))
		}
		return
	}
	Stat.ReportSuccess()
}

func panicErr(err error) {
	if err != nil {
		zap.L().Panic(err.Error())
	}
}

func UNUSED(x ...interface{}) {}
