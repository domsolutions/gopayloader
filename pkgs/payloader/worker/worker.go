package worker

import (
	"github.com/valyala/fasthttp"
	"sync"
	"time"
)

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

func (w *WorkerBase) run() {
	//w.req = requestPool.Get().(*fasthttp.Request)
	//w.resp = responsePool.Get().(*fasthttp.Response)

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
	var err error

	defer func() {
		//requestPool.Put(w.req)
		//responsePool.Put(w.resp)
		if err == nil {
			w.stats.Reqs = append(w.stats.Reqs, ReqLatency{begin, time.Now().UnixNano()})
		}
	}()

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

func (w *WorkerBase) Stats() Stats {
	return w.stats
}
