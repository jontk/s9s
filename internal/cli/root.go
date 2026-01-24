package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jontk/s9s/internal/app"
	"github.com/jontk/s9s/internal/auth"
	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/discovery"
	"github.com/jontk/s9s/internal/logging"
	"github.com/jontk/s9s/internal/mock"
	"github.com/jontk/s9s/internal/version"
	"github.com/spf13/cobra"
)

const (
	appName = "S9S - SLURM Terminal UI"
)

var (
	cfgFile          string
	useMock          bool
	noMock           bool
	debugMode        bool
	showVersion      bool
	noDiscovery      bool
	discoveryTimeout string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "s9s",
	Short: "Terminal UI for SLURM cluster management",
	Long: `s9s provides a terminal interface for managing SLURM clusters, inspired by k9s.

It allows HPC administrators and users to monitor and manage jobs, nodes, 
and cluster resources efficiently from the terminal.

Features:
• Real-time job and node monitoring
• Interactive job management (submit, cancel, hold, release)
• SSH integration for direct node access
• Export capabilities (CSV, JSON, Markdown)
• Plugin system for extensibility
• Vim-like navigation`,

	Example: `  s9s                         # Launch interactive TUI
  S9S_ENABLE_MOCK=dev s9s --mock  # Use mock SLURM for testing
  s9s setup                       # Run configuration wizard
  s9s setup --auto-discover      # Auto-discover clusters`,

	RunE: runRoot,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.s9s/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug logging")

	// Local flags
	rootCmd.Flags().BoolVar(&useMock, "mock", false, "use mock SLURM client (requires S9S_ENABLE_MOCK)")
	rootCmd.Flags().BoolVar(&noMock, "no-mock", false, "use real SLURM client (override config)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version information")
	rootCmd.Flags().BoolVar(&noDiscovery, "no-discovery", false, "disable auto-discovery of slurmrestd endpoint and token")
	rootCmd.Flags().StringVar(&discoveryTimeout, "discovery-timeout", "", "timeout for auto-discovery (e.g., 10s, 30s)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if debugMode {
		logging.Info("Debug mode enabled")
	}
}

// runRoot executes the main s9s TUI application
func runRoot(cmd *cobra.Command, args []string) error {
	// Handle version flag
	if showVersion {
		info := version.Get()
		fmt.Printf("%s version %s\n", appName, info.Short())
		return nil
	}

	// Initialize logging
	logging.Init(logging.DefaultConfig())

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply command line flag overrides
	if useMock {
		cfg.UseMockClient = true
	}
	if noMock {
		cfg.UseMockClient = false
	}
	if debugMode {
		// Debug mode enabled - would configure logging if we had logging config
		fmt.Println("Debug mode enabled")
	}

	// Validate mock usage with environment variable gating
	if err := mock.ValidateMockUsage(cfg.UseMockClient); err != nil {
		if useMock {
			// User explicitly requested mock but it's not enabled
			fmt.Printf("❌ %v\n\n", err)
			mock.SuggestMockSetup()
			return fmt.Errorf("mock mode validation failed")
		}
		// Config file has mock enabled but environment doesn't allow it
		fmt.Printf("⚠️  Mock mode disabled by environment: %v\n", err)
		fmt.Printf("   Switching to real SLURM client mode\n\n")
		cfg.UseMockClient = false
	}

	// Apply discovery flag overrides
	if noDiscovery {
		cfg.Discovery.Enabled = false
	}
	if discoveryTimeout != "" {
		cfg.Discovery.Timeout = discoveryTimeout
	}

	// Attempt auto-discovery if enabled and no endpoint/token configured
	if cfg.Discovery.Enabled && !cfg.UseMockClient {
		cfg = applyAutoDiscovery(ctx, cfg)
	}

	// Check if config is empty and suggest setup
	if len(cfg.Contexts) == 0 && !cfg.UseMockClient && cfg.Cluster.Endpoint == "" {
		fmt.Printf("⚠️  No SLURM clusters configured.\n\n")
		fmt.Printf("To get started:\n")
		fmt.Printf("  1. Run the setup wizard: %s\n", cmd.Root().CommandPath()+" setup")
		fmt.Printf("  2. Or use mock mode: %s\n", cmd.Root().CommandPath()+" --mock")
		fmt.Printf("  3. Or manually edit: ~/.s9s/config.yaml\n\n")

		if !cfg.UseMockClient {
			return fmt.Errorf("no clusters configured")
		}
	}

	// Create application instance
	s9sApp, err := app.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	// Create error channel for app errors
	errChan := make(chan error, 1)

	// Run application in goroutine
	go func() {
		err := s9sApp.Run()
		errChan <- err
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		logging.Infof("Received signal: %v. Starting graceful shutdown...", sig)

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		// Gracefully stop the application
		if err := s9sApp.Stop(); err != nil {
			logging.Errorf("Error during shutdown: %v", err)
		}

		// Cancel the main context
		cancel()

		// Wait for shutdown or timeout
		<-shutdownCtx.Done()

	case err := <-errChan:
		if err != nil {
			cancel()
			return fmt.Errorf("application error: %w", err)
		}
		// Normal shutdown (err == nil)
		cancel()
	}

	logging.Info("S9S shutdown complete")
	return nil
}

