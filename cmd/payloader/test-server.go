package payloader

import (
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
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
		}

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
