// Package metrics provides internal monitoring capabilities for the observability plugin itself
package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// PluginMetrics tracks internal metrics for the observability plugin
type PluginMetrics struct {
	// Connection metrics
	PrometheusConnections int64               // Total Prometheus connections made
	FailedConnections     int64               // Failed Prometheus connections
	ConnectionLatency     *LatencyTracker     // Connection latency tracking
	LastConnectionTime    time.Time           // Last successful connection time
	ConnectionFailures    []ConnectionFailure // Recent connection failures

	// Query metrics
	QueryCount          int64           // Total queries executed
	QueryFailures       int64           // Failed queries
	QueryLatency        *LatencyTracker // Query latency tracking
	CacheHitRate        float64         // Cache hit rate percentage
	BatchQueryCount     int64           // Batch queries executed
	StreamingQueryCount int64           // Streaming queries executed

	// Component metrics
	ComponentStatus    map[string]ComponentState // Status of each component
	ComponentRestarts  map[string]int64          // Restart count per component
	ComponentLastError map[string]ComponentError // Last error per component

	// Resource metrics
	MemoryUsage       int64   // Current memory usage in bytes
	CPUUsage          float64 // Current CPU usage percentage
	GoroutineCount    int32   // Active goroutines
	SubscriptionCount int32   // Active subscriptions
	OverlayCount      int32   // Active overlays

	// Performance metrics
	EventProcessingRate  *RateTracker // Events processed per second
	NotificationsSent    int64        // Total notifications sent
	AlertsTriggered      int64        // Alerts triggered
	DataCollectionCycles int64        // Data collection cycles completed

	// Health metrics
	UptimeStart     time.Time // Plugin start time
	LastHealthCheck time.Time // Last health check time
	HealthScore     float64   // Overall health score (0-100)

	// Lock for thread-safe operations
	mu sync.RWMutex

	// Context for cleanup
	ctx    context.Context
	cancel context.CancelFunc
}

// ComponentState represents the state of a plugin component
type ComponentState struct {
	Status      string // running, stopped, error, initializing
	LastUpdated time.Time
	Details     string // Additional status details
}

// ComponentError represents an error in a component
type ComponentError struct {
	Error     string
	Timestamp time.Time
	Severity  string // low, medium, high, critical
}

// ConnectionFailure represents a failed connection attempt
type ConnectionFailure struct {
	Timestamp time.Time
	Error     string
	Duration  time.Duration
}

// LatencyTracker tracks latency statistics
type LatencyTracker struct {
	mu         sync.RWMutex
	samples    []time.Duration
	maxSamples int
	totalCount int64
	totalTime  time.Duration
}

// RateTracker tracks rate statistics (events per second)
type RateTracker struct {
	mu         sync.RWMutex
	events     []time.Time
	windowSize time.Duration
}

