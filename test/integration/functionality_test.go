package integration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/ui/components"
)

// TestJobExportFunctionality tests the complete job export functionality
func TestJobExportFunctionality(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create exporter
	exporter := export.NewJobOutputExporter(tempDir)

	// Test comprehensive job data
	testJobData := export.JobOutputData{
		JobID:       "integration_test_1001",
		JobName:     "comprehensive_test_job",
		OutputType:  "stdout",
		Content:     generateSampleJobOutput(),
		Timestamp:   time.Now(),
		ExportedBy:  "integration_test",
		ExportTime:  time.Now(),
		ContentSize: len(generateSampleJobOutput()),
	}

	t.Run("AllExportFormats", func(t *testing.T) {
		formats := exporter.GetSupportedFormats()
		if len(formats) == 0 {
			t.Fatal("No export formats available")
		}

		for _, format := range formats {
			t.Run(string(format), func(t *testing.T) {
				result, err := exporter.Export(testJobData, format, "")
				if err != nil {
					t.Fatalf("Export failed for format %s: %v", format, err)
				}

				// Verify result properties
				if !result.Success {
					t.Errorf("Export not successful for format %s", format)
				}

				if result.Size == 0 {
					t.Errorf("Export file size is 0 for format %s", format)
				}

				if result.FilePath == "" {
					t.Errorf("Export file path is empty for format %s", format)
				}

				// Verify file exists and has content
				if stat, err := os.Stat(result.FilePath); err != nil {
					t.Errorf("Export file does not exist: %s", result.FilePath)
				} else if stat.Size() == 0 {
					t.Errorf("Export file is empty: %s", result.FilePath)
				}

				t.Logf("✅ %s export successful: %s (%d bytes)",
					format, result.FilePath, result.Size)
			})
		}
	})

	t.Run("BatchExport", func(t *testing.T) {
		// Create multiple job outputs for batch testing
		batchJobs := []export.JobOutputData{
			{
				JobID:       "batch_1",
				JobName:     "batch_test_1",
				OutputType:  "stdout",
				Content:     "Batch job 1 output\nProcessing dataset A\nCompleted successfully",
				Timestamp:   time.Now(),
				ExportedBy:  "integration_test",
				ExportTime:  time.Now(),
				ContentSize: 55,
			},
			{
				JobID:       "batch_2",
				JobName:     "batch_test_2",
				OutputType:  "stdout",
				Content:     "Batch job 2 output\nProcessing dataset B\nResults generated",
				Timestamp:   time.Now(),
				ExportedBy:  "integration_test",
				ExportTime:  time.Now(),
				ContentSize: 52,
			},
			{
				JobID:       "batch_3",
				JobName:     "batch_test_3",
				OutputType:  "stderr",
				Content:     "Warning: deprecated API used\nJob completed with warnings",
				Timestamp:   time.Now(),
				ExportedBy:  "integration_test",
				ExportTime:  time.Now(),
				ContentSize: 55,
			},
		}

		results, err := exporter.BatchExport(batchJobs, export.FormatJSON, "")
		if err != nil {
			t.Fatalf("Batch export failed: %v", err)
		}

		if len(results) != len(batchJobs) {
			t.Errorf("Expected %d results, got %d", len(batchJobs), len(results))
		}

		successCount := 0
		for i, result := range results {
			if result.Success {
				successCount++

				// Verify each file
				if _, err := os.Stat(result.FilePath); err != nil {
					t.Errorf("Batch file %d missing: %s", i, result.FilePath)
				}
			} else {
				t.Errorf("Batch export %d failed: %v", i, result.Error)
			}
		}

		if successCount != len(batchJobs) {
			t.Errorf("Expected %d successful exports, got %d", len(batchJobs), successCount)
		}

		// Test export summary
		summary := exporter.ExportSummary(results)
		if summary == "" {
			t.Error("Export summary should not be empty")
		}

		t.Logf("✅ Batch export successful: %d files", successCount)
		t.Logf("📊 Summary: %s", summary)
	})
}

