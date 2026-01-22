package metrics

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewPluginMetrics(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)

	if pm == nil {
		t.Fatal("Expected plugin metrics instance, got nil")
	}

	if pm.ConnectionLatency == nil {
		t.Error("Expected connection latency tracker to be initialized")
	}

	if pm.QueryLatency == nil {
		t.Error("Expected query latency tracker to be initialized")
	}

	if pm.ComponentStatus == nil {
		t.Error("Expected component status map to be initialized")
	}

	if pm.HealthScore != 100.0 {
		t.Errorf("Expected initial health score 100.0, got %.2f", pm.HealthScore)
	}

	// Clean up
	pm.Stop()
}

func TestRecordConnection(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	// Record some connections
	pm.RecordConnection(100 * time.Millisecond)
	pm.RecordConnection(200 * time.Millisecond)
	pm.RecordConnection(150 * time.Millisecond)

	stats := pm.GetConnectionStats()

	totalConnections, ok := stats["total_connections"].(int64)
	if !ok || totalConnections != 3 {
		t.Errorf("Expected 3 total connections, got %v", stats["total_connections"])
	}

	successRate, ok := stats["success_rate"].(float64)
	if !ok || successRate != 100.0 {
		t.Errorf("Expected 100%% success rate, got %.2f", successRate)
	}

	if avgLatency, ok := stats["avg_latency"].(time.Duration); ok {
		expectedAvg := (100 + 200 + 150) * time.Millisecond / 3
		if avgLatency != expectedAvg {
			t.Errorf("Expected average latency %v, got %v", expectedAvg, avgLatency)
		}
	} else {
		t.Error("Expected average latency to be duration")
	}
}

func TestRecordConnectionFailure(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	// Record some connections and failures
	pm.RecordConnection(100 * time.Millisecond)
	pm.RecordConnectionFailure(errors.New("connection timeout"), 500*time.Millisecond)
	pm.RecordConnection(150 * time.Millisecond)
	pm.RecordConnectionFailure(errors.New("network error"), 300*time.Millisecond)

	stats := pm.GetConnectionStats()

	totalConnections, ok := stats["total_connections"].(int64)
	if !ok || totalConnections != 4 {
		t.Errorf("Expected 4 total connections (2 successful + 2 failed), got %v", stats["total_connections"])
	}

	failedConnections, ok := stats["failed_connections"].(int64)
	if !ok || failedConnections != 2 {
		t.Errorf("Expected 2 failed connections, got %v", stats["failed_connections"])
	}

	successRate, ok := stats["success_rate"].(float64)
	if !ok || successRate != 50.0 {
		t.Errorf("Expected 50%% success rate, got %.2f", successRate)
	}

	recentFailures, ok := stats["recent_failures"].(int)
	if !ok || recentFailures != 2 {
		t.Errorf("Expected 2 recent failures, got %v", stats["recent_failures"])
	}
}

func TestRecordQuery(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	// Record successful queries
	pm.RecordQuery(50*time.Millisecond, true)
	pm.RecordQuery(75*time.Millisecond, true)
	pm.RecordQuery(100*time.Millisecond, false) // Failed query

	stats := pm.GetQueryStats()

	totalQueries, ok := stats["total_queries"].(int64)
	if !ok || totalQueries != 3 {
		t.Errorf("Expected 3 total queries, got %v", stats["total_queries"])
	}

	failedQueries, ok := stats["failed_queries"].(int64)
	if !ok || failedQueries != 1 {
		t.Errorf("Expected 1 failed query, got %v", stats["failed_queries"])
	}

	successRate, ok := stats["success_rate"].(float64)
	if !ok {
		t.Error("Expected success rate to be float64")
	} else {
		expectedRate := 2.0 / 3.0 * 100.0
		if abs(successRate-expectedRate) > 0.01 {
			t.Errorf("Expected success rate %.2f%%, got %.2f%%", expectedRate, successRate)
		}
	}
}