// NewPluginMetrics creates a new plugin metrics instance
func NewPluginMetrics(ctx context.Context) *PluginMetrics {
	metricsCtx, cancel := context.WithCancel(ctx)

	pm := &PluginMetrics{
		ConnectionLatency:   NewLatencyTracker(1000), // Keep last 1000 samples
		QueryLatency:        NewLatencyTracker(5000), // Keep last 5000 samples
		ComponentStatus:     make(map[string]ComponentState),
		ComponentRestarts:   make(map[string]int64),
		ComponentLastError:  make(map[string]ComponentError),
		EventProcessingRate: NewRateTracker(time.Minute), // 1 minute window
		UptimeStart:         time.Now(),
		LastHealthCheck:     time.Now(),
		HealthScore:         100.0, // Start with perfect health
		ctx:                 metricsCtx,
		cancel:              cancel,
	}

	// Start background monitoring
	go pm.startBackgroundTasks()

	return pm
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker(maxSamples int) *LatencyTracker {
	return &LatencyTracker{
		samples:    make([]time.Duration, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// NewRateTracker creates a new rate tracker
func NewRateTracker(windowSize time.Duration) *RateTracker {
	return &RateTracker{
		events:     make([]time.Time, 0),
		windowSize: windowSize,
	}
}

// Connection Metrics

// RecordConnection records a successful connection
func (pm *PluginMetrics) RecordConnection(latency time.Duration) {
	atomic.AddInt64(&pm.PrometheusConnections, 1)
	pm.ConnectionLatency.Record(latency)
	pm.mu.Lock()
	pm.LastConnectionTime = time.Now()
	pm.mu.Unlock()
}

// RecordConnectionFailure records a failed connection
func (pm *PluginMetrics) RecordConnectionFailure(err error, duration time.Duration) {
	atomic.AddInt64(&pm.FailedConnections, 1)

	failure := ConnectionFailure{
		Timestamp: time.Now(),
		Error:     err.Error(),
		Duration:  duration,
	}

	pm.mu.Lock()
	pm.ConnectionFailures = append(pm.ConnectionFailures, failure)

	// Keep only last 100 failures
	if len(pm.ConnectionFailures) > 100 {
		pm.ConnectionFailures = pm.ConnectionFailures[len(pm.ConnectionFailures)-100:]
	}
	pm.mu.Unlock()
}

// Query Metrics

// RecordQuery records a query execution
func (pm *PluginMetrics) RecordQuery(latency time.Duration, success bool) {
	atomic.AddInt64(&pm.QueryCount, 1)
	if !success {
		atomic.AddInt64(&pm.QueryFailures, 1)
	}
	pm.QueryLatency.Record(latency)
}

// RecordBatchQuery records a batch query execution
func (pm *PluginMetrics) RecordBatchQuery() {
	atomic.AddInt64(&pm.BatchQueryCount, 1)
}

// RecordStreamingQuery records a streaming query execution
func (pm *PluginMetrics) RecordStreamingQuery() {
	atomic.AddInt64(&pm.StreamingQueryCount, 1)
}

// UpdateCacheHitRate updates the cache hit rate
func (pm *PluginMetrics) UpdateCacheHitRate(hitRate float64) {
	pm.mu.Lock()
	pm.CacheHitRate = hitRate
	pm.mu.Unlock()
}

// Component Metrics

// UpdateComponentStatus updates the status of a component
func (pm *PluginMetrics) UpdateComponentStatus(component, status, details string) {
	pm.mu.Lock()
	pm.ComponentStatus[component] = ComponentState{
		Status:      status,
		LastUpdated: time.Now(),
		Details:     details,
	}
	pm.mu.Unlock()
}

// RecordComponentRestart records a component restart
func (pm *PluginMetrics) RecordComponentRestart(component string) {
	pm.mu.Lock()
	pm.ComponentRestarts[component]++
	pm.mu.Unlock()
}

// RecordComponentError records a component error
func (pm *PluginMetrics) RecordComponentError(component string, err error, severity string) {
	pm.mu.Lock()
	pm.ComponentLastError[component] = ComponentError{
		Error:     err.Error(),
		Timestamp: time.Now(),
		Severity:  severity,
	}
	pm.mu.Unlock()
}

// Resource Metrics

// UpdateResourceUsage updates resource usage metrics
func (pm *PluginMetrics) UpdateResourceUsage(memoryBytes int64, cpuPercent float64, goroutines int32) {
	atomic.StoreInt64(&pm.MemoryUsage, memoryBytes)
	pm.mu.Lock()
	pm.CPUUsage = cpuPercent
	pm.mu.Unlock()
	atomic.StoreInt32(&pm.GoroutineCount, goroutines)
}

// UpdateSubscriptionCount updates the active subscription count
func (pm *PluginMetrics) UpdateSubscriptionCount(count int32) {
	atomic.StoreInt32(&pm.SubscriptionCount, count)
}

// UpdateOverlayCount updates the active overlay count
func (pm *PluginMetrics) UpdateOverlayCount(count int32) {
	atomic.StoreInt32(&pm.OverlayCount, count)
}

// Performance Metrics

// RecordEvent records an event for rate tracking
func (pm *PluginMetrics) RecordEvent() {
	pm.EventProcessingRate.Record(time.Now())
}

// RecordNotification records a sent notification
func (pm *PluginMetrics) RecordNotification() {
	atomic.AddInt64(&pm.NotificationsSent, 1)
}

// RecordAlert records a triggered alert
func (pm *PluginMetrics) RecordAlert() {
	atomic.AddInt64(&pm.AlertsTriggered, 1)
}

// RecordDataCollectionCycle records a completed data collection cycle
func (pm *PluginMetrics) RecordDataCollectionCycle() {
	atomic.AddInt64(&pm.DataCollectionCycles, 1)
}

// Health Metrics

// UpdateHealthScore updates the overall health score
func (pm *PluginMetrics) UpdateHealthScore(score float64) {
	pm.mu.Lock()
	pm.HealthScore = score
	pm.LastHealthCheck = time.Now()
	pm.mu.Unlock()
}

// Getters

// GetUptime returns the plugin uptime
func (pm *PluginMetrics) GetUptime() time.Duration {
	return time.Since(pm.UptimeStart)
}

// GetConnectionStats returns connection statistics
func (pm *PluginMetrics) GetConnectionStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	successfulConnections := atomic.LoadInt64(&pm.PrometheusConnections)
	failedConnections := atomic.LoadInt64(&pm.FailedConnections)
	totalConnections := successfulConnections + failedConnections
	successRate := 100.0
	if totalConnections > 0 {
		successRate = float64(successfulConnections) / float64(totalConnections) * 100
	}

	return map[string]interface{}{
		"total_connections":    totalConnections,
		"failed_connections":   failedConnections,
		"success_rate":         successRate,
		"avg_latency":          pm.ConnectionLatency.Average(),
		"p95_latency":          pm.ConnectionLatency.Percentile(95),
		"last_connection_time": pm.LastConnectionTime,
		"recent_failures":      len(pm.ConnectionFailures),
	}
}

// GetQueryStats returns query statistics
func (pm *PluginMetrics) GetQueryStats() map[string]interface{} {
	pm.mu.RLock()
	cacheHitRate := pm.CacheHitRate
	pm.mu.RUnlock()

	totalQueries := atomic.LoadInt64(&pm.QueryCount)
	failedQueries := atomic.LoadInt64(&pm.QueryFailures)
	batchQueries := atomic.LoadInt64(&pm.BatchQueryCount)
	streamingQueries := atomic.LoadInt64(&pm.StreamingQueryCount)

	successRate := 100.0
	if totalQueries > 0 {
		successRate = float64(totalQueries-failedQueries) / float64(totalQueries) * 100
	}

	return map[string]interface{}{
		"total_queries":     totalQueries,
		"failed_queries":    failedQueries,
		"batch_queries":     batchQueries,
		"streaming_queries": streamingQueries,
		"success_rate":      successRate,
		"cache_hit_rate":    cacheHitRate,
		"avg_latency":       pm.QueryLatency.Average(),
		"p50_latency":       pm.QueryLatency.Percentile(50),
		"p95_latency":       pm.QueryLatency.Percentile(95),
		"p99_latency":       pm.QueryLatency.Percentile(99),
	}
}

// GetComponentStats returns component statistics
func (pm *PluginMetrics) GetComponentStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return map[string]interface{}{
		"component_status":   pm.ComponentStatus,
		"component_restarts": pm.ComponentRestarts,
		"component_errors":   pm.ComponentLastError,
	}
}

// GetResourceStats returns resource statistics
func (pm *PluginMetrics) GetResourceStats() map[string]interface{} {
	pm.mu.RLock()
	cpuUsage := pm.CPUUsage
	pm.mu.RUnlock()

	return map[string]interface{}{
		"memory_usage_bytes": atomic.LoadInt64(&pm.MemoryUsage),
		"cpu_usage_percent":  cpuUsage,
		"goroutine_count":    atomic.LoadInt32(&pm.GoroutineCount),
		"subscription_count": atomic.LoadInt32(&pm.SubscriptionCount),
		"overlay_count":      atomic.LoadInt32(&pm.OverlayCount),
	}
}

// GetPerformanceStats returns performance statistics
func (pm *PluginMetrics) GetPerformanceStats() map[string]interface{} {
	return map[string]interface{}{
		"event_processing_rate":  pm.EventProcessingRate.Rate(),
		"notifications_sent":     atomic.LoadInt64(&pm.NotificationsSent),
		"alerts_triggered":       atomic.LoadInt64(&pm.AlertsTriggered),
		"data_collection_cycles": atomic.LoadInt64(&pm.DataCollectionCycles),
	}
}

// GetHealthStats returns health statistics
func (pm *PluginMetrics) GetHealthStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return map[string]interface{}{
		"uptime":            pm.GetUptime(),
		"health_score":      pm.HealthScore,
		"last_health_check": pm.LastHealthCheck,
	}
}

// GetAllStats returns all statistics
func (pm *PluginMetrics) GetAllStats() map[string]interface{} {
	return map[string]interface{}{
		"connections": pm.GetConnectionStats(),
		"queries":     pm.GetQueryStats(),
		"components":  pm.GetComponentStats(),
		"resources":   pm.GetResourceStats(),
		"performance": pm.GetPerformanceStats(),
		"health":      pm.GetHealthStats(),
	}
}

// LatencyTracker methods

// Record records a latency sample
func (lt *LatencyTracker) Record(duration time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.totalCount++
	lt.totalTime += duration

	if len(lt.samples) >= lt.maxSamples {
		// Remove oldest sample
		lt.samples = lt.samples[1:]
	}
	lt.samples = append(lt.samples, duration)
}

// Average returns the average latency
func (lt *LatencyTracker) Average() time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if lt.totalCount == 0 {
		return 0
	}
	return lt.totalTime / time.Duration(lt.totalCount)
}

