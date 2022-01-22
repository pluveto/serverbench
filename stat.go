package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ReqStat struct {
	stopFlag     bool // tell the ticker to stop
	tickDuration int64
	startTime    time.Time
	endTime      time.Time

	success      int64          // num of success req during a tick
	failed       int64          // num of failed req during a tick
	maxSuccess   int64          // max num of success req during a tick
	maxFailed    int64          // max num of failed req during a tick
	totalSuccess int64          // num of success req totally
	totalFailed  int64          // num of failed req totally
	total        int64          // num of total req, given when start (this does NOT equal to totalSuccess + totalFailed)
	started      int64          // num of started workers
	stopped      int64          // num of stopped workers
	errors       map[string]int // error statistics. key is error message, value is counter

	rwMutex     sync.RWMutex
	errorsMutex sync.Mutex // lock when updating errors
}

var Stat ReqStat

func init() {
	Stat.errors = make(map[string]int)
}

func (s *ReqStat) Clear() {
	s.rwMutex.Lock()
	if s.success > s.maxSuccess {
		s.maxSuccess = s.success
	}
	s.success = 0
	s.failed = 0
	s.rwMutex.Unlock()
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

	s.rwMutex.Lock()
	s.success = 0
	s.failed = 0
	s.maxSuccess = 0
	s.maxFailed = 0
	s.totalSuccess = 0
	s.totalFailed = 0
	s.total = 0
	s.started = 0
	s.stopped = 0
	s.rwMutex.Unlock()

	s.errorsMutex.Lock()
	s.errors = make(map[string]int)
	s.errorsMutex.Unlock()

}

func Round(val float64, precision int) float64 {
	p := math.Pow10(precision)
	return math.Floor(val*p+0.5) / p
}

func Percent(a, b int64, prec int) string {
	return fmt.Sprintf(
		"%.2f", Round(float64(a*100)/float64(b), prec),
	) + "%"
}

func (s *ReqStat) Start(wg *sync.WaitGroup, total int64) {
	if total <= 0 {
		return
	}
	wg.Add(1) // make sure main coroutine will await stat ticker before stop
	defer wg.Done()
	s.tickDuration = 1000 // in ms
	s.stopFlag = false
	s.total = total
	s.startTime = time.Now()

	fname := fmt.Sprintf("stat-%s.log", time.Now().Format("2006-01-02_15_04_05"))
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY, 0777)
	panicErr(err)
	defer f.Close()

	tc := time.NewTicker(time.Microsecond * time.Duration(s.tickDuration*1000))
	defer tc.Stop()

	var tick uint64 = 0

	f.WriteString(fmt.Sprintf("Started at %s (Unix %d). Tick duration is %dms\n", s.startTime, s.startTime.Unix(), s.tickDuration))
	f.WriteString("Arguments:\n" + strings.Join(os.Args, " ") + "\n")

	f.WriteString(fmt.Sprintf("%-8s %-8s %-8s %-8s %-8s %-8s %-8s\n", "Tick", "ReqSucc", "ReqFail", "Started", "Running", "TotSucc", "Remain"))
	for {
		remain := s.total - (s.totalFailed + s.totalSuccess)
		percent := Percent(s.totalSuccess+s.totalFailed, s.total, 2)
		f.WriteString(fmt.Sprintf("%-8d %-8d %-8d %-8d %-8d %-8d %-8d(%s)\n", tick, s.success, s.failed, s.started, s.started-s.stopped, s.totalSuccess, remain, percent))
		s.Clear()
		tick += 1

		if s.stopFlag {
			break
		}
		<-tc.C
	}

	defer func(f *os.File) {
		f.WriteString(fmt.Sprintf("Stopped at %s (Unix %d)\n", s.endTime, s.endTime.Unix()))
		f.WriteString("Summary:\n")
		deltaTime := (s.endTime.Unix() - s.startTime.Unix())
		if deltaTime == 0 {
			deltaTime = 1
		}
		qps := s.totalSuccess / deltaTime
		f.WriteString(fmt.Sprintf("%d success, %d failed. %d avg qps, %d max qps\n", s.totalSuccess, s.totalFailed, qps, s.maxSuccess))

		if len(s.errors) != 0 {
			f.WriteString("Error messages statistics: \n")
			f.WriteString(fmt.Sprintf("%-8s | %s\n", "Count", "Message"))
			for err, count := range s.errors {
				f.WriteString(fmt.Sprintf("%-8d | %s\n", count, err))
			}
		}
	}(f)
}

// Stop not really stop, just set flag
func (s *ReqStat) Stop() {
	s.stopFlag = true
	s.endTime = time.Now()
}