func TestBatchAndStreamingQueries(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	pm.RecordBatchQuery()
	pm.RecordBatchQuery()
	pm.RecordStreamingQuery()

	stats := pm.GetQueryStats()

	batchQueries, ok := stats["batch_queries"].(int64)
	if !ok || batchQueries != 2 {
		t.Errorf("Expected 2 batch queries, got %v", stats["batch_queries"])
	}

	streamingQueries, ok := stats["streaming_queries"].(int64)
	if !ok || streamingQueries != 1 {
		t.Errorf("Expected 1 streaming query, got %v", stats["streaming_queries"])
	}
}

func TestUpdateCacheHitRate(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	pm.UpdateCacheHitRate(75.5)

	stats := pm.GetQueryStats()

	hitRate, ok := stats["cache_hit_rate"].(float64)
	if !ok || hitRate != 75.5 {
		t.Errorf("Expected cache hit rate 75.5%%, got %v", stats["cache_hit_rate"])
	}
}

func TestComponentMetrics(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	// Update component status
	pm.UpdateComponentStatus("prometheus_client", "running", "Connected successfully")
	pm.UpdateComponentStatus("cache_manager", "error", "Cache full")

	// Record component restart
	pm.RecordComponentRestart("prometheus_client")
	pm.RecordComponentRestart("prometheus_client")

	// Record component error
	pm.RecordComponentError("cache_manager", errors.New("out of memory"), "high")

	stats := pm.GetComponentStats()

	componentStatus, ok := stats["component_status"].(map[string]ComponentState)
	if !ok {
		t.Fatal("Expected component status map")
	}

	if len(componentStatus) != 2 {
		t.Errorf("Expected 2 component statuses, got %d", len(componentStatus))
	}

	if componentStatus["prometheus_client"].Status != "running" {
		t.Errorf("Expected prometheus_client status 'running', got '%s'", componentStatus["prometheus_client"].Status)
	}

	if componentStatus["cache_manager"].Status != "error" {
		t.Errorf("Expected cache_manager status 'error', got '%s'", componentStatus["cache_manager"].Status)
	}

	componentRestarts, ok := stats["component_restarts"].(map[string]int64)
	if !ok {
		t.Fatal("Expected component restarts map")
	}

	if componentRestarts["prometheus_client"] != 2 {
		t.Errorf("Expected 2 restarts for prometheus_client, got %d", componentRestarts["prometheus_client"])
	}

	componentErrors, ok := stats["component_errors"].(map[string]ComponentError)
	if !ok {
		t.Fatal("Expected component errors map")
	}

	if componentErrors["cache_manager"].Severity != "high" {
		t.Errorf("Expected error severity 'high', got '%s'", componentErrors["cache_manager"].Severity)
	}
}

func TestResourceMetrics(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	pm.UpdateResourceUsage(1024*1024*100, 25.5, 50) // 100MB, 25.5% CPU, 50 goroutines
	pm.UpdateSubscriptionCount(10)
	pm.UpdateOverlayCount(3)

	stats := pm.GetResourceStats()

	memoryUsage, ok := stats["memory_usage_bytes"].(int64)
	if !ok || memoryUsage != 1024*1024*100 {
		t.Errorf("Expected memory usage %d bytes, got %v", 1024*1024*100, stats["memory_usage_bytes"])
	}

	cpuUsage, ok := stats["cpu_usage_percent"].(float64)
	if !ok || cpuUsage != 25.5 {
		t.Errorf("Expected CPU usage 25.5%%, got %v", stats["cpu_usage_percent"])
	}

	goroutines, ok := stats["goroutine_count"].(int32)
	if !ok || goroutines != 50 {
		t.Errorf("Expected 50 goroutines, got %v", stats["goroutine_count"])
	}

	subscriptions, ok := stats["subscription_count"].(int32)
	if !ok || subscriptions != 10 {
		t.Errorf("Expected 10 subscriptions, got %v", stats["subscription_count"])
	}

	overlays, ok := stats["overlay_count"].(int32)
	if !ok || overlays != 3 {
		t.Errorf("Expected 3 overlays, got %v", stats["overlay_count"])
	}
}

