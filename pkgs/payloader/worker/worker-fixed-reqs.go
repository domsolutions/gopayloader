package worker

import (
	"sync"
)

type WorkerFixedReqs struct {
	*WorkerBase
}

func (w *WorkerFixedReqs) Run(wg *sync.WaitGroup) {
	defer wg.Done()

	w.config.StartTrigger.Wait()

	var i int64
	for i = 0; i < w.config.Reqs; i++ {
		select {
		case <-w.config.Ctx.Done():
			// user cancelled
			return
		default:
			w.run()
		}
	}

}
