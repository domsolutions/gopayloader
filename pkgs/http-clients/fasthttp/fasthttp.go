package fasthttp

import (
	"crypto/tls"
	"github.com/dgrr/http2"
	"github.com/domsolutions/gopayloader/pkgs/http-clients"
	"github.com/valyala/fasthttp"
	"net/url"
)

type Client struct {
	client *fasthttp.HostClient
}

type Req struct {
	req *fasthttp.Request
}

func (fh *Req) SetHeader(key, val string) {
	fh.req.Header.Set(key, val)
}

func (fh *Req) SetMethod(method string) {
	fh.req.Header.SetMethodBytes([]byte(method))
}

func (fh *Req) SetBody(body []byte) {
	fh.req.SetBody(body)
}

func (fh *Req) SetRequestURI(uri string) error {
	fh.req.SetRequestURI(uri)
	return nil
}

func (fh *Client) Do(req http_clients.Request, resp http_clients.Response) error {
	return fh.client.Do(req.(*Req).req, resp.(*fasthttp.Response))
}

func (fh *Client) NewResponse() http_clients.Response {
	return &fasthttp.Response{}
}

func (fh *Client) NewReq() http_clients.Request {
	return &Req{
		req: &fasthttp.Request{},
	}
}

func GetFastHTTPClient(config *http_clients.Config) (http_clients.GoPayLoaderClient, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.SkipVerify,
	}

	if config.MTLSCert != "" && config.MTLSKey != "" {
		cert, err := tls.LoadX509KeyPair(config.MTLSCert, config.MTLSKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	u, err := url.ParseRequestURI(config.ReqURI)
	if err != nil {
		return nil, err
	}

	client := &fasthttp.HostClient{
		Addr:                          u.Host,
		IsTLS:                         u.Scheme == "https",
		MaxConns:                      1,
		ReadTimeout:                   config.ReadTimeout,
		WriteTimeout:                  config.WriteTimeout,
		DisableHeaderNamesNormalizing: true,
		TLSConfig:                     tlsConfig,
	}

	if !config.HTTPV2 {
		return &Client{client: client}, nil
	}

	// TODO can't ctrl+c when http2 client can't connect to server which is down, just hangs
	// TODO look into how to send reqs i.e. pipelining... does it actually speed stuff up? in use by 40% so should support

	if err := http2.ConfigureClient(client, http2.ClientOpts{
		MaxResponseTime: config.ReadTimeout,
	}); err != nil {
		return nil, err
	}

	return &Client{client: client}, nil
}