// TestMultiSelectTableFunctionality tests multi-select table operations
func TestMultiSelectTableFunctionality(t *testing.T) {
	// Create multi-select table
	config := components.DefaultTableConfig()
	table := components.NewMultiSelectTable(config)

	// Test data - simulating job table data
	testData := [][]string{
		{"job1001", "user1", "gpu", "RUNNING", "2", "8G", "node01"},
		{"job1002", "user2", "cpu", "PENDING", "4", "16G", "node02"},
		{"job1003", "user1", "gpu", "COMPLETED", "8", "32G", "node03"},
		{"job1004", "user3", "cpu", "FAILED", "1", "4G", "node04"},
		{"job1005", "user2", "gpu", "RUNNING", "16", "64G", "node05"},
	}

	table.SetData(testData)

	t.Run("BasicOperations", func(t *testing.T) {
		// Test initial state
		if table.IsMultiSelectMode() {
			t.Error("Multi-select mode should be disabled by default")
		}

		if table.GetSelectionCount() != 0 {
			t.Error("Initial selection count should be 0")
		}

		// Enable multi-select mode
		table.SetMultiSelectMode(true)
		if !table.IsMultiSelectMode() {
			t.Error("Multi-select mode should be enabled")
		}

		t.Log("✅ Basic operations working")
	})

	t.Run("SelectionOperations", func(t *testing.T) {
		table.SetMultiSelectMode(true)
		table.ClearSelection()

		// Test individual row selection
		table.ToggleRow(1) // Select first data row (row 0 is header)
		if table.GetSelectionCount() != 1 {
			t.Errorf("Expected 1 selected row, got %d", table.GetSelectionCount())
		}

		if !table.IsRowSelected(0) { // Data row 0 corresponds to display row 1
			t.Error("Row 0 should be selected")
		}

		// Test multiple selection
		table.ToggleRow(2)
		table.ToggleRow(3)
		if table.GetSelectionCount() != 3 {
			t.Errorf("Expected 3 selected rows, got %d", table.GetSelectionCount())
		}

		// Test select all
		table.SelectAll()
		if table.GetSelectionCount() != len(testData) {
			t.Errorf("Select all failed: expected %d, got %d", len(testData), table.GetSelectionCount())
		}

		// Test clear selection
		table.ClearSelection()
		if table.GetSelectionCount() != 0 {
			t.Errorf("Clear selection failed: expected 0, got %d", table.GetSelectionCount())
		}

		// Test invert selection
		table.ToggleRow(1)
		table.ToggleRow(2)
		initialCount := table.GetSelectionCount()
		table.InvertSelection()
		expectedAfterInvert := len(testData) - initialCount
		if table.GetSelectionCount() != expectedAfterInvert {
			t.Errorf("Invert selection failed: expected %d, got %d", expectedAfterInvert, table.GetSelectionCount())
		}

		t.Log("✅ Selection operations working")
	})

	t.Run("DataRetrieval", func(t *testing.T) {
		table.SetMultiSelectMode(true)
		table.ClearSelection()

		// Select some rows
		table.ToggleRow(1)
		table.ToggleRow(3)

		selectedRows := table.GetSelectedRows()
		if len(selectedRows) != 2 {
			t.Errorf("Expected 2 selected row indices, got %d", len(selectedRows))
		}

		selectedData := table.GetAllSelectedData()
		if len(selectedData) != 2 {
			t.Errorf("Expected 2 selected data rows, got %d", len(selectedData))
		}

		// Verify data integrity
		for i, data := range selectedData {
			if len(data) != len(testData[0]) {
				t.Errorf("Selected data row %d has wrong column count: expected %d, got %d",
					i, len(testData[0]), len(data))
			}
		}

		t.Log("✅ Data retrieval working")
	})

	t.Run("CompatibilityMode", func(t *testing.T) {
		// Test compatibility with existing single-row operations
		table.SetMultiSelectMode(false)

		// In compatibility mode, GetSelectedData should work for single row
		data := table.GetSelectedData()
		// Should return nil or valid single row data (not crash)
		if len(data) > 10 {
			t.Error("Compatibility mode should return reasonable single row data")
		}

		t.Log("✅ Compatibility mode working")
	})

	t.Run("HintsSystem", func(t *testing.T) {
		// Test hints when multi-select is disabled
		table.SetMultiSelectMode(false)
		hints := table.GetMultiSelectHints()
		if len(hints) != 0 {
			t.Error("Should have no hints when multi-select is disabled")
		}

		// Test hints when multi-select is enabled
		table.SetMultiSelectMode(true)
		hints = table.GetMultiSelectHints()
		if len(hints) == 0 {
			t.Error("Should have hints when multi-select is enabled")
		}

		// Check for key hint elements
		hintText := fmt.Sprintf("%v", hints)
		expectedElements := []string{"Space", "Ctrl+A", "Select", "Clear"}
		for _, element := range expectedElements {
			if !containsText(hintText, element) {
				t.Errorf("Expected hint element '%s' not found in: %s", element, hintText)
			}
		}

		t.Log("✅ Hints system working")
	})
}

