package performance

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/mathutil"
)

// MetricType represents different types of metrics
type MetricType string

const (
	MetricCPU       MetricType = "cpu"
	MetricMemory    MetricType = "memory"
	MetricGoroutine MetricType = "goroutine"
	MetricLatency   MetricType = "latency"
)

// Metric represents a performance metric
type Metric struct {
	Type      MetricType
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
}

// Profiler handles performance profiling
type Profiler struct {
	metrics []Metric
	mu      sync.RWMutex
	// TODO(lint): Review unused code - field cpuProfile is unused
	// cpuProfile         *pprof.Profile
	startTime          time.Time
	operations         map[string]*OperationStats
	memBaseline        runtime.MemStats
	baselineGoroutines int
}

// OperationStats tracks statistics for a specific operation
type OperationStats struct {
	Name      string
	Count     int64
	TotalTime time.Duration
	MinTime   time.Duration
	MaxTime   time.Duration
	LastTime  time.Duration
	mu        sync.Mutex
}

// NewProfiler creates a new profiler
func NewProfiler() *Profiler {
	p := &Profiler{
		metrics:    make([]Metric, 0),
		startTime:  time.Now(),
		operations: make(map[string]*OperationStats),
	}

	// Capture baseline memory stats
	runtime.ReadMemStats(&p.memBaseline)
	p.baselineGoroutines = runtime.NumGoroutine()

	return p
}

// StartOperation starts timing an operation
func (p *Profiler) StartOperation(name string) func() {
	start := time.Now()

	return func() {
		duration := time.Since(start)
		p.recordOperation(name, duration)
	}
}

// recordOperation records an operation's performance
func (p *Profiler) recordOperation(name string, duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats, exists := p.operations[name]
	if !exists {
		stats = &OperationStats{
			Name:    name,
			MinTime: duration,
			MaxTime: duration,
		}
		p.operations[name] = stats
	}

	stats.mu.Lock()
	stats.Count++
	stats.TotalTime += duration
	stats.LastTime = duration

	if duration < stats.MinTime {
		stats.MinTime = duration
	}
	if duration > stats.MaxTime {
		stats.MaxTime = duration
	}
	stats.mu.Unlock()

	// Record as metric
	p.metrics = append(p.metrics, Metric{
		Type:      MetricLatency,
		Name:      name,
		Value:     float64(duration.Microseconds()),
		Unit:      "μs",
		Timestamp: time.Now(),
	})
}

// CaptureMemoryStats captures current memory statistics
func (p *Profiler) CaptureMemoryStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	p.mu.Lock()
	defer p.mu.Unlock()

	// Record key memory metrics
	p.metrics = append(p.metrics,
		Metric{
			Type:      MetricMemory,
			Name:      "heap_alloc",
			Value:     float64(m.HeapAlloc) / 1024 / 1024,
			Unit:      "MB",
			Timestamp: time.Now(),
		},
		Metric{
			Type:      MetricMemory,
			Name:      "heap_objects",
			Value:     float64(m.HeapObjects),
			Unit:      "objects",
			Timestamp: time.Now(),
		},
		Metric{
			Type:      MetricMemory,
			Name:      "goroutines",
			Value:     float64(runtime.NumGoroutine()),
			Unit:      "count",
			Timestamp: time.Now(),
		},
	)

	return m
}

// GetOperationStats returns statistics for all operations
func (p *Profiler) GetOperationStats() map[string]OperationSummary {
	p.mu.RLock()
	defer p.mu.RUnlock()

	summary := make(map[string]OperationSummary)

	for name, stats := range p.operations {
		stats.mu.Lock()
		avg := time.Duration(0)
		if stats.Count > 0 {
			avg = stats.TotalTime / time.Duration(stats.Count)
		}

		summary[name] = OperationSummary{
			Name:        name,
			Count:       stats.Count,
			TotalTime:   stats.TotalTime,
			AverageTime: avg,
			MinTime:     stats.MinTime,
			MaxTime:     stats.MaxTime,
			LastTime:    stats.LastTime,
		}
		stats.mu.Unlock()
	}

	return summary
}

// OperationSummary provides a summary of operation statistics
type OperationSummary struct {
	Name        string
	Count       int64
	TotalTime   time.Duration
	AverageTime time.Duration
	MinTime     time.Duration
	MaxTime     time.Duration
	LastTime    time.Duration
}

