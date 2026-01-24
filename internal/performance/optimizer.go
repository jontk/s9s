// Package performance provides performance optimization and monitoring tools.
package performance

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// OptimizationLevel represents the level of optimization
type OptimizationLevel int

const (
	// OptimizationNone indicates no optimization.
	OptimizationNone OptimizationLevel = iota
	// OptimizationLight indicates light optimization level.
	OptimizationLight
	// OptimizationModerate indicates moderate optimization level.
	OptimizationModerate
	// OptimizationAggressive indicates aggressive optimization level.
	OptimizationAggressive
)

// Optimizer provides performance optimization recommendations and automatic tuning
type Optimizer struct {
	profiler        *Profiler
	level           OptimizationLevel
	mu              sync.RWMutex
	recommendations []Recommendation
	autoTuneEnabled bool
}

// Recommendation represents a performance optimization recommendation
type Recommendation struct {
	Category    string
	Issue       string
	Impact      string // "High", "Medium", "Low"
	Suggestion  string
	AutoFixable bool
}

// NewOptimizer creates a new performance optimizer
func NewOptimizer(profiler *Profiler) *Optimizer {
	return &Optimizer{
		profiler:        profiler,
		level:           OptimizationModerate,
		recommendations: make([]Recommendation, 0),
	}
}

// SetLevel sets the optimization level
func (o *Optimizer) SetLevel(level OptimizationLevel) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.level = level
}

// EnableAutoTune enables automatic performance tuning
func (o *Optimizer) EnableAutoTune(enable bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.autoTuneEnabled = enable
}

// Analyze performs performance analysis and generates recommendations
func (o *Optimizer) Analyze() []Recommendation {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.recommendations = make([]Recommendation, 0)

	// Analyze memory usage
	o.analyzeMemory()

	// Analyze goroutines
	o.analyzeGoroutines()

	// Analyze operations
	o.analyzeOperations()

	// Analyze CPU usage
	o.analyzeCPU()

	// Apply auto-fixes if enabled
	if o.autoTuneEnabled {
		o.applyAutoFixes()
	}

	return o.recommendations
}

// analyzeMemory checks for memory-related issues
func (o *Optimizer) analyzeMemory() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check heap allocation
	heapMB := float64(m.HeapAlloc) / 1024 / 1024
	if heapMB > 500 {
		o.recommendations = append(o.recommendations, Recommendation{
			Category:    "Memory",
			Issue:       fmt.Sprintf("High heap allocation: %.2f MB", heapMB),
			Impact:      "High",
			Suggestion:  "Consider implementing object pooling for frequently allocated objects",
			AutoFixable: false,
		})
	}

	// Check GC frequency
	memDelta := o.profiler.GetMemoryDelta()
	if memDelta.GCCycles > 100 {
		o.recommendations = append(o.recommendations, Recommendation{
			Category:    "Memory",
			Issue:       fmt.Sprintf("Frequent GC cycles: %d", memDelta.GCCycles),
			Impact:      "Medium",
			Suggestion:  "Reduce allocations by reusing objects and using sync.Pool",
			AutoFixable: true,
		})
	}

	// Check for memory leaks
	leaks := o.profiler.FindMemoryLeaks()
	for _, leak := range leaks {
		o.recommendations = append(o.recommendations, Recommendation{
			Category:    "Memory",
			Issue:       leak,
			Impact:      "High",
			Suggestion:  "Investigate and fix potential memory leak",
			AutoFixable: false,
		})
	}
}

// analyzeGoroutines checks for goroutine-related issues
func (o *Optimizer) analyzeGoroutines() {
	numGoroutines := runtime.NumGoroutine()

	if numGoroutines > 1000 {
		o.recommendations = append(o.recommendations, Recommendation{
			Category:    "Concurrency",
			Issue:       fmt.Sprintf("High goroutine count: %d", numGoroutines),
			Impact:      "High",
			Suggestion:  "Use worker pools to limit concurrent goroutines",
			AutoFixable: false,
		})
	} else if numGoroutines > 500 {
		o.recommendations = append(o.recommendations, Recommendation{
			Category:    "Concurrency",
			Issue:       fmt.Sprintf("Elevated goroutine count: %d", numGoroutines),
			Impact:      "Medium",
			Suggestion:  "Consider using goroutine pools for better resource management",
			AutoFixable: false,
		})
	}
}

// analyzeOperations checks operation performance
func (o *Optimizer) analyzeOperations() {
	opStats := o.profiler.GetOperationStats()

	for name, stats := range opStats {
		// Check for slow operations
		if stats.AverageTime > 100*time.Millisecond {
			o.recommendations = append(o.recommendations, Recommendation{
				Category:    "Performance",
				Issue:       fmt.Sprintf("Slow operation '%s': avg %v", name, stats.AverageTime),
				Impact:      "High",
				Suggestion:  "Optimize the operation or add caching",
				AutoFixable: false,
			})
		} else if stats.AverageTime > 50*time.Millisecond {
			o.recommendations = append(o.recommendations, Recommendation{
				Category:    "Performance",
				Issue:       fmt.Sprintf("Operation '%s' could be faster: avg %v", name, stats.AverageTime),
				Impact:      "Medium",
				Suggestion:  "Consider optimizing or running in background",
				AutoFixable: false,
			})
		}

		// Check for high variance
		if stats.MaxTime > 10*stats.MinTime && stats.Count > 10 {
			o.recommendations = append(o.recommendations, Recommendation{
				Category:    "Performance",
				Issue:       fmt.Sprintf("High variance in '%s': min %v, max %v", name, stats.MinTime, stats.MaxTime),
				Impact:      "Medium",
				Suggestion:  "Investigate causes of performance variance",
				AutoFixable: false,
			})
		}
	}
}

