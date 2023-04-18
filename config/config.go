package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"
)

type Config struct {
	Ctx              context.Context
	ReqURI           string
	DisableKeepAlive bool
	ReqTarget        int64
	Conns            uint
	Duration         time.Duration
	MTLSKey          string
	MTLSCert         string
	SkipVerify       bool
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	Method           string
	Verbose          bool
	Ticker           time.Duration
	HTTPV2           bool
	JwtCert          string
	JwtKey           string
	JwtSub           string
	JwtIss           string
	JwtAud           string
	JwtHeader        string
	SendJWT          bool
}

func NewConfig(ctx context.Context, reqURI, mTLScert, mTLSKey string, disableKeepAlive bool, reqs int64, conns uint, totalTime time.Duration, skipVerify bool, readTimeout, writeTimeout time.Duration, method string, verbose bool, ticker time.Duration, HTTPV2 bool, jwtCert, jwtKey, jwtSub, jwtIss, jwtAud, jwtHeader string, sendJWT bool) *Config {
	return &Config{
		Ctx:              ctx,
		ReqURI:           reqURI,
		MTLSKey:          mTLSKey,
		MTLSCert:         mTLScert,
		DisableKeepAlive: disableKeepAlive,
		ReqTarget:        reqs,
		Conns:            conns,
		Duration:         totalTime,
		SkipVerify:       skipVerify,
		ReadTimeout:      readTimeout,
		WriteTimeout:     writeTimeout,
		Method:           method,
		Verbose:          verbose,
		Ticker:           ticker,
		HTTPV2:           HTTPV2,
		JwtCert:          jwtCert,
		JwtKey:           jwtKey,
		JwtSub:           jwtSub,
		JwtIss:           jwtIss,
		JwtAud:           jwtAud,
		JwtHeader:        jwtHeader,
		SendJWT:          sendJWT,
	}
}

var (
	errConnLimit = errors.New("connections can't be more than requests")
)

var allowedMethods = [4]string{
	"GET",
	"PUT",
	"POST",
	"DELETE",
}

func (c *Config) Validate() error {
	if _, err := url.ParseRequestURI(c.ReqURI); err != nil {
		return fmt.Errorf("config: invalid request uri, got error %v", err)
	}
	if int64(c.Conns) > c.ReqTarget && c.Duration == 0 {
		return errConnLimit
	}
	if int64(c.Conns) > c.ReqTarget && c.ReqTarget != 0 && c.Duration != 0 {
		return errConnLimit
	}

	if c.MTLSKey != "" {
		_, err := os.OpenFile(c.MTLSKey, os.O_RDONLY, os.ModePerm)
		if err != nil {
			if os.IsNotExist(err) {
				return errors.New("config: mTLS private key does not exist")
			}
			return fmt.Errorf("config: mTLS private key error checking file exists; %v", err)
		}
	}
	if c.MTLSCert != "" {
		_, err := os.OpenFile(c.MTLSCert, os.O_RDONLY, os.ModePerm)
		if err != nil {
			if os.IsNotExist(err) {
				return errors.New("config: mTLS cert does not exist")
			}
			return fmt.Errorf("config: mTLS cert error checking file exists; %v", err)
		}
	}

	if c.SendJWT {
		if c.JwtHeader == "" {
			return errors.New("config: empty jwt header")
		}

		if c.JwtKey != "" {
			_, err := os.OpenFile(c.JwtKey, os.O_RDONLY, os.ModePerm)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.New("config: jwt key does not exist")
				}
				return fmt.Errorf("config: jwt key error checking file exists; %v", err)
			}
		}

		if c.JwtCert != "" {
			_, err := os.OpenFile(c.JwtCert, os.O_RDONLY, os.ModePerm)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.New("config: jwt cert does not exist")
				}
				return fmt.Errorf("config: jwt cert error checking file exists; %v", err)
			}
		}
	}

	if c.Ticker == 0 {
		return errors.New("ticker value can't be zero")
	}

	if !methodAllowed(c.Method) {
		return fmt.Errorf("method %s not allowed", c.Method)
	}

	if c.WriteTimeout == 0 {
		return errors.New("write timeout is zero")
	}
	if c.ReadTimeout == 0 {
		return errors.New("read timeout is zero")
	}

	if c.ReqTarget == 0 && c.Duration == 0 {
		return errors.New("config: ReqTarget 0 and Duration 0")
	}
	return nil
}

func methodAllowed(method string) bool {
	for _, m := range allowedMethods {
		if method == m {
			return true
		}
	}
	return false
}
