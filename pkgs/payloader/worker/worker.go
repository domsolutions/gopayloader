package worker

import (
	http_clients "github.com/domsolutions/gopayloader/pkgs/http-clients"
	"sync"
	"time"
)

type Worker interface {
	Run(wg *sync.WaitGroup)
	Stats() Stats
	ReqSize() int64
	RespSize() int64
}

type WorkerBase struct {
	config     *http_clients.Config
	client     http_clients.GoPayLoaderClient
	stats      Stats
	req        http_clients.Request
	resp       http_clients.Response
	middleware func(w *WorkerBase)
	reqStats   chan<- time.Duration
}

func (w *WorkerBase) ReqSize() int64 {
	return w.req.Size()
}

func (w *WorkerBase) RespSize() int64 {
	if w.resp == nil {
		return 0
	}
	return w.resp.Size()
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
	var end int64
	var err error

	defer func() {
		if err == nil {
			w.reqStats <- time.Duration(end - begin)
		}
		if w.resp != nil {
			w.resp.Close()
		}
	}()

	if w.middleware != nil {
		w.middleware(w)
	}

	if err = w.client.Do(w.req, w.resp); err != nil {
		end = time.Now().UnixNano()
		return err
	}
	end = time.Now().UnixNano()

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
