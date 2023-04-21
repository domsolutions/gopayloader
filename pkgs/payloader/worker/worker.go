package worker

import (
	http_clients "github.com/domsolutions/gopayloader/pkgs/http-clients"
	"sync"
	"time"
)

type Worker interface {
	Run(wg *sync.WaitGroup)
	Stats() Stats
}

type WorkerBase struct {
	config     *http_clients.Config
	client     http_clients.GoPayLoaderClient
	stats      Stats
	req        http_clients.Request
	resp       http_clients.Response
	middleware func(w *WorkerBase)
}

func (w *WorkerBase) run() {
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
		if err == nil {
			w.stats.Reqs = append(w.stats.Reqs, ReqLatency{begin, time.Now().UnixNano()})
		}
	}()

	if w.middleware != nil {
		w.middleware(w)
	}

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
