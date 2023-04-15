package wrapper

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/domsolutions/gopayloader/config"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
)

func RunGoPayLoader(reqURI, mTLScert, mTLSKey string, keepAlive bool, reqs int64, conns uint, totalTime time.Duration) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configuration := config.NewConfig(ctx, reqURI, mTLScert, mTLSKey, keepAlive, reqs, conns, totalTime)
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
		fmt.Println(results)
	case err := <-errPayLoader:
		return err
	case results := <-resPayLoader:
		fmt.Println(results)
	}

	return nil
}
