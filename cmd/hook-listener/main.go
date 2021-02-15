package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nubank/workflows/pkg/hooklistener"
	"github.com/nubank/workflows/pkg/log"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

func main() {
	ctx := signals.NewContext()

	logger, err := log.NewLogger("hook-listener")
	if err != nil {
		fmt.Printf("Unable to start hook-listener: %v", err)
		os.Exit(1)
	}

	defer logger.Sync()

	ctx = logging.WithLogger(ctx, logger)
	server := hooklistener.New(ctx)
	go func() {
		logger.Info("Starting hook-listener")

		if err := server.ListenAndServe(); err != nil {
			logger.Error("Error starting hook-listener", zap.Error(err))
		}
	}()

	<-ctx.Done()

	deadline, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	server.Shutdown(deadline)

	logger.Info("Shutting down the hook-listener API")
}