// Percentile returns the specified percentile
func (lt *LatencyTracker) Percentile(percentile float64) time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.samples) == 0 {
		return 0
	}

	// Create a copy and sort
	samples := make([]time.Duration, len(lt.samples))
	copy(samples, lt.samples)

	// Simple bubble sort for small datasets
	for i := 0; i < len(samples); i++ {
		for j := 0; j < len(samples)-i-1; j++ {
			if samples[j] > samples[j+1] {
				samples[j], samples[j+1] = samples[j+1], samples[j]
			}
		}
	}

	index := int(percentile/100.0*float64(len(samples))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(samples) {
		index = len(samples) - 1
	}

	return samples[index]
}

// RateTracker methods

// Record records an event
func (rt *RateTracker) Record(timestamp time.Time) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.events = append(rt.events, timestamp)

	// Clean up old events outside the window
	cutoff := timestamp.Add(-rt.windowSize)
	for i, event := range rt.events {
		if event.After(cutoff) {
			rt.events = rt.events[i:]
			break
		}
	}
}

// Rate returns the current rate (events per second)
func (rt *RateTracker) Rate() float64 {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rt.windowSize)

	count := 0
	for _, event := range rt.events {
		if event.After(cutoff) {
			count++
		}
	}

	return float64(count) / rt.windowSize.Seconds()
}

