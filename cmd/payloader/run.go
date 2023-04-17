package payloader

import (
	"github.com/domsolutions/gopayloader/wrapper"
	"github.com/spf13/cobra"
	"time"
)

const (
	argMethod       = "method"
	argConnections  = "connections"
	argRequests     = "requests"
	argKeepAlive    = "disable-keep-alive"
	argVerifySigner = "verify"
	argTime         = "time"
	argHost         = "host"
	argMTLSKey      = "mtls-key"
	argMTLSCert     = "mtls-cert"
	argReadTimeout  = "read-timeout"
	argWriteTimeout = "write-timeout"
	argVerbose      = "verbose"
	argTicker       = "ticker"
)

var (
	method           string
	reqURI           string
	mTLSCert         string
	mTLSKey          string
	duration         time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	disableKeepAlive bool
	conns            uint
	reqs             int64
	skipVerify       bool
	verbose          bool
	ticker           time.Duration
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Load test HTTP/S server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return wrapper.RunGoPayLoader(reqURI,
			mTLSCert,
			mTLSKey,
			disableKeepAlive,
			reqs,
			conns,
			duration,
			skipVerify,
			readTimeout,
			writeTimeout,
			method,
			verbose,
			ticker)
	},
}

func init() {
	runCmd.Flags().Int64VarP(&reqs, argRequests, "r", 0, "Number of requests")
	runCmd.Flags().UintVarP(&conns, argConnections, "c", 1, "Number of simultaneous connections")
	runCmd.Flags().BoolVarP(&disableKeepAlive, argKeepAlive, "k", false, "Disable keep-alive connections")
	runCmd.Flags().BoolVar(&skipVerify, argVerifySigner, true, "Verify SSL cert signer")
	runCmd.Flags().DurationVarP(&duration, argTime, "t", 0, "Execution time window, if used with -r will uniformly distribute reqs within time window, without -r reqs are unlimited")
	runCmd.Flags().DurationVar(&readTimeout, argReadTimeout, 5*time.Second, "Read timeout")
	runCmd.Flags().DurationVar(&writeTimeout, argWriteTimeout, 5*time.Second, "Write timeout")
	runCmd.Flags().StringVar(&reqURI, argHost, "", "Request URI to run load against")
	runCmd.Flags().StringVar(&mTLSCert, argMTLSCert, "", "mTLS cert path")
	runCmd.Flags().StringVar(&mTLSKey, argMTLSKey, "", "mTLS cert private key path")
	runCmd.Flags().StringVarP(&method, argMethod, "m", "GET", "request method")
	runCmd.Flags().BoolVarP(&verbose, argVerbose, "v", false, "verbose - slows down RPS slightly for long running tests")
	runCmd.Flags().DurationVar(&ticker, argTicker, time.Second, "How often to print results while running in verbose mode")

	runCmd.MarkFlagsRequiredTogether(argMTLSCert, argMTLSKey)

	if err := runCmd.MarkFlagRequired(argHost); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(runCmd)
}
