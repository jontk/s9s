package widgets

import (
	"runtime"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/performance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCalculateCPUUsage tests the CPU usage calculation function
func TestCalculateCPUUsage(t *testing.T) {
	tests := []struct {
		name          string
		stats         map[string]performance.OperationSummary
		previousStats map[string]performance.OperationSummary
		updateInterval time.Duration
		want          float64
	}{
		{
			name:  "empty stats returns 0.0",
			stats: map[string]performance.OperationSummary{},
			want:  0.0,
		},
		{
			name: "no previous stats returns 0.0",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			previousStats:  nil,
			updateInterval: 1 * time.Second,
			want:          0.0,
		},
		{
			name: "with previous stats calculates delta correctly",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 300 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          20.0, // (300ms - 100ms) / 1s * 100 = 20%
		},
		{
			name: "caps at 100% max",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     100,
					TotalTime: 5 * time.Second,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     50,
					TotalTime: 1 * time.Second,
				},
			},
			updateInterval: 1 * time.Second,
			want:          100.0, // Should cap at 100%
		},
		{
			name: "handles new operations not in previous stats",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 300 * time.Millisecond,
				},
				"operation2": {
					Name:      "operation2",
					Count:     5,
					TotalTime: 100 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          30.0, // (200ms + 100ms) / 1s * 100 = 30%
		},
		{
			name: "handles zero update interval",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 300 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 0,
			want:          20.0, // Defaults to 1 second window
		},
		{
			name: "multiple operations",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     30,
					TotalTime: 450 * time.Millisecond,
				},
				"operation2": {
					Name:      "operation2",
					Count:     15,
					TotalTime: 250 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 300 * time.Millisecond,
				},
				"operation2": {
					Name:      "operation2",
					Count:     10,
					TotalTime: 150 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          25.0, // (150ms + 100ms) / 1s * 100 = 25%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PerformanceDashboard{
				previousStats:  tt.previousStats,
				updateInterval: tt.updateInterval,
			}

			got := pd.calculateCPUUsage(tt.stats)
			assert.Equal(t, tt.want, got, "calculateCPUUsage() = %v, want %v", got, tt.want)
		})
	}
}

// TestCalculateMemoryUsage tests the memory usage calculation function
func TestCalculateMemoryUsage(t *testing.T) {
	tests := []struct {
		name     string
		memStats runtime.MemStats
		want     float64
	}{
		{
			name: "zero Sys returns 0.0",
			memStats: runtime.MemStats{
				Sys:       0,
				HeapInuse: 1024,
			},
			want: 0.0,
		},
		{
			name: "calculates HeapInuse/Sys percentage correctly",
			memStats: runtime.MemStats{
				Sys:       1000,
				HeapInuse: 500,
			},
			want: 50.0, // 500/1000 * 100 = 50%
		},
		{
			name: "handles typical memory stats",
			memStats: runtime.MemStats{
				Sys:       10 * 1024 * 1024, // 10 MB
				HeapInuse: 8 * 1024 * 1024,  // 8 MB
			},
			want: 80.0, // 8/10 * 100 = 80%
		},
		{
			name: "handles small values",
			memStats: runtime.MemStats{
				Sys:       100,
				HeapInuse: 1,
			},
			want: 1.0,
		},
		{
			name: "handles 100% usage",
			memStats: runtime.MemStats{
				Sys:       1024,
				HeapInuse: 1024,
			},
			want: 100.0,
		},
		{
			name: "handles zero HeapInuse",
			memStats: runtime.MemStats{
				Sys:       1024,
				HeapInuse: 0,
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PerformanceDashboard{}
			got := pd.calculateMemoryUsage(&tt.memStats)
			assert.Equal(t, tt.want, got, "calculateMemoryUsage() = %v, want %v", got, tt.want)
		})
	}
}

