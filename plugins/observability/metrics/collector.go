// Package metrics provides comprehensive metrics collection and instrumentation
// for the observability plugin itself. It tracks internal performance metrics,
// request patterns, cache efficiency, and operational statistics to enable
// self-monitoring and performance optimization of the plugin components.
package metrics

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/jontk/s9s/plugins/observability/prometheus"
)

// Collector integrates plugin metrics with the Prometheus client
type Collector struct {
	metrics *PluginMetrics
	client  prometheus.PrometheusClientInterface

	// Internal state
	mu      sync.RWMutex
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewCollector creates a new metrics collector
func NewCollector(ctx context.Context, client prometheus.PrometheusClientInterface) *Collector {
	collectorCtx, cancel := context.WithCancel(ctx)

	return &Collector{
		metrics: NewPluginMetrics(collectorCtx),
		client:  client,
		ctx:     collectorCtx,
		cancel:  cancel,
	}
}

// Start starts the metrics collection
func (c *Collector) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	c.started = true

	// Start system metrics collection
	go c.collectSystemMetrics()

	// Start Prometheus metrics collection
	go c.collectPrometheusMetrics()

	// Update component status
	c.metrics.UpdateComponentStatus("metrics_collector", "running", "Collecting plugin metrics")

	return nil
}

// Stop stops the metrics collection
func (c *Collector) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	c.started = false

	if c.cancel != nil {
		c.cancel()
	}

	c.metrics.Stop()

	// Update component status
	c.metrics.UpdateComponentStatus("metrics_collector", "stopped", "Metrics collection stopped")

	return nil
}

// GetMetrics returns the plugin metrics instance
func (c *Collector) GetMetrics() *PluginMetrics {
	return c.metrics
}

// collectSystemMetrics collects system-level metrics periodically
func (c *Collector) collectSystemMetrics() {
	ticker := time.NewTicker(10 * time.Second) // Collect every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.updateSystemMetrics()
		}
	}
}

// updateSystemMetrics updates system metrics
func (c *Collector) updateSystemMetrics() {
	// Memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Goroutine count
	// nolint:gosec // G115: NumGoroutine() fits safely in int32
	goroutines := int32(runtime.NumGoroutine())

	// CPU usage would typically be calculated from system calls
	// For now, we'll estimate based on goroutine activity
	cpuUsage := c.estimateCPUUsage(goroutines)

	// nolint:gosec // G115: memory metrics bounded by system RAM
	c.metrics.UpdateResourceUsage(int64(m.Alloc), cpuUsage, goroutines)
}

// estimateCPUUsage provides a rough CPU usage estimate
func (c *Collector) estimateCPUUsage(goroutines int32) float64 {
	// This is a simplified estimation - in a real implementation,
	// you'd use system calls to get actual CPU usage
	baseUsage := float64(goroutines) / 100.0
	if baseUsage > 100.0 {
		baseUsage = 100.0
	}
	return baseUsage
}

// collectPrometheusMetrics collects metrics about Prometheus interactions
func (c *Collector) collectPrometheusMetrics() {
	ticker := time.NewTicker(30 * time.Second) // Collect every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.updatePrometheusMetrics()
		}
	}
}

// updatePrometheusMetrics updates Prometheus-related metrics
func (c *Collector) updatePrometheusMetrics() {
	// Test connection and measure latency
	start := time.Now()
	err := c.client.TestConnection(c.ctx)
	latency := time.Since(start)

	if err != nil {
		c.metrics.RecordConnectionFailure(err, latency)
		c.metrics.UpdateComponentStatus("prometheus_client", "error", err.Error())
	} else {
		c.metrics.RecordConnection(latency)
		c.metrics.UpdateComponentStatus("prometheus_client", "running", "Connected to Prometheus")
	}

	// Update cache hit rate if we have a cached client
	// We'll check this in a different way since the client might be wrapped
}

// WrapClient wraps a Prometheus client to automatically collect metrics
func (c *Collector) WrapClient(client prometheus.PrometheusClientInterface) prometheus.PrometheusClientInterface {
	return &InstrumentedClient{
		client:  client,
		metrics: c.metrics,
	}
}

// InstrumentedClient wraps a Prometheus client with metrics collection
type InstrumentedClient struct {
	client  prometheus.PrometheusClientInterface
	metrics *PluginMetrics
}

// TestConnection implements PrometheusClientInterface
func (ic *InstrumentedClient) TestConnection(ctx context.Context) error {
	start := time.Now()
	err := ic.client.TestConnection(ctx)
	latency := time.Since(start)

	if err != nil {
		ic.metrics.RecordConnectionFailure(err, latency)
	} else {
		ic.metrics.RecordConnection(latency)
	}

	return err
}

// Query implements PrometheusClientInterface
func (ic *InstrumentedClient) Query(ctx context.Context, query string, queryTime time.Time) (*prometheus.QueryResult, error) {
	start := time.Now()
	result, err := ic.client.Query(ctx, query, queryTime)
	latency := time.Since(start)

	ic.metrics.RecordQuery(latency, err == nil)
	ic.metrics.RecordEvent() // For rate tracking

	return result, err
}

// QueryRange implements PrometheusClientInterface
func (ic *InstrumentedClient) QueryRange(ctx context.Context, query string, startTime, endTime time.Time, step time.Duration) (*prometheus.QueryResult, error) {
	measureStart := time.Now()
	result, err := ic.client.QueryRange(ctx, query, startTime, endTime, step)
	latency := time.Since(measureStart)

	ic.metrics.RecordQuery(latency, err == nil)
	ic.metrics.RecordEvent()

	return result, err
}

