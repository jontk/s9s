package output

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockSSHClient implements a mock SSH client for testing
type mockSSHClient struct {
	executeFunc func(ctx context.Context, hostname, command string) (string, error)
}

func (m *mockSSHClient) ExecuteCommand(ctx context.Context, hostname, command string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, hostname, command)
	}
	return "", fmt.Errorf("mock not configured")
}

// TestRemoteFileReader_ReadRemoteFile tests reading a remote file
func TestRemoteFileReader_ReadRemoteFile(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			// Simulate file existence check
			if strings.Contains(command, "test -f") {
				return "exists", nil
			}
			// Simulate stat command
			if strings.Contains(command, "stat") {
				return "100", nil // 100 bytes
			}
			// Simulate cat command
			if strings.Contains(command, "cat") {
				return "Remote file content\nLine 2\nLine 3", nil
			}
			return "", fmt.Errorf("unexpected command: %s", command)
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	content, err := reader.ReadRemoteFile(ctx, "node01", "/path/to/file.txt")
	if err != nil {
		t.Fatalf("Failed to read remote file: %v", err)
	}

	expected := "Remote file content\nLine 2\nLine 3"
	if content != expected {
		t.Errorf("Content mismatch: got %q, want %q", content, expected)
	}
}

// TestRemoteFileReader_ReadRemoteFile_NotFound tests reading a non-existent remote file
func TestRemoteFileReader_ReadRemoteFile_NotFound(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			// Simulate file not found
			if strings.Contains(command, "test -f") {
				return "missing", nil
			}
			return "", fmt.Errorf("unexpected command: %s", command)
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	_, err := reader.ReadRemoteFile(ctx, "node01", "/path/to/missing.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

// TestRemoteFileReader_ReadRemoteFile_TooLarge tests handling of files too large for transfer
func TestRemoteFileReader_ReadRemoteFile_TooLarge(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "test -f") {
				return "exists", nil
			}
			if strings.Contains(command, "stat") {
				return "20971520", nil // 20MB
			}
			return "", fmt.Errorf("unexpected command: %s", command)
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	_, err := reader.ReadRemoteFile(ctx, "node01", "/path/to/large.txt")
	if err == nil {
		t.Fatal("Expected error for large file, got nil")
	}

	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("Expected 'too large' error, got: %v", err)
	}
}

// TestRemoteFileReader_TailRemoteFile tests reading last N lines from remote file
func TestRemoteFileReader_TailRemoteFile(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "tail -n 10") {
				return "Line 91\nLine 92\nLine 93\nLine 94\nLine 95\nLine 96\nLine 97\nLine 98\nLine 99\nLine 100", nil
			}
			return "", fmt.Errorf("unexpected command: %s", command)
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	content, err := reader.TailRemoteFile(ctx, "node01", "/path/to/file.txt", 10)
	if err != nil {
		t.Fatalf("Failed to tail remote file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 10 {
		t.Errorf("Expected 10 lines, got %d", len(lines))
	}
}

