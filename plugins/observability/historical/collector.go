// Package historical provides time-series data collection and analysis capabilities.
// It implements persistent storage for historical metrics, statistical analysis,
// trend detection, anomaly identification, and seasonal pattern recognition.
// The package supports configurable retention policies and efficient data aggregation.
package historical

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/plugins/observability/prometheus"
)

// DataPoint represents a single historical data point
type DataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     interface{}            `json:"value"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricSeries represents a time series of metric data
type MetricSeries struct {
	MetricName string            `json:"metric_name"`
	Labels     map[string]string `json:"labels"`
	DataPoints []DataPoint       `json:"data_points"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// HistoricalDataCollector collects and stores historical metric data
type HistoricalDataCollector struct {
	client          *prometheus.CachedClient
	dataDir         string
	retention       time.Duration
	collectInterval time.Duration
	maxDataPoints   int

	series   map[string]*MetricSeries
	queries  map[string]string
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

// CollectorConfig configuration for historical data collector
type CollectorConfig struct {
	DataDir         string            `json:"data_dir"`
	Retention       time.Duration     `json:"retention"`
	CollectInterval time.Duration     `json:"collect_interval"`
	MaxDataPoints   int               `json:"max_data_points"`
	Queries         map[string]string `json:"queries"`
}

// DefaultCollectorConfig returns default configuration
func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		DataDir:         "./data/historical",
		Retention:       30 * 24 * time.Hour, // 30 days
		CollectInterval: 5 * time.Minute,
		MaxDataPoints:   10000,
		Queries: map[string]string{
			"node_cpu":     `100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`,
			"node_memory":  `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100`,
			"node_load":    `node_load1`,
			"job_count":    `slurm_job_total`,
			"queue_length": `slurm_queue_pending_jobs`,
		},
	}
}

// NewHistoricalDataCollector creates a new historical data collector
func NewHistoricalDataCollector(client *prometheus.CachedClient, config CollectorConfig) (*HistoricalDataCollector, error) {
	if config.DataDir == "" {
		config.DataDir = "./data/historical"
	}

	if config.Retention == 0 {
		config.Retention = 30 * 24 * time.Hour
	}

	if config.CollectInterval == 0 {
		config.CollectInterval = 5 * time.Minute
	}

	if config.MaxDataPoints == 0 {
		config.MaxDataPoints = 10000
	}

	if config.Queries == nil {
		config.Queries = DefaultCollectorConfig().Queries
	}

	// Ensure data directory exists
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	collector := &HistoricalDataCollector{
		client:          client,
		dataDir:         config.DataDir,
		retention:       config.Retention,
		collectInterval: config.CollectInterval,
		maxDataPoints:   config.MaxDataPoints,
		series:          make(map[string]*MetricSeries),
		queries:         config.Queries,
		stopChan:        make(chan struct{}),
	}

	// Load existing data (log error but continue initialization)
	_ = collector.loadHistoricalData()

	return collector, nil
}

// Start starts the historical data collection
func (hdc *HistoricalDataCollector) Start(ctx context.Context) error {
	hdc.mu.Lock()
	defer hdc.mu.Unlock()

	if hdc.running {
		return fmt.Errorf("historical data collector is already running")
	}

	hdc.running = true
	hdc.stopChan = make(chan struct{})

	// Start collection loop
	go hdc.collectionLoop(ctx)

	// Start cleanup loop
	go hdc.cleanupLoop()

	return nil
}

// Stop stops the historical data collection
func (hdc *HistoricalDataCollector) Stop() error {
	hdc.mu.Lock()
	defer hdc.mu.Unlock()

	if !hdc.running {
		return fmt.Errorf("historical data collector is not running")
	}

	hdc.running = false
	close(hdc.stopChan)

	// Save current data (already holding lock)
	hdc.saveHistoricalDataLocked()

	return nil
}

// AddQuery adds a new query for historical collection
func (hdc *HistoricalDataCollector) AddQuery(name, query string) {
	hdc.mu.Lock()
	defer hdc.mu.Unlock()

	hdc.queries[name] = query
}

// RemoveQuery removes a query from historical collection
func (hdc *HistoricalDataCollector) RemoveQuery(name string) {
	hdc.mu.Lock()
	defer hdc.mu.Unlock()

	delete(hdc.queries, name)
	delete(hdc.series, name)
}