// applyAutoDiscovery attempts to auto-discover slurmrestd endpoint and token
// when they are not explicitly configured
func applyAutoDiscovery(ctx context.Context, cfg *config.Config) *config.Config {
	// Skip if endpoint is already configured
	if cfg.Cluster.Endpoint != "" {
		// But still try to discover token if not set
		if cfg.Cluster.Token == "" && cfg.Discovery.EnableToken {
			if token := discoverToken(ctx, cfg); token != "" {
				cfg.Cluster.Token = token
				logging.Info("Auto-discovered SLURM token")
			}
		}
		return cfg
	}

	// Try to discover endpoint
	if cfg.Discovery.EnableEndpoint {
		endpoint := discoverEndpoint(ctx, cfg)
		if endpoint != nil {
			cfg.Cluster.Endpoint = endpoint.URL
			logging.Infof("Auto-discovered slurmrestd endpoint: %s (source: %s)", endpoint.URL, endpoint.Source)

			// Set API version if not already set
			if cfg.Cluster.APIVersion == "" {
				cfg.Cluster.APIVersion = "v0.0.43"
			}

			// Create a default context if none exists
			if len(cfg.Contexts) == 0 {
				cfg.Contexts = append(cfg.Contexts, config.ContextConfig{
					Name:    "default",
					Cluster: cfg.Cluster,
				})
				cfg.CurrentContext = "default"
			}
		}
	}

	// Try to discover token if not set
	if cfg.Cluster.Token == "" && cfg.Discovery.EnableToken {
		if token := discoverToken(ctx, cfg); token != "" {
			cfg.Cluster.Token = token
			logging.Info("Auto-discovered SLURM token")
		}
	}

	return cfg
}

// discoverEndpoint attempts to discover the slurmrestd endpoint
func discoverEndpoint(ctx context.Context, cfg *config.Config) *discovery.DiscoveredEndpoint {
	timeout, err := time.ParseDuration(cfg.Discovery.Timeout)
	if err != nil {
		timeout = 10 * time.Second
	}

	adCfg := discovery.AutoDiscoveryConfig{
		Enabled:        true,
		EnableEndpoint: true,
		Timeout:        timeout,
		DefaultPort:    cfg.Discovery.DefaultPort,
		ScontrolPath:   cfg.Discovery.ScontrolPath,
		CacheDuration:  5 * time.Minute,
	}

	ad := discovery.NewAutoDiscoveryWithConfig(adCfg)

	endpoint, err := ad.DiscoverEndpoint(ctx)
	if err != nil {
		if debugMode {
			logging.Errorf("Auto-discovery failed: %v", err)
		}
		return nil
	}

	return endpoint
}

// discoverToken attempts to discover or generate a SLURM JWT token
func discoverToken(ctx context.Context, cfg *config.Config) string {
	timeout, err := time.ParseDuration(cfg.Discovery.Timeout)
	if err != nil {
		timeout = 10 * time.Second
	}

	tdCfg := auth.TokenDiscoveryConfig{
		Enabled:       true,
		ScontrolPath:  cfg.Discovery.ScontrolPath,
		Timeout:       timeout,
		TokenLifespan: 3600, // 1 hour
	}

	td := auth.NewTokenDiscoveryWithConfig(tdCfg)

	token, err := td.DiscoverToken(ctx, cfg.CurrentContext)
	if err != nil {
		if debugMode {
			logging.Errorf("Token discovery failed: %v", err)
		}
		return ""
	}

	return token.Token
}
