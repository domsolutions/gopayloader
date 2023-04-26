package payloader

import (
	"context"
	"crypto/tls"
	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
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
	// give time for server to spin up
	time.Sleep(1 * time.Second)
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

func TestPayLoader_RunFastHTTP2(t *testing.T) {
	type fields struct {
		config *config.Config
	}
	tests := []struct {
		name    string
		fields  fields
		want    *GoPayloaderResults
		wantErr bool
	}{
		{
			name: "GET 10 connections for 2100 requests",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        "https://localhost:8889",
				ReqTarget:     2100,
				Conns:         10,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        "fasthttp-2",
				VerboseTicker: time.Second,
				SkipVerify:    true,
				Verbose:       true,
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
				ReqURI:        "https://localhost:8889",
				Conns:         10,
				Duration:      2 * time.Second,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "POST",
				Client:        "fasthttp-2",
				VerboseTicker: time.Second,
				SkipVerify:    true,
				Verbose:       true,
			}},
		},
		{
			name: "PUT 10 connections for 1 second long test with 100 requests",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        "https://localhost:8889",
				Conns:         10,
				ReqTarget:     100,
				Duration:      1 * time.Second,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "PUT",
				Client:        "fasthttp-2",
				VerboseTicker: time.Second,
				SkipVerify:    true,
				Verbose:       true,
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
				ReqURI:        "https://localhost:8889",
				ReqTarget:     2100,
				Conns:         10,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        "fasthttp-2",
				VerboseTicker: time.Second,
				JwtHeader:     "some-jwt",
				JwtKey:        filepath.Join("..", "..", "test", "private-key.pem"),
				SendJWT:       true,
				SkipVerify:    true,
				Verbose:       true,
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
		})
	}
}

func TestPayLoader_RunFastHTTP1(t *testing.T) {
	type fields struct {
		config *config.Config
	}
	tests := []struct {
		name    string
		fields  fields
		want    *GoPayloaderResults
		wantErr bool
	}{
		{
			name: "GET 10 connections for 2100 requests",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        "http://localhost:8888",
				ReqTarget:     2100,
				Conns:         10,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        "fasthttp-1",
				VerboseTicker: time.Second,
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
				ReqURI:        "http://localhost:8888",
				Conns:         10,
				Duration:      2 * time.Second,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "POST",
				Client:        "fasthttp-1",
				VerboseTicker: time.Second,
			}},
		},
		{
			name: "PUT 10 connections for 1 second long test with 100 requests",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        "http://localhost:8888",
				Conns:         10,
				ReqTarget:     100,
				Duration:      1 * time.Second,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "PUT",
				Client:        "fasthttp-1",
				VerboseTicker: time.Second,
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
				ReqURI:        "http://localhost:8888",
				ReqTarget:     2100,
				Conns:         10,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        "fasthttp-1",
				VerboseTicker: time.Second,
				JwtHeader:     "some-jwt",
				JwtKey:        filepath.Join("..", "..", "test", "private-key.pem"),
				SendJWT:       true,
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
		})
	}
}
