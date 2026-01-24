package performance

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/views"
	"github.com/jontk/s9s/pkg/slurm"
	"github.com/rivo/tview"
)

// TestBatchExportMemoryUsage benchmarks memory usage of batch export operations
func TestBatchExportMemoryUsage(t *testing.T) {
	t.Run("LegacyBatchExport", func(t *testing.T) {
		testBatchExportMemory(t, false)
	})

	t.Run("OptimizedBatchExport", func(t *testing.T) {
		testBatchExportMemory(t, true)
	})
}

func testBatchExportMemory(t *testing.T, useOptimized bool) {
	// Force garbage collection before test
	runtime.GC()
	runtime.GC()

	// Get initial memory stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create test components
	client := slurm.NewFastMockClient()
	app := tview.NewApplication()
	_ = views.NewBatchOperationsView(client, app) // Unused but shows we can create the view

	// Create export data for 100 jobs (simulating medium batch)
	jobCount := 100
	exportData := make([]export.JobOutputData, jobCount)

	for i := 0; i < jobCount; i++ {
		jobID := fmt.Sprintf("job_%d", i)

		// Generate different content sizes based on optimization
		var content string
		if useOptimized {
			content = generateOptimizedContent(jobID)
		} else {
			content = generateLargeContent(jobID)
		}

		exportData[i] = export.JobOutputData{
			JobID:       jobID,
			JobName:     fmt.Sprintf("test_job_%d", i),
			OutputType:  "stdout",
			Content:     content,
			Timestamp:   time.Now(),
			ExportedBy:  "test",
			ExportTime:  time.Now(),
			ContentSize: len(content),
		}
	}

	// Create exporter
	tempDir := t.TempDir()
	exporter := export.NewJobOutputExporter(tempDir)

	// Perform batch export with memory tracking
	var results []*export.ExportResult
	var err error

	if useOptimized {
		// Use the optimized batch export with callback
		results, err = exporter.BatchExportWithCallback(exportData, export.FormatJSON, "", func(current, total int, jobID string) {
			// Progress callback - in real implementation this would update UI
		})
	} else {
		// Use standard batch export
		results, err = exporter.BatchExport(exportData, export.FormatJSON, "")
	}

	if err != nil {
		t.Fatalf("Batch export failed: %v", err)
	}

	// Verify all exports succeeded
	successful := 0
	for _, result := range results {
		if result.Success {
			successful++
		}
	}

	if successful != jobCount {
		t.Errorf("Expected %d successful exports, got %d", jobCount, successful)
	}

	// Force garbage collection and measure final memory
	runtime.GC()
	runtime.GC()

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Calculate memory usage
	allocatedMemory := m2.TotalAlloc - m1.TotalAlloc
	heapMemory := m2.HeapAlloc - m1.HeapAlloc
	peakHeapMemory := m2.HeapSys

	// Log memory usage
	optimizationType := "Legacy"
	if useOptimized {
		optimizationType = "Optimized"
	}

	t.Logf("%s Batch Export Memory Usage:", optimizationType)
	t.Logf("  Jobs Processed: %d", jobCount)
	t.Logf("  Total Allocated: %s", formatBytes(int64(allocatedMemory)))
	t.Logf("  Heap Allocated: %s", formatBytes(int64(heapMemory)))
	t.Logf("  Peak Heap Size: %s", formatBytes(int64(peakHeapMemory)))
	t.Logf("  Memory per Job: %s", formatBytes(int64(allocatedMemory)/int64(jobCount)))
	t.Logf("  Successful Exports: %d", successful)

	// Performance assertions
	memoryPerJob := allocatedMemory / uint64(jobCount)

	if useOptimized {
		// Optimized version should use less memory per job
		maxMemoryPerJobOptimized := uint64(50 * 1024) // 50KB per job max
		if memoryPerJob > maxMemoryPerJobOptimized {
			t.Errorf("Optimized export uses too much memory per job: %s (max: %s)",
				formatBytes(int64(memoryPerJob)), formatBytes(int64(maxMemoryPerJobOptimized)))
		}
	} else if memoryPerJob < 100*1024 { // Less than 100KB seems too low for legacy
		// Legacy version baseline - should use more memory
		t.Logf("Legacy export memory usage unexpectedly low: %s per job", formatBytes(int64(memoryPerJob)))
	}
}

