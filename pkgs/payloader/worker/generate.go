package worker

import (
	"fmt"
	http_clients "github.com/domsolutions/gopayloader/pkgs/http-clients"
	"github.com/domsolutions/gopayloader/pkgs/http-clients/fasthttp"
	"github.com/domsolutions/gopayloader/pkgs/http-clients/nethttp"
	"os"
	"strings"
)

const (
	HttpClientNetHTTP   = "nethttp"
	HttpClientNetHTTP3  = "nethttp-3"
	HttpClientFastHTTP1 = "fasthttp-1"
	HttpClientFastHTTP2 = "fasthttp-2"
)

type TotalRequestsComplete int64

type ResponseCode int

type Stats struct {
	CompletedReqs int64
	FailedReqs    int64
	Responses     map[ResponseCode]int64
	Errors        map[string]uint
}

func NewWorker(config *http_clients.Config) (Worker, error) {
	client, err := getClient(config)
	if err != nil {
		return nil, err
	}

	resp := client.NewResponse()
	req, err := getReq(client, config)
	if err != nil {
		return nil, err
	}

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

func getReq(client http_clients.GoPayLoaderClient, config *http_clients.Config) (http_clients.Request, error) {
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

func jwtMiddleware(w *WorkerBase) {
	select {
	case <-w.config.Ctx.Done():
		// user cancelled
		return
	//case err := <-w.config.JwtStreamErr:
	//	pterm.Error.Printf("Failed to get jwts from cache, got error; %v \n", err)
	//	return TODO fix
	case jwt := <-w.config.JwtStreamReceiver:
		w.req.SetHeader(w.config.JWTHeader, jwt)
	}
}

func baseConfig(config *http_clients.Config, client http_clients.GoPayLoaderClient, req http_clients.Request, resp http_clients.Response) *WorkerBase {
	return &WorkerBase{
		config:   config,
		req:      req,
		resp:     resp,
		client:   client,
		reqStats: config.ReqStats,
		stats: Stats{
			Responses: make(map[ResponseCode]int64),
			Errors:    make(map[string]uint),
		},
	}
}

func getClient(config *http_clients.Config) (http_clients.GoPayLoaderClient, error) {
	switch config.Client {
	case HttpClientNetHTTP:
		return nethttp.GetNetHTTPClient(config)
	case HttpClientNetHTTP3:
		return nethttp.GetNetHTTP3Client(config)
	case HttpClientFastHTTP1:
		return fasthttp.GetFastHTTPClient1(config)
	case HttpClientFastHTTP2:
		return fasthttp.GetFastHTTPClient2(config)
	}
	return nil, fmt.Errorf("client %s not recognised", config.Client)
}
