package fasthttp

import (
	"crypto/tls"
	"github.com/domsolutions/gopayloader/pkgs/http-clients"
	"github.com/valyala/fasthttp"
	"net"
	"net/url"
)

type Client struct {
	client *fasthttp.HostClient
	http2  bool
}

type Req struct {
	req *fasthttp.Request
}

type Resp struct {
	resp *fasthttp.Response
}

func (r *Resp) StatusCode() int {
	return r.resp.StatusCode()
}

func (r *Resp) Size() int64 {
	var size = int64(len(r.resp.Body()))
	size += int64(len(r.resp.Header.Header()))
	return size
}

func (r *Resp) Close() {
	r.resp.CloseBodyStream()
}

func (fh *Req) SetHeader(key, val string) {
	fh.req.Header.Set(key, val)
}

func (fh *Req) Size() int64 {
	size := len(fh.req.Body()) + 2 // 2 for the \r\n that separates the headers and body.
	fh.req.Header.VisitAll(func(key, value []byte) {
		size += len(key) + len(value) + 2 // 2 for the \r\n that separates the headers.
	})
	return int64(size)
}

func (fh *Req) SetMethod(method string) {
	fh.req.Header.SetMethodBytes([]byte(method))
}

func (fh *Req) SetBody(body []byte) {
	fh.req.SetBody(body)
}

func (fh *Client) Do(req http_clients.Request, resp http_clients.Response) error {
	return fh.client.Do(req.(*Req).req, resp.(*Resp).resp)
}

func (c *Client) HTTP2() bool {
	return c.http2
}

func (c *Client) CloseConns() {
	c.client.CloseIdleConnections()
}

func (fh *Client) NewResponse() http_clients.Response {
	// TODO: buffer pool
	return &Resp{resp: &fasthttp.Response{}}
}

func (fh *Client) NewReq(method, url string) (http_clients.Request, error) {
	// TODO: buffer pool
	r := &fasthttp.Request{}
	r.SetRequestURI(url)
	r.Header.SetMethodBytes([]byte(method))
	return &Req{
		req: r,
	}, nil
}

func GetFastHTTPClient1(config *http_clients.Config) (http_clients.GoPayLoaderClient, error) {
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
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, config.ReadTimeout)
		},
	}

	return &Client{client: client, http2: false}, nil
}
