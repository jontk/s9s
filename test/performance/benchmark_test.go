package performance_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/performance"
	"github.com/jontk/s9s/internal/streaming"
)

// BenchmarkCircularBuffer tests circular buffer performance
func BenchmarkCircularBuffer(b *testing.B) {
	tests := []struct {
		name     string
		capacity int
		lineSize int
	}{
		{"Small_1K", 1000, 80},
		{"Medium_10K", 10000, 80},
		{"Large_100K", 100000, 80},
		{"LongLines_10K", 10000, 1000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			buffer := streaming.NewCircularBuffer(tt.capacity)
			line := strings.Repeat("x", tt.lineSize)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buffer.AppendString(line)
			}

			b.ReportMetric(float64(buffer.Size()), "final_size")
			b.ReportMetric(float64(buffer.TotalLines()), "total_lines")
		})
	}
}

// BenchmarkEventBus tests event bus performance
func BenchmarkEventBus(b *testing.B) {
	tests := []struct {
		name        string
		subscribers int
		concurrent  bool
	}{
		{"Single_1Sub", 1, false},
		{"Single_10Sub", 10, false},
		{"Single_100Sub", 100, false},
		{"Concurrent_10Sub", 10, true},
		{"Concurrent_100Sub", 100, true},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			eventBus := streaming.NewEventBus()

			// Create subscribers
			channels := make([]chan<- streaming.StreamEvent, tt.subscribers)
			for i := 0; i < tt.subscribers; i++ {
				ch := make(chan streaming.StreamEvent, 100)
				eventBus.Subscribe("job1", "stdout", ch)
				channels[i] = ch

				// Drain channel in background
				go func(c chan streaming.StreamEvent) {
					for range c {
						// Just consume
					}
				}(ch)
			}

			event := &streaming.StreamEvent{
				JobID:      "job1",
				OutputType: "stdout",
				Content:    "Test line of output data",
				Timestamp:  time.Now(),
			}

			b.ResetTimer()

			if tt.concurrent {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						eventBus.Publish(event)
					}
				})
			} else {
				for i := 0; i < b.N; i++ {
					eventBus.Publish(event)
				}
			}

			// Cleanup
			eventBus.UnsubscribeAll("job1", "stdout")
		})
	}
}

// BenchmarkStringOperations tests common string operations
func BenchmarkStringOperations(b *testing.B) {
	tests := []struct {
		name      string
		operation func()
	}{
		{
			name: "StringBuilder_1000Lines",
			operation: func() {
				var builder strings.Builder
				for i := 0; i < 1000; i++ {
					builder.WriteString(fmt.Sprintf("Line %d: Some log output text\n", i))
				}
				_ = builder.String()
			},
		},
		{
			name: "StringConcat_1000Lines",
			operation: func() {
				result := ""
				for i := 0; i < 1000; i++ {
					result += fmt.Sprintf("Line %d: Some log output text\n", i)
				}
				_ = result
			},
		},
		{
			name: "SliceAppend_1000Lines",
			operation: func() {
				lines := make([]string, 0, 1000)
				for i := 0; i < 1000; i++ {
					lines = append(lines, fmt.Sprintf("Line %d: Some log output text", i))
				}
				_ = strings.Join(lines, "\n")
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tt.operation()
			}
		})
	}
}

// BenchmarkMapOperations tests map performance patterns
func BenchmarkMapOperations(b *testing.B) {
	tests := []struct {
		name      string
		operation func()
	}{
		{
			name: "MapWithMutex_1000Ops",
			operation: func() {
				m := make(map[string]string)
				for i := 0; i < 1000; i++ {
					key := fmt.Sprintf("key_%d", i)
					m[key] = fmt.Sprintf("value_%d", i)
					_ = m[key]
				}
			},
		},
		{
			name: "PreallocMap_1000Ops",
			operation: func() {
				m := make(map[string]string, 1000)
				for i := 0; i < 1000; i++ {
					key := fmt.Sprintf("key_%d", i)
					m[key] = fmt.Sprintf("value_%d", i)
					_ = m[key]
				}
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tt.operation()
			}
		})
	}
}

