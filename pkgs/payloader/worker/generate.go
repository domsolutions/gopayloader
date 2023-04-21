package worker

import (
	"context"
	"crypto/tls"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
	"net/url"
	"sync"
	"time"
)

const (
	ReqBegin = 0
	ReqEnd   = 1
)

type TotalRequestsComplete int64

type Config struct {
	ReqURI            string
	DisableKeepAlive  bool
	SkipVerify        bool
	MTLSKey           string
	MTLSCert          string
	ReqTarget         int64
	Ctx               context.Context
	StartTrigger      *sync.WaitGroup
	Until             time.Duration
	ReqEvery          time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	Method            string
	Verbose           bool
	HTTPV2            bool
	JwtStreamReceiver <-chan string
	JwtStreamErr      <-chan error
	JWTHeader         string
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

func (c *Config) ReqLimitedOnly() bool {
	return c.Until == 0 && c.ReqTarget != 0
}

func (c *Config) UnlimitedReqs() bool {
	return c.Until != 0 && c.ReqTarget == 0
}

func NewWorker(config *Config) (Worker, error) {
	client, err := getClient(config)
	if err != nil {
		return nil, err
	}

	resp := &fasthttp.Response{}
	req := getReq(config)

	if config.ReqLimitedOnly() {
		if config.JwtStreamReceiver != nil {
			w := &WorkerFixedReqs{baseConfig(config, client, req, resp)}
			w.middleware = jwtMiddleware
			return w, nil
		}
		return &WorkerFixedReqs{baseConfig(config, client, req, resp)}, nil
	}

	if config.UnlimitedReqs() {
		return &WorkerFixedTime{baseConfig(config, client, req, resp)}, nil
	}

	w := &WorkerFixedTimeRequests{baseConfig(config, client, req, resp)}
	if config.JwtStreamReceiver != nil {
		w.middleware = jwtMiddleware
	}
	return w, nil
}

func getReq(config *Config) *fasthttp.Request {
	req := &fasthttp.Request{}
	req.SetRequestURI(config.ReqURI)
	if config.DisableKeepAlive {
		req.Header.Add(fasthttp.HeaderConnection, "close")
	}
	if config.Method != "GET" {
		req.Header.SetMethodBytes([]byte(config.Method))
	}
	return req
}

func jwtMiddleware(w *WorkerBase) {
	select {
	case <-w.config.Ctx.Done():
		// user cancelled
		return
	//case err := <-w.config.JwtStreamErr:
	//	pterm.Error.Printf("Failed to get jwts from cache, got error; %v \n", err)
	//	return TODO fix
	case jwt := <-w.config.JwtStreamReceiver:
		w.req.Header.Set(w.config.JWTHeader, jwt)
	}
}

func baseConfig(config *Config, client *fasthttp.HostClient, req *fasthttp.Request, resp *fasthttp.Response) *WorkerBase {
	return &WorkerBase{
		config: config,
		req:    req,
		resp:   resp,
		client: client,
		stats: Stats{
			Responses: make(map[ResponseCode]int64),
			Errors:    make(map[string]uint),
		},
	}
}

func getClient(config *Config) (*fasthttp.HostClient, error) {
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

	// TODO implement HTTPv3??? from github.com/quic-go/quic-go in use by 25% of websites!! should probably support it
	// //
	//if config.HTTPV3 {
	//	return &http.Client{
	//		Transport: &http3.RoundTripper{},
	//	}, nil
	//}

	if !config.HTTPV2 {
		return client, nil
	}

	// TODO can't ctrl+c when http2 client can't connect to server which is down, just hangs
	// TODO look into how to send reqs i.e. pipelining... does it actually speed stuff up? in use by 40% so should support

	if err := http2.ConfigureClient(client, http2.ClientOpts{
		MaxResponseTime: config.ReadTimeout,
	}); err != nil {
		return nil, err
	}

	return client, nil
}
