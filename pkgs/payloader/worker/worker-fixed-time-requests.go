package worker

import (
	"sync"
	"time"
)

type WorkerFixedTimeRequests struct {
	*WorkerBase
}

func (w *WorkerFixedTimeRequests) Run(wg *sync.WaitGroup) {
	defer wg.Done()

	w.config.StartTrigger.Wait()
	tickerDeadline := time.NewTicker(w.config.Until)
	newReq := time.NewTicker(w.config.ReqEvery)

	for {
		select {
		case <-w.config.Ctx.Done():
			// user cancelled
			return
		case <-tickerDeadline.C:
			return
		case <-newReq.C:
			w.run()
		}
	}

}
