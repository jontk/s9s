package output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockPathResolver implements a mock path resolver for testing
type mockPathResolver struct {
	resolveFunc func(jobID, outputType string) (string, bool, string, error)
}

func (m *mockPathResolver) ResolveOutputPath(jobID, outputType string) (string, bool, string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(jobID, outputType)
	}
	return "", false, "", fmt.Errorf("mock not configured")
}

// TestJobOutputReader_ReadJobOutput_Local tests reading local job output
func TestJobOutputReader_ReadJobOutput_Local(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	content := "Job output line 1\nJob output line 2\nJob output line 3\n"
	err := os.WriteFile(testFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock path resolver that returns local path
	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil // not remote
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	result, err := reader.ReadJobOutput(ctx, "12345", "stdout")
	if err != nil {
		t.Fatalf("Failed to read job output: %v", err)
	}

	if result.Content != content {
		t.Errorf("Content mismatch: got %q, want %q", result.Content, content)
	}

	if result.Source != "local" {
		t.Errorf("Source mismatch: got %s, want local", result.Source)
	}

	if !result.Metadata.Exists {
		t.Error("Expected metadata.Exists to be true")
	}

	if result.Metadata.IsLocal != true {
		t.Error("Expected metadata.IsLocal to be true")
	}
}

// TestJobOutputReader_ReadJobOutput_Remote tests reading remote job output
func TestJobOutputReader_ReadJobOutput_Remote(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "test -f") {
				return "exists", nil
			}
			if strings.Contains(command, "stat") {
				return "100 1609459200", nil
			}
			if strings.Contains(command, "cat") {
				return "Remote job output\nLine 2\n", nil
			}
			return "", fmt.Errorf("unexpected command: %s", command)
		},
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return "/remote/path/job_12345.out", true, "node01", nil // remote on node01
		},
	}

	reader := NewJobOutputReader(mockResolver, mockSSH)
	ctx := context.Background()

	result, err := reader.ReadJobOutput(ctx, "12345", "stdout")
	if err != nil {
		t.Fatalf("Failed to read remote job output: %v", err)
	}

	if !strings.Contains(result.Content, "Remote job output") {
		t.Errorf("Expected remote content, got: %s", result.Content)
	}

	if result.Source != "remote:node01" {
		t.Errorf("Source mismatch: got %s, want remote:node01", result.Source)
	}

	if result.Metadata.IsLocal {
		t.Error("Expected metadata.IsLocal to be false for remote file")
	}

	if result.Metadata.NodeID != "node01" {
		t.Errorf("NodeID mismatch: got %s, want node01", result.Metadata.NodeID)
	}
}

// TestJobOutputReader_ReadJobOutput_NotFound tests handling of missing files
func TestJobOutputReader_ReadJobOutput_NotFound(t *testing.T) {
	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return "/nonexistent/file.txt", false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	_, err := reader.ReadJobOutput(ctx, "12345", "stdout")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestJobOutputReader_ReadPartial_TailMode tests reading with tail mode
func TestJobOutputReader_ReadPartial_TailMode(t *testing.T) {
	// Create temporary test file with many lines (large enough to trigger tail mode)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	var lines []string
	for i := 1; i <= 100; i++ {
		// Make each line long enough so total exceeds MaxBytes
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

	// Read with tail mode - set low MaxBytes to trigger tail mode
	opts := ReadOptions{
		MaxBytes: 500, // File is much larger than this
		MaxLines: 10,
		TailMode: true,
	}

	result, err := reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read with tail mode: %v", err)
	}

	// Should have at most 10 lines
	resultLines := strings.Split(strings.TrimSpace(result.Content), "\n")
	if len(resultLines) > 10 {
		t.Errorf("Expected at most 10 lines, got %d", len(resultLines))
	}

	// Verify it's the last lines (line 91-100)
	if !strings.Contains(resultLines[len(resultLines)-1], "Line 100") {
		t.Errorf("Expected last line to contain 'Line 100', got: %s", resultLines[len(resultLines)-1])
	}
}

// TestJobOutputReader_ReadPartial_LargeFile tests handling of large files
func TestJobOutputReader_ReadPartial_LargeFile(t *testing.T) {
	// Create a file that exceeds the limit
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.out")
	content := strings.Repeat("A", 200) // Small but we'll set a low limit
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

	// Set low limit
	opts := ReadOptions{
		MaxBytes: 50, // File is 200 bytes
		MaxLines: 100,
		TailMode: true,
	}

	result, err := reader.ReadPartial(ctx, "12345", "stdout", opts)
	if err != nil {
		t.Fatalf("Failed to read large file: %v", err)
	}

	if !result.Truncated {
		t.Error("Expected Truncated to be true for large file")
	}
}

// TestJobOutputReader_GetMetadata tests getting file metadata
func TestJobOutputReader_GetMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	content := "Test content"
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

	metadata, err := reader.GetMetadata(ctx, "12345", "stdout")
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
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
}

// TestJobOutputReader_GetMetadata_Remote tests getting remote file metadata
func TestJobOutputReader_GetMetadata_Remote(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "stat") {
				return "2048 1609459200", nil
			}
			return "", fmt.Errorf("unexpected command: %s", command)
		},
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return "/remote/job_12345.out", true, "node02", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, mockSSH)
	ctx := context.Background()

	metadata, err := reader.GetMetadata(ctx, "12345", "stdout")
	if err != nil {
		t.Fatalf("Failed to get remote metadata: %v", err)
	}

	if !metadata.Exists {
		t.Error("Expected remote file to exist")
	}

	if metadata.Size != 2048 {
		t.Errorf("Size mismatch: got %d, want 2048", metadata.Size)
	}

	if metadata.IsLocal {
		t.Error("Expected file to be remote")
	}

	if metadata.NodeID != "node02" {
		t.Errorf("NodeID mismatch: got %s, want node02", metadata.NodeID)
	}
}

// TestJobOutputReader_PathResolutionError tests handling of path resolution errors
func TestJobOutputReader_PathResolutionError(t *testing.T) {
	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return "", false, "", fmt.Errorf("job not found")
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)
	ctx := context.Background()

	_, err := reader.ReadJobOutput(ctx, "99999", "stdout")
	if err == nil {
		t.Fatal("Expected error from path resolution, got nil")
	}

	if !strings.Contains(err.Error(), "failed to resolve") {
		t.Errorf("Expected path resolution error, got: %v", err)
	}
}

// TestJobOutputReader_ContextCancellation tests context cancellation
func TestJobOutputReader_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "job_12345.out")
	err := os.WriteFile(testFile, []byte("content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockResolver := &mockPathResolver{
		resolveFunc: func(jobID, outputType string) (string, bool, string, error) {
			return testFile, false, "", nil
		},
	}

	reader := NewJobOutputReader(mockResolver, nil)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = reader.ReadJobOutput(ctx, "12345", "stdout")
	if err == nil {
		t.Fatal("Expected error for canceled context, got nil")
	}

	// Error is wrapped, so check if it contains "context canceled"
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}
