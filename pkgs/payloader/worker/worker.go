package worker

import (
	"context"
	"crypto/tls"
	"github.com/domsolutions/fasthttp"
	"net/url"
	"sync"
	"time"
)

var (
	requestPool  *sync.Pool
	responsePool *sync.Pool
)

const (
	ReqBegin = 0
	ReqEnd   = 1
)

type TotalRequestsComplete int64

type Worker interface {
	Run(wg *sync.WaitGroup)
	Stats() Stats
}

type WorkerBase struct {
	config *Config
	client *fasthttp.HostClient
	stats  Stats
	req    *fasthttp.Request
	resp   *fasthttp.Response
}

type Config struct {
	ReqURI           string
	DisableKeepAlive bool
	SkipVerify       bool
	MTLSKey          string
	MTLSCert         string
	Reqs             int64
	Ctx              context.Context
	StartTrigger     *sync.WaitGroup
	Until            time.Duration
	ReqEvery         time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	Method           string
	Verbose          bool
}

type ResponseCode int

type ReqLatency [2]int64

type Stats struct {
	CompletedReqs int64
	FailedReqs    int64
	Reqs          []ReqLatency
	Responses     map[ResponseCode]int64
	Errors        map[string]uint
}

func (c *Config) TimeLimited() bool {
	return c.Until != 0
}

func NewWorker(config *Config) (Worker, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.SkipVerify,
	}

	if config.MTLSCert != "" && config.MTLSKey != "" {
		cert, err := tls.LoadX509KeyPair(config.MTLSCert, config.MTLSKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	u, err := url.ParseRequestURI(config.ReqURI)
	if err != nil {
		return nil, err
	}

	client := &fasthttp.HostClient{
		Addr:                          u.Host,
		IsTLS:                         u.Scheme == "https",
		MaxConns:                      1,
		ReadTimeout:                   config.ReadTimeout,
		WriteTimeout:                  config.WriteTimeout,
		DisableHeaderNamesNormalizing: true,
		TLSConfig:                     tlsConfig,
	}

	if responsePool == nil {
		responsePool = &sync.Pool{New: func() any {
			return &fasthttp.Response{}
		}}
	}

	if requestPool == nil {
		requestPool = &sync.Pool{New: func() any {
			req := &fasthttp.Request{}
			if config.DisableKeepAlive {
				req.Header.Add(fasthttp.HeaderConnection, "close")
			}
			if config.Method != "GET" {
				req.Header.SetMethodBytes([]byte(config.Method))
			}
			req.SetRequestURI(config.ReqURI)
			return req
		}}
	}

	if !config.TimeLimited() {
		return &WorkerFixedReqs{baseConfig(config, client)}, nil
	}

	if config.Reqs == 0 {
		return &WorkerFixedTime{baseConfig(config, client)}, nil
	}
	return &WorkerFixedTimeRequests{baseConfig(config, client)}, nil
}

func baseConfig(config *Config, client *fasthttp.HostClient) *WorkerBase {
	return &WorkerBase{
		config: config,
		client: client,
		stats: Stats{
			Responses: make(map[ResponseCode]int64),
			Errors:    make(map[string]uint),
		},
	}
}

func (w *WorkerBase) run() {
	w.req = requestPool.Get().(*fasthttp.Request)
	w.resp = responsePool.Get().(*fasthttp.Response)

	err := w.process()
	if err != nil {
		if _, ok := w.stats.Errors[err.Error()]; ok {
			w.stats.Errors[err.Error()]++
		} else {
			w.stats.Errors[err.Error()] = 1
		}
		w.stats.FailedReqs++
		return
	}
	w.stats.CompletedReqs++
}

func (w *WorkerBase) process() error {
	begin := time.Now().UnixNano()

	defer func() {
		requestPool.Put(w.req)
		responsePool.Put(w.resp)
		w.stats.Reqs = append(w.stats.Reqs, ReqLatency{begin, time.Now().UnixNano()})
	}()

	if err := w.client.Do(w.req, w.resp); err != nil {
		return err
	}

	status := w.resp.StatusCode()
	_, ok := w.stats.Responses[(ResponseCode(status))]
	if ok {
		w.stats.Responses[(ResponseCode(status))]++
		return nil
	}
	w.stats.Responses[(ResponseCode(status))] = 1
	return nil
}

func (w *WorkerBase) Stats() Stats {
	return w.stats
}
