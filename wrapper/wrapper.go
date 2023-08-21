package wrapper

import (
	"context"
	"errors"
	"github.com/domsolutions/gopayloader/pkgs/payloader/output/cli"
	"github.com/domsolutions/gopayloader/version"
	"github.com/pterm/pterm"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
)

func RunGoPayLoader(reqURI, mTLScert, mTLSKey string, disableKeepAlive bool, reqs int64, conns uint, totalTime time.Duration, skipVerify bool, readTimeout, writeTimeout time.Duration, method string, verbose bool, ticker time.Duration, jwtKID, jwtKey, jwtSub, jwtCustomClaimsJSON, jwtIss, jwtAud, jwtHeader, jwtsFilename string, headers []string, body, bodyFile string, client string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := config.NewConfig(ctx,
		reqURI,
		mTLScert,
		mTLSKey,
		disableKeepAlive,
		reqs,
		conns,
		totalTime,
		skipVerify,
		readTimeout,
		writeTimeout,
		method,
		verbose,
		ticker,
		jwtKID, jwtKey, jwtSub, jwtCustomClaimsJSON, jwtIss, jwtAud, jwtHeader, jwtsFilename, headers, body, bodyFile, client)
	if err := conf.Validate(); err != nil {
		return err
	}

	pterm.DefaultBasicText.Printf(pterm.LightYellow("Gopayloader v%s HTTP/JWT authentication benchmark tool \n"), version.Version)
	pterm.DefaultBasicText.Println("https://github.com/domsolutions/gopayloader")

	if verbose {
		pterm.EnableDebugMessages()
		pterm.Warning.Println("In verbose mode RPS will be slightly lower due to monitoring, more noticeable in longer running tests")
	}

	payload := payloader.NewPayLoader(conf)
	errPayLoader := make(chan error)
	resPayLoader := make(chan *payloader.GoPayloaderResults)

	go func() {
		results, err := payload.Run()
		if err != nil {
			errPayLoader <- err
			return
		}
		resPayLoader <- results
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case <-c:
		// user pressed ctrl+c
		cancel()
		timeout := 5 * time.Second
		deadline := time.Now().Add(timeout)

		pterm.Info.Printf("User aborted; waiting %s for results before exiting \n", timeout)

		ctx, c := context.WithDeadline(context.Background(), deadline)
		defer c()

		select {
		case results := <-resPayLoader:
			cli.Display(results)
		case err := <-errPayLoader:
			// user may have cancelled during jwt generation, so there will be no results
			return err
		case <-ctx.Done():
			return errors.New("timeout exceeded, failed to get payload results")
		}
	case err := <-errPayLoader:
		return err
	case results := <-resPayLoader:
		cli.Display(results)
	}

	return nil
}
