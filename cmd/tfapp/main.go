// Package main is the entry point for the tfapp application.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tfapp/internal/cli"
	apperrors "tfapp/internal/errors"
)

func main() {
	// Create a context that can be cancelled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Parse command-line flags
	flags := cli.ParseFlags()

	// Create and run the application
	app := cli.NewApp()
	if err := app.Run(ctx, flags); err != nil {
		apperrors.ExitWithError(err, 1)
	}
}