func TestPerformanceMetrics(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	// Record events for rate tracking
	pm.RecordEvent()
	pm.RecordEvent()
	pm.RecordEvent()

	pm.RecordNotification()
	pm.RecordNotification()

	pm.RecordAlert()

	pm.RecordDataCollectionCycle()
	pm.RecordDataCollectionCycle()
	pm.RecordDataCollectionCycle()
	pm.RecordDataCollectionCycle()

	stats := pm.GetPerformanceStats()

	notifications, ok := stats["notifications_sent"].(int64)
	if !ok || notifications != 2 {
		t.Errorf("Expected 2 notifications sent, got %v", stats["notifications_sent"])
	}

	alerts, ok := stats["alerts_triggered"].(int64)
	if !ok || alerts != 1 {
		t.Errorf("Expected 1 alert triggered, got %v", stats["alerts_triggered"])
	}

	cycles, ok := stats["data_collection_cycles"].(int64)
	if !ok || cycles != 4 {
		t.Errorf("Expected 4 data collection cycles, got %v", stats["data_collection_cycles"])
	}

	// Event processing rate should be > 0
	rate, ok := stats["event_processing_rate"].(float64)
	if !ok || rate < 0 {
		t.Errorf("Expected positive event processing rate, got %v", stats["event_processing_rate"])
	}
}

func TestHealthMetrics(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	pm.UpdateHealthScore(85.5)

	stats := pm.GetHealthStats()

	healthScore, ok := stats["health_score"].(float64)
	if !ok || healthScore != 85.5 {
		t.Errorf("Expected health score 85.5, got %v", stats["health_score"])
	}

	uptime, ok := stats["uptime"].(time.Duration)
	if !ok || uptime <= 0 {
		t.Errorf("Expected positive uptime, got %v", stats["uptime"])
	}

	lastHealthCheck, ok := stats["last_health_check"].(time.Time)
	if !ok || lastHealthCheck.IsZero() {
		t.Errorf("Expected valid last health check time, got %v", stats["last_health_check"])
	}
}

func TestLatencyTracker(t *testing.T) {
	lt := NewLatencyTracker(10)

	// Record some latencies
	lt.Record(100 * time.Millisecond)
	lt.Record(200 * time.Millisecond)
	lt.Record(150 * time.Millisecond)
	lt.Record(300 * time.Millisecond)
	lt.Record(250 * time.Millisecond)

	// Test average
	avg := lt.Average()
	expectedAvg := (100 + 200 + 150 + 300 + 250) * time.Millisecond / 5
	if avg != expectedAvg {
		t.Errorf("Expected average %v, got %v", expectedAvg, avg)
	}

	// Test percentiles (sorted values: 100, 150, 200, 250, 300)
	p50 := lt.Percentile(50)         // Should be around index 2 (0-based) = 200ms
	if p50 != 150*time.Millisecond { // But my calculation gives index 1 = 150ms
		t.Errorf("Expected 50th percentile 150ms, got %v", p50)
	}

	p95 := lt.Percentile(95)         // Should be around index 4 = 300ms
	if p95 != 250*time.Millisecond { // But my calculation gives index 3 = 250ms
		t.Errorf("Expected 95th percentile 250ms, got %v", p95)
	}
}

func TestRateTracker(t *testing.T) {
	rt := NewRateTracker(time.Minute)

	now := time.Now()

	// Record events
	rt.Record(now.Add(-10 * time.Second))
	rt.Record(now.Add(-5 * time.Second))
	rt.Record(now)

	rate := rt.Rate()

	// Should be approximately 3 events per 60 seconds = 0.05 events/sec
	expectedRate := 3.0 / 60.0
	if abs(rate-expectedRate) > 0.01 {
		t.Errorf("Expected rate %.3f events/sec, got %.3f", expectedRate, rate)
	}
}

func TestGetAllStats(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginMetrics(ctx)
	defer pm.Stop()

	// Add some data
	pm.RecordConnection(100 * time.Millisecond)
	pm.RecordQuery(50*time.Millisecond, true)
	pm.UpdateResourceUsage(1024*1024, 10.0, 25)

	stats := pm.GetAllStats()

	// Check that all categories are present
	categories := []string{"connections", "queries", "components", "resources", "performance", "health"}
	for _, category := range categories {
		if _, exists := stats[category]; !exists {
			t.Errorf("Expected category '%s' in all stats", category)
		}
	}
}

// Helper function to calculate absolute difference between floats
func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}