// GetMemoryDelta returns memory changes since baseline
func (p *Profiler) GetMemoryDelta() MemoryDelta {
	var current runtime.MemStats
	runtime.ReadMemStats(&current)

	return MemoryDelta{
		HeapAllocDelta:   mathutil.Uint64ToInt64(current.HeapAlloc) - mathutil.Uint64ToInt64(p.memBaseline.HeapAlloc),
		HeapObjectsDelta: mathutil.Uint64ToInt64(current.HeapObjects) - mathutil.Uint64ToInt64(p.memBaseline.HeapObjects),
		GoroutinesDelta:  runtime.NumGoroutine() - p.baselineGoroutines,
		GCCycles:         current.NumGC - p.memBaseline.NumGC,
	}
}

// MemoryDelta represents memory changes
type MemoryDelta struct {
	HeapAllocDelta   int64
	HeapObjectsDelta int64
	GoroutinesDelta  int
	GCCycles         uint32
}

// Report generates a performance report
func (p *Profiler) Report() string {
	report := "Performance Report\n"
	report += "==================\n"
	report += fmt.Sprintf("Uptime: %v\n\n", time.Since(p.startTime))

	// Memory stats
	memDelta := p.GetMemoryDelta()
	report += "Memory Changes:\n"
	report += fmt.Sprintf("  Heap Alloc Delta: %+d MB\n", memDelta.HeapAllocDelta/1024/1024)
	report += fmt.Sprintf("  Heap Objects Delta: %+d\n", memDelta.HeapObjectsDelta)
	report += fmt.Sprintf("  Goroutines Delta: %+d\n", memDelta.GoroutinesDelta)
	report += fmt.Sprintf("  GC Cycles: %d\n\n", memDelta.GCCycles)

	// Operation stats
	opStats := p.GetOperationStats()
	if len(opStats) > 0 {
		report += "Operation Statistics:\n"
		for _, stats := range opStats {
			report += fmt.Sprintf("  %s:\n", stats.Name)
			report += fmt.Sprintf("    Count: %d\n", stats.Count)
			report += fmt.Sprintf("    Avg: %v, Min: %v, Max: %v\n",
				stats.AverageTime, stats.MinTime, stats.MaxTime)
		}
	}

	return report
}

// FindMemoryLeaks looks for potential memory leaks
func (p *Profiler) FindMemoryLeaks() []string {
	var issues []string

	memDelta := p.GetMemoryDelta()

	// Check for excessive heap growth
	if memDelta.HeapAllocDelta > 100*1024*1024 { // 100MB
		issues = append(issues, fmt.Sprintf("High memory growth: %d MB", memDelta.HeapAllocDelta/1024/1024))
	}

	// Check for goroutine leaks
	if memDelta.GoroutinesDelta > 100 {
		issues = append(issues, fmt.Sprintf("Potential goroutine leak: %d new goroutines", memDelta.GoroutinesDelta))
	}

	// Check for excessive objects
	if memDelta.HeapObjectsDelta > 1000000 {
		issues = append(issues, fmt.Sprintf("High object count increase: %d objects", memDelta.HeapObjectsDelta))
	}

	return issues
}

// BenchmarkOperation runs a benchmark on an operation
func BenchmarkOperation(name string, iterations int, operation func()) BenchmarkResult {
	// Warm up
	for i := 0; i < 10; i++ {
		operation()
	}

	// Force GC before benchmark
	runtime.GC()
	runtime.Gosched()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	for i := 0; i < iterations; i++ {
		operation()
	}

	elapsed := time.Since(start)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	return BenchmarkResult{
		Name:        name,
		Iterations:  iterations,
		TotalTime:   elapsed,
		TimePerOp:   elapsed / time.Duration(iterations),
		AllocsPerOp: mathutil.Uint64ToInt64(memAfter.Mallocs-memBefore.Mallocs) / int64(iterations),
		BytesPerOp:  mathutil.Uint64ToInt64(memAfter.TotalAlloc-memBefore.TotalAlloc) / int64(iterations),
	}
}

// BenchmarkResult contains benchmark results
type BenchmarkResult struct {
	Name        string
	Iterations  int
	TotalTime   time.Duration
	TimePerOp   time.Duration
	AllocsPerOp int64
	BytesPerOp  int64
}

// String returns a string representation of benchmark result
func (r BenchmarkResult) String() string {
	return fmt.Sprintf("%s: %d iterations in %v (%v/op, %d allocs/op, %d B/op)",
		r.Name, r.Iterations, r.TotalTime, r.TimePerOp, r.AllocsPerOp, r.BytesPerOp)
}
