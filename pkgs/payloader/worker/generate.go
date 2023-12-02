package worker

import (
	"fmt"
	http_clients "github.com/domsolutions/gopayloader/pkgs/http-clients"
	"github.com/domsolutions/gopayloader/pkgs/http-clients/fasthttp"
	"github.com/domsolutions/gopayloader/pkgs/http-clients/nethttp"
	"os"
	"strings"
	"sync"
)

const (
	HttpClientNetHTTP   = "nethttp"
	HttpClientNetHTTP2  = "nethttp2"
	HttpClientNetHTTP3  = "nethttp3"
	HttpClientFastHTTP1 = "fasthttp"
)

type TotalRequestsComplete int64

type ResponseCode int

type Stats struct {
	CompletedReqs int64
	FailedReqs    int64
	Responses     *sync.Map
	Errors        *sync.Map
}

func NewWorker(config *http_clients.Config) (Worker, error) {
	client, err := http(config)
	if err != nil {
		return nil, err
	}

	if config.ReqLimitedOnly() {
		if config.JwtStreamReceiver != nil {
			w := &WorkerFixedReqs{baseConfig(config, client)}
			w.middleware = jwtMiddleware
			return w, nil
		}
		return &WorkerFixedReqs{baseConfig(config, client)}, nil
	}

	if config.UnlimitedReqs() {
		return &WorkerFixedTime{baseConfig(config, client)}, nil
	}

	w := &WorkerFixedTimeRequests{baseConfig(config, client)}
	if config.JwtStreamReceiver != nil {
		w.middleware = jwtMiddleware
	}
	return w, nil
}

func newReq(client http_clients.GoPayLoaderClient, config *http_clients.Config) (http_clients.Request, error) {
	req, err := client.NewReq(config.Method, config.ReqURI)
	if err != nil {
		return nil, err
	}

	if config.DisableKeepAlive {
		req.SetHeader("Connection", "close")
	}
	if len(config.Headers) > 0 {
		for _, h := range config.Headers {
			header := strings.Split(h, ":")
			req.SetHeader(header[0], header[1])
		}
	}

	if len(config.Body) > 0 {
		req.SetBody([]byte(config.Body))
	}

	if len(config.BodyFile) > 0 {
		bb, err := os.ReadFile(config.BodyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read body file %v", err)
		}
		req.SetBody(bb)
	}

	return req, nil
}

func jwtMiddleware(w *WorkerBase, req http_clients.Request) {
	select {
	case jwt := <-w.config.JwtStreamReceiver:
		req.SetHeader(w.config.JWTHeader, jwt)
	}
}

func baseConfig(config *http_clients.Config, client http_clients.GoPayLoaderClient) *WorkerBase {
	return &WorkerBase{
		config:     config,
		client:     client,
		parallel:   config.Parallel,
		parallelWg: &sync.WaitGroup{},
		reqStats:   config.ReqStats,
		method:     config.Method,
		url:        config.ReqURI,
		stats: Stats{
			Responses: &sync.Map{},
			Errors:    &sync.Map{},
		},
		statsSuccessLock: &sync.Mutex{},
		statsErrorLock:   &sync.Mutex{},
	}
}

func http(config *http_clients.Config) (http_clients.GoPayLoaderClient, error) {
	switch config.Client {
	case HttpClientNetHTTP:
		return nethttp.GetNetHTTPClient(config)
	case HttpClientNetHTTP2:
		return nethttp.GetNetHTTP2Client(config)
	case HttpClientNetHTTP3:
		return nethttp.GetNetHTTP3Client(config)
	case HttpClientFastHTTP1:
		return fasthttp.GetFastHTTPClient1(config)
	}
	return nil, fmt.Errorf("client %s not recognised", config.Client)
}
