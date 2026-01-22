package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/initialization"
	"gopkg.in/yaml.v3"
)

func main() {
	// Set up master key for testing (32 bytes base64 encoded)
	// This is just a test key: "test-master-key-32-bytes-long!!!!"
	_ = os.Setenv("OBSERVABILITY_MASTER_KEY", "dGVzdC1tYXN0ZXIta2V5LTMyLWJ5dGVzLWxvbmchISEh")

	// Load configuration
	configData, err := os.ReadFile("test-config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Merge with defaults and validate
	cfg.MergeWithDefaults()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	fmt.Println("Configuration loaded successfully")
	fmt.Printf("Prometheus endpoint: %s\n", cfg.Prometheus.Endpoint)

	// Initialize components
	ctx := context.Background()
	manager := initialization.NewManagerWithContext(ctx, &cfg)

	fmt.Println("\nInitializing components...")
	components, err := manager.InitializeComponents()
	if err != nil {
		log.Fatalf("Failed to initialize components: %v", err)
	}

	fmt.Println("✓ Components initialized successfully")

	// Test Prometheus connectivity
	fmt.Println("\nTesting Prometheus connectivity...")
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Query for available metrics
	query := "up"
	result, err := components.Client.Query(testCtx, query, time.Now())
	if err != nil {
		log.Fatalf("Failed to query Prometheus: %v", err)
	}

	fmt.Printf("✓ Successfully connected to Prometheus\n")
	fmt.Printf("✓ Query result for 'up': %v\n", result)

	// Test some SLURM-specific metrics
	fmt.Println("\nChecking for SLURM metrics...")
	queries := []string{
		"node_cpu_seconds_total",
		"node_memory_MemTotal_bytes",
		"node_load1",
		"slurm_job_states",
		"slurm_node_states",
	}

	for _, q := range queries {
		result, err := components.Client.Query(testCtx, fmt.Sprintf("count(%s)", q), time.Now())
		if err != nil {
			fmt.Printf("✗ %s: %v\n", q, err)
		} else {
			fmt.Printf("✓ %s: %v\n", q, result)
		}
	}

	// Test caching
	fmt.Println("\nTesting cache functionality...")
	start := time.Now()
	_, _ = components.CachedClient.Query(testCtx, "node_cpu_seconds_total", time.Now())
	firstQueryTime := time.Since(start)

	start = time.Now()
	_, _ = components.CachedClient.Query(testCtx, "node_cpu_seconds_total", time.Now())
	secondQueryTime := time.Since(start)

	fmt.Printf("✓ First query: %v\n", firstQueryTime)
	fmt.Printf("✓ Second query (cached): %v\n", secondQueryTime)
	fmt.Printf("✓ Cache speedup: %.2fx\n", float64(firstQueryTime)/float64(secondQueryTime))

	// Get cache stats
	stats := components.CachedClient.CacheStats()
	fmt.Printf("✓ Cache stats: %+v\n", stats)

	// Test historical data collection
	fmt.Println("\nTesting historical data collection...")
	historicalStats := components.HistoricalCollector.GetCollectorStats()
	fmt.Printf("✓ Historical collector stats: %+v\n", historicalStats)

	// Test external API if enabled
	if components.ExternalAPI != nil {
		fmt.Printf("\nStarting external API on port %d...\n", cfg.ExternalAPI.Port)
		if err := components.ExternalAPI.Start(ctx); err != nil {
			log.Printf("Failed to start external API: %v", err)
		} else {
			fmt.Println("✓ External API started successfully")
			fmt.Printf("  Test endpoints:\n")
			fmt.Printf("  - http://localhost:%d/health\n", cfg.ExternalAPI.Port)
			fmt.Printf("  - http://localhost:%d/api/v1/status\n", cfg.ExternalAPI.Port)
			fmt.Printf("  - http://localhost:%d/api/v1/metrics/query?query=up\n", cfg.ExternalAPI.Port)
		}
	}

	// Keep running for a bit to test
	fmt.Println("\nPlugin is running. Press Ctrl+C to stop...")
	select {}
}