// TestProfilerIntegration tests the profiler integration
func TestProfilerIntegration(t *testing.T) {
	profiler := performance.NewProfiler()

	// Simulate some operations
	for i := 0; i < 100; i++ {
		done := profiler.StartOperation("test_operation")
		time.Sleep(time.Millisecond)
		done()
	}

	// Capture memory stats
	profiler.CaptureMemoryStats()

	// Get operation stats
	stats := profiler.GetOperationStats()
	if len(stats) == 0 {
		t.Error("Expected operation statistics")
	}

	// Check for memory leaks
	leaks := profiler.FindMemoryLeaks()
	t.Logf("Potential issues found: %v", leaks)

	// Generate report
	report := profiler.Report()
	if report == "" {
		t.Error("Expected non-empty report")
	}
	t.Log(report)
}

// TestOptimizer tests the performance optimizer
func TestOptimizer(t *testing.T) {
	profiler := performance.NewProfiler()
	optimizer := performance.NewOptimizer(profiler)

	// Enable auto-tune
	optimizer.EnableAutoTune(true)
	optimizer.SetLevel(performance.OptimizationModerate)

	// Simulate some load
	for i := 0; i < 50; i++ {
		done := profiler.StartOperation("heavy_operation")
		// Simulate work
		data := make([]byte, 1024*1024) // 1MB allocation
		_ = data
		time.Sleep(2 * time.Millisecond)
		done()
	}

	// Analyze performance
	recommendations := optimizer.Analyze()

	t.Logf("Found %d recommendations", len(recommendations))
	for _, rec := range recommendations {
		t.Logf("[%s] %s: %s (Impact: %s)", rec.Category, rec.Issue, rec.Suggestion, rec.Impact)
	}

	// Get summary
	summary := optimizer.GetOptimizationSummary()
	t.Log(summary)
}

// BenchmarkObjectPool tests object pool performance
func BenchmarkObjectPool(b *testing.B) {
	type TestObject struct {
		Data [1024]byte
		ID   int
	}

	tests := []struct {
		name    string
		usePool bool
	}{
		{"WithPool", true},
		{"WithoutPool", false},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var pool *performance.ObjectPool[*TestObject]

			if tt.usePool {
				pool = performance.NewObjectPool(func() *TestObject {
					return &TestObject{}
				})
			}

			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					var obj *TestObject

					if tt.usePool {
						obj = pool.Get()
					} else {
						obj = &TestObject{}
					}

					// Use the object
					obj.ID = 42
					obj.Data[0] = 1

					if tt.usePool {
						pool.Put(obj)
					}
				}
			})
		})
	}
}

// BenchmarkConcurrentOperations tests concurrent operation patterns
func BenchmarkConcurrentOperations(b *testing.B) {
	tests := []struct {
		name      string
		workers   int
		operation func(id int)
	}{
		{
			name:    "Sequential",
			workers: 1,
			operation: func(id int) {
				// Simulate work
				sum := 0
				for i := 0; i < 1000; i++ {
					sum += i * id
				}
				_ = sum
			},
		},
		{
			name:    "Parallel_4Workers",
			workers: 4,
			operation: func(id int) {
				// Simulate work
				sum := 0
				for i := 0; i < 1000; i++ {
					sum += i * id
				}
				_ = sum
			},
		},
		{
			name:    "Parallel_8Workers",
			workers: 8,
			operation: func(id int) {
				// Simulate work
				sum := 0
				for i := 0; i < 1000; i++ {
					sum += i * id
				}
				_ = sum
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.SetParallelism(tt.workers)

			if tt.workers == 1 {
				for i := 0; i < b.N; i++ {
					tt.operation(i)
				}
			} else {
				b.RunParallel(func(pb *testing.PB) {
					id := 0
					for pb.Next() {
						tt.operation(id)
						id++
					}
				})
			}
		})
	}
}
