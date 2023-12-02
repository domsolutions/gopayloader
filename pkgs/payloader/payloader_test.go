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
	golanghttp2 "golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var (
	testServerHTTP3 httpv3server.Server
	testFastHTTP    fasthttp.Server
	crtPath         string
	keyPath         string
)

func init() {
	crtPath = filepath.Join("..", "..", "test", "server.crt")
	keyPath = filepath.Join("..", "..", "test", "server.key")

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
	crt, err := os.ReadFile(crtPath)
	if err != nil {
		log.Fatal(err)
	}

	key, err := os.ReadFile(keyPath)
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
	testFastHTTP = fasthttp.Server{
		ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
			log.Printf("Got error from req %+v", ctx)
			log.Println(err)
		},
		ConnState: func(conn net.Conn, state fasthttp.ConnState) {
			switch state {
			case fasthttp.StateNew:
				log.Printf("New conn from %s \n", conn.RemoteAddr().String())
			case fasthttp.StateClosed:
				log.Printf("Closed conn from %s \n", conn.RemoteAddr().String())
			case fasthttp.StateIdle:
				log.Printf("Idle conn from %s \n", conn.RemoteAddr().String())
			}
		},
		Handler: func(c *fasthttp.RequestCtx) {
			_, err = c.WriteString("hello")
			if err != nil {
				log.Println(err)
			}
		},
	}

	if err := testFastHTTP.ListenAndServe(addr); err != nil {
		log.Println(err)
	}
}

func testStartHTTP3Server(addr string) {
	var err error
	testServerHTTP3 = httpv3server.Server{
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

	if err := testServerHTTP3.ListenAndServe(); err != nil {
		log.Println(err)
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

	err = golanghttp2.ConfigureServer(server, &golanghttp2.Server{})
	if err != nil {
		panic(err)
	}

	if err := server.ListenAndServeTLS(crtPath, keyPath); err != nil {
		log.Println(err)
	}
}

func TestPayLoader_RunFastHTTP1NonSSL(t *testing.T) {
	testPayLoader_Run(t, "http://localhost:8888", "fasthttp", nil)
}

func TestPayLoader_RunFastHTTP1SSL(t *testing.T) {
	testPayLoader_Run(t, "https://localhost:8889", "fasthttp", nil)
}

func TestPayLoader_RunNetHTT21SSL(t *testing.T) {
	testPayLoader_Run(t, "https://localhost:8889", "nethttp2", nil)
}

func TestPayLoader_RunNetHTTP1SSL(t *testing.T) {
	testPayLoader_Run(t, "https://localhost:8889", "nethttp", nil)
}

func TestPayLoader_RunNetHTTP3(t *testing.T) {
	testPayLoader_Run(t, "https://localhost:8890", "nethttp3", func() {
		testServerHTTP3.Close()
	})
}

func testPayLoader_Run(t *testing.T, addr, client string, cleanup func()) {
	type fields struct {
		config *config.Config
	}

	type tcase struct {
		name    string
		fields  fields
		want    *GoPayloaderResults
		wantErr error
		check   func(t *testing.T)
	}

	tests := []tcase{
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
			name: "PUT 10 connections for 5 second long test with 100 requests",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				Conns:         10,
				ReqTarget:     100,
				Duration:      5 * time.Second,
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
				f, err := os.OpenFile(filepath.Join(JwtCacheDir, "gopayloader-jwtstore-b7d91ad840dd089d10e2c3bbad56b43f0c558f4ec93a81b05b9f1fa9c8d4ad6a.txt"), os.O_RDONLY, os.ModePerm)
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
					t.Fatalf("file size 0 for jwt cache store for %s \n", "gopayloader-jwtstore-b7d91ad840dd089d10e2c3bbad56b43f0c558f4ec93a81b05b9f1fa9c8d4ad6a.txt")
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
				JwtIss:              "some-issuer",
				JwtKID:              "13325575tevdfbdsfsf",
				JwtKey:              filepath.Join("..", "..", "test", "private-key-jwt.pem"),
				SkipVerify:          true,
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
				f, err := os.OpenFile(filepath.Join(JwtCacheDir, "gopayloader-jwtstore-4f12b598aa74a10a1a94931f0f93ef9f7afb43e138060ed6ce7f5c9906447c1f.txt"), os.O_RDONLY, os.ModePerm)
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
					t.Fatalf("file size 0 for jwt cache store for %s \n", "gopayloader-jwtstore-4f12b598aa74a10a1a94931f0f93ef9f7afb43e138060ed6ce7f5c9906447c1f.txt")
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
				f, err := os.OpenFile(filepath.Join(JwtCacheDir, "gopayloader-jwtstore-52496875054792ed64a436091bb4734fef6f159c9f4db038e843cbf8c7fa717b.txt"), os.O_RDONLY, os.ModePerm)
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
					t.Fatalf("file size 0 for jwt cache store for %s \n", "gopayloader-jwtstore-52496875054792ed64a436091bb4734fef6f159c9f4db038e843cbf8c7fa717b.txt")
				}
			},
		},
		{
			name: "GET using JWT file only",
			fields: fields{config: &config.Config{
				Ctx:           context.Background(),
				ReqURI:        addr,
				ReqTarget:     10,
				Conns:         1,
				ReadTimeout:   5 * time.Second,
				WriteTimeout:  5 * time.Second,
				Method:        "GET",
				Client:        client,
				VerboseTicker: time.Second,
				Headers:       []string{"content-type: application/json"},
				JwtHeader:     "Authorization",
				JwtsFilename:  filepath.Join("..", "..", "test", "jwtstestfile.txt"),
				SkipVerify:    true,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 10,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 10,
				},
				Errors: nil,
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

	if client == "nethttp2" || client == "nethttp3" {
		tests = append(tests, tcase{
			name: "PARALLEL - GET 10 connections for 210 requests",
			fields: fields{config: &config.Config{
				Parallel:      true,
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
		})
	}

	if cleanup != nil {
		t.Cleanup(cleanup)
	}

	for _, tt := range tests {
		tt := tt
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
