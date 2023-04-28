# Gopayloader

[![Build status](https://github.com/domsolutions/gopayloader/actions/workflows/go.yml/badge.svg)](https://github.com/domsolutions/gopayloader/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/domsolutions/gopayloader)](https://goreportcard.com/report/github.com/domsolutions/gopayloader)
[![GoDoc](https://godoc.org/github.com/domsolutions/gopayloader?status.svg)](http://godoc.org/github.com/domsolutions/gopayloader)

Gopayloader is an HTTP/S benchmarking tool. Inspired by [bombardier](https://github.com/codesenberg/bombardier/) it also uses [fasthttp](https://github.com/valyala/fasthttp) which allows for fast creation and sending of requests due to low allocations and lots of other improvements. But with 
added improvement of also supporting fashttp for HTTP/2.
It uses this client by default, a different client can be used with `--client` flag.

Supports all HTTP versions, using [quic-go](https://github.com/quic-go/quic-go) for HTTP/3 client with `--client nethttp-3`. For HTTP/2 can use fasthttp with `--client fasthttp-2` or standard core golang `net/http` with `--client nethttp`

Supports ability to generate custom JWTs to send in headers with payload (only limited by HDD size). This can be useful if the service being
tested is JWT authenticated. Each JWT generated will be unique as contains a unique `jti` in claims i.e.

```json
{
  "jti": "8f2d1472-084c-4662-ae74-04e0f1de4993"
}
```

A private key is supplied as a flag with optional flags to set other claims i.e. `sub` `aud`, `iss`. It will then check if the required number of jwts has already
been generated in a previous test by checking a file on disk. All JWTs are saved on disk in cache, this allows
huge number of jwts to be generated without affecting in-memory use of gopayloader. Once all jwts have been generated
the tests begin, and jwts are streamed from disk to requests. This keeps the memory footprint low. The other major benefit to pre-generating
is all of CPU cycles can be dedicated to sending the requests thus achieving higher RPS.


## Installation

Can install with (supported go versions >= 1.19)

```shell
go install github.com/domsolutions/gopayloader@latest 
```

Or download pre-compiled binaries from [releases](https://github.com/domsolutions/gopayloader/releases)

## Usage

To list all available flags run;

```shell
./gopayloader run --help

Load test HTTP/S server - supports HTTP/1.1 HTTP/2 HTTP/3

Usage:
  gopayloader run <host>(host format - protocol://host:port/path i.e. https://localhost:443/some-path) [flags]

Flags:
  -b, --body string              request body
      --body-file string         read request body from file
      --client string            fasthttp-1 for fast http/1.1 requests
                                 fasthttp-2 for fast http/2 requests 
                                 nethttp for standard net/http requests supporting http/1.1 http/2
                                 nethttp-3 for standard net/http requests supporting http/3 using quic-go (default "fasthttp-1")
  -c, --connections uint         Number of simultaneous connections (default 1)
  -k, --disable-keep-alive       Disable keep-alive connections
  -H, --headers strings          headers to send in request, can have multiple i.e -H 'content-type:application/json' -H' connection:close'
  -h, --help                     help for run
      --jwt-aud string           JWT audience (aud) claim
      --jwt-header string        JWT header field name
      --jwt-iss string           JWT issuer (iss) claim
      --jwt-key string           JWT signing private key path
      --jwt-kid string           JWT KID
      --jwt-sub string           JWT subject (sub) claim
  -m, --method string            request method (default "GET")
      --mtls-cert string         mTLS cert path
      --mtls-key string          mTLS cert private key path
      --read-timeout duration    Read timeout (default 5s)
  -r, --requests int             Number of requests
      --skip-verify              Skip verify SSL cert signer
      --ticker duration          How often to print results while running in verbose mode (default 1s)
  -t, --time duration            Execution time window, if used with -r will uniformly distribute reqs within time window, without -r reqs are unlimited
  -v, --verbose                  verbose - slows down RPS slightly for long running tests
      --write-timeout duration   Write timeout (default 5s)

```

By default, it runs in quiet mode to dedicate all CPU cycles to sending requests to achieve max RPS. Verbose
mode can be enabled with `-v` flag.

To run `1000000` requests across `150` connections;

```shell
./gopayloader run http://localhost:8081 -c 150 -r 1000000 

Gopayloader v0.1.0 HTTP/JWT authentication benchmark tool 
https://github.com/domsolutions/gopayloader

 INFO  Running 1,000,000 request/s with 150 connection/s against http://localhost:8081
 SUCCESS  Payload complete, calculating results
 SUCCESS  Gopayloader results 

+-----------------------+-------------------------------+
| METRIC                | RESULT                        |
+-----------------------+-------------------------------+
| Total time            | 18.983561852s                 |
| Start time            | Thu, 27 Apr 2023 12:04:47 BST |
| End time              | Thu, 27 Apr 2023 12:05:06 BST |
| Completed requests    | 1000000                       |
| Failed requests       | 0                             |
+-----------------------+-------------------------------+
| Average RPS           | 52677.153                     |
| Max RPS               | 54411                         |
| Min RPS               | 47096                         |
+-----------------------+-------------------------------+
| Req size (bytes)      | 42                            |
| Req size/second (MB)  | 2.225                         |
| Req total size (MB)   | 40.054                        |
+-----------------------+-------------------------------+
| Resp size (bytes)     | 135                           |
| Resp size/second (MB) | 7.153                         |
| Resp total size (MB)  | 128.746                       |
+-----------------------+-------------------------------+
| Average latency       | 2.816219ms                    |
| Max latency           | 62.938092ms                   |
| Min latency           | 74.879µs                      |
+-----------------------+-------------------------------+
| Response code; 200    | 1000000                       |
+-----------------------+-------------------------------+
```

To run `1000000` requests across `150` connections with jwts (jwts will only be generated when number of requests are specified);

Example header jwt generated;

```json
{
  "alg": "ES256",
  "kid": "3434645743124",
  "typ": "JWT"
}
```

example body;

```json
{
  "aud": "some-audience",
  "exp": 1714130039,
  "iss": "some-issuer",
  "jti": "05181473-bbd6-4d21-8942-d86c2e972b2b",
  "sub": "my-subject"
}
```

Note `jti` will be different for each jwt.

Will set jwt value = header field `my-jwt` and sign with key `./private-key.pem` and KID `3434645743124`

`./gopayloader run http://localhost:8081 -c 150 -r 1000000 --jwt-header "my-jwt" --jwt-key ./private-key.pem --jwt-kid 3434645743124 --jwt-sub "my-subject" --jwt-aud "some-audience" --jwt-iss "some-issuer"`

```shell
Gopayloader v0.1.0 HTTP/JWT authentication benchmark tool 
https://github.com/domsolutions/gopayloader

 INFO  Sending jwts with requests, checking for jwts in cache
 INFO  Generating batch of 1000000 JWTs and saving to disk
 INFO  Running 1,000,000 request/s with 150 connection/s against http://localhost:8081
 SUCCESS  Payload complete, calculating results
 SUCCESS  Gopayloader results 

+-----------------------+-------------------------------+
| METRIC                | RESULT                        |
+-----------------------+-------------------------------+
| Total time            | 19.679817482s                 |
| Start time            | Thu, 27 Apr 2023 12:14:48 BST |
| End time              | Thu, 27 Apr 2023 12:15:08 BST |
| Completed requests    | 1000000                       |
| Failed requests       | 0                             |
+-----------------------+-------------------------------+
| Average RPS           | 50813.479                     |
| Max RPS               | 52708                         |
| Min RPS               | 48098                         |
+-----------------------+-------------------------------+
| Req size (bytes)      | 370                           |
| Req size/second (MB)  | 18.572                        |
| Req total size (MB)   | 352.859                       |
+-----------------------+-------------------------------+
| Resp size (bytes)     | 135                           |
| Resp size/second (MB) | 6.776                         |
| Resp total size (MB)  | 128.746                       |
+-----------------------+-------------------------------+
| Average latency       | 2.92205ms                     |
| Max latency           | 78.047387ms                   |
| Min latency           | 80.768µs                      |
+-----------------------+-------------------------------+
| Response code; 200    | 1000000                       |
+-----------------------+-------------------------------+
```

To remove all generated jwts;

```shell
./gopayloader clear-cache 
```

## Benchmark comparisons

All tests are running against below HTTP/1.1 server;

```shell
./gopayloader http-server -p 8081 -s 1 --fasthttp-1
```

Tested running for `30 seconds` reqs over `125` connections

Gopayloader tested with 
```shell
./gopayloader run http://localhost:8081 -c 125 --time 30s 
```

achieved mean RPS of `53098` 

| Tool                                                     | Cmd                                                      | Mean RPS | Gopayloader improvement               |
|----------------------------------------------------------|----------------------------------------------------------|----------|---------------------------------------|
| [k6](https://github.com/grafana/k6)                      | `k6 run --vus 125 --duration 30s k6.js`                  | 15268    | <span style="color:green">235%</span> |
| [bombardier](https://github.com/codesenberg/bombardier/) | `bombardier http://localhost:8081 -c 125 --duration=30s` | 51311    | <span style="color:green">3.4%</span> |
| [hey](https://github.com/rakyll/hey)                     | `hey -z 30s -c 125 http://localhost:8081`                | 22644    | <span style="color:green">134%</span> |
