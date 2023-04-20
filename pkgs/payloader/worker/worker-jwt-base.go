package worker

import (
	"github.com/valyala/fasthttp"
	"time"
)

type WorkerJWTBase struct {
	config    *Config
	client    *fasthttp.HostClient
	stats     Stats
	req       *fasthttp.Request
	resp      *fasthttp.Response
	jwtStream <-chan string
	jwtHeader string
	jwt       string
}

func (w *WorkerJWTBase) run() {
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

func (w *WorkerJWTBase) setJWT() {
	select {
	case <-w.config.Ctx.Done():
		// user cancelled
		return
	case w.jwt = <-w.jwtStream:
		w.req.Header.Set(w.jwtHeader, w.jwt)
	}
}

func (w *WorkerJWTBase) process() error {
	begin := time.Now().UnixNano()
	var err error

	defer func() {
		if err == nil {
			w.stats.Reqs = append(w.stats.Reqs, ReqLatency{begin, time.Now().UnixNano()})
		}
	}()

	w.setJWT()
	if err = w.client.Do(w.req, w.resp); err != nil {
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

func (w *WorkerJWTBase) Stats() Stats {
	return w.stats
}
