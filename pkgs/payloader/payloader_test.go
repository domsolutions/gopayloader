package payloader

import (
	"context"
	"crypto/tls"
	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/quic-go/quic-go"
	httpv3server "github.com/quic-go/quic-go/http3"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func init() {
	go testStartHTTP1Server("localhost:8888")
	go testStartHTTP2Server("localhost:8889")
	go testStartHTTP3Server("localhost:8890")
	// give time for server to spin up
	time.Sleep(1 * time.Second)

	err := os.RemoveAll(JwtCacheDir)
	if err != nil {
		log.Printf("Failed to clear jwt cache dir; %v", err)
	}
}

func tlsConfig() *tls.Config {
	crt, err := os.ReadFile(filepath.Join("..", "..", "test", "server.crt"))
	if err != nil {
		log.Fatal(err)
	}

	key, err := os.ReadFile(filepath.Join("..", "..", "test", "server.key"))
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

func testStartHTTP1Server(addr string) {
	var err error
	server := fasthttp.Server{
		Handler: func(c *fasthttp.RequestCtx) {
			_, err = c.WriteString("hello")
			if err != nil {
				log.Println(err)
			}
		},
	}

	if err := server.ListenAndServe(addr); err != nil {
		log.Fatal(err)
	}
}

func testStartHTTP3Server(addr string) {
	var err error
	server := httpv3server.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err = w.Write([]byte("hello"))
			if err != nil {
				log.Println(err)
			}
		}),
		Addr: addr,
		QuicConfig: &quic.Config{
			EnableDatagrams: true,
		},
		TLSConfig: tlsConfig(),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func testStartHTTP2Server(addr string) {
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		TLSConfig:    tlsConfig(),
	}
	var err error

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte("hello"))
		if err != nil {
			log.Println(err)
		}
	})

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
	}
}

func TestPayLoader_RunFastHTTP1NonSSL(t *testing.T) {
	testPayLoader_Run(t, "http://localhost:8888", "fasthttp-1")
}

func TestPayLoader_RunFastHTTP1SSL(t *testing.T) {
	testPayLoader_Run(t, "https://localhost:8889", "fasthttp-1")
}

func TestPayLoader_RunNetHTTP1SSL(t *testing.T) {
	testPayLoader_Run(t, "https://localhost:8889", "nethttp")
}

func TestPayLoader_RunFastHTTP2SSL(t *testing.T) {
	testPayLoader_Run(t, "https://localhost:8889", "fasthttp-2")
}

func TestPayLoader_RunNetHTTP3(t *testing.T) {
	testPayLoader_Run(t, "https://localhost:8890", "nethttp-3")
}

func testPayLoader_Run(t *testing.T, addr, client string) {
	type fields struct {
		config *config.Config
	}
	tests := []struct {
		name    string
		fields  fields
		want    *GoPayloaderResults
		wantErr bool
		check   func(t *testing.T)
	}{
		{
			name: "GET 10 connections for 2100 requests",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				ReqTarget:     2100,
				Conns:         10,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        client,
				VerboseTicker: time.Second,
				SkipVerify:    true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 2100,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 2100,
				},
				Errors: nil,
			},
		},
		{
			name: "POST 10 connections for 2 second long test",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				Conns:         10,
				Duration:      2 * time.Second,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "POST",
				Client:        client,
				VerboseTicker: time.Second,
				SkipVerify:    true,
			}},
		},
		{
			name: "PUT 10 connections for 1 second long test with 100 requests",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				Conns:         10,
				ReqTarget:     100,
				Duration:      1 * time.Second,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "PUT",
				Client:        client,
				VerboseTicker: time.Second,
				SkipVerify:    true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 100,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 100,
				},
				Errors: nil,
			},
		},
		{
			name: "GET 10 connections for 2100 requests with jwts",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				ReqTarget:     2100,
				Conns:         10,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        client,
				VerboseTicker: time.Second,
				JwtHeader:     "some-jwt",
				JwtKey:        filepath.Join("..", "..", "test", "private-key-jwt.pem"),
				SkipVerify:    true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 2100,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 2100,
				},
				Errors: nil,
			},
			check: func(t *testing.T) {
				_, err := os.OpenFile(filepath.Join(JwtCacheDir, "gopayloader-jwtstore-672c3f20f01d56f616b14d9c2b213590abea414a1d19c516b1269ceb0232b345.txt"), os.O_RDONLY, os.ModePerm)
				if err != nil {
					if os.IsNotExist(err) {
						t.Fatal(err)
					}
					t.Fatal(err)
				}
			},
		},
		{
			name: "GET 101 connections for 2100 requests with jwts with all available jwt fields and header",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				ReqTarget:     2100,
				Conns:         101,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        client,
				VerboseTicker: time.Second,
				Headers:       []string{"content-type: application/json"},
				JwtHeader:     "some-jwt",
				JwtAud:        "some-aud",
				JwtSub:        "some-subject",
				JwtIss:        "some-issuer",
				JwtKID:        "13325575tevdfbdsfsf",
				JwtKey:        filepath.Join("..", "..", "test", "private-key-jwt.pem"),
				SkipVerify:    true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 2100,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 2100,
				},
				Errors: nil,
			},
			check: func(t *testing.T) {
				_, err := os.OpenFile(filepath.Join(JwtCacheDir, "gopayloader-jwtstore-ee6963c0246fe92609c0a80921c3ffe35e4d487c4b494d38bcdff151efc41ff4.txt"), os.O_RDONLY, os.ModePerm)
				if err != nil {
					if os.IsNotExist(err) {
						t.Fatal(err)
					}
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPayLoader(tt.fields.config)
			got, err := p.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.fields.config.Duration != 0 && tt.fields.config.ReqTarget == 0 {
				if got.CompletedReqs == 0 {
					t.Errorf("got %d completed requests, wanted > 0", got.CompletedReqs)
				}
				if got.FailedReqs != 0 {
					t.Errorf("got %d failed requests, wanted 0", got.FailedReqs)
				}
				if got.Responses[200] != got.CompletedReqs {
					t.Errorf("got %d failed requests, wanted %d", got.Responses[200], got.CompletedReqs)
				}
				return
			}

			if tt.want.CompletedReqs != got.CompletedReqs {
				t.Errorf("wanted completed reqs %d got %d", tt.want.CompletedReqs, got.CompletedReqs)
			}
			if tt.want.FailedReqs != got.FailedReqs {
				t.Errorf("wanted failed reqs %d got %d", tt.want.FailedReqs, got.FailedReqs)
			}
			if !reflect.DeepEqual(tt.want.Responses, got.Responses) {
				t.Errorf("response codes not expected")
			}

			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}
