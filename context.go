package main

type RequestContext struct {
	WorkerID   int64
	BatchID    int64
	RequestSeq int64
}