// TestRemoteFileReader_TailRemoteFile_NotFound tests tailing a non-existent remote file
func TestRemoteFileReader_TailRemoteFile_NotFound(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			return "", fmt.Errorf("tail: /path/to/missing.txt: No such file or directory")
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	_, err := reader.TailRemoteFile(ctx, "node01", "/path/to/missing.txt", 10)
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

// TestRemoteFileReader_HeadRemoteFile tests reading first N lines from remote file
func TestRemoteFileReader_HeadRemoteFile(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "head -n 5") {
				return "Line 1\nLine 2\nLine 3\nLine 4\nLine 5", nil
			}
			return "", fmt.Errorf("unexpected command: %s", command)
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	content, err := reader.HeadRemoteFile(ctx, "node01", "/path/to/file.txt", 5)
	if err != nil {
		t.Fatalf("Failed to head remote file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}
}

// TestRemoteFileReader_GetRemoteFileInfo tests getting metadata for remote file
func TestRemoteFileReader_GetRemoteFileInfo(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			if strings.Contains(command, "stat") {
				// Return "size modtime" format
				return "1024 1609459200", nil // 1KB, 2021-01-01 00:00:00 UTC
			}
			return "", fmt.Errorf("unexpected command: %s", command)
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	metadata, err := reader.GetRemoteFileInfo(ctx, "node01", "/path/to/file.txt")
	if err != nil {
		t.Fatalf("Failed to get remote file info: %v", err)
	}

	if !metadata.Exists {
		t.Error("Expected file to exist")
	}

	if metadata.Size != 1024 {
		t.Errorf("Size mismatch: got %d, want 1024", metadata.Size)
	}

	if metadata.IsLocal {
		t.Error("Expected file to be remote")
	}

	if metadata.NodeID != "node01" {
		t.Errorf("NodeID mismatch: got %s, want node01", metadata.NodeID)
	}

	expectedTime := time.Unix(1609459200, 0)
	if !metadata.ModTime.Equal(expectedTime) {
		t.Errorf("ModTime mismatch: got %v, want %v", metadata.ModTime, expectedTime)
	}
}

// TestRemoteFileReader_GetRemoteFileInfo_NotFound tests getting info for non-existent remote file
func TestRemoteFileReader_GetRemoteFileInfo_NotFound(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			return "", fmt.Errorf("stat: cannot stat '/path/to/missing.txt': No such file or directory")
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	metadata, err := reader.GetRemoteFileInfo(ctx, "node01", "/path/to/missing.txt")
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}

	if metadata.Exists {
		t.Error("Expected file to not exist")
	}

	if metadata.NodeID != "node01" {
		t.Errorf("NodeID mismatch: got %s, want node01", metadata.NodeID)
	}
}

// TestRemoteFileReader_NilSSHClient tests handling of nil SSH client
func TestRemoteFileReader_NilSSHClient(t *testing.T) {
	reader := NewRemoteFileReader(nil)
	ctx := context.Background()

	// Test ReadRemoteFile
	_, err := reader.ReadRemoteFile(ctx, "node01", "/path/to/file.txt")
	if err == nil {
		t.Error("Expected error for nil SSH client, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Expected 'not configured' error, got: %v", err)
	}

	// Test TailRemoteFile
	_, err = reader.TailRemoteFile(ctx, "node01", "/path/to/file.txt", 10)
	if err == nil {
		t.Error("Expected error for nil SSH client, got nil")
	}

	// Test HeadRemoteFile
	_, err = reader.HeadRemoteFile(ctx, "node01", "/path/to/file.txt", 10)
	if err == nil {
		t.Error("Expected error for nil SSH client, got nil")
	}

	// Test GetRemoteFileInfo
	_, err = reader.GetRemoteFileInfo(ctx, "node01", "/path/to/file.txt")
	if err == nil {
		t.Error("Expected error for nil SSH client, got nil")
	}
}

// TestRemoteFileReader_SSHError tests handling of SSH errors
func TestRemoteFileReader_SSHError(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			return "", fmt.Errorf("ssh: connect to host node01 port 22: Connection refused")
		},
	}

	reader := NewRemoteFileReader(mockSSH)
	ctx := context.Background()

	_, err := reader.ReadRemoteFile(ctx, "node01", "/path/to/file.txt")
	if err == nil {
		t.Fatal("Expected SSH error, got nil")
	}

	if !strings.Contains(err.Error(), "Connection refused") {
		t.Errorf("Expected connection error, got: %v", err)
	}
}

// TestRemoteFileReader_ContextCancellation tests handling of context cancellation
func TestRemoteFileReader_ContextCancellation(t *testing.T) {
	mockSSH := &mockSSHClient{
		executeFunc: func(ctx context.Context, hostname, command string) (string, error) {
			// Check if context is canceled
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
				return "exists", nil
			}
		},
	}

	reader := NewRemoteFileReader(mockSSH)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := reader.TailRemoteFile(ctx, "node01", "/path/to/file.txt", 10)
	if err == nil {
		t.Fatal("Expected error for canceled context, got nil")
	}
}

// TestEscapePath tests shell path escaping
func TestEscapePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/simple/path.txt", "/simple/path.txt"},
		{"/path with spaces/file.txt", "/path with spaces/file.txt"},
		{"/path/with'quote.txt", "/path/with'\\''quote.txt"},
		{"/path/with'multiple'quotes.txt", "/path/with'\\''multiple'\\''quotes.txt"},
	}

	for _, tt := range tests {
		result := escapePath(tt.input)
		if result != tt.expected {
			t.Errorf("escapePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
