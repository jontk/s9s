package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/performance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobOutputExporter(t *testing.T) {
	tempDir := t.TempDir()
	exporter := NewJobOutputExporter(tempDir)

	t.Run("ExportText", func(t *testing.T) {
		data := JobOutputData{
			JobID:       "12345",
			JobName:     "test-job",
			OutputType:  "stdout",
			Content:     "This is test output\nLine 2\nLine 3",
			Timestamp:   time.Now(),
			ExportedBy:  "test",
			ExportTime:  time.Now(),
			ContentSize: 50,
		}

		result, err := exporter.Export(data, FormatText, "")
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.FilePath, ".txt")

		// Verify file exists and contains expected content
		content, err := os.ReadFile(result.FilePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Job ID: 12345")
		assert.Contains(t, string(content), "This is test output")
	})

	t.Run("ExportJSON", func(t *testing.T) {
		data := JobOutputData{
			JobID:      "12345",
			JobName:    "test-job",
			OutputType: "stdout",
			Content:    "Test content",
		}

		result, err := exporter.Export(data, FormatJSON, "")
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.FilePath, ".json")
	})

	t.Run("ExportCSV", func(t *testing.T) {
		data := JobOutputData{
			JobID:      "12345",
			JobName:    "test-job",
			OutputType: "stdout",
			Content:    "Line 1\nLine 2\nLine 3",
		}

		result, err := exporter.Export(data, FormatCSV, "")
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.FilePath, ".csv")
	})

	t.Run("GetSupportedFormats", func(t *testing.T) {
		formats := exporter.GetSupportedFormats()
		assert.Contains(t, formats, FormatText)
		assert.Contains(t, formats, FormatJSON)
		assert.Contains(t, formats, FormatCSV)
		assert.Contains(t, formats, FormatMarkdown)
		assert.Contains(t, formats, FormatHTML)
	})
}

func TestPerformanceExporter(t *testing.T) {
	tempDir := t.TempDir()
	exporter := NewPerformanceExporter(tempDir)

	// Create mock profiler and optimizer
	profiler := performance.NewProfiler()
	optimizer := performance.NewOptimizer(profiler)

	// Simulate some operations
	for i := 0; i < 5; i++ {
		stop := profiler.StartOperation("TestOp1")
		time.Sleep(10 * time.Millisecond)
		stop()

		stop = profiler.StartOperation("TestOp2")
		time.Sleep(5 * time.Millisecond)
		stop()
	}

	t.Run("ExportText", func(t *testing.T) {
		result, err := exporter.ExportPerformanceReport(profiler, optimizer, FormatText, "")
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.FilePath, ".txt")

		// Verify file exists and contains expected content
		content, err := os.ReadFile(result.FilePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Performance Report")
		assert.Contains(t, string(content), "System Metrics")
		assert.Contains(t, string(content), "Operation Statistics")
	})

	t.Run("ExportJSON", func(t *testing.T) {
		result, err := exporter.ExportPerformanceReport(profiler, optimizer, FormatJSON, "")
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.FilePath, ".json")
	})

	t.Run("ExportHTML", func(t *testing.T) {
		result, err := exporter.ExportPerformanceReport(profiler, optimizer, FormatHTML, "")
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.FilePath, ".html")

		// Verify HTML structure
		content, err := os.ReadFile(result.FilePath)
		require.NoError(t, err)
		htmlContent := string(content)
		assert.Contains(t, htmlContent, "<!DOCTYPE html>")
		assert.Contains(t, htmlContent, "S9s Performance Report")
		assert.Contains(t, htmlContent, "<table>")
	})

	t.Run("BatchExport", func(t *testing.T) {
		formats := []ExportFormat{FormatText, FormatJSON, FormatCSV}
		results, err := exporter.BatchExportReports(profiler, optimizer, formats)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		for _, result := range results {
			assert.True(t, result.Success)
			assert.FileExists(t, result.FilePath)
		}
	})
}

// TestExportDialogIntegration tests are in the widgets package

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024, "1.5 MB"},
	}

	for _, test := range tests {
		result := formatBytes(test.bytes)
		assert.Equal(t, test.expected, result, "formatBytes(%d) should return %s", test.bytes, test.expected)
	}
}

func TestExportFilename(t *testing.T) {
	exporter := NewJobOutputExporter("")

	// Test filename generation
	filename := exporter.generateFilename("12345", "test job/name", "stdout", FormatText)
	assert.Contains(t, filename, "job_12345_test_job_name_stdout")
	assert.Contains(t, filename, ".txt")
	assert.NotContains(t, filename, "/") // Special chars should be replaced
}

func BenchmarkExportPerformance(b *testing.B) {
	tempDir := b.TempDir()
	exporter := NewPerformanceExporter(tempDir)
	profiler := performance.NewProfiler()
	optimizer := performance.NewOptimizer(profiler)

	// Generate some test data
	for i := 0; i < 100; i++ {
		stop := profiler.StartOperation(strings.Repeat("Operation", i%10))
		time.Sleep(time.Microsecond)
		stop()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exporter.ExportPerformanceReport(profiler, optimizer, FormatJSON, filepath.Join(tempDir, "bench.json"))
	}
}
