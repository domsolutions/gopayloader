package wrapper

import (
	"context"
	"github.com/domsolutions/gopayloader/pkgs/payloader/output/cli"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
)

func RunGoPayLoader(reqURI, mTLScert, mTLSKey string, disableKeepAlive bool, reqs int64, conns uint, totalTime time.Duration, skipVerify bool, readTimeout, writeTimeout time.Duration, method string, verbose bool, ticker time.Duration) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configuration := config.NewConfig(ctx, reqURI, mTLScert, mTLSKey, disableKeepAlive, reqs, conns, totalTime, skipVerify, readTimeout, writeTimeout, method, verbose, ticker)
	if err := configuration.Validate(); err != nil {
		return err
	}

	payload := payloader.NewPayLoader(configuration)
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
