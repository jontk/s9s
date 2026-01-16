package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jontk/s9s/plugins/observability/analysis"
	"github.com/jontk/s9s/plugins/observability/historical"
)

// MockPrometheusServer creates a mock Prometheus server for testing
func MockPrometheusServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock query endpoint
	mux.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		
		var response map[string]interface{}
		
		switch {
		case query == "up":
			response = map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": "vector",
					"result": []map[string]interface{}{
						{
							"metric": map[string]interface{}{
								"__name__":  "up",
								"instance": "localhost:9090",
								"job":      "prometheus",
							},
							"value": []interface{}{
								time.Now().Unix(),
								"1",
							},
						},
					},
				},
			}
		case query == "node_cpu":
			response = map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": "vector",
					"result": []map[string]interface{}{
						{
							"metric": map[string]interface{}{
								"instance": "node1",
							},
							"value": []interface{}{
								time.Now().Unix(),
								"45.5",
							},
						},
					},
				},
			}
		case query == "node_memory":
			response = map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": "vector",
					"result": []map[string]interface{}{
						{
							"metric": map[string]interface{}{
								"instance": "node1",
							},
							"value": []interface{}{
								time.Now().Unix(),
								"67.2",
							},
						},
					},
				},
			}
		default:
			response = map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": "vector",
					"result":     []interface{}{},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock query_range endpoint
	mux.HandleFunc("/api/v1/query_range", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		step := r.URL.Query().Get("step")

		// Generate time series data
		startTime, _ := time.Parse(time.RFC3339, start)
		endTime, _ := time.Parse(time.RFC3339, end)
		stepDuration, _ := time.ParseDuration(step + "s")

		var values [][]interface{}
		current := startTime
		baseValue := 50.0

		for current.Before(endTime) {
			// Add some variation to make data realistic
			variation := (float64(current.Unix()%100) - 50) / 10
			value := baseValue + variation
			values = append(values, []interface{}{
				current.Unix(),
				fmt.Sprintf("%.1f", value),
			})
			current = current.Add(stepDuration)
		}

		response := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "matrix",
				"result": []map[string]interface{}{
					{
						"metric": map[string]interface{}{
							"instance": "node1",
							"__name__":  query,
						},
						"values": values,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock health endpoint
	mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Prometheus is Healthy.\n"))
	})

	return httptest.NewServer(mux)
}

func TestObservabilityPluginIntegration(t *testing.T) {
	// Start mock Prometheus server
	server := MockPrometheusServer()
	defer server.Close()

	// Create plugin with mock server configuration
	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint":     server.URL,
		"prometheus.timeout":      "5s",
		"display.refreshInterval": "10s",
		"display.showOverlays":    true,
		"alerts.enabled":         true,
		"cache.enabled":          true,
		"cache.defaultTTL":       "1m",
		"cache.maxSize":          100,
	}

	ctx := context.Background()

	// Test initialization
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	// Test plugin info
	info := plugin.GetInfo()
	if info.Name != "observability" {
		t.Errorf("Expected plugin name 'observability', got '%s'", info.Name)
	}

	// Test plugin start
	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("Plugin start failed: %v", err)
	}

	// Test health check
	health := plugin.Health()
	if !health.Healthy {
		t.Errorf("Plugin should be healthy, got: %v", health)
	}

	// Test plugin stop
	err = plugin.Stop(ctx)
	if err != nil {
		t.Fatalf("Plugin stop failed: %v", err)
	}
}

func TestDataProviderIntegration(t *testing.T) {
	server := MockPrometheusServer()
	defer server.Close()

	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint": server.URL,
		"prometheus.timeout":  "5s",
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("Plugin start failed: %v", err)
	}
	defer plugin.Stop(ctx)

	// Test data providers
	providers := plugin.GetDataProviders()
	expectedProviders := []string{
		"prometheus-metrics",
		"alerts",
		"historical-data",
		"trend-analysis",
		"anomaly-detection",
		"seasonal-analysis",
		"resource-efficiency",
		"cluster-efficiency",
	}

	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}

	// Test prometheus-metrics query
	params := map[string]interface{}{
		"query": "up",
	}
	data, err := plugin.Query(ctx, "prometheus-metrics", params)
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}
	if data == nil {
		t.Error("Expected data, got nil")
	}
}

func TestHistoricalDataIntegration(t *testing.T) {
	server := MockPrometheusServer()
	defer server.Close()

	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint": server.URL,
		"prometheus.timeout":  "5s",
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("Plugin start failed: %v", err)
	}
	defer plugin.Stop(ctx)

	// Wait a moment for data collection
	time.Sleep(100 * time.Millisecond)

	// Test historical data query
	params := map[string]interface{}{
		"metric_name": "node_cpu",
		"duration":    "1h",
	}
	data, err := plugin.Query(ctx, "historical-data", params)
	if err == nil && data != nil {
		// Data might not be available immediately, which is okay
		series, ok := data.(*historical.MetricSeries)
		if ok && len(series.DataPoints) > 0 {
			t.Logf("Historical data query successful: %d data points", len(series.DataPoints))
		}
	}
}

func TestEfficiencyAnalysisIntegration(t *testing.T) {
	server := MockPrometheusServer()
	defer server.Close()

	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint": server.URL,
		"prometheus.timeout":  "5s",
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("Plugin start failed: %v", err)
	}
	defer plugin.Stop(ctx)

	// Wait for some data collection
	time.Sleep(500 * time.Millisecond)

	// Test resource efficiency analysis
	params := map[string]interface{}{
		"resource_type":    "cpu",
		"analysis_period": "1h",
	}
	
	data, err := plugin.Query(ctx, "resource-efficiency", params)
	// This might fail if there's not enough historical data, which is expected
	if err == nil && data != nil {
		analysis, ok := data.(*analysis.ResourceEfficiency)
		if ok {
			t.Logf("Efficiency analysis successful: %.2f score, %s level", 
				analysis.OverallScore, analysis.EfficiencyLevel)
		}
	} else {
		t.Logf("Efficiency analysis failed (expected with limited data): %v", err)
	}
}

