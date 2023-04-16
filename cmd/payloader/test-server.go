package payloader

import (
	"github.com/domsolutions/fasthttp"
	"github.com/spf13/cobra"
	"log"
	"strconv"
	"strings"
)

var (
	port         int
	responseSize int
)

var runServerCmd = &cobra.Command{
	Use:   "http-server",
	Short: "Start a local HTTP server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		response := strings.Repeat("a", responseSize)
		addr := "localhost:" + strconv.Itoa(port)
		log.Println("Starting HTTP server on:", addr)

		server := fasthttp.Server{
			Handler: func(c *fasthttp.RequestCtx) {
				_, err := c.WriteString(response)
				if err != nil {
					log.Println(err)
				}
			},
			ErrorHandler:                       nil,
			HeaderReceived:                     nil,
			ContinueHandler:                    nil,
			Name:                               "",
			Concurrency:                        0,
			ReadBufferSize:                     0,
			WriteBufferSize:                    0,
			ReadTimeout:                        0,
			WriteTimeout:                       0,
			IdleTimeout:                        0,
			MaxConnsPerIP:                      0,
			MaxRequestsPerConn:                 0,
			MaxKeepaliveDuration:               0,
			MaxIdleWorkerDuration:              0,
			TCPKeepalivePeriod:                 0,
			MaxRequestBodySize:                 0,
			DisableKeepalive:                   false,
			TCPKeepalive:                       true,
			ReduceMemoryUsage:                  false,
			GetOnly:                            false,
			DisablePreParseMultipartForm:       false,
			LogAllErrors:                       false,
			SecureErrorLogMessage:              false,
			DisableHeaderNamesNormalizing:      false,
			SleepWhenConcurrencyLimitsExceeded: 0,
			NoDefaultServerHeader:              false,
			NoDefaultDate:                      false,
			NoDefaultContentType:               false,
			KeepHijackedConns:                  false,
			CloseOnShutdown:                    false,
			StreamRequestBody:                  false,
			ConnState:                          nil,
			Logger:                             nil,
			TLSConfig:                          nil,
			FormValueFunc:                      nil,
		}
		//}
		//err := fasthttp.ListenAndServe(addr)
		//if err != nil {
		//	return err
		//}

		if err := server.ListenAndServe(addr); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	runServerCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port")
	runServerCmd.Flags().IntVarP(&responseSize, "response-size", "s", 10, "Response size")
	rootCmd.AddCommand(runServerCmd)
}
