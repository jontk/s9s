package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jontk/s9s/internal/app"
	"github.com/jontk/s9s/internal/config"
)

const (
	// Version information
	version = "0.1.0"
	appName = "S9S - SLURM Terminal UI"
)

func main() {
	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("%s version %s\n", appName, version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create application instance
	s9sApp, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Create error channel for app errors
	errChan := make(chan error, 1)

	// Run application in goroutine
	go func() {
		if err := s9sApp.Run(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v. Starting graceful shutdown...", sig)
		
		// Create shutdown context with timeout
		_, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		// Gracefully stop the application
		if err := s9sApp.Stop(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}

		// Cancel the main context
		cancel()

	case err := <-errChan:
		log.Printf("Application error: %v", err)
		cancel()
		os.Exit(1)
	}

	log.Println("S9S shutdown complete")
}