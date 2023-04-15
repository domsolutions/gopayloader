package payloader

import (
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
	TotalTime                time.Duration
	StartTime                time.Time
	StopTime                 time.Time
	CompletedReqs            int64
	FailedReqs               int64
	TimePerReqNanoseconds    []int64
	AverageTimePerReqSeconds time.Duration
	MeanRPS                  float64
	Responses                map[worker.ResponseCode]int64
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
	reqsPerWorker := p.config.Reqs / int64(p.config.Conns)
	remainderReqs := p.config.Reqs % int64(p.config.Conns)

	workersComplete := &sync.WaitGroup{}
	workersComplete.Add(int(p.config.Conns))

	startTrigger := &sync.WaitGroup{}
	startTrigger.Add(1)

	var reqEvery time.Duration
	if p.config.Duration != 0 && p.config.Reqs != 0 {
		reqEvery = time.Duration(int64(p.config.Duration) / reqsPerWorker)
	}

	workers := make([]worker.Worker, p.config.Conns)
	var conn uint
	for conn = 0; conn < p.config.Conns; conn++ {
		c := &worker.Config{
			ReqURI:       p.config.ReqURI,
			KeepAlive:    p.config.KeepAlive,
			MTLSKey:      p.config.MTLSKey,
			MTLSCert:     p.config.MTLSCert,
			Reqs:         reqsPerWorker,
			Ctx:          p.config.Ctx,
			StartTrigger: startTrigger,
			Until:        p.config.Duration,
			ReqEvery:     reqEvery,
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

	// wait for reqs to complete
	workersComplete.Wait()
	p.stopTimer()
	return p.getResults(workers)
}

func (p *PayLoader) getResults(workers []worker.Worker) (*Results, error) {
	results := &Results{
		StartTime: p.startTime,
		StopTime:  p.stopTime,
		TotalTime: p.startTime.Sub(p.stopTime),
		Responses: make(map[worker.ResponseCode]int64),
	}

	for _, w := range workers {
		stats := w.Stats()
		results.CompletedReqs += stats.CompletedReqs
		results.FailedReqs += stats.FailedReqs
		results.TimePerReqNanoseconds = append(results.TimePerReqNanoseconds, stats.TimePerReq...)

		for code, hit := range stats.Responses {
			if _, ok := results.Responses[code]; ok {
				results.Responses[code] += hit
			} else {
				results.Responses[code] = hit
			}
		}
	}

	var totalLatency int64 = 0
	for _, r := range results.TimePerReqNanoseconds {
		totalLatency += r
	}
	results.AverageTimePerReqSeconds = time.Duration(totalLatency/int64(len(results.TimePerReqNanoseconds))) / time.Second

	results.MeanRPS = float64(results.CompletedReqs) / float64(results.TotalTime/time.Second)

	return results, nil
}

func (p *PayLoader) Run() (*Results, error) {
	return p.handleReqs()
}