// GetHistoricalData returns historical data for a metric
func (hdc *HistoricalDataCollector) GetHistoricalData(metricName string, start, end time.Time) (*MetricSeries, error) {
	hdc.mu.RLock()
	defer hdc.mu.RUnlock()

	series, exists := hdc.series[metricName]
	if !exists {
		return nil, fmt.Errorf("metric not found: %s", metricName)
	}

	// Filter data points by time range
	filteredSeries := &MetricSeries{
		MetricName: series.MetricName,
		Labels:     series.Labels,
		DataPoints: make([]DataPoint, 0),
		CreatedAt:  series.CreatedAt,
		UpdatedAt:  series.UpdatedAt,
	}

	for _, dp := range series.DataPoints {
		if dp.Timestamp.After(start) && dp.Timestamp.Before(end) {
			filteredSeries.DataPoints = append(filteredSeries.DataPoints, dp)
		}
	}

	return filteredSeries, nil
}

// GetAvailableMetrics returns list of available historical metrics
func (hdc *HistoricalDataCollector) GetAvailableMetrics() []string {
	hdc.mu.RLock()
	defer hdc.mu.RUnlock()

	metrics := make([]string, 0, len(hdc.series))
	for name := range hdc.series {
		metrics = append(metrics, name)
	}

	sort.Strings(metrics)
	return metrics
}

// GetMetricStatistics returns statistics for a metric
func (hdc *HistoricalDataCollector) GetMetricStatistics(metricName string, duration time.Duration) (map[string]interface{}, error) {
	end := time.Now()
	start := end.Add(-duration)

	series, err := hdc.GetHistoricalData(metricName, start, end)
	if err != nil {
		return nil, err
	}

	if len(series.DataPoints) == 0 {
		return map[string]interface{}{
			"count": 0,
			"error": "no data points in range",
		}, nil
	}

	// Calculate statistics
	var sum, min, max float64
	validPoints := 0
	first := true

	for _, dp := range series.DataPoints {
		if val, ok := convertToFloat64(dp.Value); ok {
			if first {
				min = val
				max = val
				first = false
			} else {
				if val < min {
					min = val
				}
				if val > max {
					max = val
				}
			}
			sum += val
			validPoints++
		}
	}

	if validPoints == 0 {
		return map[string]interface{}{
			"count": len(series.DataPoints),
			"error": "no valid numeric data points",
		}, nil
	}

	avg := sum / float64(validPoints)

	return map[string]interface{}{
		"count":    validPoints,
		"min":      min,
		"max":      max,
		"average":  avg,
		"sum":      sum,
		"timespan": duration.String(),
		"first":    series.DataPoints[0].Timestamp,
		"last":     series.DataPoints[len(series.DataPoints)-1].Timestamp,
	}, nil
}

// GetCollectorStats returns collector statistics
func (hdc *HistoricalDataCollector) GetCollectorStats() map[string]interface{} {
	hdc.mu.RLock()
	defer hdc.mu.RUnlock()

	totalDataPoints := 0
	oldestTimestamp := time.Now()
	newestTimestamp := time.Time{}

	for _, series := range hdc.series {
		totalDataPoints += len(series.DataPoints)
		if len(series.DataPoints) > 0 {
			firstPoint := series.DataPoints[0].Timestamp
			lastPoint := series.DataPoints[len(series.DataPoints)-1].Timestamp

			if firstPoint.Before(oldestTimestamp) {
				oldestTimestamp = firstPoint
			}
			if lastPoint.After(newestTimestamp) {
				newestTimestamp = lastPoint
			}
		}
	}

	return map[string]interface{}{
		"running":            hdc.running,
		"metrics_count":      len(hdc.series),
		"total_data_points":  totalDataPoints,
		"collect_interval":   hdc.collectInterval.String(),
		"retention_period":   hdc.retention.String(),
		"max_data_points":    hdc.maxDataPoints,
		"oldest_data":        oldestTimestamp,
		"newest_data":        newestTimestamp,
		"data_directory":     hdc.dataDir,
		"queries_configured": len(hdc.queries),
	}
}

// collectionLoop runs the data collection process
func (hdc *HistoricalDataCollector) collectionLoop(ctx context.Context) {
	ticker := time.NewTicker(hdc.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hdc.stopChan:
			return
		case <-ticker.C:
			hdc.collectData(ctx)
		}
	}
}

