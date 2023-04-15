package worker

import (
	"sync"
	"time"
)

type WorkerFixedTime struct {
	*WorkerBase
}

func (w *WorkerFixedTime) Run(wg *sync.WaitGroup) {
	defer wg.Done()

	w.config.StartTrigger.Wait()
	ticker := time.NewTicker(w.config.Until)

	for {
		select {
		case <-w.config.Ctx.Done():
			// user cancelled
			return
		case <-ticker.C:
			return
		default:
			w.run()
		}
	}

}
