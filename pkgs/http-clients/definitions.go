package http_clients

import (
	"context"
	"sync"
	"time"
)

type Request interface {
	SetHeader(key, val string)
	SetMethod(method string)
	SetBody(body []byte)
	SetRequestURI(uri string)
}

type Response interface {
	StatusCode() int
}

type GoPayLoaderClient interface {
	Do(req Request, resp Response) error
	NewReq() Request
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
	HTTPV2            bool
	JwtStreamReceiver <-chan string
	JwtStreamErr      <-chan error
	JWTHeader         string
	Headers           []string
	Body              string
	BodyFile          string
}

func (c *Config) ReqLimitedOnly() bool {
	return c.Until == 0 && c.ReqTarget != 0
}

func (c *Config) UnlimitedReqs() bool {
	return c.Until != 0 && c.ReqTarget == 0
}