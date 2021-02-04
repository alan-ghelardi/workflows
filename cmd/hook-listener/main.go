package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nubank/workflows/pkg/hooklistener"
	"github.com/nubank/workflows/pkg/log"
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
	eventHandler := hooklistener.NewEventHandler()
	routes := hooklistener.Routes(eventHandler)
	server := hooklistener.NewServer(ctx, routes)
	go func() {
		logger.Info("Starting hook listener API")

		if err := server.ListenAndServe(); err != nil {
			logger.Error(err)
		}
	}()

	<-ctx.Done()

	deadline, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	server.Shutdown(deadline)

	logger.Info("Shutting down the hook listener API")
}
