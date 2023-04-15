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

type Worker interface {
	Run(wg *sync.WaitGroup)
	Stats() Stats
}

type WorkerBase struct {
	config *Config
	client *fasthttp.HostClient
	stats  Stats
}

type Config struct {
	ReqURI       string
	KeepAlive    bool
	MTLSKey      string
	MTLSCert     string
	Reqs         int64
	Ctx          context.Context
	StartTrigger *sync.WaitGroup
	Until        time.Duration
	ReqEvery     time.Duration
}

type ResponseCode int

type Stats struct {
	CompletedReqs int64
	FailedReqs    int64
	TimePerReq    []int64
	Responses     map[ResponseCode]int64
}

func NewWorker(config *Config) (Worker, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
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
		Addr:     u.Host,
		IsTLS:    u.Scheme == "https",
		MaxConns: 1,
		//ReadTimeout:                   opts.timeout,
		//WriteTimeout:                  opts.timeout,
		DisableHeaderNamesNormalizing: true,
		TLSConfig:                     tlsConfig,
		//Dial: fasthttpDialFunc(
		//	opts.bytesRead, opts.bytesWritten,
		//),
	}

	if responsePool == nil {
		responsePool = &sync.Pool{New: func() any {
			return &fasthttp.Response{}
		}}
	}

	if requestPool == nil {
		requestPool = &sync.Pool{New: func() any {
			req := &fasthttp.Request{}
			req.SetRequestURI(config.ReqURI)
			return req
		}}
	}

	if config.Until != 0 {
		if config.Reqs == 0 {
			return &WorkerFixedTime{&WorkerBase{
				config: config,
				client: client,
			}}, nil
		}
		return &WorkerFixedTimeRequests{&WorkerBase{
			config: config,
			client: client,
		}}, nil
	}

	return &WorkerFixedReqs{&WorkerBase{
		config: config,
		client: client,
	}}, nil
}

func (w *WorkerBase) run() {
	req := requestPool.Get().(*fasthttp.Request)
	resp := requestPool.Get().(*fasthttp.Response)

	err := w.process(req, resp)
	if err != nil {
		// TODO store error?
		w.stats.FailedReqs++
		return
	}
	w.stats.CompletedReqs++
}

func (w *WorkerBase) process(req *fasthttp.Request, resp *fasthttp.Response) error {
	begin := time.Now()

	defer func() {
		requestPool.Put(req)
		responsePool.Put(resp)
		w.stats.TimePerReq = append(w.stats.TimePerReq, time.Now().Sub(begin).Nanoseconds())
	}()

	if err := w.client.Do(req, resp); err != nil {
		return err
	}
	if _, ok := w.stats.Responses[ResponseCode(resp.StatusCode())]; ok {
		w.stats.Responses[ResponseCode(resp.StatusCode())]++
	} else {
		w.stats.Responses[ResponseCode(resp.StatusCode())] = 1
	}

	return nil
}

func (w *WorkerBase) Stats() Stats {
	return w.stats
}
