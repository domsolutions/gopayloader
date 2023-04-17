package cli

import (
	"fmt"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/jedib0t/go-pretty/v6/table"
	"os"
	"time"
)

func Display(results *payloader.Results) error {
	fmt.Print("Gopayloader results \n\n")

	displayOverview(results)
	displayRPS(results.RPS)
	displayLatency(results.Latency)
	displayResponseCodes(results.Responses)

	if len(results.Errors) > 0 {
		displayErrors(results.Errors)
	}
	return nil
}

func displayOverview(results *payloader.Results) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Metric", "Result"})
	t.AppendRows([]table.Row{
		{"Total time", results.Total},
		{"Start time", results.Start.Format(time.RFC1123)},
		{"End time", results.End.Format(time.RFC1123)},
		{"Completed requests", results.CompletedReqs},
		{"Failed requests", results.FailedReqs},
	})
	t.AppendSeparator()
	t.Render()
}

func displayErrors(errors map[string]uint) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Error", "Count"})

	rows := make([]table.Row, 0)
	for err, count := range errors {
		rows = append(rows, table.Row{err, count})
	}

	t.AppendRows(rows)
	t.AppendSeparator()
	t.Render()
}

func displayResponseCodes(resps map[worker.ResponseCode]int64) {
	response := table.NewWriter()
	response.SetOutputMirror(os.Stdout)
	response.AppendHeader(table.Row{"Response code", "Count"})

	rows := make([]table.Row, 0)
	for code, freq := range resps {
		rows = append(rows, table.Row{code, freq})
	}

	response.AppendRows(rows)
	response.AppendSeparator()
	response.Render()
}

func displayLatency(results payloader.Latency) {
	latency := table.NewWriter()
	latency.SetOutputMirror(os.Stdout)
	latency.AppendHeader(table.Row{"Latency", "Count"})

	latency.AppendRows([]table.Row{
		{"Average", results.Average},
		{"Max", results.Max},
		{"Min", results.Min},
	})

	latency.AppendSeparator()
	latency.Render()
}

func displayRPS(results payloader.RPS) {
	rps := table.NewWriter()
	rps.SetOutputMirror(os.Stdout)
	rps.AppendHeader(table.Row{"RPS", "Count"})

	rps.AppendRows([]table.Row{
		{"Average", results.Average},
		{"Max", results.Max},
		{"Min", results.Min},
	})

	rps.AppendSeparator()
	rps.Render()
}
