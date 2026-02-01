package output

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLocalFileReader_ReadFile tests reading a complete file
func TestLocalFileReader_ReadFile(t *testing.T) {
	reader := NewLocalFileReader()

	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read the file
	ctx := context.Background()
	result, err := reader.ReadFile(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if result != content {
		t.Errorf("Content mismatch: got %q, want %q", result, content)
	}
}

// TestLocalFileReader_ReadFile_NotFound tests reading a non-existent file
func TestLocalFileReader_ReadFile_NotFound(t *testing.T) {
	reader := NewLocalFileReader()

	ctx := context.Background()
	_, err := reader.ReadFile(ctx, "/nonexistent/file.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

// TestLocalFileReader_ReadFile_TooLarge tests handling of files too large for memory
func TestLocalFileReader_ReadFile_TooLarge(t *testing.T) {
	reader := NewLocalFileReader()
	reader.maxMemoryBuffer = 100 // Set low limit for testing

	// Create a file larger than the limit
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	content := strings.Repeat("A", 200)
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read the file
	ctx := context.Background()
	_, err = reader.ReadFile(ctx, testFile)
	if err == nil {
		t.Fatal("Expected error for large file, got nil")
	}

	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("Expected 'too large' error, got: %v", err)
	}
}

// TestLocalFileReader_TailFile tests reading last N lines
func TestLocalFileReader_TailFile(t *testing.T) {
	reader := NewLocalFileReader()

	// Create a file with multiple lines
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	lines := []string{}
	for i := 1; i <= 100; i++ {
		lines = append(lines, "Line "+string(rune('0'+i)))
	}
	content := strings.Join(lines, "\n") + "\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read last 10 lines
	ctx := context.Background()
	result, err := reader.TailFile(ctx, testFile, 10)
	if err != nil {
		t.Fatalf("Failed to tail file: %v", err)
	}

	// Count lines in result
	resultLines := strings.Split(strings.TrimSpace(result), "\n")
	if len(resultLines) > 10 {
		t.Errorf("Expected at most 10 lines, got %d", len(resultLines))
	}
}

// TestLocalFileReader_HeadFile tests reading first N lines
func TestLocalFileReader_HeadFile(t *testing.T) {
	reader := NewLocalFileReader()

	// Create a file with multiple lines
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read first 3 lines
	ctx := context.Background()
	result, err := reader.HeadFile(ctx, testFile, 3)
	if err != nil {
		t.Fatalf("Failed to head file: %v", err)
	}

	expected := "Line 1\nLine 2\nLine 3\n"
	if result != expected {
		t.Errorf("Content mismatch: got %q, want %q", result, expected)
	}
}

// TestLocalFileReader_ReadRange tests reading a byte range
func TestLocalFileReader_ReadRange(t *testing.T) {
	reader := NewLocalFileReader()

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "0123456789ABCDEFGHIJ"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read bytes 5-14 (10 bytes)
	ctx := context.Background()
	result, err := reader.ReadRange(ctx, testFile, 5, 10)
	if err != nil {
		t.Fatalf("Failed to read range: %v", err)
	}

	expected := "56789ABCDE"
	if result != expected {
		t.Errorf("Content mismatch: got %q, want %q", result, expected)
	}
}

// TestLocalFileReader_GetFileInfo tests getting file metadata
func TestLocalFileReader_GetFileInfo(t *testing.T) {
	reader := NewLocalFileReader()

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Test content"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get file info
	metadata, err := reader.GetFileInfo(testFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if !metadata.Exists {
		t.Error("Expected file to exist")
	}

	if metadata.Size != int64(len(content)) {
		t.Errorf("Size mismatch: got %d, want %d", metadata.Size, len(content))
	}

	if !metadata.IsLocal {
		t.Error("Expected file to be local")
	}

	if metadata.Path != testFile {
		t.Errorf("Path mismatch: got %s, want %s", metadata.Path, testFile)
	}
}

// TestLocalFileReader_GetFileInfo_NotFound tests getting info for non-existent file
func TestLocalFileReader_GetFileInfo_NotFound(t *testing.T) {
	reader := NewLocalFileReader()

	metadata, err := reader.GetFileInfo("/nonexistent/file.txt")
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}

	if metadata.Exists {
		t.Error("Expected file to not exist")
	}
}

// TestLocalFileReader_ContextCancellation tests handling of context cancellation
func TestLocalFileReader_ContextCancellation(t *testing.T) {
	reader := NewLocalFileReader()

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("Test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to read with canceled context
	_, err = reader.ReadFile(ctx, testFile)
	if err == nil {
		t.Fatal("Expected error for canceled context, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestLocalFileReader_ContextTimeout tests handling of context timeout
func TestLocalFileReader_ContextTimeout(t *testing.T) {
	reader := NewLocalFileReader()

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("Test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create context with very short timeout (already expired)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout

	// Try to read with timed-out context
	_, err = reader.ReadFile(ctx, testFile)
	if err == nil {
		t.Fatal("Expected error for timed-out context, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
	}
}
