package performance

import (
	"fmt"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/pkg/slurm"
)

// BenchmarkMultiSelectOperations benchmarks various multi-select table operations
func BenchmarkMultiSelectOperations(b *testing.B) {
	sizes := []int{100, 1000, 5000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("TableSize_%d", size), func(b *testing.B) {
			// Create table with test data
			config := components.DefaultTableConfig()
			table := components.NewMultiSelectTable(config)

			// Generate test data
			testData := make([][]string, size)
			for i := 0; i < size; i++ {
				testData[i] = []string{
					fmt.Sprintf("job%d", i),
					fmt.Sprintf("user%d", i%10),
					[]string{"RUNNING", "PENDING", "COMPLETED", "FAILED"}[i%4],
					fmt.Sprintf("node%d", i%50),
					fmt.Sprintf("%d", (i%32)+1),
					fmt.Sprintf("%dG", (i%64)+4),
				}
			}

			table.SetData(testData)
			table.SetMultiSelectMode(true)

			b.Run("SelectAll", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					table.SelectAll()
					table.ClearSelection()
				}
			})

			b.Run("ToggleRows", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					// Select every 10th row
					for j := 0; j < size; j += 10 {
						table.ToggleRow(j + 1) // +1 for header
					}
					table.ClearSelection()
				}
			})

			b.Run("InvertSelection", func(b *testing.B) {
				// Pre-select some rows
				for j := 0; j < size/2; j += 5 {
					table.ToggleRow(j + 1)
				}

				for i := 0; i < b.N; i++ {
					table.InvertSelection()
				}

				table.ClearSelection()
			})

			b.Run("GetSelectedData", func(b *testing.B) {
				// Pre-select 10% of rows
				for j := 0; j < size; j += 10 {
					table.ToggleRow(j + 1)
				}

				for i := 0; i < b.N; i++ {
					_ = table.GetAllSelectedData()
				}

				table.ClearSelection()
			})
		})
	}
}

// BenchmarkJobExportOperations benchmarks job export operations
func BenchmarkJobExportOperations(b *testing.B) {
	tempDir := b.TempDir()
	exporter := export.NewJobOutputExporter(tempDir)

	// Generate different sized job outputs
	jobSizes := []int{1000, 10000, 100000, 1000000} // characters

	for _, size := range jobSizes {
		jobData := export.JobOutputData{
			JobID:       fmt.Sprintf("benchmark_job_%d", size),
			JobName:     "benchmark_test",
			OutputType:  "stdout",
			Content:     generateJobOutput(size),
			Timestamp:   time.Now(),
			ExportedBy:  "benchmark",
			ExportTime:  time.Now(),
			ContentSize: size,
		}

		b.Run(fmt.Sprintf("Size_%d_chars", size), func(b *testing.B) {
			formats := []export.ExportFormat{
				export.FormatText,
				export.FormatJSON,
				export.FormatCSV,
				export.FormatMarkdown,
			}

			for _, format := range formats {
				b.Run(string(format), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_, err := exporter.Export(jobData, format, "")
						if err != nil {
							b.Errorf("Export failed: %v", err)
						}
					}
				})
			}
		})
	}
}