// TestCalculateNetworkUsage tests the network usage calculation function
func TestCalculateNetworkUsage(t *testing.T) {
	tests := []struct {
		name           string
		stats          map[string]performance.OperationSummary
		previousStats  map[string]performance.OperationSummary
		updateInterval time.Duration
		want           float64
	}{
		{
			name:  "nil stats returns 0.0",
			stats: nil,
			want:  0.0,
		},
		{
			name: "no previous stats returns 0.0",
			stats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			previousStats:  nil,
			updateInterval: 1 * time.Second,
			want:          0.0,
		},
		{
			name: "identifies network operations - ssh",
			stats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          1.0, // 10 ops/s * 0.1 MB/op = 1.0 MB/s
		},
		{
			name: "identifies network operations - api",
			stats: map[string]performance.OperationSummary{
				"api_call": {
					Name:      "api_call",
					Count:     30,
					TotalTime: 300 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"api_call": {
					Name:      "api_call",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          2.0, // 20 ops/s * 0.1 MB/op = 2.0 MB/s
		},
		{
			name: "identifies network operations - network in name",
			stats: map[string]performance.OperationSummary{
				"network_request": {
					Name:      "network_request",
					Count:     15,
					TotalTime: 150 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"network_request": {
					Name:      "network_request",
					Count:     5,
					TotalTime: 50 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          1.0, // 10 ops/s * 0.1 MB/op = 1.0 MB/s
		},
		{
			name: "calculates delta ops correctly - multiple network operations",
			stats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
				"api_call": {
					Name:      "api_call",
					Count:     30,
					TotalTime: 300 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
				"api_call": {
					Name:      "api_call",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          3.0, // (10 + 20) ops/s * 0.1 MB/op = 3.0 MB/s
		},
		{
			name: "ignores non-network operations",
			stats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
				"local_compute": {
					Name:      "local_compute",
					Count:     50,
					TotalTime: 500 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
				"local_compute": {
					Name:      "local_compute",
					Count:     25,
					TotalTime: 250 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          1.0, // Only counts ssh_connect: 10 ops/s * 0.1 = 1.0 MB/s
		},
		{
			name: "handles new network operations not in previous stats",
			stats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
				"api_new": {
					Name:      "api_new",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          2.0, // (10 + 10) ops/s * 0.1 = 2.0 MB/s
		},
		{
			name: "handles zero update interval",
			stats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 0,
			want:          1.0, // Defaults to 1 second window: 10 ops/s * 0.1 = 1.0 MB/s
		},
		{
			name: "handles case insensitivity",
			stats: map[string]performance.OperationSummary{
				"SSH_CONNECT": {
					Name:      "SSH_CONNECT",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
				"API_CALL": {
					Name:      "API_CALL",
					Count:     30,
					TotalTime: 300 * time.Millisecond,
				},
				"Network_Request": {
					Name:      "Network_Request",
					Count:     15,
					TotalTime: 150 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"SSH_CONNECT": {
					Name:      "SSH_CONNECT",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
				"API_CALL": {
					Name:      "API_CALL",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
				"Network_Request": {
					Name:      "Network_Request",
					Count:     5,
					TotalTime: 50 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          4.0, // (10 + 20 + 10) ops/s * 0.1 = 4.0 MB/s
		},
		{
			name: "handles 2 second interval",
			stats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     40,
					TotalTime: 400 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"ssh_connect": {
					Name:      "ssh_connect",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
			},
			updateInterval: 2 * time.Second,
			want:          1.0, // 20 ops / 2s = 10 ops/s * 0.1 = 1.0 MB/s
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PerformanceDashboard{
				previousStats:  tt.previousStats,
				updateInterval: tt.updateInterval,
			}

			got := pd.calculateNetworkUsage(tt.stats)
			assert.InDelta(t, tt.want, got, 0.01, "calculateNetworkUsage() = %v, want %v", got, tt.want)
		})
	}
}

// TestCalculateOpsRate tests the operations rate calculation function
func TestCalculateOpsRate(t *testing.T) {
	tests := []struct {
		name           string
		stats          map[string]performance.OperationSummary
		previousStats  map[string]performance.OperationSummary
		updateInterval time.Duration
		want           float64
	}{
		{
			name:  "nil stats returns 0.0",
			stats: nil,
			want:  0.0,
		},
		{
			name: "no previous stats returns 0.0",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			previousStats:  nil,
			updateInterval: 1 * time.Second,
			want:          0.0,
		},
		{
			name: "calculates delta operations correctly - single operation",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          10.0, // 10 ops / 1s = 10 ops/s
		},
		{
			name: "calculates delta operations correctly - multiple operations",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     30,
					TotalTime: 300 * time.Millisecond,
				},
				"operation2": {
					Name:      "operation2",
					Count:     25,
					TotalTime: 250 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
				"operation2": {
					Name:      "operation2",
					Count:     15,
					TotalTime: 150 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          20.0, // (10 + 10) ops / 1s = 20 ops/s
		},
		{
			name: "converts to ops/second based on updateInterval",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     40,
					TotalTime: 400 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
			},
			updateInterval: 2 * time.Second,
			want:          10.0, // 20 ops / 2s = 10 ops/s
		},
		{
			name: "handles new operations not in previous stats",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
				"operation2": {
					Name:      "operation2",
					Count:     15,
					TotalTime: 150 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          25.0, // (10 + 15) ops / 1s = 25 ops/s
		},
		{
			name: "handles zero update interval",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     30,
					TotalTime: 300 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
			},
			updateInterval: 0,
			want:          10.0, // Defaults to 1 second window: 10 ops / 1s = 10 ops/s
		},
		{
			name: "handles operations removed from stats",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     30,
					TotalTime: 300 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     20,
					TotalTime: 200 * time.Millisecond,
				},
				"operation2": {
					Name:      "operation2",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          10.0, // Only counts operation1: 10 ops / 1s = 10 ops/s
		},
		{
			name: "handles high operation rate",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     1500,
					TotalTime: 1500 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     500,
					TotalTime: 500 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          1000.0, // 1000 ops / 1s = 1000 ops/s
		},
		{
			name: "handles fractional seconds interval",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     15,
					TotalTime: 150 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 500 * time.Millisecond,
			want:          10.0, // 5 ops / 0.5s = 10 ops/s
		},
		{
			name: "handles zero delta operations",
			stats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			previousStats: map[string]performance.OperationSummary{
				"operation1": {
					Name:      "operation1",
					Count:     10,
					TotalTime: 100 * time.Millisecond,
				},
			},
			updateInterval: 1 * time.Second,
			want:          0.0, // No new operations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PerformanceDashboard{
				previousStats:  tt.previousStats,
				updateInterval: tt.updateInterval,
			}

			got := pd.calculateOpsRate(tt.stats)
			assert.InDelta(t, tt.want, got, 0.01, "calculateOpsRate() = %v, want %v", got, tt.want)
		})
	}
}

// TestAddToHistory tests the history management function
func TestAddToHistory(t *testing.T) {
	tests := []struct {
		name       string
		maxHistory int
		initial    []float64
		add        []float64
		wantLen    int
		wantLast   float64
	}{
		{
			name:       "adds to empty history",
			maxHistory: 10,
			initial:    []float64{},
			add:        []float64{1.0},
			wantLen:    1,
			wantLast:   1.0,
		},
		{
			name:       "adds multiple values",
			maxHistory: 10,
			initial:    []float64{1.0},
			add:        []float64{2.0, 3.0},
			wantLen:    3,
			wantLast:   3.0,
		},
		{
			name:       "respects max history limit",
			maxHistory: 3,
			initial:    []float64{1.0, 2.0, 3.0},
			add:        []float64{4.0},
			wantLen:    3,
			wantLast:   4.0,
		},
		{
			name:       "removes oldest when exceeding limit",
			maxHistory: 2,
			initial:    []float64{1.0, 2.0},
			add:        []float64{3.0, 4.0},
			wantLen:    2,
			wantLast:   4.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PerformanceDashboard{
				maxHistory: tt.maxHistory,
			}
			history := make([]float64, len(tt.initial))
			copy(history, tt.initial)

			for _, val := range tt.add {
				pd.addToHistory(&history, val)
			}

			require.Equal(t, tt.wantLen, len(history), "history length mismatch")
			assert.Equal(t, tt.wantLast, history[len(history)-1], "last value mismatch")
		})
	}
}

// TestCalculateAverage tests the average calculation helper
func TestCalculateAverage(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
	}{
		{
			name: "empty data returns 0",
			data: []float64{},
			want: 0.0,
		},
		{
			name: "single value",
			data: []float64{5.0},
			want: 5.0,
		},
		{
			name: "multiple values",
			data: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			want: 3.0,
		},
		{
			name: "decimal values",
			data: []float64{1.5, 2.5, 3.5},
			want: 2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PerformanceDashboard{}
			got := pd.calculateAverage(tt.data)
			assert.InDelta(t, tt.want, got, 0.01, "calculateAverage() = %v, want %v", got, tt.want)
		})
	}
}

// TestCalculateMax tests the max calculation helper
func TestCalculateMax(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
	}{
		{
			name: "empty data returns 0",
			data: []float64{},
			want: 0.0,
		},
		{
			name: "single value",
			data: []float64{5.0},
			want: 5.0,
		},
		{
			name: "multiple values",
			data: []float64{1.0, 5.0, 3.0, 2.0, 4.0},
			want: 5.0,
		},
		{
			name: "negative values",
			data: []float64{-5.0, -1.0, -10.0, -3.0},
			want: -1.0,
		},
		{
			name: "mixed values",
			data: []float64{-5.0, 0.0, 10.0, 3.0},
			want: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PerformanceDashboard{}
			got := pd.calculateMax(tt.data)
			assert.Equal(t, tt.want, got, "calculateMax() = %v, want %v", got, tt.want)
		})
	}
}

// BenchmarkCalculateCPUUsage benchmarks CPU usage calculation
func BenchmarkCalculateCPUUsage(b *testing.B) {
	stats := map[string]performance.OperationSummary{
		"operation1": {
			Name:      "operation1",
			Count:     100,
			TotalTime: 1000 * time.Millisecond,
		},
		"operation2": {
			Name:      "operation2",
			Count:     200,
			TotalTime: 2000 * time.Millisecond,
		},
	}
	previousStats := map[string]performance.OperationSummary{
		"operation1": {
			Name:      "operation1",
			Count:     50,
			TotalTime: 500 * time.Millisecond,
		},
		"operation2": {
			Name:      "operation2",
			Count:     100,
			TotalTime: 1000 * time.Millisecond,
		},
	}

	pd := &PerformanceDashboard{
		previousStats:  previousStats,
		updateInterval: 1 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pd.calculateCPUUsage(stats)
	}
}

// BenchmarkCalculateNetworkUsage benchmarks network usage calculation
func BenchmarkCalculateNetworkUsage(b *testing.B) {
	stats := map[string]performance.OperationSummary{
		"ssh_connect": {
			Name:      "ssh_connect",
			Count:     100,
			TotalTime: 1000 * time.Millisecond,
		},
		"api_call": {
			Name:      "api_call",
			Count:     200,
			TotalTime: 2000 * time.Millisecond,
		},
		"local_operation": {
			Name:      "local_operation",
			Count:     300,
			TotalTime: 3000 * time.Millisecond,
		},
	}
	previousStats := map[string]performance.OperationSummary{
		"ssh_connect": {
			Name:      "ssh_connect",
			Count:     50,
			TotalTime: 500 * time.Millisecond,
		},
		"api_call": {
			Name:      "api_call",
			Count:     100,
			TotalTime: 1000 * time.Millisecond,
		},
		"local_operation": {
			Name:      "local_operation",
			Count:     150,
			TotalTime: 1500 * time.Millisecond,
		},
	}

	pd := &PerformanceDashboard{
		previousStats:  previousStats,
		updateInterval: 1 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pd.calculateNetworkUsage(stats)
	}
}

// BenchmarkCalculateOpsRate benchmarks operations rate calculation
func BenchmarkCalculateOpsRate(b *testing.B) {
	stats := map[string]performance.OperationSummary{
		"operation1": {
			Name:      "operation1",
			Count:     1000,
			TotalTime: 1000 * time.Millisecond,
		},
		"operation2": {
			Name:      "operation2",
			Count:     2000,
			TotalTime: 2000 * time.Millisecond,
		},
		"operation3": {
			Name:      "operation3",
			Count:     3000,
			TotalTime: 3000 * time.Millisecond,
		},
	}
	previousStats := map[string]performance.OperationSummary{
		"operation1": {
			Name:      "operation1",
			Count:     500,
			TotalTime: 500 * time.Millisecond,
		},
		"operation2": {
			Name:      "operation2",
			Count:     1000,
			TotalTime: 1000 * time.Millisecond,
		},
		"operation3": {
			Name:      "operation3",
			Count:     1500,
			TotalTime: 1500 * time.Millisecond,
		},
	}

	pd := &PerformanceDashboard{
		previousStats:  previousStats,
		updateInterval: 1 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pd.calculateOpsRate(stats)
	}
}
