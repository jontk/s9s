package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jontk/s9s/plugins/observability/prometheus"
)

// MockPrometheusClient for testing
type MockPrometheusClient struct {
	shouldFail      bool
	latency         time.Duration
	connectionCount int
}

func (m *MockPrometheusClient) TestConnection(ctx context.Context) error {
	m.connectionCount++
	if m.shouldFail {
		return errors.New("mock connection failure")
	}
	time.Sleep(m.latency)
	return nil
}

func (m *MockPrometheusClient) Query(ctx context.Context, query string, time time.Time) (*prometheus.QueryResult, error) {
	if m.shouldFail {
		return nil, errors.New("mock query failure")
	}
	return &prometheus.QueryResult{Status: "success"}, nil
}

func (m *MockPrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*prometheus.QueryResult, error) {
	if m.shouldFail {
		return nil, errors.New("mock query range failure")
	}
	return &prometheus.QueryResult{Status: "success"}, nil
}

func (m *MockPrometheusClient) Series(ctx context.Context, matches []string, start, end time.Time) ([]map[string]string, error) {
	if m.shouldFail {
		return nil, errors.New("mock series failure")
	}
	return []map[string]string{{"__name__": "test_metric"}}, nil
}

func (m *MockPrometheusClient) Labels(ctx context.Context) ([]string, error) {
	if m.shouldFail {
		return nil, errors.New("mock labels failure")
	}
	return []string{"__name__", "job", "instance"}, nil
}

func (m *MockPrometheusClient) BatchQuery(ctx context.Context, queries map[string]string, ts time.Time) (map[string]*prometheus.QueryResult, error) {
	if m.shouldFail {
		return nil, errors.New("mock batch query failure")
	}

	results := make(map[string]*prometheus.QueryResult)
	for name := range queries {
		results[name] = &prometheus.QueryResult{Status: "success"}
	}
	return results, nil
}

func TestNewCollector(t *testing.T) {
	ctx := context.Background()
	client := &MockPrometheusClient{}

	collector := NewCollector(ctx, client)
	defer func() { _ = collector.Stop() }()

	if collector == nil {
		t.Fatal("Expected collector instance, got nil")
	}

	if collector.metrics == nil {
		t.Error("Expected metrics instance to be initialized")
	}

	if collector.client != client {
		t.Error("Expected client to be set correctly")
	}
}

func TestCollectorStartStop(t *testing.T) {
	ctx := context.Background()
	client := &MockPrometheusClient{}

	collector := NewCollector(ctx, client)

	// Start collector
	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Check that it's marked as started
	collector.mu.RLock()
	started := collector.started
	collector.mu.RUnlock()

	if !started {
		t.Error("Expected collector to be marked as started")
	}

	// Check component status
	stats := collector.metrics.GetComponentStats()
	componentStatus := stats["component_status"].(map[string]ComponentState)

	if componentStatus["metrics_collector"].Status != "running" {
		t.Errorf("Expected metrics_collector status 'running', got '%s'", componentStatus["metrics_collector"].Status)
	}

	// Stop collector
	err = collector.Stop()
	if err != nil {
		t.Fatalf("Failed to stop collector: %v", err)
	}

	// Check that it's marked as stopped
	collector.mu.RLock()
	started = collector.started
	collector.mu.RUnlock()

	if started {
		t.Error("Expected collector to be marked as stopped")
	}

	// Check component status after stop
	stats = collector.metrics.GetComponentStats()
	componentStatus = stats["component_status"].(map[string]ComponentState)

	if componentStatus["metrics_collector"].Status != "stopped" {
		t.Errorf("Expected metrics_collector status 'stopped', got '%s'", componentStatus["metrics_collector"].Status)
	}
}

func TestCollectorDoubleStartStop(t *testing.T) {
	ctx := context.Background()
	client := &MockPrometheusClient{}

	collector := NewCollector(ctx, client)
	defer func() { _ = collector.Stop() }()

	// Start twice - should not fail
	err1 := collector.Start()
	err2 := collector.Start()

	if err1 != nil {
		t.Fatalf("First start failed: %v", err1)
	}

	if err2 != nil {
		t.Fatalf("Second start failed: %v", err2)
	}

	// Stop twice - should not fail
	err3 := collector.Stop()
	err4 := collector.Stop()

	if err3 != nil {
		t.Fatalf("First stop failed: %v", err3)
	}

	if err4 != nil {
		t.Fatalf("Second stop failed: %v", err4)
	}
}

func TestInstrumentedClient(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockPrometheusClient{latency: 10 * time.Millisecond}

	collector := NewCollector(ctx, mockClient)
	defer func() { _ = collector.Stop() }()

	instrumentedClient := collector.WrapClient(mockClient)

	if instrumentedClient == nil {
		t.Fatal("Expected instrumented client, got nil")
	}

	// Test connection
	err := instrumentedClient.TestConnection(ctx)
	if err != nil {
		t.Errorf("Expected successful connection, got error: %v", err)
	}

	// Check metrics
	connectionStats := collector.metrics.GetConnectionStats()
	totalConnections := connectionStats["total_connections"].(int64)
	if totalConnections != 1 {
		t.Errorf("Expected 1 connection recorded, got %d", totalConnections)
	}

	// Test query
	result, err := instrumentedClient.Query(ctx, "up", time.Now())
	if err != nil {
		t.Errorf("Expected successful query, got error: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected query status 'success', got '%s'", result.Status)
	}

	// Check query metrics
	queryStats := collector.metrics.GetQueryStats()
	totalQueries := queryStats["total_queries"].(int64)
	if totalQueries != 1 {
		t.Errorf("Expected 1 query recorded, got %d", totalQueries)
	}

	// Test batch query
	queries := map[string]string{
		"cpu": "cpu_usage",
		"mem": "memory_usage",
	}

	batchResults, err := instrumentedClient.BatchQuery(ctx, queries, time.Now())
	if err != nil {
		t.Errorf("Expected successful batch query, got error: %v", err)
	}

	if len(batchResults) != 2 {
		t.Errorf("Expected 2 batch results, got %d", len(batchResults))
	}

	// Check batch query metrics
	queryStats = collector.metrics.GetQueryStats()
	batchQueries := queryStats["batch_queries"].(int64)
	if batchQueries != 1 {
		t.Errorf("Expected 1 batch query recorded, got %d", batchQueries)
	}
}