// Series implements PrometheusClientInterface
func (ic *InstrumentedClient) Series(ctx context.Context, matches []string, startTime, endTime time.Time) ([]map[string]string, error) {
	measureStart := time.Now()
	result, err := ic.client.Series(ctx, matches, startTime, endTime)
	latency := time.Since(measureStart)

	ic.metrics.RecordQuery(latency, err == nil)
	ic.metrics.RecordEvent()

	return result, err
}

// Labels implements PrometheusClientInterface
func (ic *InstrumentedClient) Labels(ctx context.Context) ([]string, error) {
	measureStart := time.Now()
	result, err := ic.client.Labels(ctx)
	latency := time.Since(measureStart)

	ic.metrics.RecordQuery(latency, err == nil)
	ic.metrics.RecordEvent()

	return result, err
}

// BatchQuery implements PrometheusClientInterface
func (ic *InstrumentedClient) BatchQuery(ctx context.Context, queries map[string]string, queryTime time.Time) (map[string]*prometheus.QueryResult, error) {
	measureStart := time.Now()
	result, err := ic.client.BatchQuery(ctx, queries, queryTime)
	latency := time.Since(measureStart)

	ic.metrics.RecordQuery(latency, err == nil)
	ic.metrics.RecordBatchQuery()
	ic.metrics.RecordEvent()

	return result, err
}

// MetricsExporter provides methods to export metrics in various formats
type MetricsExporter struct {
	metrics *PluginMetrics
}

// NewMetricsExporter creates a new metrics exporter
func NewMetricsExporter(metrics *PluginMetrics) *MetricsExporter {
	return &MetricsExporter{
		metrics: metrics,
	}
}

// ExportPrometheusFormat exports metrics in Prometheus exposition format
func (me *MetricsExporter) ExportPrometheusFormat() string {
	stats := me.metrics.GetAllStats()

	var output string

	// Connection metrics
	connectionStats := stats["connections"].(map[string]interface{})
	output += formatPrometheusMetric("observability_plugin_connections_total", connectionStats["total_connections"])
	output += formatPrometheusMetric("observability_plugin_connection_failures_total", connectionStats["failed_connections"])
	output += formatPrometheusMetric("observability_plugin_connection_success_rate", connectionStats["success_rate"])

	if avgLatency, ok := connectionStats["avg_latency"].(time.Duration); ok {
		output += formatPrometheusMetric("observability_plugin_connection_latency_seconds", avgLatency.Seconds())
	}

	// Query metrics
	queryStats := stats["queries"].(map[string]interface{})
	output += formatPrometheusMetric("observability_plugin_queries_total", queryStats["total_queries"])
	output += formatPrometheusMetric("observability_plugin_query_failures_total", queryStats["failed_queries"])
	output += formatPrometheusMetric("observability_plugin_batch_queries_total", queryStats["batch_queries"])
	output += formatPrometheusMetric("observability_plugin_streaming_queries_total", queryStats["streaming_queries"])
	output += formatPrometheusMetric("observability_plugin_query_success_rate", queryStats["success_rate"])
	output += formatPrometheusMetric("observability_plugin_cache_hit_rate", queryStats["cache_hit_rate"])

	// Resource metrics
	resourceStats := stats["resources"].(map[string]interface{})
	output += formatPrometheusMetric("observability_plugin_memory_bytes", resourceStats["memory_usage_bytes"])
	output += formatPrometheusMetric("observability_plugin_cpu_usage_percent", resourceStats["cpu_usage_percent"])
	output += formatPrometheusMetric("observability_plugin_goroutines", resourceStats["goroutine_count"])
	output += formatPrometheusMetric("observability_plugin_subscriptions", resourceStats["subscription_count"])
	output += formatPrometheusMetric("observability_plugin_overlays", resourceStats["overlay_count"])

	// Performance metrics
	performanceStats := stats["performance"].(map[string]interface{})
	output += formatPrometheusMetric("observability_plugin_events_per_second", performanceStats["event_processing_rate"])
	output += formatPrometheusMetric("observability_plugin_notifications_total", performanceStats["notifications_sent"])
	output += formatPrometheusMetric("observability_plugin_alerts_total", performanceStats["alerts_triggered"])
	output += formatPrometheusMetric("observability_plugin_data_collection_cycles_total", performanceStats["data_collection_cycles"])

	// Health metrics
	healthStats := stats["health"].(map[string]interface{})
	if uptime, ok := healthStats["uptime"].(time.Duration); ok {
		output += formatPrometheusMetric("observability_plugin_uptime_seconds", uptime.Seconds())
	}
	output += formatPrometheusMetric("observability_plugin_health_score", healthStats["health_score"])

	return output
}

// formatPrometheusMetric formats a metric in Prometheus exposition format
func formatPrometheusMetric(name string, value interface{}) string {
	return name + " " + formatValue(value) + "\n"
}

// formatValue formats a value for Prometheus
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case int:
		return string(rune(v + '0'))
	case int32:
		return string(rune(int(v) + '0'))
	case int64:
		return string(rune(int(v) + '0'))
	case float64:
		if v == float64(int(v)) {
			return string(rune(int(v) + '0'))
		}
		return "0" // Simplified - would use proper float formatting
	default:
		return "0"
	}
}