// analyzeCPU checks CPU-related issues
func (o *Optimizer) analyzeCPU() {
	// Set GOMAXPROCS recommendation based on optimization level
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()

	if maxProcs != numCPU && o.level >= OptimizationModerate {
		o.recommendations = append(o.recommendations, Recommendation{
			Category:    "CPU",
			Issue:       fmt.Sprintf("GOMAXPROCS (%d) != NumCPU (%d)", maxProcs, numCPU),
			Impact:      "Medium",
			Suggestion:  "Set GOMAXPROCS to match CPU count for better parallelism",
			AutoFixable: true,
		})
	}
}

// applyAutoFixes applies automatic performance fixes
func (o *Optimizer) applyAutoFixes() {
	for i, rec := range o.recommendations {
		if !rec.AutoFixable {
			continue
		}

		switch rec.Category {
		case "CPU":
			if rec.Issue == fmt.Sprintf("GOMAXPROCS (%d) != NumCPU (%d)", runtime.GOMAXPROCS(0), runtime.NumCPU()) {
				runtime.GOMAXPROCS(runtime.NumCPU())
				o.recommendations[i].Suggestion += " [AUTO-FIXED]"
			}
		case "Memory":
			if rec.Issue == fmt.Sprintf("Frequent GC cycles: %d", o.profiler.GetMemoryDelta().GCCycles) {
				// Adjust GC percentage for less frequent GC
				if o.level >= OptimizationModerate {
					debug.SetGCPercent(200) // Less frequent GC
					o.recommendations[i].Suggestion += " [AUTO-FIXED: GC percent set to 200]"
				}
			}
		}
	}
}

// GetOptimizationSummary returns a summary of optimizations
func (o *Optimizer) GetOptimizationSummary() string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	summary := "Performance Optimization Summary\n"
	summary += "================================\n"
	summary += fmt.Sprintf("Optimization Level: %s\n", o.getLevelString())
	summary += fmt.Sprintf("Auto-Tune: %v\n\n", o.autoTuneEnabled)

	highCount, mediumCount, lowCount := 0, 0, 0
	for _, rec := range o.recommendations {
		switch rec.Impact {
		case "High":
			highCount++
		case "Medium":
			mediumCount++
		case "Low":
			lowCount++
		}
	}

	summary += "Issues Found:\n"
	summary += fmt.Sprintf("  High Impact: %d\n", highCount)
	summary += fmt.Sprintf("  Medium Impact: %d\n", mediumCount)
	summary += fmt.Sprintf("  Low Impact: %d\n", lowCount)

	return summary
}

// getLevelString returns string representation of optimization level
func (o *Optimizer) getLevelString() string {
	switch o.level {
	case OptimizationNone:
		return "None"
	case OptimizationLight:
		return "Light"
	case OptimizationModerate:
		return "Moderate"
	case OptimizationAggressive:
		return "Aggressive"
	default:
		return "Unknown"
	}
}

// TuneForBatchOperations optimizes settings for batch operations
func (o *Optimizer) TuneForBatchOperations() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.level >= OptimizationLight {
		// Increase GC threshold for batch operations
		debug.SetGCPercent(400)

		// Use all available CPUs
		runtime.GOMAXPROCS(runtime.NumCPU())

		o.recommendations = append(o.recommendations, Recommendation{
			Category:    "Batch",
			Issue:       "Batch operation mode",
			Impact:      "Info",
			Suggestion:  "Tuned for batch operations: GC=400%, GOMAXPROCS=NumCPU",
			AutoFixable: false,
		})
	}
}

// TuneForInteractive optimizes settings for interactive use
func (o *Optimizer) TuneForInteractive() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.level >= OptimizationLight {
		// Lower GC threshold for more responsive behavior
		debug.SetGCPercent(100)

		// Leave one CPU for system responsiveness
		if runtime.NumCPU() > 2 {
			runtime.GOMAXPROCS(runtime.NumCPU() - 1)
		}

		o.recommendations = append(o.recommendations, Recommendation{
			Category:    "Interactive",
			Issue:       "Interactive mode",
			Impact:      "Info",
			Suggestion:  "Tuned for interactive use: GC=100%, GOMAXPROCS=NumCPU-1",
			AutoFixable: false,
		})
	}
}

// ObjectPool provides a generic object pool for reducing allocations
type ObjectPool[T any] struct {
	pool    sync.Pool
	factory func() T
}

// NewObjectPool creates a new object pool
func NewObjectPool[T any](factory func() T) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return factory()
			},
		},
		factory: factory,
	}
}

// Get retrieves an object from the pool
func (p *ObjectPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns an object to the pool
func (p *ObjectPool[T]) Put(obj T) {
	p.pool.Put(obj)
}