// startBackgroundTasks starts background monitoring tasks
func (pm *PluginMetrics) startBackgroundTasks() {
	ticker := time.NewTicker(30 * time.Second) // Update every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.collectSystemMetrics()
			pm.calculateHealthScore()
		}
	}
}

// collectSystemMetrics collects system-level metrics
func (pm *PluginMetrics) collectSystemMetrics() {
	// This would typically use runtime.ReadMemStats() and other system calls
	// For now, we'll simulate the collection

	// Update goroutine count (this would be runtime.NumGoroutine())
	// atomic.StoreInt32(&pm.GoroutineCount, int32(runtime.NumGoroutine()))
}

// calculateHealthScore calculates the overall health score
func (pm *PluginMetrics) calculateHealthScore() {
	score := 100.0

	// Reduce score based on various factors
	failureRate := float64(atomic.LoadInt64(&pm.FailedConnections)) / float64(atomic.LoadInt64(&pm.PrometheusConnections)+1)
	if failureRate > 0.1 { // More than 10% failure rate
		score -= failureRate * 50 // Reduce by up to 50 points
	}

	queryFailureRate := float64(atomic.LoadInt64(&pm.QueryFailures)) / float64(atomic.LoadInt64(&pm.QueryCount)+1)
	if queryFailureRate > 0.05 { // More than 5% query failure rate
		score -= queryFailureRate * 30 // Reduce by up to 30 points
	}

	// Check component health
	pm.mu.RLock()
	errorCount := 0
	for _, status := range pm.ComponentStatus {
		if status.Status == "error" {
			errorCount++
		}
	}
	pm.mu.RUnlock()

	if errorCount > 0 {
		score -= float64(errorCount) * 10 // Reduce by 10 points per component error
	}

	if score < 0 {
		score = 0
	}

	pm.UpdateHealthScore(score)
}

// Stop stops the plugin metrics
func (pm *PluginMetrics) Stop() {
	if pm.cancel != nil {
		pm.cancel()
	}
}
