package output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestJobOutputReader_WithCaching tests caching integration
func TestJobOutputReader_WithCaching(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	content := "Cached output content\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	// First read - should miss cache
	opts := ReadOptions{
		CacheEnabled: true,
	}

	result1, err := reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Second read - should hit cache
	result2, err := reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read from cache: %v", err)
	}

	if result1.Content != result2.Content {
		t.Error("Cached content doesn't match original")
	}

	// Check cache stats
	stats := reader.cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.Hits)
	}
}

// TestJobOutputReader_CacheDisabled tests reading without cache
func TestJobOutputReader_CacheDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	content := "No cache content\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	// Read with cache disabled
	opts := ReadOptions{
		CacheEnabled: false,
	}

	_, err = reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Cache should be empty
	stats := reader.cache.Stats()
	if stats.Entries != 0 {
		t.Errorf("Expected 0 cache entries, got %d", stats.Entries)
	}
}

// TestJobOutputReader_ForceRefresh tests cache bypass
func TestJobOutputReader_ForceRefresh(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	content := "Original content\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	// First read - populate cache
	opts := ReadOptions{
		CacheEnabled: true,
	}

	_, err = reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Modify file
	newContent := "Updated content\n"
	err = os.WriteFile(testFile, []byte(newContent), 0600)
	if err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	// Read with force refresh
	opts.ForceRefresh = true
	result, err := reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read with force refresh: %v", err)
	}

	// Should get updated content, not cached
	if !strings.Contains(result.Content, "Updated") {
		t.Error("Expected force refresh to bypass cache and get updated content")
	}
}

// TestJobOutputReader_ClearCache tests cache clearing
func TestJobOutputReader_ClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	err := os.WriteFile(testFile, []byte("Content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	// Populate cache
	opts := ReadOptions{CacheEnabled: true}
	_, err = reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Verify cache has entry
	stats := reader.cache.Stats()
	if stats.Entries != 1 {
		t.Fatalf("Expected 1 cache entry, got %d", stats.Entries)
	}

	// Clear cache
	reader.ClearCache()

	// Verify cache is empty
	stats = reader.cache.Stats()
	if stats.Entries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.Entries)
	}
}

// TestJobOutputReader_SetCache tests custom cache
func TestJobOutputReader_SetCache(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	err := os.WriteFile(testFile, []byte("Content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)

	// Set custom cache with short TTL
	customCache := NewOutputCache(1*time.Second, 5)
	reader.SetCache(customCache)

	ctx := context.Background()
	opts := ReadOptions{CacheEnabled: true}

	// Read to populate cache
	_, err = reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Verify custom cache was used
	stats := customCache.Stats()
	if stats.Entries != 1 {
		t.Errorf("Expected 1 entry in custom cache, got %d", stats.Entries)
	}
}

// TestLocalFileReader_TailFile_SmallFile tests tail on file smaller than estimate
func TestLocalFileReader_TailFile_SmallFile(t *testing.T) {
	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "small.txt")
	content := "Line 1\nLine 2\nLine 3\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	result, err := reader.TailFile(ctx, testFile, 10) // Request more lines than file has
	if err != nil {
		t.Fatalf("Failed to tail small file: %v", err)
	}

	// Should return all content
	if result != content {
		t.Errorf("Content mismatch for small file: got %q, want %q", result, content)
	}
}

// TestRemoteFileReader_HeadRemoteFile_NotFound tests head on non-existent remote file
func TestRemoteFileReader_HeadRemoteFile_NotFound(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			return "", nil // Return empty string for head command
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	result, err := reader.HeadRemoteFile(ctx, "node01", "/missing.txt", 10)
	if err != nil {
		t.Fatalf("Expected no error for head (SSH command succeeded), got: %v", err)
	}

	// Should return empty string
	if result != "" {
		t.Errorf("Expected empty result, got: %q", result)
	}
}

// TestLocalFileReader_ReadRange_BeyondEOF tests reading range beyond file size
func TestLocalFileReader_ReadRange_BeyondEOF(t *testing.T) {
	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "short.txt")
	content := "SHORT"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	// Try to read beyond file size
	result, err := reader.ReadRange(ctx, testFile, 0, 100)
	if err != nil {
		t.Fatalf("Failed to read range: %v", err)
	}

	// Should return only available content
	if result != content {
		t.Errorf("Content mismatch: got %q, want %q", result, content)
	}
}

