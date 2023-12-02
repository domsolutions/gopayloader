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
	statsSuccessLock *sync.Mutex
	statsErrorLock   *sync.Mutex
	config           *http_clients.Config
	client           http_clients.GoPayLoaderClient
	stats            Stats
	middleware       func(w *WorkerBase, req http_clients.Request)
	reqStats         chan<- time.Duration
	parallel         bool
	method           string
	url              string
	reqSize          int64
	respSize         int64
	parallelWg       *sync.WaitGroup
}

func (w *WorkerBase) ReqSize() int64 {
	return w.reqSize
}

func (w *WorkerBase) RespSize() int64 {
	return w.respSize
}

func (w *WorkerBase) updateErrStats(err error) {
	w.statsErrorLock.Lock()
	defer w.statsErrorLock.Unlock()

	val, ok := w.stats.Errors.Load(err.Error())
	if ok {
		w.stats.Errors.Store(err.Error(), val.(uint64)+1)
	} else {
		w.stats.Errors.Store(err.Error(), uint64(1))
	}

	w.stats.FailedReqs.Add(1)
}

func (w *WorkerBase) run() {
	if w.parallel {
		w.parallelWg.Add(1)
		go func() {
			defer w.parallelWg.Done()

			err := w.process()
			if err != nil {
				w.updateErrStats(err)
			}
		}()
		return
	}

	err := w.process()
	if err != nil {
		w.updateErrStats(err)
	}
}

func (w *WorkerBase) process() error {
	begin := time.Now().UnixNano()
	var end int64
	var err error

	req, err := newReq(w.client, w.config)
	if err != nil {
		return err
	}

	resp := w.client.NewResponse()

	defer func() {
		if err == nil {
			w.reqStats <- time.Duration(end - begin)
			// this frees up the connection to be used by other requests
			resp.Close()
		}
	}()

	if w.middleware != nil {
		w.middleware(w, req)
	}

	if err = w.client.Do(req, resp); err != nil {
		end = time.Now().UnixNano()
		return err
	}
	end = time.Now().UnixNano()

	w.updateRespStats(req, resp)
	return nil
}

func (w *WorkerBase) updateRespStats(req http_clients.Request, resp http_clients.Response) {
	w.statsSuccessLock.Lock()
	defer w.statsSuccessLock.Unlock()

	if w.reqSize == 0 {
		w.reqSize = req.Size()
	}

	if w.respSize == 0 {
		w.respSize = resp.Size()
	}

	w.stats.CompletedReqs.Add(1)

	val, ok := w.stats.Responses.Load(ResponseCode(resp.StatusCode()))
	if ok {
		w.stats.Responses.Store(ResponseCode(resp.StatusCode()), val.(int64)+1)
		return
	}

	w.stats.Responses.Store(ResponseCode(resp.StatusCode()), int64(1))
}

func (w *WorkerBase) Stats() Stats {
	return w.stats
}
