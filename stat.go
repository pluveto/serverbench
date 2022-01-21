package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ReqStat struct {
	success      int64
	failed       int64
	totalSuccess int64
	totalFailed  int64
	stopFlag     bool
	started      int64
	stopped      int64
	errors       map[string]int
	errorsMutex  sync.Mutex
}

var Stat ReqStat

func init() {
	Stat.errors = make(map[string]int)
}

func (s *ReqStat) Clear() {
	atomic.StoreInt64(&s.success, 0)
	atomic.StoreInt64(&s.failed, 0)
}

func (s *ReqStat) ReportWorkerStarted() {
	atomic.AddInt64(&s.started, 1)
}

func (s *ReqStat) ReportWorkerStopped() {
	atomic.AddInt64(&s.stopped, 1)
}

func (s *ReqStat) ReportSuccess() {
	atomic.AddInt64(&s.success, 1)
	atomic.AddInt64(&s.totalSuccess, 1)
}

func (s *ReqStat) ReportFailed(message string) {
	message = strings.TrimSpace(message)
	atomic.AddInt64(&s.failed, 1)
	atomic.AddInt64(&s.totalFailed, 1)

	s.errorsMutex.Lock()
	defer s.errorsMutex.Unlock()
	_, ok := s.errors[message]
	if !ok {
		s.errors[message] = 1
	} else {
		s.errors[message] += 1
	}
	// if len(message) > 0 {
	// 	fmt.Println(message)
	// }
}

func (s *ReqStat) ClearAll() {
	atomic.StoreInt64(&s.success, 0)
	atomic.StoreInt64(&s.failed, 0)
	atomic.StoreInt64(&s.totalSuccess, 0)
	atomic.StoreInt64(&s.totalFailed, 0)
}

func (s *ReqStat) Start(wg *sync.WaitGroup) {
	var tickDuration int = 1000 // ms
	wg.Add(1)
	s.stopFlag = false
	tc := time.NewTicker(time.Microsecond * time.Duration(tickDuration*1000))
	defer tc.Stop()
	fname := fmt.Sprintf("stat-%s.log", time.Now().Format("2006-01-02_15_04_05"))
	f, err := os.OpenFile(fname, os.O_CREATE, 0777)
	panicErr(err)
	defer f.Close()

	var tick uint64 = 0

	f.WriteString(fmt.Sprintf("Started at %s (Unix %d) tick duration %dms\n", time.Now().UTC(), time.Now().Unix(), tickDuration))
	f.WriteString("Arguments:\n" + strings.Join(os.Args, " ") + "\n")

	f.WriteString(fmt.Sprintf("%-8s %-8s %-8s %-8s %-8s\n", "Tick", "ReqSucc", "ReqFail", "Started", "Running"))
	for {
		f.WriteString(fmt.Sprintf("%-8d %-8d %-8d %-8d %-8d\n", tick, s.success, s.failed, s.started, s.started-s.stopped))
		s.Clear()
		tick += 1

		if s.stopFlag {
			wg.Done()
			break
		}
		<-tc.C
	}
	f.WriteString(fmt.Sprintf("Summary: %d success, %d failed\n", s.totalSuccess, s.totalFailed))

	if len(s.errors) != 0 {
		f.WriteString("Error messages statistics: \n")
		f.WriteString(fmt.Sprintf("%-8s | %s\n", "Count", "Message"))
		for err, count := range s.errors {
			f.WriteString(fmt.Sprintf("%-8d | %s\n", count, err))
		}
	}
}

func (s *ReqStat) Stop() {
	s.stopFlag = true
}
