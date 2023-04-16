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
	Reqs             int64
	Conns            uint
	Duration         time.Duration
	MTLSKey          string
	MTLSCert         string
	SkipVerify       bool
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	Method           string
	Verbose          bool
}

func NewConfig(ctx context.Context, reqURI, mTLScert, mTLSKey string, disableKeepAlive bool, reqs int64, conns uint, totalTime time.Duration, skipVerify bool, readTimeout, writeTimeout time.Duration, method string, verbose bool) *Config {
	return &Config{
		Ctx:              ctx,
		ReqURI:           reqURI,
		MTLSKey:          mTLSKey,
		MTLSCert:         mTLScert,
		DisableKeepAlive: disableKeepAlive,
		Reqs:             reqs,
		Conns:            conns,
		Duration:         totalTime,
		SkipVerify:       skipVerify,
		ReadTimeout:      readTimeout,
		WriteTimeout:     writeTimeout,
		Method:           method,
		Verbose:          verbose,
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
	if int64(c.Conns) > c.Reqs && c.Duration == 0 {
		return errConnLimit
	}
	if int64(c.Conns) > c.Reqs && c.Reqs != 0 && c.Duration != 0 {
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

	if !methodAllowed(c.Method) {
		return fmt.Errorf("method %s not allowed", c.Method)
	}

	if c.WriteTimeout == 0 {
		return errors.New("write timeout is zero")
	}
	if c.ReadTimeout == 0 {
		return errors.New("read timeout is zero")
	}

	if c.Reqs == 0 && c.Duration == 0 {
		return errors.New("config: Reqs 0 and Duration 0")
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
