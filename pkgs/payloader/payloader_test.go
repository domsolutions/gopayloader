package payloader

import (
	"context"
	"crypto/tls"
	"errors"
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
		wantErr error
		check   func(t *testing.T)
	}{
		{
			name: "GET 10 connections for 210 requests",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				ReqTarget:     210,
				Conns:         10,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        client,
				VerboseTicker: time.Second,
				SkipVerify:    true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 210,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 210,
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
			name: "GET 10 connections for 210 requests with jwts",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				ReqTarget:     210,
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
				CompletedReqs: 210,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 210,
				},
				Errors: nil,
			},
			check: func(t *testing.T) {
				f, err := os.OpenFile(filepath.Join(JwtCacheDir, "gopayloader-jwtstore-18b6ccea4495c0ee5ec82463959343375bc9cd14da1a69ed6e68fe55bc43e3e6.txt"), os.O_RDONLY, os.ModePerm)
				if err != nil {
					if os.IsNotExist(err) {
						t.Fatal(err)
					}
					t.Fatal(err)
				}
				stat, err := f.Stat()
				if err != nil {
					t.Fatal(err)
				}
				if stat.Size() == 0 {
					t.Fatalf("file size 0 for jwt cache store for %s \n", "gopayloader-jwtstore-18b6ccea4495c0ee5ec82463959343375bc9cd14da1a69ed6e68fe55bc43e3e6.txt")
				}
			},
		},
		{
			name: "GET 101 connections for 210 requests with jwts with all available jwt fields and header",
			fields: fields{config: &config.Config{
				Ctx:                 context.Background(),
				ReqURI:              addr,
				ReqTarget:           210,
				Conns:               11,
				ReadTimeout:         5 * time.Second,
				WriteTimeout:        5 * time.Second,
				Method:              "GET",
				Client:              client,
				VerboseTicker:       time.Second,
				Headers:             []string{"content-type: application/json"},
				JwtHeader:           "some-jwt",
				JwtAud:              "some-aud",
				JwtSub:              "some-subject",
				JwtCustomClaimsJSON: "{\"custom-claim1\": \"abc\", \"custom-claim2\": \"def\"}",
				JwtIss:        	     "some-issuer",
				JwtKID:        	     "13325575tevdfbdsfsf",
				JwtKey:        	     filepath.Join("..", "..", "test", "private-key-jwt.pem"),
				SkipVerify:    	     true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 210,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 210,
				},
				Errors: nil,
			},
			check: func(t *testing.T) {
				f, err := os.OpenFile(filepath.Join(JwtCacheDir, "gopayloader-jwtstore-5b2b1d2712e9f97ef8f249571e178f56e7b56d56dfeb5eeed8a0cbeb364ef653.txt"), os.O_RDONLY, os.ModePerm)
				if err != nil {
					if os.IsNotExist(err) {
						t.Fatal(err)
					}
					t.Fatal(err)
				}
				stat, err := f.Stat()
				if err != nil {
					t.Fatal(err)
				}
				if stat.Size() == 0 {
					t.Fatalf("file size 0 for jwt cache store for %s \n", "gopayloader-jwtstore-5b2b1d2712e9f97ef8f249571e178f56e7b56d56dfeb5eeed8a0cbeb364ef653.txt")
				}
			},
		},
		{
			name: "GET RSA JWT",
			fields: fields{config: &config.Config{
				Ctx:                 context.Background(),
				ReqURI:              addr,
				ReqTarget:           10,
				Conns:               1,
				ReadTimeout:         5 * time.Second,
				WriteTimeout:        5 * time.Second,
				Method:              "GET",
				Client:              client,
				VerboseTicker:       time.Second,
				Headers:             []string{"content-type: application/json"},
				JwtHeader:           "some-jwt",
				JwtAud:              "some-aud",
				JwtSub:              "some-subject",
				JwtCustomClaimsJSON: "",
				JwtIss:              "some-issuer",
				JwtKID:              "13325575tevdfbdsfsf",
				JwtKey:              filepath.Join("..", "..", "test", "rsa.private"),
				SkipVerify:          true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 10,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 10,
				},
				Errors: nil,
			},
			check: func(t *testing.T) {
				f, err := os.OpenFile(filepath.Join(JwtCacheDir, "gopayloader-jwtstore-f356c646122f9103d45dc1db2c1dbb3b5c8da423dc38b76cd475875438de8cbf.txt"), os.O_RDONLY, os.ModePerm)
				if err != nil {
					if os.IsNotExist(err) {
						t.Fatal(err)
					}
					t.Fatal(err)
				}
				stat, err := f.Stat()
				if err != nil {
					t.Fatal(err)
				}
				if stat.Size() == 0 {
					t.Fatalf("file size 0 for jwt cache store for %s \n", "gopayloader-jwtstore-f356c646122f9103d45dc1db2c1dbb3b5c8da423dc38b76cd475875438de8cbf.txt")
				}
			},
		},
		{
			name: "Error hostname incorrect format - missing port",
			fields: fields{config: &config.Config{
				Ctx:                 context.Background(),
				ReqURI:              "http://localhost/",
				ReqTarget:           210,
				Conns:               101,
				ReadTimeout:         5 * time.Second,
				WriteTimeout:        5 * time.Second,
				Method:              "GET",
				Client:              client,
				VerboseTicker:       time.Second,
				Headers:             []string{"content-type: application/json"},
				JwtHeader:           "some-jwt",
				JwtAud:              "some-aud",
				JwtSub:              "some-subject",
				JwtCustomClaimsJSON: "{\"custom-claim1\": \"abc\", \"custom-claim2\": \"def\"}",
				JwtIss:              "some-issuer",
				JwtKID:              "13325575tevdfbdsfsf",
				JwtKey:              filepath.Join("..", "..", "test", "private-key-jwt.pem"),
				SkipVerify:          true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 0,
				FailedReqs:    0,
				Errors:        nil,
			},
			wantErr: errors.New("url not in correct format http://localhost/ needs to be like protocol://host:port/path i.e. https://localhost:443/some-path"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPayLoader(tt.fields.config)
			got, err := p.Run()
			if err != nil {
				if tt.wantErr == nil {
					t.Errorf("Run() error = %v, wanted no error", err)
					return
				}
				if tt.wantErr.Error() != err.Error() {
					t.Errorf("Run() error = %v, wanted error %v", err, tt.wantErr)
					return
				}
			}
			if err != nil {
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