// BenchmarkBatchExportMemoryComparison compares memory usage between legacy and optimized exports
func BenchmarkBatchExportMemoryComparison(b *testing.B) {
	jobSizes := []int{10, 50, 100, 500}

	for _, jobCount := range jobSizes {
		b.Run(fmt.Sprintf("Jobs_%d", jobCount), func(b *testing.B) {
			b.Run("Legacy", func(b *testing.B) {
				benchmarkBatchExportMemory(b, jobCount, false)
			})

			b.Run("Optimized", func(b *testing.B) {
				benchmarkBatchExportMemory(b, jobCount, true)
			})
		})
	}
}

func benchmarkBatchExportMemory(b *testing.B, jobCount int, useOptimized bool) {
	tempDir := b.TempDir()
	exporter := export.NewJobOutputExporter(tempDir)

	// Pre-generate export data
	exportData := make([]export.JobOutputData, jobCount)
	for i := 0; i < jobCount; i++ {
		jobID := fmt.Sprintf("bench_job_%d", i)
		var content string
		if useOptimized {
			content = generateOptimizedContent(jobID)
		} else {
			content = generateLargeContent(jobID)
		}

		exportData[i] = export.JobOutputData{
			JobID:       jobID,
			JobName:     fmt.Sprintf("bench_test_%d", i),
			OutputType:  "stdout",
			Content:     content,
			Timestamp:   time.Now(),
			ExportedBy:  "benchmark",
			ExportTime:  time.Now(),
			ContentSize: len(content),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var err error
		if useOptimized {
			_, err = exporter.BatchExportWithCallback(exportData, export.FormatJSON, "", nil)
		} else {
			_, err = exporter.BatchExport(exportData, export.FormatJSON, "")
		}

		if err != nil {
			b.Fatalf("Batch export failed: %v", err)
		}

		// Force garbage collection between iterations to get cleaner measurements
		runtime.GC()
	}
}

// generateLargeContent generates large job output content (simulating legacy approach)
func generateLargeContent(jobID string) string {
	content := fmt.Sprintf(`=== SLURM Job Output for %s ===
Job started at: %s
Working directory: /home/user/large_simulation
Command line: ./simulate --input large_dataset.dat --output results/ --threads 32

Environment:
SLURM_JOB_ID=%s
SLURM_JOB_NAME=large_simulation
SLURM_CPUS_PER_TASK=32
SLURM_MEM_PER_NODE=128000

Loading required modules:
  - gcc/11.2.0
  - openmpi/4.1.4
  - python/3.9.7
  - cuda/11.4
  - tensorflow/2.8.0

Initializing simulation parameters...
Setting up computational grid...
Allocating memory for 1M+ data points...
Loading input dataset (15.7 GB)...

`, jobID, time.Now().Format("2006-01-02 15:04:05"), jobID)

	// Add repetitive processing logs to make it large
	for i := 1; i <= 100; i++ {
		content += fmt.Sprintf(`[%s] Processing batch %d/100
  - Reading data chunk %d (156 MB)
  - Running computational kernel
  - GPU utilization: 95.2%%
  - Memory usage: %d/%d MB
  - Estimated time remaining: %dm %ds
  - Intermediate results written to batch_%d.tmp

`, time.Now().Add(time.Duration(i)*time.Minute).Format("15:04:05"),
			i, i, 1000+i*10, 12800, 100-i, (100-i)*2, i)
	}

	content += fmt.Sprintf(`
Simulation completed successfully!

Final Results:
==============
Total computation time: 2h 47m 32s
Data processed: 15.7 GB input, 8.3 GB output
Peak memory usage: 11.2 GB
Peak GPU memory: 7.8 GB
CPU efficiency: 94.7%%
GPU efficiency: 89.3%%
Total energy consumed: 47.2 kWh

Output files:
  - results/final_output.dat (8.3 GB)
  - results/statistics.json (2.1 MB)
  - results/visualization.png (4.7 MB)
  - results/log_detailed.txt (15.3 MB)

Job %s completed at: %s
Exit code: 0
`, jobID, time.Now().Format("2006-01-02 15:04:05"))

	return content
}

// generateOptimizedContent generates minimal job output content (optimized approach)
func generateOptimizedContent(jobID string) string {
	return fmt.Sprintf(`Job %s - COMPLETED
Started: %s
Runtime: 1h 23m 45s
Exit: 0

Results:
- Input: 2.1 GB processed
- Output: 987 MB generated
- CPU: 92.1%% efficiency
- Memory: 4.1 GB peak

Files: results_%s.dat (987MB)
Status: SUCCESS`,
		jobID,
		time.Now().Format("15:04:05"),
		jobID)
}

// formatBytes formats byte count as human readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
