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

func CreateTransport() *http.Transport {
	/*
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
		return HTTPTransport */
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   120 * time.Second,
			KeepAlive: 120 * time.Second,
		}).DialContext,
		MaxIdleConns:        0,                 // 最大连接数,默认0无穷大
		MaxIdleConnsPerHost: 0,                 // 对每个host的最大连接数量(MaxIdleConnsPerHost<=MaxIdleConns)
		IdleConnTimeout:     120 * time.Second, // 多长时间未使用自动关闭连接
	}
	return tr
}

func worker(wc *WorkerContext, agp ArgProvider) {
	fmt.Printf("worker %d started\n", wc.WorkerID)
	cli := http.DefaultClient // &http.Client{Transport: createTransport(), Timeout: 120 * time.Second}
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
		codeStr := fmt.Sprintf("Status code %d", resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			fmt.Println(codeStr + string(body))
			Stat.ReportFailed(codeStr + string(body))
		} else {
			Stat.ReportFailed(codeStr)
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
