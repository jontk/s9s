package integration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/views"
	"github.com/jontk/s9s/pkg/slurm"
	"github.com/rivo/tview"
)

// TestMultiSelectBatchOperationsIntegration tests the integration between multi-select and batch operations
func TestMultiSelectBatchOperationsIntegration(t *testing.T) {
	// Create mock client
	mockClient := slurm.NewMockClient()

	// Create application for testing
	app := tview.NewApplication()

	// Create jobs view
	jobsView := views.NewJobsView(mockClient)
	jobsView.SetApp(app)

	// Create pages for modal handling
	pages := tview.NewPages()
	jobsView.SetPages(pages)

	// Simulate some job data - jobs are pre-populated in the mock client
	// The mock client automatically includes sample jobs from populateSampleData()
	// No need to add jobs manually since MockClient already contains test data

	// Test multi-select functionality
	t.Run("MultiSelectMode", func(t *testing.T) {
		// Refresh to load data
		if err := jobsView.Refresh(); err != nil {
			t.Fatalf("Failed to refresh jobs view: %v", err)
		}

		// Verify the view was created successfully
		if jobsView == nil {
			t.Error("Jobs view not initialized")
		}

		// Verify UI components are accessible
		if jobsView.Render() == nil {
			t.Error("Jobs view render failed")
		}

		t.Log("Multi-select integration components initialized successfully")
	})
}

// TestJobOutputExportIntegration tests job output export functionality
func TestJobOutputExportIntegration(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create exporter
	exporter := export.NewJobOutputExporter(tempDir)

	// Test data
	testJobData := export.JobOutputData{
		JobID:       "1001",
		JobName:     "test_job_integration",
		OutputType:  "stdout",
		Content:     "Test job output content\nLine 2 of output\nFinal line",
		Timestamp:   time.Now(),
		ExportedBy:  "test_user",
		ExportTime:  time.Now(),
		ContentSize: 50,
	}

	// Test all export formats
	formats := []export.ExportFormat{
		export.FormatText,
		export.FormatJSON,
		export.FormatCSV,
		export.FormatMarkdown,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			result, err := exporter.Export(testJobData, format, "")
			if err != nil {
				t.Fatalf("Export failed for format %s: %v", format, err)
			}

			if !result.Success {
				t.Errorf("Export not marked as successful for format %s", format)
			}

			if result.Size == 0 {
				t.Errorf("Export file size is 0 for format %s", format)
			}

			// Verify file exists
			if _, err := os.Stat(result.FilePath); os.IsNotExist(err) {
				t.Errorf("Export file does not exist: %s", result.FilePath)
			}

			t.Logf("Successfully exported %s format to %s (%d bytes)",
				format, result.FilePath, result.Size)
		})
	}
}

// TestBatchExportIntegration tests batch export functionality
func TestBatchExportIntegration(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create exporter
	exporter := export.NewJobOutputExporter(tempDir)

	// Create multiple job outputs for batch testing
	testJobs := []export.JobOutputData{
		{
			JobID:       "2001",
			JobName:     "batch_job_1",
			OutputType:  "stdout",
			Content:     "Output from batch job 1\nProcessing data...\nCompleted successfully",
			Timestamp:   time.Now(),
			ExportedBy:  "test_user",
			ExportTime:  time.Now(),
			ContentSize: 65,
		},
		{
			JobID:       "2002",
			JobName:     "batch_job_2",
			OutputType:  "stdout",
			Content:     "Output from batch job 2\nRunning calculations...\nResults saved",
			Timestamp:   time.Now(),
			ExportedBy:  "test_user",
			ExportTime:  time.Now(),
			ContentSize: 58,
		},
		{
			JobID:       "2003",
			JobName:     "batch_job_3",
			OutputType:  "stderr",
			Content:     "Warning: deprecated function used\nJob completed with warnings",
			Timestamp:   time.Now(),
			ExportedBy:  "test_user",
			ExportTime:  time.Now(),
			ContentSize: 62,
		},
	}

	t.Run("BatchExport", func(t *testing.T) {
		results, err := exporter.BatchExport(testJobs, export.FormatJSON, tempDir)
		if err != nil {
			t.Fatalf("Batch export failed: %v", err)
		}

		if len(results) != len(testJobs) {
			t.Errorf("Expected %d results, got %d", len(testJobs), len(results))
		}

		successCount := 0
		var totalSize int64

		for i, result := range results {
			if result.Success {
				successCount++
				totalSize += result.Size

				// Verify file exists
				if _, err := os.Stat(result.FilePath); os.IsNotExist(err) {
					t.Errorf("Batch export file %d does not exist: %s", i, result.FilePath)
				}
			} else {
				t.Errorf("Batch export %d failed: %v", i, result.Error)
			}
		}

		if successCount != len(testJobs) {
			t.Errorf("Expected %d successful exports, got %d", len(testJobs), successCount)
		}

		t.Logf("Batch export successful: %d files, total size: %d bytes", successCount, totalSize)

		// Test export summary
		summary := exporter.ExportSummary(results)
		if summary == "" {
			t.Error("Export summary is empty")
		}

		t.Logf("Export summary: %s", summary)
	})
}

