package payloader

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"github.com/domsolutions/http2"
	"github.com/quic-go/quic-go"
	httpv3server "github.com/quic-go/quic-go/http3"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	golanghttp2 "golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	port         int
	responseSize int
	fasthttp1    bool
	fasthttp2    bool
	nethttp2     bool
	httpv3       bool
	debug        bool
)

var (
	serverCert string
	privateKey string
	crt        []byte
	key        []byte
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	serverCert = filepath.Join(wd, "/cmd/payloader/cert/server.crt")
	privateKey = filepath.Join(wd, "/cmd/payloader/cert/server.key")
}

func tlsConfig() *tls.Config {
	var err error
	crt, err = os.ReadFile(serverCert)
	if err != nil {
		log.Fatal(err)
	}

	key, err = os.ReadFile(privateKey)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		log.Fatal(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "localhost",
	}
}

var runServerCmd = &cobra.Command{
	Use:   "http-server",
	Short: "Start a local HTTP server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		response := strings.Repeat("a", responseSize)
		addr := "localhost:" + strconv.Itoa(port)
		log.Println("Starting HTTP server on:", addr)

		if fasthttp1 {
			var err error

			server := fasthttp.Server{
				ConnState: func(c net.Conn, s fasthttp.ConnState) {
					if debug {
						if s == fasthttp.StateNew {
							log.Println("new conn")
						}
					}
				},
				Handler: func(c *fasthttp.RequestCtx) {
					_, err = c.WriteString(response)
					if err != nil {
						log.Println(err)
					}
					if debug {
						log.Printf("%s\n", c.Request.Header.String())
						log.Printf("%s\n", c.Request.Body())
					}
				},
			}

			errs := make(chan error)
			go func() {
				if err := server.ListenAndServe(addr); err != nil {
					log.Println(err)
					errs <- err
				}
			}()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)

			select {
			case <-c:
				log.Println("User cancelled, shutting down")
			case err := <-errs:
				log.Printf("Got error from server; %v \n", err)
			}

			server.Shutdown()
			return nil
		}

		if fasthttp2 {
			var err error

			server := fasthttp.Server{
				ErrorHandler: func(c *fasthttp.RequestCtx, err error) {
					log.Println(err)
					c.WriteString(err.Error())
				},
				Handler: func(c *fasthttp.RequestCtx) {
					_, err = c.WriteString(response)
					if err != nil {
						log.Println(err)
					}
					if debug {
						log.Printf("%s\n", c.Request.Header.String())
						log.Printf("%s\n", c.Request.Body())
					}
				},
			}

			tlsConfig()
			err = server.AppendCertEmbed(crt, key)
			if err != nil {
				log.Fatalln(err)
			}

			http2.ConfigureServer(&server, http2.ServerConfig{
				Debug: debug,
			})

			errs := make(chan error)
			go func() {
				if err := server.ListenAndServeTLSEmbed(addr, crt, key); err != nil {
					log.Println(err)
					errs <- err
				}
			}()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)

			select {
			case <-c:
				log.Println("User cancelled, shutting down")
			case err := <-errs:
				log.Printf("Got error from server; %v \n", err)
			}

			server.Shutdown()
			return nil
		}

		if nethttp2 {
			server := &http.Server{
				Addr:         addr,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				TLSConfig:    tlsConfig(),
				ConnState: func(c net.Conn, s http.ConnState) {
					if !debug {
						return
					}
					switch s {
					case http.StateNew:
						log.Println("NEW conn")
					case http.StateClosed:
						log.Println("CLOSED conn")
					case http.StateHijacked:
						log.Println("HIJACKED conn")
					}
				},
			}
			var err error

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				_, err = w.Write([]byte(response))
				if err != nil {
					log.Println(err)
				}
				if debug {
					log.Printf("%+v\n", r.Header)
					log.Printf("%+v\n", r.Body)
				}
			})

			err = golanghttp2.ConfigureServer(server, &golanghttp2.Server{})
			if err != nil {
				return err
			}

			errs := make(chan error)
			go func() {
				if err := server.ListenAndServeTLS(serverCert, privateKey); err != nil {
					errs <- err
				}
			}()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)

			select {
			case <-c:
				log.Println("User cancelled, shutting down")
			case err := <-errs:
				log.Printf("Got error from server; %v \n", err)
			}

			server.Shutdown(context.Background())
			return nil
		}

		if httpv3 {
			var err error

			quicConf := &quic.Config{
				EnableDatagrams: true,
			}

			tlsConfigServer := tlsConfig()

			server := httpv3server.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, err = w.Write([]byte(response))
					if err != nil {
						log.Println(err)
					}
					if debug {
						log.Printf("%+v\n", r.Header)
					}
				}),
				Addr:       addr,
				QuicConfig: quicConf,
				TLSConfig:  tlsConfigServer,
			}

			if err := server.ListenAndServe(); err != nil {
				log.Fatal(err)
			}
		}

		return errors.New("http option not recognised")
	},
}

func init() {
	runServerCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port")
	runServerCmd.Flags().IntVarP(&responseSize, "response-size", "s", 10, "Response size")
	runServerCmd.Flags().BoolVar(&fasthttp1, "fasthttp-1", false, "Fasthttp HTTP/1.1 server")
	runServerCmd.Flags().BoolVar(&fasthttp2, "fasthttp-2", false, "Fasthttp HTTP/2 server")
	runServerCmd.Flags().BoolVar(&nethttp2, "netHTTP-2", false, "net/http HTTP/2 server")
	runServerCmd.Flags().BoolVar(&httpv3, "http-3", false, "HTTP/3 server")
	runServerCmd.Flags().BoolVarP(&debug, "verbose", "v", false, "print logs")
	rootCmd.AddCommand(runServerCmd)
}

type MyWriteCloser struct {
	*bufio.Writer
}

func (mwc *MyWriteCloser) Close() error {
	// Noop
	return nil
}
