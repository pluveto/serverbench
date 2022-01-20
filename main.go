package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/akamensky/argparse"
	"go.uber.org/zap"
)

func initLogger() {
	logger := zap.NewExample()
	defer logger.Sync()

	undo := zap.ReplaceGlobals(logger)
	defer undo()
}

func loadOptions() Option {
	parser := argparse.NewParser("swvbench", "A network service benchmark tool")
	workerNum := parser.Int("w", "worker-num", &argparse.Options{Required: true, Default: 100})
	batchSize := parser.Int("b", "batch-size", &argparse.Options{Required: true, Default: 1})
	endpoint := parser.String("e", "endpoint", &argparse.Options{Required: true, Help: "EndPoint of service"})
	err := parser.Parse(os.Args)
	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	return Option{
		MaxCPUNum:    1,
		WorkerNum:    int64(*workerNum),
		BatchSize:    int64(*batchSize),
		EndPoint:     *endpoint,
		EndPointType: EndPoint_HTTP,
	}
}

func loadAgp() ArgProvider {
	p := BasicProvider{}
	p.FromFile("sample.header.json", "sample.json")
	return p
}

func benchmark(opt Option, agp ArgProvider) {
	runtime.GOMAXPROCS(opt.MaxCPUNum)

	var wg sync.WaitGroup
	var twg sync.WaitGroup

	go Stat.Start(&twg)

	var workerID int64
	for workerID = 1; workerID <= opt.WorkerNum; workerID++ {
		wg.Add(1)
		go func(wid_ int64) {
			defer wg.Done()
			worker(&WorkerContext{
				WorkerID:       wid_,
				BatchSize:      opt.BatchSize,
				TargetLocation: opt.EndPoint,
			}, agp)
		}(workerID)
	}
	wg.Wait()
	Stat.Stop()
	twg.Wait()
}

func main() {
	initLogger()
	benchmark(loadOptions(), loadAgp())
}