// TestSSHIntegrationBasic tests basic SSH integration functionality
func TestSSHIntegrationBasic(t *testing.T) {
	// Skip if SSH is not available
	if os.Getenv("SSH_TEST_ENABLED") != "true" {
		t.Skip("SSH integration tests disabled. Set SSH_TEST_ENABLED=true to enable.")
	}

	// This test would require actual SSH connectivity
	// For now, we test the component initialization

	t.Run("SSHClientInitialization", func(t *testing.T) {
		mockClient := slurm.NewMockClient()
		app := tview.NewApplication()

		// Create nodes view (which includes SSH functionality)
		nodesView := views.NewNodesView(mockClient)
		nodesView.SetApp(app)

		// Verify SSH components are initialized
		// Note: This would be expanded with actual SSH testing in a full test suite
		if err := nodesView.Refresh(); err != nil {
			t.Fatalf("Failed to refresh nodes view: %v", err)
		}

		t.Log("SSH integration components initialized successfully")
	})
}

// TestMultiSelectTableCompatibility tests compatibility with existing functionality
func TestMultiSelectTableCompatibility(t *testing.T) {
	// Create multi-select table
	config := components.DefaultTableConfig()
	table := components.NewMultiSelectTable(config)

	// Test data
	testData := [][]string{
		{"job1", "user1", "running", "node1"},
		{"job2", "user2", "pending", "node2"},
		{"job3", "user1", "completed", "node3"},
	}

	table.SetData(testData)

	t.Run("SingleRowCompatibility", func(t *testing.T) {
		// Test that single-row selection still works (compatibility mode)
		table.SetMultiSelectMode(false)

		// Simulate row selection
		data := table.GetSelectedData()
		// In compatibility mode, should return single row or nil
		if data != nil && len(data) > 4 {
			t.Error("Compatibility mode should return single row data")
		}

		t.Log("Single-row compatibility verified")
	})

	t.Run("MultiSelectMode", func(t *testing.T) {
		// Test multi-select functionality
		table.SetMultiSelectMode(true)

		// Test selection operations
		table.ToggleRow(1) // Select first data row
		table.ToggleRow(2) // Select second data row

		if table.GetSelectionCount() != 2 {
			t.Errorf("Expected 2 selected rows, got %d", table.GetSelectionCount())
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

		t.Log("Multi-select functionality verified")
	})

	t.Run("KeyboardShortcuts", func(t *testing.T) {
		// Test keyboard shortcuts
		hints := table.GetMultiSelectHints()
		if len(hints) == 0 {
			t.Error("Multi-select hints should not be empty when multi-select is enabled")
		}

		expectedHints := []string{"Space", "Ctrl+A", "Select All", "Clear", "Invert"}
		hintText := ""
		for _, hint := range hints {
			hintText += hint + " "
		}

		for _, expected := range expectedHints {
			if !containsString(hintText, expected) {
				t.Errorf("Expected hint '%s' not found in: %s", expected, hintText)
			}
		}

		t.Log("Keyboard shortcuts verified")
	})
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (len(s) == len(substr) && s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr || containsString(s[1:], substr)))
}

// BenchmarkMultiSelectPerformance benchmarks multi-select operations
func BenchmarkMultiSelectPerformance(b *testing.B) {
	// Create table with large dataset
	config := components.DefaultTableConfig()
	table := components.NewMultiSelectTable(config)

	// Generate large test dataset
	largeData := make([][]string, 10000)
	for i := 0; i < 10000; i++ {
		largeData[i] = []string{
			fmt.Sprintf("job%d", i),
			fmt.Sprintf("user%d", i%100),
			"running",
			fmt.Sprintf("node%d", i%50),
		}
	}

	table.SetData(largeData)
	table.SetMultiSelectMode(true)

	b.Run("SelectAll", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			table.SelectAll()
			table.ClearSelection()
		}
	})

	b.Run("ToggleRows", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Select random rows
			for j := 0; j < 100; j++ {
				table.ToggleRow(j*10 + 1) // +1 for header offset
			}
			table.ClearSelection()
		}
	})
}
