package payloader

import (
	"context"
	"fmt"
	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	argConnections = "connections"
	argRequests    = "requests"
	argKeepAlive   = "keep-alive"
	argTime        = "time"
	argHost        = "host"
	argMTLSKey     = "mtls-key"
	argMTLSCert    = "mtls-cert"
)

var (
	reqURI    string
	mTLSCert  string
	mTLSKey   string
	duration  time.Duration
	keepAlive bool
	conns     int
	reqs      int64
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Load test HTTP/S server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := config.NewConfig(ctx, reqURI, mTLSCert, mTLSKey, keepAlive, reqs, conns, duration)
		if err := config.Validate(); err != nil {
			return err
		}

		payload := payloader.NewPayLoader(config)
		errPayLoader := make(chan error)
		resPayLoader := make(chan *payloader.Results)

		go func() {
			results, err := payload.Run()
			if err != nil {
				errPayLoader <- err
				return
			}
			resPayLoader <- results
		}()

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		select {
		case <-c:
			// user pressed ctrl+c
			cancel()
			results := <-resPayLoader
			fmt.Println(results)
		case err := <-errPayLoader:
			return err
		case results := <-resPayLoader:
			fmt.Println(results)
		}

		return nil
	},
}

func init() {
	runCmd.Flags().Int64VarP(&reqs, argRequests, "r", 0, "Number of requests")
	runCmd.Flags().IntVarP(&conns, argConnections, "c", 1, "Number of simultaneous connections")
	runCmd.Flags().BoolVarP(&keepAlive, argKeepAlive, "k", true, "Reuse existing connections")
	runCmd.Flags().DurationVarP(&duration, argTime, "t", 0, "Execution time window, if used with -r will uniformly distribute reqs within time window, without -r reqs are unlimited")
	// TODO fix -h flag not allowed, already in use??
	runCmd.Flags().StringVarP(&reqURI, argHost, "s", "", "Request URI to run load against")
	runCmd.Flags().StringVarP(&mTLSCert, argMTLSCert, "mc", "", "mTLS cert path")
	runCmd.Flags().StringVarP(&mTLSKey, argMTLSKey, "mk", "", "mTLS cert private key path")

	runCmd.MarkFlagsRequiredTogether(argMTLSCert, argMTLSKey)

	if err := runCmd.MarkFlagRequired(argHost); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(runCmd)
}
