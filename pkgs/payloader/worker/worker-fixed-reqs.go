package worker

import (
	"sync"
)

type WorkerFixedReqs struct {
	*WorkerBase
}

func (w *WorkerFixedReqs) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	defer w.client.CloseConns()

	w.config.StartTrigger.Wait()

	var i int64
	for i = 0; i < w.config.ReqTarget; i++ {
		select {
		case <-w.config.Ctx.Done():
			// user cancelled
			return
		default:
			w.run()
		}
	}

	if w.parallel {
		w.parallelWg.Wait()
	}
}