// TestPerformanceWithLargeDataset tests performance with realistic dataset sizes
func TestPerformanceWithLargeDataset(t *testing.T) {
	// Create table with larger dataset
	config := components.DefaultTableConfig()
	table := components.NewMultiSelectTable(config)

	// Generate realistic dataset size (1000 jobs)
	largeData := make([][]string, 1000)
	for i := 0; i < 1000; i++ {
		largeData[i] = []string{
			fmt.Sprintf("job%04d", i),
			fmt.Sprintf("user%d", i%20),
			// nolint:gosec // G602: modulo 3 guarantees safe index
			[]string{"cpu", "gpu", "mem"}[i%3],
			// nolint:gosec // G602: modulo 4 guarantees safe index
			[]string{"RUNNING", "PENDING", "COMPLETED", "FAILED"}[i%4],
			fmt.Sprintf("%d", (i%16)+1),
			fmt.Sprintf("%dG", (i%64)+4),
			fmt.Sprintf("node%03d", i%100),
		}
	}

	// Time the data loading
	start := time.Now()
	table.SetData(largeData)
	loadTime := time.Since(start)

	if loadTime > time.Second {
		t.Errorf("Data loading took too long: %v", loadTime)
	}

	table.SetMultiSelectMode(true)

	// Time select all operation
	start = time.Now()
	table.SelectAll()
	selectAllTime := time.Since(start)

	if selectAllTime > 100*time.Millisecond {
		t.Errorf("Select all took too long: %v", selectAllTime)
	}

	if table.GetSelectionCount() != len(largeData) {
		t.Errorf("Select all failed: expected %d, got %d", len(largeData), table.GetSelectionCount())
	}

	// Time clear operation
	start = time.Now()
	table.ClearSelection()
	clearTime := time.Since(start)

	if clearTime > 50*time.Millisecond {
		t.Errorf("Clear selection took too long: %v", clearTime)
	}

	t.Logf("✅ Performance test passed - Load: %v, SelectAll: %v, Clear: %v",
		loadTime, selectAllTime, clearTime)
}

// generateSampleJobOutput creates realistic job output for testing
func generateSampleJobOutput() string {
	return `SLURM Job Output - Integration Test
------------------------------------

Job Started: ` + time.Now().Format("2006-01-02 15:04:05") + `
Working Directory: /home/user/projects/simulation
Command: ./run_simulation --input data.txt --output results.txt

Loading modules...
  - gcc/11.2.0
  - openmpi/4.1.1
  - python/3.9.7

Initializing simulation parameters...
Grid size: 1024x1024
Time steps: 10000
Threads: 16

Starting computation...
[INFO] Step    1000/10000 - Progress:  10.0% - ETA: 0:12:34
[INFO] Step    2000/10000 - Progress:  20.0% - ETA: 0:10:45
[INFO] Step    3000/10000 - Progress:  30.0% - ETA: 0:09:12
[INFO] Step    4000/10000 - Progress:  40.0% - ETA: 0:07:33
[INFO] Step    5000/10000 - Progress:  50.0% - ETA: 0:06:15
[INFO] Step    6000/10000 - Progress:  60.0% - ETA: 0:04:52
[INFO] Step    7000/10000 - Progress:  70.0% - ETA: 0:03:28
[INFO] Step    8000/10000 - Progress:  80.0% - ETA: 0:02:11
[INFO] Step    9000/10000 - Progress:  90.0% - ETA: 0:01:05
[INFO] Step   10000/10000 - Progress: 100.0% - Completed!

Simulation completed successfully!
Results written to: results.txt
Statistics written to: stats.csv
Visualization saved to: plot.png

Performance Summary:
  Total runtime: 0:15:23
  Average time per step: 0.092 seconds
  Peak memory usage: 12.3 GB
  CPU efficiency: 87.2%
  I/O operations: 156,432

Exit code: 0
`
}

// Helper function for string containment check
func containsText(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(s) == len(substr) && s == substr ||
			len(s) > len(substr) && (s[:len(substr)] == substr || containsText(s[1:], substr)))
}
