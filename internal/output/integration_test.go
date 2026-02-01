package output

import (
	"context"
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
