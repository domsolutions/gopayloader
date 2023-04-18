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

	// TODO optimise 3 loops
	pterm.Debug.Println("Calculating max/min RPS")
	reqsPerSecond := make(map[time.Duration]uint64)
	for t := results.Start; t.Before(results.End); t = t.Add(time.Second) {
		begin := t.UnixNano()
		end := t.Add(time.Second).UnixNano()

		for _, w := range workers {
			stats := w.Stats()
			for _, l := range stats.Reqs {
				if l[worker.ReqBegin] >= begin && l[worker.ReqEnd] <= end {
					if _, ok := reqsPerSecond[time.Duration(t.Unix())]; ok {
						reqsPerSecond[time.Duration(t.Unix())]++
					} else {
						reqsPerSecond[time.Duration(t.Unix())] = 1
					}
				}
			}
		}
	}

	// TODO standard deviation, histogram

	if len(reqsPerSecond) > 0 {
		results.RPS.Min = reqsPerSecond[0]

		for _, val := range reqsPerSecond {
			if val > results.RPS.Max {
				results.RPS.Max = val
			}
			if val < results.RPS.Min {
				results.RPS.Min = val
			}
		}

		results.RPS.Average = float64(results.CompletedReqs) / (float64(results.Total) / float64(time.Second))
	}

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

	return results, nil
}