func TestInstrumentedClientFailures(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockPrometheusClient{shouldFail: true}

	collector := NewCollector(ctx, mockClient)
	defer func() { _ = collector.Stop() }()

	instrumentedClient := collector.WrapClient(mockClient)

	// Test failed connection
	err := instrumentedClient.TestConnection(ctx)
	if err == nil {
		t.Error("Expected connection failure, got success")
	}

	// Check connection failure metrics
	connectionStats := collector.metrics.GetConnectionStats()
	failedConnections := connectionStats["failed_connections"].(int64)
	if failedConnections != 1 {
		t.Errorf("Expected 1 failed connection recorded, got %d", failedConnections)
	}

	// Test failed query
	_, err = instrumentedClient.Query(ctx, "up", time.Now())
	if err == nil {
		t.Error("Expected query failure, got success")
	}

	// Check query failure metrics
	queryStats := collector.metrics.GetQueryStats()
	failedQueries := queryStats["failed_queries"].(int64)
	if failedQueries != 1 {
		t.Errorf("Expected 1 failed query recorded, got %d", failedQueries)
	}
}

func TestMetricsCollection(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockPrometheusClient{latency: 5 * time.Millisecond}

	collector := NewCollector(ctx, mockClient)
	defer func() { _ = collector.Stop() }()

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Manually trigger system metrics collection
	collector.updateSystemMetrics()

	// Check that system metrics are being collected
	resourceStats := collector.metrics.GetResourceStats()

	memoryUsage := resourceStats["memory_usage_bytes"].(int64)
	if memoryUsage <= 0 {
		t.Errorf("Expected positive memory usage, got %d", memoryUsage)
	}

	goroutines := resourceStats["goroutine_count"].(int32)
	if goroutines <= 0 {
		t.Errorf("Expected positive goroutine count, got %d", goroutines)
	}

	cpuUsage := resourceStats["cpu_usage_percent"].(float64)
	if cpuUsage < 0 {
		t.Errorf("Expected non-negative CPU usage, got %.2f", cpuUsage)
	}
}

func TestMetricsExporter(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	// Add some test data
	pm.RecordConnection(100 * time.Millisecond)
	pm.RecordQuery(50*time.Millisecond, true)
	pm.UpdateResourceUsage(1024*1024, 25.5, 10)
	pm.UpdateCacheHitRate(75.0)

	exporter := NewMetricsExporter(pm)
	output := exporter.ExportPrometheusFormat()

	if output == "" {
		t.Error("Expected non-empty Prometheus format output")
	}

	// Check for some expected metrics in the output
	expectedMetrics := []string{
		"observability_plugin_connections_total",
		"observability_plugin_queries_total",
		"observability_plugin_memory_bytes",
		"observability_plugin_cache_hit_rate",
	}

	for _, metric := range expectedMetrics {
		if !contains(output, metric) {
			t.Errorf("Expected metric '%s' in output", metric)
		}
	}
}

func TestPrometheusMetricsCollection(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockPrometheusClient{}

	collector := NewCollector(ctx, mockClient)
	defer func() { _ = collector.Stop() }()

	// Manually trigger Prometheus metrics collection
	collector.updatePrometheusMetrics()

	// Check that connection was attempted
	if mockClient.connectionCount != 1 {
		t.Errorf("Expected 1 connection attempt, got %d", mockClient.connectionCount)
	}

	// Check component status
	stats := collector.metrics.GetComponentStats()
	componentStatus := stats["component_status"].(map[string]ComponentState)

	if componentStatus["prometheus_client"].Status != "running" {
		t.Errorf("Expected prometheus_client status 'running', got '%s'", componentStatus["prometheus_client"].Status)
	}
}

func TestPrometheusMetricsCollectionFailure(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockPrometheusClient{shouldFail: true}

	collector := NewCollector(ctx, mockClient)
	defer func() { _ = collector.Stop() }()

	// Manually trigger Prometheus metrics collection
	collector.updatePrometheusMetrics()

	// Check that connection was attempted
	if mockClient.connectionCount != 1 {
		t.Errorf("Expected 1 connection attempt, got %d", mockClient.connectionCount)
	}

	// Check component status shows error
	stats := collector.metrics.GetComponentStats()
	componentStatus := stats["component_status"].(map[string]ComponentState)

	if componentStatus["prometheus_client"].Status != "error" {
		t.Errorf("Expected prometheus_client status 'error', got '%s'", componentStatus["prometheus_client"].Status)
	}

	// Check that connection failure was recorded
	connectionStats := collector.metrics.GetConnectionStats()
	failedConnections := connectionStats["failed_connections"].(int64)
	if failedConnections != 1 {
		t.Errorf("Expected 1 failed connection, got %d", failedConnections)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