// TestLocalFileReader_TailFile_LargeFile tests tail on large files with seek
func TestLocalFileReader_TailFile_LargeFile(t *testing.T) {
	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Create a file > 1MB to trigger seek logic
	var lines []string
	for i := 1; i <= 20000; i++ {
		lines = append(lines, fmt.Sprintf("Line %05d with some padding text to make it larger", i))
	}
	content := strings.Join(lines, "\n") + "\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	result, err := reader.TailFile(ctx, testFile, 10)
	if err != nil {
		t.Fatalf("Failed to tail large file: %v", err)
	}

	// Should contain last lines
	if !strings.Contains(result, "Line 20000") {
		t.Error("Expected result to contain last line")
	}

	resultLines := strings.Split(strings.TrimSpace(result), "\n")
	if len(resultLines) > 10 {
		t.Errorf("Expected at most 10 lines, got %d", len(resultLines))
	}
}

// TestLocalFileReader_HeadFile_Error tests error cases for HeadFile
func TestLocalFileReader_HeadFile_Error(t *testing.T) {
	reader := NewLocalFileReader()

	ctx := context.Background()

	// Test with non-existent file
	_, err := reader.HeadFile(ctx, "/nonexistent/file.txt", 10)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestLocalFileReader_ReadRange_Error tests error cases for ReadRange
func TestLocalFileReader_ReadRange_Error(t *testing.T) {
	reader := NewLocalFileReader()

	ctx := context.Background()

	// Test with non-existent file
	_, err := reader.ReadRange(ctx, "/nonexistent/file.txt", 0, 100)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = reader.ReadRange(canceledCtx, "/tmp/test.txt", 0, 100)
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

// TestJobOutputReader_SetCache_Integration tests SetCache integration
func TestJobOutputReader_SetCache_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	err := os.WriteFile(testFile, []byte("Content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)

	// Set custom cache
	customCache := NewOutputCache(10*time.Minute, 50)
	reader.SetCache(customCache)

	// Verify SetCache worked
	if reader.cache != customCache {
		t.Error("SetCache did not update cache reference")
	}

	ctx := context.Background()
	opts := ReadOptions{CacheEnabled: true}

	// Read to populate custom cache
	_, err = reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Verify entry in custom cache
	stats := customCache.Stats()
	if stats.Entries != 1 {
		t.Errorf("Expected 1 entry in custom cache, got %d", stats.Entries)
	}
}

// TestJobOutputReader_ReadPartial_HeadMode tests reading with head mode
func TestJobOutputReader_ReadPartial_HeadMode(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	var lines []string
	for i := 1; i <= 100; i++ {
		lines = append(lines, fmt.Sprintf("Line %03d: %s", i, strings.Repeat("X", 50)))
	}
	content := strings.Join(lines, "\n") + "\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	// Read with head mode (not tail)
	opts := ReadOptions{
		MaxBytes: 500,
		MaxLines: 10,
		TailMode: false, // Head mode
	}

	result, err := reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read with head mode: %v", err)
	}

	// Should have first lines
	resultLines := strings.Split(strings.TrimSpace(result.Content), "\n")
	if len(resultLines) > 10 {
		t.Errorf("Expected at most 10 lines, got %d", len(resultLines))
	}

	// Verify it's the first lines
	if !strings.Contains(resultLines[0], "Line 001") {
		t.Errorf("Expected first line to contain 'Line 001', got: %s", resultLines[0])
	}
}

// TestRemoteFileReader_HeadRemoteFile_Error tests error cases
func TestRemoteFileReader_HeadRemoteFile_Error(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			return "", fmt.Errorf("ssh: Connection refused")
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	_, err := reader.HeadRemoteFile(ctx, "node01", "/path/to/file.txt", 10)
	if err == nil {
		t.Fatal("Expected SSH error, got nil")
	}

	if !strings.Contains(err.Error(), "Connection refused") {
		t.Errorf("Expected connection error, got: %v", err)
	}
}
