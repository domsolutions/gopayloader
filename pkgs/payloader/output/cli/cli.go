package cli

import (
	"fmt"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/jedib0t/go-pretty/v6/table"
	"os"
	"strconv"
	"time"
)

func Display(results *payloader.Results) {
	fmt.Print("Gopayloader results \n\n")

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	displayOverview(results, t)
	displayRPS(results.RPS, t)
	displayLatency(results.Latency, t)
	displayResponseCodes(results.Responses, t)

	if len(results.Errors) > 0 {
		displayErrors(results.Errors, t)
	}

	t.Render()
}

func displayOverview(results *payloader.Results, t table.Writer) {
	t.AppendHeader(table.Row{"Metric", "Result"})
	t.AppendRows([]table.Row{
		{"Total time", results.Total},
		{"Start time", results.Start.Format(time.RFC1123)},
		{"End time", results.End.Format(time.RFC1123)},
		{"Completed requests", results.CompletedReqs},
		{"Failed requests", results.FailedReqs},
	})
	t.AppendSeparator()
}

func displayErrors(errors map[string]uint, t table.Writer) {
	rows := make([]table.Row, 0)
	for err, count := range errors {
		rows = append(rows, table.Row{"Error; " + err, count})
	}
	t.AppendRows(rows)
	t.AppendSeparator()
}

func displayResponseCodes(resps map[worker.ResponseCode]int64, t table.Writer) {
	rows := make([]table.Row, 0)
	for code, freq := range resps {
		rows = append(rows, table.Row{"Response code; " + strconv.Itoa(int(code)), freq})
	}
	t.AppendRows(rows)
	t.AppendSeparator()
}

func displayLatency(results payloader.Latency, t table.Writer) {
	t.AppendRows([]table.Row{
		{"Average latency", results.Average},
		{"Max latency", results.Max},
		{"Min latency", results.Min},
	})
	t.AppendSeparator()
}

func displayRPS(results payloader.RPS, t table.Writer) {
	t.AppendRows([]table.Row{
		{"Average RPS", results.Average},
		{"Max RPS", results.Max},
		{"Min RPS", results.Min},
	})

	t.AppendSeparator()
}
