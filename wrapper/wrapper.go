package wrapper

import (
	"context"
	"github.com/domsolutions/gopayloader/pkgs/payloader/output/cli"
	"github.com/pterm/pterm"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
)

func RunGoPayLoader(reqURI, mTLScert, mTLSKey string, disableKeepAlive bool, reqs int64, conns uint, totalTime time.Duration, skipVerify bool, readTimeout, writeTimeout time.Duration, method string, verbose bool, ticker time.Duration, HTTPV2 bool, jwtCert, jwtKey, jwtSub, jwtIss, jwtAud, jwtHeader string, sendJWT bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := config.NewConfig(ctx, reqURI, mTLScert, mTLSKey, disableKeepAlive, reqs, conns, totalTime, skipVerify, readTimeout, writeTimeout, method, verbose, ticker, HTTPV2, jwtCert, jwtKey, jwtSub, jwtIss, jwtAud, jwtHeader, sendJWT)
	if err := conf.Validate(); err != nil {
		return err
	}

	if verbose {
		pterm.EnableDebugMessages()
		pterm.Warning.Println("In verbose mode RPS will be slightly lower, especially for long running tests")
	}

	payload := payloader.NewPayLoader(conf)
	errPayLoader := make(chan error)
	resPayLoader := make(chan *payloader.Results)

	go func() {
		results, err := payload.Run()
		if err != nil {
			errPayLoader <- err
			return
		}
		resPayLoader <- results
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case <-c:
		// user pressed ctrl+c
		cancel()
		results := <-resPayLoader
		if err := cli.Display(results); err != nil {
			return err
		}
	case err := <-errPayLoader:
		return err
	case results := <-resPayLoader:
		if err := cli.Display(results); err != nil {
			return err
		}
	}

	return nil
}
