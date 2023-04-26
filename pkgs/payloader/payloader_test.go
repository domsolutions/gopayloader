package payloader

import (
	"context"
	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader/worker"
	"github.com/valyala/fasthttp"
	"log"
	"reflect"
	"testing"
	"time"
)

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

func TestPayLoader_Run(t *testing.T) {
	go testStartHTTP1Server("localhost:8888")
	// give time for server to spin up
	time.Sleep(1 * time.Second)

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
			name: "fasthttp-1 10 connections for 21 requests",
			fields: fields{config: &config.Config{
				Ctx:          context.Background(),
				ReqURI:       "http://localhost:8888",
				ReqTarget:    21,
				Conns:        10,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
				Method:       "GET",
				Client:       "fasthttp-1",
				Ticker:       time.Second,
			}},
			want: &GoPayloaderResults{
				CompletedReqs: 21,
				FailedReqs:    0,
				Responses: map[worker.ResponseCode]int64{
					200: 21,
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
