// Package models defines data structures and types for representing metrics,
// nodes, jobs, and other observability entities. It provides standardized
// interfaces for metric collection, type-safe data handling, and consistent
// serialization across the observability plugin components.
package models

import (
	"fmt"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCPU         MetricType = "cpu"
	MetricTypeMemory      MetricType = "memory"
	MetricTypeDisk        MetricType = "disk"
	MetricTypeNetwork     MetricType = "network"
	MetricTypeLoad        MetricType = "load"
	MetricTypeTemperature MetricType = "temperature"
	MetricTypeCustom      MetricType = "custom"
)

// MetricValue represents a single metric measurement
type MetricValue struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Unit      string                 `json:"unit"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TimeSeries represents a time series of metric values
type TimeSeries struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Values []MetricValue     `json:"values"`
	Unit   string            `json:"unit"`
	Type   MetricType        `json:"type"`
}

// Add adds a metric value to the time series
func (ts *TimeSeries) Add(value MetricValue) {
	ts.Values = append(ts.Values, value)
}

// Latest returns the most recent value
func (ts *TimeSeries) Latest() *MetricValue {
	if len(ts.Values) == 0 {
		return nil
	}
	return &ts.Values[len(ts.Values)-1]
}

// Average calculates the average value over the time series
func (ts *TimeSeries) Average() float64 {
	if len(ts.Values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range ts.Values {
		sum += v.Value
	}
	return sum / float64(len(ts.Values))
}

// Min returns the minimum value in the time series
func (ts *TimeSeries) Min() float64 {
	if len(ts.Values) == 0 {
		return 0
	}

	min := ts.Values[0].Value
	for _, v := range ts.Values[1:] {
		if v.Value < min {
			min = v.Value
		}
	}
	return min
}

// Max returns the maximum value in the time series
func (ts *TimeSeries) Max() float64 {
	if len(ts.Values) == 0 {
		return 0
	}

	max := ts.Values[0].Value
	for _, v := range ts.Values[1:] {
		if v.Value > max {
			max = v.Value
		}
	}
	return max
}

// MetricCollection represents a collection of related metrics
type MetricCollection struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Type       string                `json:"type"` // "node", "job", "cluster"
	Metrics    map[string]TimeSeries `json:"metrics"`
	LastUpdate time.Time             `json:"last_update"`
}

// NewMetricCollection creates a new metric collection
func NewMetricCollection(id, name, collectionType string) *MetricCollection {
	return &MetricCollection{
		ID:      id,
		Name:    name,
		Type:    collectionType,
		Metrics: make(map[string]TimeSeries),
	}
}

// AddMetric adds a time series to the collection
func (mc *MetricCollection) AddMetric(name string, ts TimeSeries) {
	mc.Metrics[name] = ts
	mc.LastUpdate = time.Now()
}

// GetMetric retrieves a specific metric by name
func (mc *MetricCollection) GetMetric(name string) (*TimeSeries, bool) {
	ts, exists := mc.Metrics[name]
	return &ts, exists
}

// ResourceMetrics represents aggregated resource metrics
type ResourceMetrics struct {
	CPU       CPUMetrics     `json:"cpu"`
	Memory    MemoryMetrics  `json:"memory"`
	Disk      DiskMetrics    `json:"disk"`
	Network   NetworkMetrics `json:"network"`
	Timestamp time.Time      `json:"timestamp"`
}

// CPUMetrics represents CPU-related metrics
type CPUMetrics struct {
	Usage     float64 `json:"usage"`     // Percentage (0-100)
	Cores     int     `json:"cores"`     // Number of cores
	Load1m    float64 `json:"load_1m"`   // 1-minute load average
	Load5m    float64 `json:"load_5m"`   // 5-minute load average
	Load15m   float64 `json:"load_15m"`  // 15-minute load average
	Throttled float64 `json:"throttled"` // Throttled percentage
	System    float64 `json:"system"`    // System CPU percentage
	User      float64 `json:"user"`      // User CPU percentage
	IOWait    float64 `json:"io_wait"`   // IO wait percentage
	Limit     float64 `json:"limit"`     // CPU limit (cores or millicores)
}

// MemoryMetrics represents memory-related metrics
type MemoryMetrics struct {
	Total     uint64  `json:"total"`      // Total memory in bytes
	Used      uint64  `json:"used"`       // Used memory in bytes
	Available uint64  `json:"available"`  // Available memory in bytes
	Cache     uint64  `json:"cache"`      // Cache memory in bytes
	Buffer    uint64  `json:"buffer"`     // Buffer memory in bytes
	Usage     float64 `json:"usage"`      // Usage percentage (0-100)
	SwapTotal uint64  `json:"swap_total"` // Total swap in bytes
	SwapUsed  uint64  `json:"swap_used"`  // Used swap in bytes
	Limit     uint64  `json:"limit"`      // Memory limit in bytes (for containers/jobs)
}

// DiskMetrics represents disk I/O metrics
type DiskMetrics struct {
	ReadBytesPerSec  float64 `json:"read_bytes_per_sec"`  // Bytes/sec
	WriteBytesPerSec float64 `json:"write_bytes_per_sec"` // Bytes/sec
	ReadOpsPerSec    float64 `json:"read_ops_per_sec"`    // Operations/sec
	WriteOpsPerSec   float64 `json:"write_ops_per_sec"`   // Operations/sec
	IOUtilization    float64 `json:"io_utilization"`      // Percentage (0-100)
}

// NetworkMetrics represents network I/O metrics
type NetworkMetrics struct {
	ReceiveBytesPerSec    float64 `json:"receive_bytes_per_sec"`    // Bytes/sec
	TransmitBytesPerSec   float64 `json:"transmit_bytes_per_sec"`   // Bytes/sec
	ReceivePacketsPerSec  float64 `json:"receive_packets_per_sec"`  // Packets/sec
	TransmitPacketsPerSec float64 `json:"transmit_packets_per_sec"` // Packets/sec
	ReceiveErrors         uint64  `json:"receive_errors"`           // Total errors
	TransmitErrors        uint64  `json:"transmit_errors"`          // Total errors
}

// AggregationFunc represents a function for aggregating metric values
type AggregationFunc func([]float64) float64

// CommonAggregations provides common aggregation functions
var CommonAggregations = map[string]AggregationFunc{
	"avg": func(values []float64) float64 {
		if len(values) == 0 {
			return 0
		}
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))
	},
	"min": func(values []float64) float64 {
		if len(values) == 0 {
			return 0
		}
		min := values[0]
		for _, v := range values[1:] {
			if v < min {
				min = v
			}
		}
		return min
	},
	"max": func(values []float64) float64 {
		if len(values) == 0 {
			return 0
		}
		max := values[0]
		for _, v := range values[1:] {
			if v > max {
				max = v
			}
		}
		return max
	},
	"sum": func(values []float64) float64 {
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum
	},
	"count": func(values []float64) float64 {
		return float64(len(values))
	},
}

// FormatValue formats a metric value with appropriate units
func FormatValue(value float64, unit string) string {
	switch unit {
	case "bytes":
		return formatBytes(value)
	case "bytes/sec":
		return formatBytes(value) + "/s"
	case "percent":
		return fmt.Sprintf("%.1f%%", value)
	case "cores":
		return fmt.Sprintf("%.2f", value)
	case "load":
		return fmt.Sprintf("%.2f", value)
	default:
		return fmt.Sprintf("%.2f %s", value, unit)
	}
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	if bytes < 1 {
		return "0 B"
	}

	i := 0
	for bytes >= 1024 && i < len(units)-1 {
		bytes /= 1024
		i++
	}

	if i == 0 {
		return fmt.Sprintf("%.0f %s", bytes, units[i])
	}
	return fmt.Sprintf("%.1f %s", bytes, units[i])
}

// GetColorForUsage returns a color based on usage percentage
func GetColorForUsage(usage float64) string {
	switch {
	case usage >= 90:
		return "red"
	case usage >= 75:
		return "yellow"
	case usage >= 50:
		return "orange"
	default:
		return "green"
	}
}

// Alert represents a monitoring alert
type Alert struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Severity   string            `json:"severity"` // "info", "warning", "critical"
	State      string            `json:"state"`    // "pending", "firing", "resolved"
	Message    string            `json:"message"`
	Resolution string            `json:"resolution,omitempty"`
	Source     string            `json:"source"`
	Metric     string            `json:"metric"`
	Value      float64           `json:"value"`
	Threshold  float64           `json:"threshold"`
	Timestamp  time.Time         `json:"timestamp"`
	Resolved   bool              `json:"resolved"`
	ResolvedAt time.Time         `json:"resolved_at,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}
