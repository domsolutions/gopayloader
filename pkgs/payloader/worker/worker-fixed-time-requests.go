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

	w.config.StartTrigger.Wait()
	deadline, _ := context.WithTimeout(context.Background(), w.config.Until)
	newReq := time.NewTicker(w.config.ReqEvery)

	for {
		select {
		case <-w.config.Ctx.Done():
			// user cancelled
			return
		case <-deadline.Done():
			// required reqs were not completed in time period, finish reqs
			if w.config.ReqTarget != w.stats.CompletedReqs+w.stats.FailedReqs {
				w.run()
				continue
			}
			return
		case <-newReq.C:
			w.run()
		}
	}

}
