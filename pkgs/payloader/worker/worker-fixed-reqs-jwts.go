package worker

import (
	"sync"
)

type WorkerFixedReqsJwts struct {
	*WorkerJWTBase
}

func (w *WorkerFixedReqsJwts) Run(wg *sync.WaitGroup) {
	defer wg.Done()

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
}