// BenchmarkBatchExportOperations benchmarks batch export operations
func BenchmarkBatchExportOperations(b *testing.B) {
	tempDir := b.TempDir()
	exporter := export.NewJobOutputExporter(tempDir)

	batchSizes := []int{10, 50, 100, 500}
	jobOutputSize := 10000 // Standard job output size

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			// Generate batch data
			batchJobs := make([]export.JobOutputData, batchSize)
			for i := 0; i < batchSize; i++ {
				batchJobs[i] = export.JobOutputData{
					JobID:       fmt.Sprintf("batch_job_%d", i),
					JobName:     fmt.Sprintf("batch_test_%d", i),
					OutputType:  "stdout",
					Content:     generateJobOutput(jobOutputSize),
					Timestamp:   time.Now(),
					ExportedBy:  "benchmark",
					ExportTime:  time.Now(),
					ContentSize: jobOutputSize,
				}
			}

			for i := 0; i < b.N; i++ {
				_, err := exporter.BatchExport(batchJobs, export.FormatJSON, "")
				if err != nil {
					b.Errorf("Batch export failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkMockClientOperations benchmarks SLURM mock client operations
func BenchmarkMockClientOperations(b *testing.B) {
	mockClient := slurm.NewFastMockClient() // Use fast client for performance tests

	b.Run("ListJobs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := mockClient.Jobs().List(nil)
			if err != nil {
				b.Errorf("List jobs failed: %v", err)
			}
		}
	})

	b.Run("ListNodes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := mockClient.Nodes().List(nil)
			if err != nil {
				b.Errorf("List nodes failed: %v", err)
			}
		}
	})

	b.Run("GetJobOutput", func(b *testing.B) {
		// Get first job ID from mock data
		jobs, _ := mockClient.Jobs().List(nil)
		if len(jobs.Jobs) == 0 {
			b.Skip("No jobs available for testing")
		}
		jobID := jobs.Jobs[0].ID

		for i := 0; i < b.N; i++ {
			_, err := mockClient.Jobs().GetOutput(jobID)
			if err != nil {
				b.Errorf("Get job output failed: %v", err)
			}
		}
	})

	b.Run("FilteredJobList", func(b *testing.B) {
		// Test with various filters
		filters := []struct {
			name string
			opts *dao.ListJobsOptions
		}{
			{"ByState", &dao.ListJobsOptions{States: []string{"RUNNING"}}},
			{"ByUser", &dao.ListJobsOptions{Users: []string{"alice"}}},
			{"ByPartition", &dao.ListJobsOptions{Partitions: []string{"compute"}}},
			{"Multiple", &dao.ListJobsOptions{
				States:     []string{"RUNNING", "PENDING"},
				Users:      []string{"alice", "bob"},
				Partitions: []string{"compute", "gpu"},
			}},
		}

		for _, filter := range filters {
			b.Run(filter.name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, err := mockClient.Jobs().List(filter.opts)
					if err != nil {
						b.Errorf("Filtered job list failed: %v", err)
					}
				}
			})
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage for different operations
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("MultiSelectTable_LargeDataset", func(b *testing.B) {
		config := components.DefaultTableConfig()
		table := components.NewMultiSelectTable(config)

		// Generate large dataset
		const dataSize = 50000
		testData := make([][]string, dataSize)
		for i := 0; i < dataSize; i++ {
			testData[i] = []string{
				fmt.Sprintf("job%d", i),
				fmt.Sprintf("user%d", i%100),
				[]string{"RUNNING", "PENDING", "COMPLETED", "FAILED"}[i%4],
				fmt.Sprintf("node%d", i%500),
				fmt.Sprintf("%d", (i%64)+1),
				fmt.Sprintf("%dG", (i%128)+4),
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			table.SetData(testData)
			table.SetMultiSelectMode(true)
			table.SelectAll()
			_ = table.GetAllSelectedData()
			table.ClearSelection()
		}
	})

	b.Run("JobExport_LargeContent", func(b *testing.B) {
		tempDir := b.TempDir()
		exporter := export.NewJobOutputExporter(tempDir)

		// Generate very large job output
		const contentSize = 1000000 // 1MB
		largeJobData := export.JobOutputData{
			JobID:       "memory_test_job",
			JobName:     "memory_benchmark",
			OutputType:  "stdout",
			Content:     generateJobOutput(contentSize),
			Timestamp:   time.Now(),
			ExportedBy:  "benchmark",
			ExportTime:  time.Now(),
			ContentSize: contentSize,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := exporter.Export(largeJobData, export.FormatJSON, "")
			if err != nil {
				b.Errorf("Large export failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("ConcurrentTableOperations", func(b *testing.B) {
		config := components.DefaultTableConfig()
		table := components.NewMultiSelectTable(config)

		// Generate test data
		testData := make([][]string, 1000)
		for i := 0; i < 1000; i++ {
			testData[i] = []string{
				fmt.Sprintf("job%d", i),
				fmt.Sprintf("user%d", i%10),
				[]string{"RUNNING", "PENDING", "COMPLETED", "FAILED"}[i%4],
				fmt.Sprintf("node%d", i%20),
			}
		}

		table.SetData(testData)
		table.SetMultiSelectMode(true)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate concurrent operations
				table.ToggleRow(1)
				table.GetSelectionCount()
				table.IsRowSelected(0)
				table.ToggleRow(1) // Toggle back
			}
		})
	})

	b.Run("ConcurrentMockClientAccess", func(b *testing.B) {
		mockClient := slurm.NewFastMockClient() // Use fast client for performance tests

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate concurrent client operations
				_, _ = mockClient.Jobs().List(nil)
				_, _ = mockClient.Nodes().List(nil)
			}
		})
	})
}

// Helper function to generate job output of specified size
func generateJobOutput(size int) string {
	const chunk = "Processing data chunk... simulation running... results calculated...\n"
	chunkSize := len(chunk)

	if size <= chunkSize {
		return chunk[:size]
	}

	chunks := size / chunkSize
	remainder := size % chunkSize

	result := ""
	for i := 0; i < chunks; i++ {
		result += chunk
	}

	if remainder > 0 {
		result += chunk[:remainder]
	}

	return result
}

// BenchmarkStringOperations benchmarks string operations used in the application
func BenchmarkStringOperations(b *testing.B) {
	// Test string formatting operations used in job display
	b.Run("JobStringFormatting", func(b *testing.B) {
		jobData := []string{
			"job12345",
			"alice",
			"RUNNING",
			"compute",
			"16",
			"64G",
			"node001-016",
		}

		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("%-12s %-10s %-12s %-10s %5s %8s %-20s",
				jobData[0], jobData[1], jobData[2], jobData[3],
				jobData[4], jobData[5], jobData[6])
		}
	})

	b.Run("FilterMatching", func(b *testing.B) {
		filter := "running"
		testStrings := []string{
			"RUNNING",
			"PENDING",
			"COMPLETED",
			"FAILED",
			"running_job",
			"test_running_simulation",
		}

		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				_ = contains(s, filter)
			}
		}
	})
}

// Helper function for string containment (case-insensitive)
func contains(s, substr string) bool {
	// Simple case-insensitive containment check
	s = toLower(s)
	substr = toLower(substr)
	return len(s) >= len(substr) &&
		(len(s) == len(substr) && s == substr ||
		 len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

// Simple ASCII lowercase conversion for benchmarking
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}