// collectData collects data for all configured queries
func (hdc *HistoricalDataCollector) collectData(ctx context.Context) {
	if hdc.client == nil {
		return
	}

	timestamp := time.Now()

	hdc.mu.RLock()
	queries := make(map[string]string)
	for name, query := range hdc.queries {
		queries[name] = query
	}
	hdc.mu.RUnlock()

	// Execute batch query
	results, err := hdc.client.BatchQuery(ctx, queries, timestamp)
	if err != nil {
		// Log error but continue
		return
	}

	hdc.mu.Lock()
	defer hdc.mu.Unlock()

	// Process results
	for metricName, result := range results {
		series, exists := hdc.series[metricName]
		if !exists {
			series = &MetricSeries{
				MetricName: metricName,
				Labels:     make(map[string]string),
				DataPoints: make([]DataPoint, 0),
				CreatedAt:  timestamp,
				UpdatedAt:  timestamp,
			}
			hdc.series[metricName] = series
		}

		// Add new data point
		dataPoint := DataPoint{
			Timestamp: timestamp,
			Value:     result,
			Metadata: map[string]interface{}{
				"query": hdc.queries[metricName],
			},
		}

		series.DataPoints = append(series.DataPoints, dataPoint)
		series.UpdatedAt = timestamp

		// Trim data points if necessary
		if len(series.DataPoints) > hdc.maxDataPoints {
			// Remove oldest points
			removeCount := len(series.DataPoints) - hdc.maxDataPoints
			series.DataPoints = series.DataPoints[removeCount:]
		}
	}

	// Periodic save (every 10 collections)
	if timestamp.Unix()%(int64(hdc.collectInterval.Seconds())*10) == 0 {
		go hdc.saveHistoricalData()
	}
}

// cleanupLoop runs periodic cleanup of old data
func (hdc *HistoricalDataCollector) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-hdc.stopChan:
			return
		case <-ticker.C:
			hdc.cleanupOldData()
		}
	}
}

// cleanupOldData removes data points older than retention period
func (hdc *HistoricalDataCollector) cleanupOldData() {
	cutoff := time.Now().Add(-hdc.retention)

	hdc.mu.Lock()
	defer hdc.mu.Unlock()

	for _, series := range hdc.series {
		// Find first data point to keep
		keepIndex := 0
		for i, dp := range series.DataPoints {
			if dp.Timestamp.After(cutoff) {
				keepIndex = i
				break
			}
		}

		// Remove old data points
		if keepIndex > 0 {
			series.DataPoints = series.DataPoints[keepIndex:]
		}
	}
}

// saveHistoricalData saves historical data to disk
func (hdc *HistoricalDataCollector) saveHistoricalData() {
	hdc.mu.RLock()
	defer hdc.mu.RUnlock()
	hdc.saveHistoricalDataLocked()
}

// saveHistoricalDataLocked saves historical data to disk (assumes lock is already held)
func (hdc *HistoricalDataCollector) saveHistoricalDataLocked() {
	for name, series := range hdc.series {
		filename := filepath.Join(hdc.dataDir, fmt.Sprintf("%s.json", name))
		data, err := json.MarshalIndent(series, "", "  ")
		if err != nil {
			continue
		}

		// Write to temporary file first
		tempFile := filename + ".tmp"
		if err := os.WriteFile(tempFile, data, 0644); err != nil {
			continue
		}

		// Atomic rename
		_ = os.Rename(tempFile, filename)
	}
}

// loadHistoricalData loads historical data from disk
func (hdc *HistoricalDataCollector) loadHistoricalData() error {
	files, err := filepath.Glob(filepath.Join(hdc.dataDir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var series MetricSeries
		if err := json.Unmarshal(data, &series); err != nil {
			continue
		}

		// Extract metric name from filename
		basename := filepath.Base(file)
		metricName := basename[:len(basename)-5] // Remove .json extension

		hdc.series[metricName] = &series
	}

	return nil
}

// ExportData exports historical data in various formats
func (hdc *HistoricalDataCollector) ExportData(metricName string, start, end time.Time, format string) ([]byte, error) {
	series, err := hdc.GetHistoricalData(metricName, start, end)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(series, "", "  ")
	case "csv":
		return hdc.exportToCSV(series)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// exportToCSV exports data to CSV format
func (hdc *HistoricalDataCollector) exportToCSV(series *MetricSeries) ([]byte, error) {
	var csv strings.Builder

	// Header
	csv.WriteString("timestamp,value\n")

	// Data
	for _, dp := range series.DataPoints {
		csv.WriteString(fmt.Sprintf("%s,%v\n", dp.Timestamp.Format(time.RFC3339), dp.Value))
	}

	return []byte(csv.String()), nil
}

// convertToFloat64 attempts to convert a value to float64
func convertToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}
