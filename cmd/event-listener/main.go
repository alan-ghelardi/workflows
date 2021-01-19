package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nubank/workflows/pkg/listener"
	"github.com/nubank/workflows/pkg/log"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

func main() {
	ctx := signals.NewContext()

	logger, err := log.NewLogger("event-listener")
	if err != nil {
		fmt.Printf("Unable to start event-listener: %v", err)
		os.Exit(1)
	}

	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Fatalw("Error flushing logger", zap.Error(err))
		}
	}()

	ctx = logging.WithLogger(ctx, logger)
	eventListener := listener.New()
	routes := listener.Routes(eventListener)
	server := listener.NewServer(ctx, routes)
	go func() {
		logger.Info("Starting event listener API")

		if err := server.ListenAndServe(); err != nil {
			logger.Error(err)
		}
	}()

	<-ctx.Done()

	deadline, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	server.Shutdown(deadline)

	logger.Info("Shutting down the event listener API")
}
