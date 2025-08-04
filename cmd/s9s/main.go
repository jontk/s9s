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

	// Handle help flag
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Printf("%s\n\nUsage: %s [options]\n\n", appName, os.Args[0])
		fmt.Println("Options:")
		fmt.Println("  --version, -v    Show version information")
		fmt.Println("  --help, -h       Show this help message")
		fmt.Println("  --mock           Use mock SLURM client for testing")
		fmt.Println("  --no-mock        Use real SLURM client")
		fmt.Println("\nConfiguration:")
		fmt.Println("  Configuration file: ~/.s9s/config.yaml")
		fmt.Println("  See config.example.yaml for configuration options")
		fmt.Println("\nKeyboard shortcuts:")
		fmt.Println("  F1               Show help")
		fmt.Println("  F10              Configuration settings")
		fmt.Println("  Tab              Switch between views")
		fmt.Println("  1-9              Switch to specific views")
		fmt.Println("  :                Command mode")
		fmt.Println("  q                Quit")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Process command line arguments
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--mock":
			cfg.UseMockClient = true
		case "--no-mock":
			cfg.UseMockClient = false
		}
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
		err := s9sApp.Run()
		// Send error to channel (nil means normal shutdown)
		errChan <- err
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
		if err != nil {
			log.Printf("Application error: %v", err)
			cancel()
			os.Exit(1)
		}
		// Normal shutdown (err == nil)
		cancel()
	}

	log.Println("S9S shutdown complete")
}
