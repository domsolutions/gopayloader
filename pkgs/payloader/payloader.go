package payloader

import (
	"context"
	"fmt"
	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"sync"
	"time"
)

type PayLoader struct {
	config    *config.Config
	startTime time.Time
	stopTime  time.Time
}

type Results struct {
	Total         time.Duration
	Start         time.Time
	End           time.Time
	CompletedReqs int64
	FailedReqs    int64
	LatencyPerReq []time.Duration
	RPS           RPS
	Latency       Latency
	Responses     map[worker.ResponseCode]int64
	Errors        map[string]uint
}

type RPS struct {
	Average float64
	Max     uint64
	Min     uint64
}

type Latency struct {
	Average time.Duration
	Max     time.Duration
	Min     time.Duration
}

func NewPayLoader(config *config.Config) *PayLoader {
	return &PayLoader{config: config}
}

func (p *PayLoader) startTimer() {
	p.startTime = time.Now()
}

func (p *PayLoader) stopTimer() {
	p.stopTime = time.Now()
}

func (p *PayLoader) startWorkers(wg *sync.WaitGroup) {
	wg.Done()
}

func (p *PayLoader) handleReqs() (*Results, error) {
	reqsPerWorker := p.config.ReqTarget / int64(p.config.Conns)
	remainderReqs := p.config.ReqTarget % int64(p.config.Conns)

	workersComplete := &sync.WaitGroup{}
	workersComplete.Add(int(p.config.Conns))

	startTrigger := &sync.WaitGroup{}
	startTrigger.Add(1)

	var reqEvery time.Duration
	if p.config.Duration != 0 && p.config.ReqTarget != 0 {
		reqEvery = time.Duration(int64(p.config.Duration) / reqsPerWorker)
	}

	workers := make([]worker.Worker, p.config.Conns)
	//reports := make([]<-chan worker.TotalRequestsComplete, 0)

	var conn uint
	for conn = 0; conn < p.config.Conns; conn++ {
		c := &worker.Config{
			ReqURI:           p.config.ReqURI,
			DisableKeepAlive: p.config.DisableKeepAlive,
			MTLSKey:          p.config.MTLSKey,
			MTLSCert:         p.config.MTLSCert,
			Reqs:             reqsPerWorker,
			Ctx:              p.config.Ctx,
			StartTrigger:     startTrigger,
			Until:            p.config.Duration,
			ReqEvery:         reqEvery,
			ReadTimeout:      p.config.ReadTimeout,
			WriteTimeout:     p.config.WriteTimeout,
			Method:           p.config.Method,
			Verbose:          p.config.Verbose,
		}
		if conn == 0 {
			c.Reqs += remainderReqs
		}

		w, err := worker.NewWorker(c)
		if err != nil {
			return nil, err
		}

		workers[conn] = w
		go w.Run(workersComplete)
	}

	p.startWorkers(startTrigger)
	p.startTimer()

	ctx, stopResultsPrinter := context.WithCancel(context.Background())
	defer stopResultsPrinter()
	if p.config.Verbose {
		go p.displayProgress(ctx, workers)
	}

	// wait for reqs to complete
	workersComplete.Wait()
	p.stopTimer()
	if p.config.Verbose {
		stopResultsPrinter()
	}
	return p.getResults(workers)
}

func (p *PayLoader) displayProgress(ctx context.Context, workers []worker.Worker) {
	tick := time.NewTicker(p.config.Ticker)
	var stats worker.Stats
	var totalSuccess int64 = 0
	var totalError int64 = 0

	for {
		select {
		case <-ctx.Done():
			// workers finished
			return
		case <-p.config.Ctx.Done():
			// user cancelled
			return
		case <-tick.C:
			totalSuccess = 0
			totalError = 0

			for _, w := range workers {
				stats = w.Stats()
				totalSuccess += stats.CompletedReqs
				totalError += stats.FailedReqs
			}
			if totalSuccess > 0 {
				fmt.Printf("%d requests successfully complete\n", totalSuccess)
			}
			if totalError > 0 {
				fmt.Printf("%d requests failed\n", totalError)
			}
		}
	}
}

func (p *PayLoader) getResults(workers []worker.Worker) (*Results, error) {
	results := &Results{
		Start:     p.startTime,
		End:       p.stopTime,
		Total:     p.stopTime.Sub(p.startTime),
		Responses: make(map[worker.ResponseCode]int64),
		Errors:    make(map[string]uint),
	}

	for _, w := range workers {
		stats := w.Stats()
		results.CompletedReqs += stats.CompletedReqs
		results.FailedReqs += stats.FailedReqs

		for _, l := range stats.Reqs {
			results.LatencyPerReq = append(results.LatencyPerReq, time.Duration(l[1]-l[0]))
		}

		for err, count := range stats.Errors {
			if _, ok := results.Errors[err]; ok {
				results.Errors[err] += count
			} else {
				results.Errors[err] = count
			}
		}

		for code, val := range stats.Responses {
			if _, ok := results.Responses[code]; ok {
				results.Responses[code] += val
			} else {
				results.Responses[code] = val
			}
		}

	}

	// TODO optimise 3 loops

	reqsPerSecond := make(map[time.Duration]uint64)
	for t := results.Start; t.Before(results.End); t = t.Add(time.Second) {
		begin := t.UnixNano()
		end := t.Add(time.Second).UnixNano()

		for _, w := range workers {
			stats := w.Stats()
			for _, l := range stats.Reqs {
				if l[worker.ReqBegin] >= begin && l[worker.ReqEnd] <= end {
					if _, ok := reqsPerSecond[time.Duration(t.Unix())]; ok {
						reqsPerSecond[time.Duration(t.Unix())]++
					} else {
						reqsPerSecond[time.Duration(t.Unix())] = 1
					}
				}
			}
		}
	}

	if len(reqsPerSecond) > 0 {
		results.RPS.Min = reqsPerSecond[0]

		for _, val := range reqsPerSecond {
			if val > results.RPS.Max {
				results.RPS.Max = val
			}
			if val < results.RPS.Min {
				results.RPS.Min = val
			}
		}

		results.RPS.Average = float64(results.CompletedReqs) / (float64(results.Total) / float64(time.Second))
	}

	if len(results.LatencyPerReq) > 0 {
		var totalLatency time.Duration = 0
		results.Latency.Min = results.LatencyPerReq[0]

		for _, r := range results.LatencyPerReq {
			if r > results.Latency.Max {
				results.Latency.Max = r
			}
			if r < results.Latency.Min {
				results.Latency.Min = r
			}
			totalLatency += r
		}

		results.Latency.Average = totalLatency / time.Duration(len(results.LatencyPerReq))
	}

	return results, nil
}

func (p *PayLoader) Run() (*Results, error) {
	return p.handleReqs()
}
