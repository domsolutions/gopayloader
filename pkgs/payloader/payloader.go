package payloader

import (
	"context"
	"errors"
	"github.com/domsolutions/gopayloader/config"
	http_clients "github.com/domsolutions/gopayloader/pkgs/http-clients"
	jwt_generator "github.com/domsolutions/gopayloader/pkgs/jwt-generator"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/pterm/pterm"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	cacheDir = "gopayloader"
)

var (
	JwtCacheDir string
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	JwtCacheDir = filepath.Join(homeDir, ".cache", cacheDir)
}

type PayLoader struct {
	config    *config.Config
	startTime time.Time
	stopTime  time.Time
}

type GoPayloaderResults struct {
	Total         time.Duration
	Start         time.Time
	End           time.Time
	CompletedReqs int64
	FailedReqs    int64
	RPS           RPS
	Latency       Latency
	Responses     map[worker.ResponseCode]int64
	Errors        map[string]uint
	ReqByteSize   ByteSize
	RespByteSize  ByteSize
}

type ByteSize struct {
	Single    int64
	Total     int64
	PerSecond int64
}

type RPS struct {
	Average float64
	Max     int64
	Min     int64
}

type Latency struct {
	Average time.Duration
	Max     time.Duration
	Min     time.Duration
	Total   time.Duration
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

func (p *PayLoader) handleReqs() (*GoPayloaderResults, error) {
	var jwtStreamErrs <-chan error
	var jwtStream <-chan string

	if p.config.SendJWT && p.config.ReqTarget != 0 {
		if JwtCacheDir == "" {
			pterm.Error.Println("Can't save jwts if no cache directory")
			return nil, errors.New("cache directory couldn't be determined")
		}

		pterm.Info.Printf("Sending jwts with requests, checking for jwts in cache\n")

		jwt := jwt_generator.NewJWTGenerator(&jwt_generator.Config{
			Ctx:                 p.config.Ctx,
			Kid:                 p.config.JwtKID,
			JwtKeyPath:          p.config.JwtKey,
			JwtSub:              p.config.JwtSub,
			JwtCustomClaimsJSON: p.config.JwtCustomClaimsJSON,
			JwtIss:              p.config.JwtIss,
			JwtAud:              p.config.JwtAud,
		})

		if err := os.MkdirAll(JwtCacheDir, 0755); err != nil {
			return nil, err
		}
		if err := jwt.Generate(p.config.ReqTarget, JwtCacheDir, false); err != nil {
			return nil, err
		}
		jwtStream, jwtStreamErrs = jwt.JWTS(p.config.ReqTarget)
	}

	reqsPerWorker := p.config.ReqTarget / int64(p.config.Conns)
	remainderReqs := p.config.ReqTarget % int64(p.config.Conns)

	workersComplete := &sync.WaitGroup{}
	workersComplete.Add(int(p.config.Conns))

	startTrigger := &sync.WaitGroup{}
	startTrigger.Add(1)

	var reqEvery time.Duration
	printer := message.NewPrinter(language.English)

	if p.config.Duration != 0 && p.config.ReqTarget != 0 {
		reqEvery = time.Duration(float64(p.config.Duration) / (float64(p.config.ReqTarget) / float64(p.config.Conns)))
		msg := printer.Sprintf("Running requests every %s for every %d connection/s for total %d request/s against %s\n",
			reqEvery.String(), int(p.config.Conns), p.config.ReqTarget, p.config.ReqURI)
		pterm.Info.Printf(msg)
	} else if p.config.Duration != 0 && p.config.ReqTarget == 0 {
		reqEvery = time.Duration(float64(p.config.Duration) / (float64(p.config.ReqTarget) / float64(p.config.Conns)))
		msg := printer.Sprintf("Running requests for %s for %d connection/s against %s\n",
			p.config.Duration, int(p.config.Conns), p.config.ReqURI)
		pterm.Info.Printf(msg)
	} else {
		msg := printer.Sprintf("Running %d request/s with %d connection/s against %s\n", p.config.ReqTarget, int(p.config.Conns), p.config.ReqURI)
		pterm.Info.Printf(msg)
	}

	workers := make([]worker.Worker, p.config.Conns)
	reqStats := make(chan time.Duration, 1000000)

	var conn uint
	for conn = 0; conn < p.config.Conns; conn++ {
		c := &http_clients.Config{
			ReqURI:           p.config.ReqURI,
			DisableKeepAlive: p.config.DisableKeepAlive,
			SkipVerify:       p.config.SkipVerify,
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
			Headers:          p.config.Headers,
			Body:             p.config.Body,
			BodyFile:         p.config.BodyFile,
			ReqStats:         reqStats,
			Client:           p.config.Client,
		}

		// evenly distribute remainder reqs
		if remainderReqs > 0 {
			c.ReqTarget++
			remainderReqs--
		}

		if p.config.SendJWT {
			c.JwtStreamReceiver = jwtStream
			c.JwtStreamErr = jwtStreamErrs
			c.JWTHeader = p.config.JwtHeader
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

	ctx, stopStatsCalc := context.WithCancel(context.Background())
	defer stopStatsCalc()
	if p.config.Verbose {
		go p.displayProgress(ctx, workers, int(p.config.ReqTarget), p.config.Duration)
	}

	results := &GoPayloaderResults{}
	go p.calcReqStats(ctx, reqStats, results)

	workersComplete.Wait()
	// load test complete, close idle connections

	pterm.Success.Printf("Payload complete, calculating results\n")

	p.stopTimer()
	stopStatsCalc()

	return p.ComputeResults(workers, results)
}

func (p *PayLoader) calcReqStats(ctx context.Context, recv <-chan time.Duration, result *GoPayloaderResults) {
	var t time.Duration
	var rps int64 = 0
	timer := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			// req finished
			return
		case <-timer.C:
			// new RPS
			if rps > result.RPS.Max {
				result.RPS.Max = rps
			}
			if rps < result.RPS.Min || result.RPS.Min == 0 {
				result.RPS.Min = rps
			}
			rps = 0
		case t = <-recv:
			rps++
			if t > result.Latency.Max {
				result.Latency.Max = t
			}
			if t < result.Latency.Min || result.Latency.Min == 0 {
				result.Latency.Min = t
			}
			result.Latency.Total += t
		}
	}
}

func (p *PayLoader) displayProgress(ctx context.Context, workers []worker.Worker, reqTarget int, endTime time.Duration) {
	tick := time.NewTicker(p.config.VerboseTicker)
	var stats worker.Stats
	var prevSuccess, prevError int64 = 0, 0
	var progress *pterm.ProgressbarPrinter

	displayStats, err := pterm.DefaultArea.Start(
		pterm.Red(pterm.Sprintf("0 requests failed\n")),
		pterm.Green(pterm.Sprintf("0 requests successful")))
	if err != nil {
		pterm.Error.Printf("Failed to create display stats area, got error; %v \n", err)
		return
	}

	defer displayStats.Stop()
	progress, err = p.getProgressBar(endTime, reqTarget)
	if err != nil {
		return
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
				progress.Add(int(p.config.VerboseTicker.Seconds()))
			} else {
				progress.Add(int(success-prevSuccess) + int(errs-prevError))
			}

			prevSuccess = success
			prevError = errs
		}
	}

}

func (p *PayLoader) getProgressBar(endTime time.Duration, reqTarget int) (*pterm.ProgressbarPrinter, error) {
	if endTime != 0 {
		progress, err := pterm.DefaultProgressbar.
			WithTotal(int(endTime.Seconds())).
			WithShowElapsedTime().
			WithElapsedTimeRoundingFactor(time.Second).
			WithTitle("Sending requests for " + endTime.String()).Start()
		if err != nil {
			pterm.Error.Printf("Failed to create progress bar, got error; %v \n", err)
			return nil, err
		}
		return progress, nil
	}

	progress, err := pterm.DefaultProgressbar.WithTotal(reqTarget).WithTitle("Sending " + strconv.Itoa(reqTarget) + " requests").Start()
	if err != nil {
		pterm.Error.Printf("Failed to create progress bar, got error; %v \n", err)
		return nil, err
	}
	return progress, nil
}

func (p *PayLoader) Run() (*GoPayloaderResults, error) {
	if err := p.config.Validate(); err != nil {
		return nil, err
	}
	return p.handleReqs()
}
