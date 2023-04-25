package cli

import (
	"fmt"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pterm/pterm"
	"os"
	"strconv"
	"time"
)

func Display(results *payloader.GoPayloaderResults) {
	pterm.Success.Printf("Gopayloader results \n\n")
	fmt.Println("")

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	displayOverview(results, t)
	displayRPS(results.RPS, t)
	displayReqSize(results.ReqByteSize, t)
	displayRespSize(results.RespByteSize, t)
	displayLatency(results.Latency, t)
	displayResponseCodes(results.Responses, t)

	if len(results.Errors) > 0 {
		displayErrors(results.Errors, t)
	}

	t.Render()
}

func displayOverview(results *payloader.GoPayloaderResults, t table.Writer) {
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

func displayReqSize(req payloader.ByteSize, t table.Writer) {
	rows := make([]table.Row, 0)
	rows = append(rows, table.Row{"Req size (bytes)", req.Single})
	rows = append(rows, table.Row{"Req size/second (MB)", fmt.Sprintf("%.3f", float64(req.PerSecond)/(1024*1024))})
	rows = append(rows, table.Row{"Req total size (MB)", fmt.Sprintf("%.3f", float64(req.Total)/float64(1024*1024))})
	t.AppendRows(rows)
	t.AppendSeparator()
}

func displayRespSize(resp payloader.ByteSize, t table.Writer) {
	rows := make([]table.Row, 0)
	rows = append(rows, table.Row{"Resp size (bytes)", resp.Single})
	rows = append(rows, table.Row{"Resp size/second (MB)", fmt.Sprintf("%.3f", float64(resp.PerSecond)/(1024*1024))})
	rows = append(rows, table.Row{"Resp total size (MB)", fmt.Sprintf("%.3f", float64(resp.Total)/float64(1024*1024))})
	t.AppendRows(rows)
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
		{"Average RPS", fmt.Sprintf("%.3f", results.Average)},
		{"Max RPS", results.Max},
		{"Min RPS", results.Min},
	})

	t.AppendSeparator()
}
