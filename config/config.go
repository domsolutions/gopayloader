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
	Ctx       context.Context
	ReqURI    string
	KeepAlive bool
	Reqs      int64
	Conns     uint
	Duration  time.Duration
	MTLSKey   string
	MTLSCert  string
}

func NewConfig(ctx context.Context, reqURI, mTLScert, mTLSKey string, keepAlive bool, reqs int64, conns uint, totalTime time.Duration) *Config {
	return &Config{
		Ctx:       ctx,
		ReqURI:    reqURI,
		MTLSKey:   mTLSKey,
		MTLSCert:  mTLScert,
		KeepAlive: keepAlive,
		Reqs:      reqs,
		Conns:     conns,
		Duration:  totalTime,
	}
}

var (
	errConnLimit = errors.New("connections can't be more than requests")
)

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

	if c.Reqs == 0 && c.Duration == 0 {
		return errors.New("config: Reqs 0 and Duration 0")
	}
	return nil
}
