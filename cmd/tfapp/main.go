// Package main is the entry point for the tfapp application.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tfapp/internal/cli"
	"tfapp/internal/config"
	apperrors "tfapp/internal/errors"
	"tfapp/internal/ui"
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
		fmt.Printf("%s\nReceived interrupt signal, shutting down...%s\n", ui.ColorInfo, ui.ColorReset)
		cancel()
	}()

	// Load configuration
	cfg, configCreated, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sWarning: Error loading configuration: %s%s\n", ui.ColorWarning, err, ui.ColorReset)
		fmt.Fprintf(os.Stderr, "%sContinuing with default settings...%s\n", ui.ColorWarning, ui.ColorReset)
		cfg = config.DefaultConfig()
	}

	// Display a message if the config was created
	if configCreated {
		configPath, _ := config.ConfigFilePath()
		fmt.Printf("%sCreated default configuration file at %s%s\n", ui.ColorInfo, configPath, ui.ColorReset)
		fmt.Printf("%sYou can edit this file to customize the application colors.%s\n", ui.ColorInfo, ui.ColorReset)
		fmt.Println()
	}

	// Initialize UI colors from configuration
	ui.InitColors(cfg)

	// Parse command-line flags
	flags := cli.ParseFlags()

	// Create and run the application
	app := cli.NewApp()
	if err := app.Run(ctx, flags); err != nil {
		apperrors.ExitWithError(err, 1)
	}
}
