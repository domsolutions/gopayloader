package worker

import (
	"context"
	"sync"
	"time"
)

type WorkerFixedTimeRequests struct {
	*WorkerBase
}

func (w *WorkerFixedTimeRequests) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	defer w.client.CloseConns()

	w.config.StartTrigger.Wait()
	deadline, c := context.WithTimeout(context.Background(), w.config.Until)
	defer c()
	newReq := time.NewTicker(w.config.ReqEvery)

	for {
		select {
		case <-w.config.Ctx.Done():
			// user cancelled
			return
		case <-deadline.Done():
			// required reqs were not completed in time period, finish reqs
			if w.config.ReqTarget != w.stats.CompletedReqs.Load()+w.stats.FailedReqs.Load() {
				w.run()
				continue
			}

			if w.parallel {
				w.parallelWg.Wait()
			}
			return
		case <-newReq.C:
			w.run()
		}
	}

}
