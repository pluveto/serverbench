package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type ReqStat struct {
	success      int64
	failed       int64
	totalSuccess int64
	totalFailed  int64
	stopFlag     bool
	writeLock    sync.Mutex
}

var Stat ReqStat

func (s *ReqStat) Clear() {
	s.writeLock.Lock()
	s.success = 0
	s.failed = 0
	s.writeLock.Unlock()
}

func (s *ReqStat) ReportSuccess() {
	s.writeLock.Lock()
	s.success += 1
	s.totalSuccess += 1
	s.writeLock.Unlock()
}

func (s *ReqStat) ReportFailed() {
	s.writeLock.Lock()
	s.failed += 1
	s.totalFailed += 1
	s.writeLock.Unlock()
}

func (s *ReqStat) ClearAll() {
	s.writeLock.Lock()
	s.success = 0
	s.failed = 0
	s.totalSuccess = 0
	s.totalFailed = 0
	s.writeLock.Unlock()
}

func (s *ReqStat) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	s.stopFlag = false
	tc := time.NewTicker(time.Second)
	defer tc.Stop()
	fname := fmt.Sprintf("stat-%s.log", time.Now().Format("2006-01-02_15_04_05"))
	f, err := os.OpenFile(fname, os.O_CREATE, 0777)
	panicErr(err)
	defer f.Close()

	var tick uint64 = 0

	f.WriteString(
		fmt.Sprintf("Started at %s (Unix %d)\n", time.Now().UTC(), time.Now().Unix()),
	)
	f.WriteString("Time\tReqSucc\tReqFail\n")
	for {
		f.WriteString(fmt.Sprintf("%d\t\t%d\t\t%d\n", tick, s.success, s.failed))
		s.Clear()
		tick += 1

		if s.stopFlag {
			wg.Done()
			break
		}
		<-tc.C
	}
	f.WriteString(fmt.Sprintf("total: %d succ, %d fail\n", s.totalSuccess, s.totalFailed))
	f.WriteString("Error stat: \n")
	for err, count := range Errors {
		f.WriteString(fmt.Sprintf("%d | %s\n", count, err))
	}
}

func (s *ReqStat) Stop() {
	s.stopFlag = true
}
