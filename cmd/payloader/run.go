package payloader

import (
	"errors"
	"github.com/domsolutions/gopayloader/wrapper"
	"github.com/spf13/cobra"
	"time"
)

const (
	argMethod       = "method"
	argConnections  = "connections"
	argRequests     = "requests"
	argKeepAlive    = "disable-keep-alive"
	argVerifySigner = "skip-verify"
	argTime         = "time"
	argMTLSKey      = "mtls-key"
	argMTLSCert     = "mtls-cert"
	argReadTimeout  = "read-timeout"
	argWriteTimeout = "write-timeout"
	argVerbose      = "verbose"
	argTicker       = "ticker"
	argHTTPV2       = "http-v2"
	argJWTKey       = "jwt-key"
	argJWTSUb       = "jwt-sub"
	argJWTIss       = "jwt-iss"
	argJWTAud       = "jwt-aud"
	argJWTHeader    = "jwt-header"
	argJWTEnable    = "jwt-enable"
	argJWTKid       = "jwt-kid"
	argClearCache   = "clear-cache"
	argHeaders      = "headers"
	argBody         = "body"
	argBodyFile     = "body-file"
)

var (
	method           string
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
	HTTPV2           bool
	jwtKey           string
	jwtSub           string
	jwtIss           string
	jwtAud           string
	jwtHeader        string
	sendJWT          bool
	jwtKID           string
	clearCache       bool
	headers          *[]string
	body             string
	bodyFile         string
)

var runCmd = &cobra.Command{
	Use:   "run <host>",
	Short: "Load test HTTP/S server",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("no request uri specified as argument")
		}
		return nil
	},
	Long: ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		reqURI := args[0]
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
			ticker,
			HTTPV2,
			jwtKID,
			jwtKey,
			jwtSub,
			jwtIss,
			jwtAud,
			jwtHeader,
			sendJWT,
			clearCache,
			*headers,
			body,
			bodyFile)
	},
}

func init() {
	runCmd.Flags().Int64VarP(&reqs, argRequests, "r", 0, "Number of requests")
	runCmd.Flags().UintVarP(&conns, argConnections, "c", 1, "Number of simultaneous connections")
	runCmd.Flags().BoolVarP(&disableKeepAlive, argKeepAlive, "k", false, "Disable keep-alive connections")
	// TODO test http/2 works - just hangs
	runCmd.Flags().BoolVar(&HTTPV2, argHTTPV2, false, "Use HTTP/2")
	runCmd.Flags().BoolVar(&skipVerify, argVerifySigner, false, "Skip verify SSL cert signer")
	runCmd.Flags().DurationVarP(&duration, argTime, "t", 0, "Execution time window, if used with -r will uniformly distribute reqs within time window, without -r reqs are unlimited")
	runCmd.Flags().DurationVar(&readTimeout, argReadTimeout, 5*time.Second, "Read timeout")
	runCmd.Flags().DurationVar(&writeTimeout, argWriteTimeout, 5*time.Second, "Write timeout")
	runCmd.Flags().StringVarP(&method, argMethod, "m", "GET", "request method")
	runCmd.Flags().StringVarP(&body, argBody, "b", "", "request body")
	runCmd.Flags().StringVar(&bodyFile, argBodyFile, "", "read request body from file")
	runCmd.Flags().BoolVarP(&verbose, argVerbose, "v", false, "verbose - slows down RPS slightly for long running tests")
	runCmd.Flags().DurationVar(&ticker, argTicker, time.Second, "How often to print results while running in verbose mode")

	runCmd.Flags().StringVar(&mTLSCert, argMTLSCert, "", "mTLS cert path")
	runCmd.Flags().StringVar(&mTLSKey, argMTLSKey, "", "mTLS cert private key path")

	headers = runCmd.Flags().StringSliceP(argHeaders, "H", []string{}, "headers to send in request, can have multiple i.e -H 'content-type:application/json' -H' connection:close'")

	// TODO in stats, bytes sent/received... received means reading body, possibly rps reduce

	runCmd.Flags().StringVar(&jwtKID, argJWTKid, "", "JWT KID")
	runCmd.Flags().StringVar(&jwtKey, argJWTKey, "", "JWT signing private key path")
	runCmd.Flags().StringVar(&jwtAud, argJWTAud, "", "JWT audience (aud) claim")
	runCmd.Flags().StringVar(&jwtIss, argJWTIss, "", "JWT issuer (iss) claim")
	runCmd.Flags().StringVar(&jwtSub, argJWTSUb, "", "JWT subject (sub) claim")
	runCmd.Flags().StringVar(&jwtHeader, argJWTHeader, "", "JWT header field name")
	runCmd.Flags().BoolVar(&sendJWT, argJWTEnable, false, "Send JWTs with requests")
	runCmd.Flags().BoolVar(&clearCache, argClearCache, false, "Delete all generated jwts")

	runCmd.MarkFlagsRequiredTogether(argMTLSCert, argMTLSKey)
	runCmd.MarkFlagsRequiredTogether(argJWTKey, argJWTEnable)
	runCmd.MarkFlagsMutuallyExclusive(argBody, argBodyFile)

	rootCmd.AddCommand(runCmd)
}
