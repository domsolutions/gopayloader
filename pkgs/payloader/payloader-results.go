package payloader

import (
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/pterm/pterm"
	"time"
)

func (p *PayLoader) ComputeResults(workers []worker.Worker, results *GoPayloaderResults) (*GoPayloaderResults, error) {
	results.Start = p.startTime
	results.End = p.stopTime
	results.Total = p.stopTime.Sub(p.startTime)
	results.Errors = make(map[string]uint)
	results.Responses = make(map[worker.ResponseCode]int64)

	pterm.Debug.Println("Calculating response code statistics")

	for _, w := range workers {
		stats := w.Stats()
		results.CompletedReqs += stats.CompletedReqs
		results.FailedReqs += stats.FailedReqs

		for err, count := range stats.Errors {
			if _, ok := results.Errors[err]; ok {
				results.Errors[err] += count
			} else {
				results.Errors[err] = count
			}
		}

		for code, val := range stats.Responses {
			if _, ok := results.Responses[code]; ok {
				results.Responses[code] += val
			} else {
				results.Responses[code] = val
			}
		}
	}

	if results.CompletedReqs > 0 {
		results.Latency.Average = results.Latency.Total / time.Duration(results.CompletedReqs)
		results.RPS.Average = float64(results.CompletedReqs) / (float64(results.Total) / float64(time.Second))

		results.ReqByteSize.Single = workers[0].ReqSize()
		results.ReqByteSize.Total = workers[0].ReqSize() * results.CompletedReqs
		if numSeconds := int64(results.Total / time.Second); numSeconds == 0 {
			results.ReqByteSize.PerSecond = workers[0].ReqSize() * results.CompletedReqs
		} else {
			results.ReqByteSize.PerSecond = (workers[0].ReqSize() * results.CompletedReqs) / int64(results.Total/time.Second)
		}

		results.RespByteSize.Single = workers[0].RespSize()
		results.RespByteSize.Total = workers[0].RespSize() * results.CompletedReqs
		if numSeconds := int64(results.Total / time.Second); numSeconds == 0 {
			results.RespByteSize.PerSecond = workers[0].RespSize() * results.CompletedReqs
		} else {
			results.RespByteSize.PerSecond = (workers[0].RespSize() * results.CompletedReqs) / int64(results.Total/time.Second)
		}
	}

	return results, nil
}
