package http_clients

import (
	"context"
	"sync"
	"time"
)

type Request interface {
	SetHeader(key, val string)
	SetBody(body []byte)
	Size() int64
}

type Response interface {
	StatusCode() int
}

type GoPayLoaderClient interface {
	Do(req Request, resp Response) error
	NewReq(method, url string) (Request, error)
	NewResponse() Response
}

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
	JwtStreamReceiver <-chan string
	JwtStreamErr      <-chan error
	JWTHeader         string
	Headers           []string
	Body              string
	BodyFile          string
	NetHTTP           bool
	HTTPV3            bool
	ReqStats          chan<- time.Duration
	Client            string
}

func (c *Config) ReqLimitedOnly() bool {
	return c.Until == 0 && c.ReqTarget != 0
}

func (c *Config) UnlimitedReqs() bool {
	return c.Until != 0 && c.ReqTarget == 0
}
