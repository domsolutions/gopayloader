package payloader

import (
	"github.com/domsolutions/gopayloader/wrapper"
	"github.com/spf13/cobra"
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
	conns     uint
	reqs      int64
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Load test HTTP/S server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return wrapper.RunGoPayLoader(reqURI, mTLSCert, mTLSKey, keepAlive, reqs, conns, duration)
	},
}

func init() {
	runCmd.Flags().Int64VarP(&reqs, argRequests, "r", 0, "Number of requests")
	runCmd.Flags().UintVarP(&conns, argConnections, "c", 1, "Number of simultaneous connections")
	runCmd.Flags().BoolVarP(&keepAlive, argKeepAlive, "k", true, "Reuse existing connections")
	runCmd.Flags().DurationVarP(&duration, argTime, "t", 0, "Execution time window, if used with -r will uniformly distribute reqs within time window, without -r reqs are unlimited")
	// TODO fix -h flag not allowed, already in use?? by help!!!
	runCmd.Flags().StringVarP(&reqURI, argHost, "s", "", "Request URI to run load against")
	runCmd.Flags().StringVar(&mTLSCert, argMTLSCert, "", "mTLS cert path")
	runCmd.Flags().StringVar(&mTLSKey, argMTLSKey, "", "mTLS cert private key path")

	runCmd.MarkFlagsRequiredTogether(argMTLSCert, argMTLSKey)

	if err := runCmd.MarkFlagRequired(argHost); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(runCmd)
}
