package payloader

import (
	"errors"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/domsolutions/gopayloader/wrapper"
	"github.com/spf13/cobra"
	"time"
)

const (
	argMethod       	 = "method"
	argConnections  	 = "connections"
	argRequests     	 = "requests"
	argKeepAlive    	 = "disable-keep-alive"
	argVerifySigner 	 = "skip-verify"
	argTime         	 = "time"
	argMTLSKey      	 = "mtls-key"
	argMTLSCert     	 = "mtls-cert"
	argReadTimeout  	 = "read-timeout"
	argWriteTimeout 	 = "write-timeout"
	argVerbose      	 = "verbose"
	argTicker       	 = "ticker"
	argJWTKey       	 = "jwt-key"
	argJWTSUb       	 = "jwt-sub"
	argJWTCustomClaims = "jwt-claims"
	argJWTIss       	 = "jwt-iss"
	argJWTAud       	 = "jwt-aud"
	argJWTHeader    	 = "jwt-header"
	argJWTKid       	 = "jwt-kid"
	argJWTsFilename    = "jwts-filename"
	argHeaders      	 = "headers"
	argBody         	 = "body"
	argBodyFile     	 = "body-file"
	argClient       	 = "client"
)

var (
	client           string
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
	jwtKey           string
	jwtSub           string
	jwtCustomClaims  string
	jwtIss           string
	jwtAud           string
	jwtHeader        string
	jwtKID           string
	jwtsFilename		 string
	headers          *[]string
	body             string
	bodyFile         string
)

var runCmd = &cobra.Command{
	Use:   "run <host>(host format - protocol://host:port/path i.e. https://localhost:443/some-path)",
	Short: "Load test HTTP/S server - supports HTTP/1.1 HTTP/2 HTTP/3",
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
			jwtKID,
			jwtKey,
			jwtSub,
			jwtCustomClaims,
			jwtIss,
			jwtAud,
			jwtHeader,
			jwtsFilename,
			*headers,
			body,
			bodyFile,
			client)
	},
}

func init() {
	runCmd.Flags().Int64VarP(&reqs, argRequests, "r", 0, "Number of requests")
	runCmd.Flags().UintVarP(&conns, argConnections, "c", 1, "Number of simultaneous connections")
	runCmd.Flags().BoolVarP(&disableKeepAlive, argKeepAlive, "k", false, "Disable keep-alive connections")

	runCmd.Flags().BoolVar(&skipVerify, argVerifySigner, false, "Skip verify SSL cert signer")
	runCmd.Flags().DurationVarP(&duration, argTime, "t", 0, "Execution time window, if used with -r will uniformly distribute reqs within time window, without -r reqs are unlimited")
	runCmd.Flags().DurationVar(&readTimeout, argReadTimeout, 5*time.Second, "Read timeout")
	runCmd.Flags().DurationVar(&writeTimeout, argWriteTimeout, 5*time.Second, "Write timeout")
	runCmd.Flags().StringVarP(&method, argMethod, "m", "GET", "request method")
	runCmd.Flags().StringVarP(&body, argBody, "b", "", "request body")
	runCmd.Flags().StringVar(&bodyFile, argBodyFile, "", "read request body from file")
	runCmd.Flags().BoolVarP(&verbose, argVerbose, "v", false, "verbose - slows down RPS slightly for long running tests")
	runCmd.Flags().DurationVar(&ticker, argTicker, time.Second, "How often to print results while running in verbose mode")
	headers = runCmd.Flags().StringSliceP(argHeaders, "H", []string{}, "headers to send in request, can have multiple i.e -H 'content-type:application/json' -H' connection:close'")
	runCmd.Flags().StringVar(&mTLSCert, argMTLSCert, "", "mTLS cert path")
	runCmd.Flags().StringVar(&mTLSKey, argMTLSKey, "", "mTLS cert private key path")

	runCmd.Flags().StringVar(&client, argClient, worker.HttpClientFastHTTP1, worker.HttpClientFastHTTP1+` for fast http/1.1 requests
`+worker.HttpClientFastHTTP2+` for fast http/2 requests 
`+worker.HttpClientNetHTTP+` for standard net/http requests supporting http/1.1 http/2
`+worker.HttpClientNetHTTP3+` for standard net/http requests supporting http/3 using quic-go`)

	runCmd.Flags().StringVar(&jwtKID, argJWTKid, "", "JWT KID")
	runCmd.Flags().StringVar(&jwtKey, argJWTKey, "", "JWT signing private key path")
	runCmd.Flags().StringVar(&jwtAud, argJWTAud, "", "JWT audience (aud) claim")
	runCmd.Flags().StringVar(&jwtIss, argJWTIss, "", "JWT issuer (iss) claim")
	runCmd.Flags().StringVar(&jwtSub, argJWTSUb, "", "JWT subject (sub) claim")
	runCmd.Flags().StringVar(&jwtCustomClaims, argJWTCustomClaims, "", "JWT custom claims")
	runCmd.Flags().StringVarP(&jwtsFilename, argJWTsFilename, "f", "", "File name in .cache where the JWTs to use are stored")
	runCmd.Flags().StringVar(&jwtHeader, argJWTHeader, "", "JWT header field name")

	runCmd.MarkFlagsRequiredTogether(argMTLSCert, argMTLSKey)
	runCmd.MarkFlagsRequiredTogether(argJWTKey, argJWTHeader)
	runCmd.MarkFlagsMutuallyExclusive(argBody, argBodyFile)

	rootCmd.AddCommand(runCmd)
}
