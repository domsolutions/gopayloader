package payloader

import (
	"context"
	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/pterm/pterm"
	"strconv"
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
		reqEvery = time.Duration(float64(p.config.Duration) / (float64(p.config.ReqTarget) / float64(p.config.Conns)))
		pterm.Debug.Printf("Running requests every %s for every %d connection/s\n", reqEvery.String(), int(p.config.Conns))
	}

	workers := make([]worker.Worker, p.config.Conns)

	var conn uint
	for conn = 0; conn < p.config.Conns; conn++ {
		c := &worker.Config{
			ReqURI:           p.config.ReqURI,
			DisableKeepAlive: p.config.DisableKeepAlive,
			MTLSKey:          p.config.MTLSKey,
			MTLSCert:         p.config.MTLSCert,
			ReqTarget:        reqsPerWorker,
			Ctx:              p.config.Ctx,
			StartTrigger:     startTrigger,
			Until:            p.config.Duration,
			ReqEvery:         reqEvery,
			ReadTimeout:      p.config.ReadTimeout,
			WriteTimeout:     p.config.WriteTimeout,
			Method:           p.config.Method,
			Verbose:          p.config.Verbose,
			HTTPV2:           p.config.HTTPV2,
		}
		if conn == 0 {
			c.ReqTarget += remainderReqs
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
		go p.displayProgress(ctx, workers, int(p.config.ReqTarget), p.config.Duration)
	}

	workersComplete.Wait()
	pterm.Debug.Printf("\nPayload complete, calculating results\n")

	p.stopTimer()
	if p.config.Verbose {
		stopResultsPrinter()
	}

	plResults := NewPayLoaderResults(p)
	return plResults.ComputeResults(workers)
}

func (p *PayLoader) displayProgress(ctx context.Context, workers []worker.Worker, reqTarget int, endTime time.Duration) {
	tick := time.NewTicker(p.config.Ticker)
	var stats worker.Stats
	var prevSuccess int64 = 0
	var prevError int64 = 0
	var progress *pterm.ProgressbarPrinter

	displayStats, err := pterm.DefaultArea.Start(
		pterm.Red(pterm.Sprintf("0 requests failed\n")),
		pterm.Green(pterm.Sprintf("0 requests successful")))
	if err != nil {
		pterm.Error.Printf("Failed to create display stats area, got error; %v \n", err)
		return
	}

	defer displayStats.Stop()

	if endTime != 0 {
		progress, err = pterm.DefaultProgressbar.
			WithTotal(int(endTime.Seconds())).
			WithShowElapsedTime().
			WithElapsedTimeRoundingFactor(time.Second).
			WithTitle("Sending requests for " + endTime.String()).Start()
		if err != nil {
			pterm.Error.Printf("Failed to create progress bar, got error; %v \n", err)
			return
		}
	} else {
		progress, err = pterm.DefaultProgressbar.WithTotal(reqTarget).WithTitle("Sending " + strconv.Itoa(reqTarget) + " requests").Start()
		if err != nil {
			pterm.Error.Printf("Failed to create progress bar, got error; %v \n", err)
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			// workers finished
			return
		case <-p.config.Ctx.Done():
			// user cancelled
			return
		case <-tick.C:
			var errs int64 = 0
			var success int64 = 0

			for _, w := range workers {
				stats = w.Stats()
				errs += stats.FailedReqs
				success += stats.CompletedReqs
			}

			displayStats.Update(
				pterm.Red(pterm.Sprintf("%d requests failed\n", errs)),
				pterm.Green(pterm.Sprintf("%d requests successful", success)))

			if endTime != 0 {
				progress.Add(int(p.config.Ticker.Seconds()))
			} else {
				progress.Add(int(success-prevSuccess) + int(errs-prevError))
			}

			prevSuccess = success
			prevError = errs
		}
	}
}

func (p *PayLoader) Run() (*Results, error) {
	return p.handleReqs()
}