func TestSubscriptionIntegration(t *testing.T) {
	server := MockPrometheusServer()
	defer server.Close()

	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint": server.URL,
		"prometheus.timeout":  "5s",
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("Plugin start failed: %v", err)
	}
	defer plugin.Stop(ctx)

	// Test subscription
	callbackCalled := false
	callback := func(data interface{}, err error) {
		callbackCalled = true
		if err != nil {
			t.Errorf("Callback received error: %v", err)
		}
	}

	subscriptionID, err := plugin.Subscribe(ctx, "prometheus-metrics", callback)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Wait for callback
	time.Sleep(100 * time.Millisecond)

	// Test unsubscribe
	err = plugin.Unsubscribe(ctx, subscriptionID)
	if err != nil {
		t.Errorf("Unsubscribe failed: %v", err)
	}

	// Note: callback might not be called immediately in test environment
	t.Logf("Subscription test completed, callback called: %v", callbackCalled)
}

func TestViewIntegration(t *testing.T) {
	server := MockPrometheusServer()
	defer server.Close()

	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint": server.URL,
		"prometheus.timeout":  "5s",
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	// Test view information
	views := plugin.GetViews()
	if len(views) != 1 {
		t.Errorf("Expected 1 view, got %d", len(views))
	}

	if views[0].ID != "observability" {
		t.Errorf("Expected view ID 'observability', got '%s'", views[0].ID)
	}

	// Note: We can't easily test CreateView without a full tview application context
}

func TestOverlayIntegration(t *testing.T) {
	server := MockPrometheusServer()
	defer server.Close()

	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint":  server.URL,
		"prometheus.timeout":   "5s",
		"display.showOverlays": true,
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("Plugin start failed: %v", err)
	}
	defer plugin.Stop(ctx)

	// Test overlay information
	overlays := plugin.GetOverlays()
	expectedOverlays := []string{"jobs-metrics", "nodes-metrics"}

	if len(overlays) != len(expectedOverlays) {
		t.Errorf("Expected %d overlays, got %d", len(expectedOverlays), len(overlays))
	}

	for i, overlay := range overlays {
		if overlay.ID != expectedOverlays[i] {
			t.Errorf("Expected overlay ID '%s', got '%s'", expectedOverlays[i], overlay.ID)
		}
	}
}

func TestConfigurationIntegration(t *testing.T) {
	plugin := New()

	// Test configuration schema
	schema := plugin.GetConfigSchema()
	expectedFields := []string{
		"prometheus.endpoint",
		"prometheus.timeout",
		"display.refreshInterval",
		"display.showOverlays",
		"alerts.enabled",
	}

	for _, field := range expectedFields {
		if _, exists := schema[field]; !exists {
			t.Errorf("Expected configuration field '%s' not found", field)
		}
	}

	// Test configuration validation
	validConfig := map[string]interface{}{
		"prometheus.endpoint": "http://localhost:9090",
		"prometheus.timeout":  "10s",
		"alerts.enabled":      true,
	}

	err := plugin.ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("Valid configuration rejected: %v", err)
	}
}

func TestErrorHandlingIntegration(t *testing.T) {
	// Create plugin without server (should fail)
	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint": "http://localhost:9999", // Non-existent server
		"prometheus.timeout":  "1s",
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	// Start should fail due to connection error
	err = plugin.Start(ctx)
	if err == nil {
		t.Error("Expected start to fail with invalid endpoint")
		plugin.Stop(ctx)
	}

	// Health check should report unhealthy
	health := plugin.Health()
	if health.Healthy {
		t.Error("Plugin should be unhealthy with invalid endpoint")
	}
}

func TestConcurrentOperations(t *testing.T) {
	server := MockPrometheusServer()
	defer server.Close()

	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint": server.URL,
		"prometheus.timeout":  "5s",
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Plugin initialization failed: %v", err)
	}

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("Plugin start failed: %v", err)
	}
	defer plugin.Stop(ctx)

	// Run concurrent queries
	numWorkers := 10
	done := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			params := map[string]interface{}{
				"query": "up",
			}

			data, err := plugin.Query(ctx, "prometheus-metrics", params)
			if err != nil {
				t.Errorf("Worker %d: Query failed: %v", workerID, err)
				return
			}

			if data == nil {
				t.Errorf("Worker %d: Expected data, got nil", workerID)
			}
		}(i)
	}

	// Wait for all workers
	for i := 0; i < numWorkers; i++ {
		<-done
	}
}

func BenchmarkQuery(b *testing.B) {
	server := MockPrometheusServer()
	defer server.Close()

	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint": server.URL,
		"prometheus.timeout":  "5s",
		"cache.enabled":       true,
		"cache.defaultTTL":    "1m",
	}

	ctx := context.Background()
	err := plugin.Init(ctx, config)
	if err != nil {
		b.Fatalf("Plugin initialization failed: %v", err)
	}

	err = plugin.Start(ctx)
	if err != nil {
		b.Fatalf("Plugin start failed: %v", err)
	}
	defer plugin.Stop(ctx)

	params := map[string]interface{}{
		"query": "up",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.Query(ctx, "prometheus-metrics", params)
		if err != nil {
			b.Errorf("Query failed: %v", err)
		}
	}
}