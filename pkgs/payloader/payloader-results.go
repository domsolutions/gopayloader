package payloader

import (
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/pterm/pterm"
	"time"
)

type PayloaderResults struct {
	*PayLoader
}

func NewPayLoaderResults(pl *PayLoader) *PayloaderResults {
	return &PayloaderResults{pl}
}

func (p *PayloaderResults) ComputeResults(workers []worker.Worker) (*Results, error) {
	results := &Results{
		Start:     p.startTime,
		End:       p.stopTime,
		Total:     p.stopTime.Sub(p.startTime),
		Responses: make(map[worker.ResponseCode]int64),
		Errors:    make(map[string]uint),
	}

	pterm.Debug.Println("Calculating response code statistics")
	for _, w := range workers {
		stats := w.Stats()
		results.CompletedReqs += stats.CompletedReqs
		results.FailedReqs += stats.FailedReqs

		for _, l := range stats.Reqs {
			results.LatencyPerReq = append(results.LatencyPerReq, time.Duration(l[1]-l[0]))
		}

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

	pterm.Debug.Println("Calculating max/min RPS")
	calcMaxMinRPS(results, workers)

	if len(results.LatencyPerReq) > 0 {
		pterm.Debug.Println("Calculating max/min latency")

		var totalLatency time.Duration = 0
		results.Latency.Min = results.LatencyPerReq[0]

		for _, r := range results.LatencyPerReq {
			if r > results.Latency.Max {
				results.Latency.Max = r
			}
			if r < results.Latency.Min {
				results.Latency.Min = r
			}
			totalLatency += r
		}

		results.Latency.Average = totalLatency / time.Duration(len(results.LatencyPerReq))
	}

	results.ReqByteSize.Single = workers[0].ReqSize()
	results.ReqByteSize.Total = workers[0].ReqSize() * results.CompletedReqs
	if numSeconds := int64(results.Total / time.Second); numSeconds == 0 {
		results.ReqByteSize.PerSecond = workers[0].ReqSize() * results.CompletedReqs
	} else {
		results.ReqByteSize.PerSecond = (workers[0].ReqSize() * results.CompletedReqs) / int64(results.Total/time.Second)
	}

	return results, nil
}

func calcMaxMinRPS(results *Results, workers []worker.Worker) {
	reqsPerSecond := make(map[time.Duration]uint64)

	calcRPSForRange := func(latencies []worker.ReqLatency, startTime time.Time) []worker.ReqLatency {
		begin := startTime.UnixNano()
		end := startTime.Add(time.Second).UnixNano()
		outOfBoundsLatencies := make([]worker.ReqLatency, 0)

		for _, l := range latencies {
			if l[worker.ReqBegin] >= begin && l[worker.ReqEnd] <= end {
				if _, ok := reqsPerSecond[time.Duration(startTime.Unix())]; ok {
					reqsPerSecond[time.Duration(startTime.Unix())]++
				} else {
					reqsPerSecond[time.Duration(startTime.Unix())] = 1
				}
				continue
			}
			outOfBoundsLatencies = append(outOfBoundsLatencies, l)
		}
		return outOfBoundsLatencies
	}

	reqs := make([]worker.ReqLatency, 0)
	for _, w := range workers {
		reqs = append(reqs, w.Stats().Reqs...)
	}

	for t := results.Start; t.Before(results.End); t = t.Add(500 * time.Millisecond) {
		reqs = calcRPSForRange(reqs, t)
	}

	// TODO standard deviation, histogram

	if len(reqsPerSecond) > 0 {
		for _, val := range reqsPerSecond {
			if val > results.RPS.Max {
				results.RPS.Max = val
			}
			if val < results.RPS.Min || results.RPS.Min == 0 {
				results.RPS.Min = val
			}
		}
		results.RPS.Average = float64(results.CompletedReqs) / (float64(results.Total) / float64(time.Second))
	}
}
