package output

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLocalFileReader_HeadFile_EmptyFile tests reading head of empty file
func TestLocalFileReader_HeadFile_EmptyFile(t *testing.T) {
	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")
	err := os.WriteFile(testFile, []byte(""), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	result, err := reader.HeadFile(ctx, testFile, 10)
	if err != nil {
		t.Fatalf("Failed to read head of empty file: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty result for empty file, got: %q", result)
	}
}

// TestLocalFileReader_TailFile_ExactLines tests tail with exact number of lines
func TestLocalFileReader_TailFile_ExactLines(t *testing.T) {
	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "exact.txt")
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	result, err := reader.TailFile(ctx, testFile, 5)
	if err != nil {
		t.Fatalf("Failed to tail file: %v", err)
	}

	// Should return all lines (at least 4, possibly 5 depending on final newline handling)
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) < 4 || len(lines) > 5 {
		t.Errorf("Expected 4-5 lines, got %d", len(lines))
	}

	// Verify we got the last lines
	if !strings.Contains(result, "Line 5") {
		t.Error("Expected result to contain Line 5")
	}
}

// TestLocalFileReader_ReadRange_ZeroLength tests reading zero bytes
func TestLocalFileReader_ReadRange_ZeroLength(t *testing.T) {
	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("Content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	result, err := reader.ReadRange(ctx, testFile, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read zero-length range: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty result for zero-length read, got: %q", result)
	}
}

// TestLocalFileReader_ReadRange_OffsetAtEnd tests reading from end of file
func TestLocalFileReader_ReadRange_OffsetAtEnd(t *testing.T) {
	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "1234567890"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	// Read from position 10 (end of file) - EOF is acceptable behavior
	result, err := reader.ReadRange(ctx, testFile, 10, 10)

	// Either no error with empty result, or EOF error is acceptable
	if err != nil {
		if !strings.Contains(err.Error(), "EOF") {
			t.Fatalf("Expected EOF or no error, got: %v", err)
		}
		// EOF is expected when reading at end, test passes
		return
	}

	if result != "" {
		t.Errorf("Expected empty result when reading from EOF, got: %q", result)
	}
}

// TestLocalFileReader_HeadFile_SingleLine tests reading single line
func TestLocalFileReader_HeadFile_SingleLine(t *testing.T) {
	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "single.txt")
	content := "Only one line\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	result, err := reader.HeadFile(ctx, testFile, 1)
	if err != nil {
		t.Fatalf("Failed to read head: %v", err)
	}

	if result != content {
		t.Errorf("Content mismatch: got %q, want %q", result, content)
	}
}

// TestLocalFileReader_TailFile_PermissionDenied tests permission errors
func TestLocalFileReader_TailFile_PermissionDenied(t *testing.T) {
	// Skip on systems where we can't change permissions reliably (Windows, root user)
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	reader := NewLocalFileReader()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "noperm.txt")
	err := os.WriteFile(testFile, []byte("Content"), 0000) // No permissions
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	_, err = reader.TailFile(ctx, testFile, 10)
	if err == nil {
		t.Error("Expected permission error, got nil")
	}
}

// TestRemoteFileReader_ReadRemoteFile_ParseError tests handling of unparseable stat output
func TestRemoteFileReader_ReadRemoteFile_ParseError(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "test -f") {
				return "exists", nil
			}
			if strings.Contains(command, "stat") {
				return "invalid-size-output", nil // Not a number
			}
			return "", nil
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	_, err := reader.ReadRemoteFile(ctx, "node01", "/path/to/file.txt")
	if err == nil {
		t.Fatal("Expected parse error, got nil")
	}

	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// TestRemoteFileReader_GetRemoteFileInfo_ParseError tests parse error in stat
func TestRemoteFileReader_GetRemoteFileInfo_ParseError(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			return "not-valid-format", nil
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	_, err := reader.GetRemoteFileInfo(ctx, "node01", "/path/to/file.txt")
	if err == nil {
		t.Fatal("Expected parse error, got nil")
	}
}

// TestJobOutputReader_GetMetadata_NotFound tests metadata for missing file
func TestJobOutputReader_GetMetadata_NotFound(t *testing.T) {
	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return "/nonexistent/file.txt", false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	metadata, err := reader.GetMetadata(ctx, "12345", "stdout")
	if err != nil {
		t.Fatalf("Expected no error for non-existent file metadata, got: %v", err)
	}

	if metadata.Exists {
		t.Error("Expected Exists to be false for non-existent file")
	}
}

// TestJobOutputReader_ReadPartial_RemoteHeadMode tests remote file with head mode
func TestJobOutputReader_ReadPartial_RemoteHeadMode(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "stat") {
				return "10000 1609459200", nil // 10KB file
			}
			if strings.Contains(command, "head -n") {
				return "Line 1\nLine 2\nLine 3\n", nil
			}
			return "", nil
		},
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return "/remote/path/job.out", true, "node01", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, mockSSH)
	ctx := context.Background()

	opts := ReadOptions{
		MaxBytes: 500, // File is 10KB, so it's "large"
		MaxLines: 3,
		TailMode: false, // Head mode
	}

	result, err := reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read remote with head mode: %v", err)
	}

	if !strings.Contains(result.Content, "Line 1") {
		t.Error("Expected head mode to return first lines")
	}
}

// TestJobOutputReader_ReadPartial_RemoteTailMode tests remote file with tail mode
func TestJobOutputReader_ReadPartial_RemoteTailMode(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "stat") {
				return "10000 1609459200", nil
			}
			if strings.Contains(command, "tail -n") {
				return "Line 98\nLine 99\nLine 100\n", nil
			}
			return "", nil
		},
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return "/remote/path/job.out", true, "node01", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, mockSSH)
	ctx := context.Background()

	opts := ReadOptions{
		MaxBytes: 500,
		MaxLines: 3,
		TailMode: true,
	}

	result, err := reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read remote with tail mode: %v", err)
	}

	if !strings.Contains(result.Content, "Line 100") {
		t.Error("Expected tail mode to return last lines")
	}
}

// TestOutputCache_GetAfterEviction tests getting entry after it's been evicted
func TestOutputCache_GetAfterEviction(t *testing.T) {
	cache := NewOutputCache(5*time.Minute, 2) // Very small cache

	content1 := &OutputContent{Content: "First"}
	content2 := &OutputContent{Content: "Second"}
	content3 := &OutputContent{Content: "Third"}

	cache.Set("key1", content1, 0)
	time.Sleep(10 * time.Millisecond)
	cache.Set("key2", content2, 0)
	time.Sleep(10 * time.Millisecond)

	// This should evict key1
	cache.Set("key3", content3, 0)

	// Try to get evicted entry
	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted")
	}

	// key2 and key3 should still exist
	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to still exist")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to still exist")
	}
}
