package output

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SSHExecutor defines the interface for SSH command execution
type SSHExecutor interface {
	ExecuteCommand(ctx context.Context, hostname, command string) (string, error)
}

// RemoteFileReader handles reading files via SSH
type RemoteFileReader struct {
	sshClient      SSHExecutor
	maxTransfer    int64
	timeout        time.Duration
	useCompression bool
}

// NewRemoteFileReader creates a new remote file reader
func NewRemoteFileReader(sshClient SSHExecutor) *RemoteFileReader {
	return &RemoteFileReader{
		sshClient:      sshClient,
		maxTransfer:    10 * 1024 * 1024, // 10MB max transfer
		timeout:        30 * time.Second,
		useCompression: true,
	}
}

// ReadRemoteFile reads file via SSH cat command
func (r *RemoteFileReader) ReadRemoteFile(ctx context.Context, nodeID, path string) (string, error) {
	if r.sshClient == nil {
		return "", fmt.Errorf("SSH client not configured")
	}

	// First check if file exists
	checkCmd := fmt.Sprintf("test -f '%s' && echo exists || echo missing", escapePath(path))
	output, err := r.sshClient.ExecuteCommand(ctx, nodeID, checkCmd)
	if err != nil {
		return "", fmt.Errorf("failed to check file existence: %w", err)
	}

	if strings.TrimSpace(output) == "missing" {
		return "", fmt.Errorf("file does not exist on remote node: %s", path)
	}

	// Get file size
	sizeCmd := fmt.Sprintf("stat -c %%s '%s' 2>/dev/null || stat -f %%z '%s'", escapePath(path), escapePath(path))
	sizeOutput, err := r.sshClient.ExecuteCommand(ctx, nodeID, sizeCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get file size: %w", err)
	}

	var size int64
	_, err = fmt.Sscanf(strings.TrimSpace(sizeOutput), "%d", &size)
	if err != nil {
		return "", fmt.Errorf("failed to parse file size: %w", err)
	}

	// Check if file is too large
	if size > r.maxTransfer {
		return "", fmt.Errorf("file too large for remote transfer (%d bytes, max %d): use TailRemoteFile instead",
			size, r.maxTransfer)
	}

	// Read the file content
	catCmd := fmt.Sprintf("cat '%s'", escapePath(path))
	content, err := r.sshClient.ExecuteCommand(ctx, nodeID, catCmd)
	if err != nil {
		return "", fmt.Errorf("failed to read remote file: %w", err)
	}

	return content, nil
}

// TailRemoteFile reads last N lines via SSH tail
func (r *RemoteFileReader) TailRemoteFile(ctx context.Context, nodeID, path string, lines int) (string, error) {
	if r.sshClient == nil {
		return "", fmt.Errorf("SSH client not configured")
	}

	// Use tail command to read last N lines
	tailCmd := fmt.Sprintf("tail -n %d '%s'", lines, escapePath(path))
	output, err := r.sshClient.ExecuteCommand(ctx, nodeID, tailCmd)
	if err != nil {
		// Check if it's a "file not found" error
		if strings.Contains(err.Error(), "No such file") {
			return "", fmt.Errorf("file does not exist on remote node: %s", path)
		}
		return "", fmt.Errorf("failed to tail remote file: %w", err)
	}

	return output, nil
}

// HeadRemoteFile reads first N lines via SSH head
func (r *RemoteFileReader) HeadRemoteFile(ctx context.Context, nodeID, path string, lines int) (string, error) {
	if r.sshClient == nil {
		return "", fmt.Errorf("SSH client not configured")
	}

	// Use head command to read first N lines
	headCmd := fmt.Sprintf("head -n %d '%s'", lines, escapePath(path))
	output, err := r.sshClient.ExecuteCommand(ctx, nodeID, headCmd)
	if err != nil {
		// Check if it's a "file not found" error
		if strings.Contains(err.Error(), "No such file") {
			return "", fmt.Errorf("file does not exist on remote node: %s", path)
		}
		return "", fmt.Errorf("failed to head remote file: %w", err)
	}

	return output, nil
}

// GetRemoteFileInfo returns metadata about a remote file
func (r *RemoteFileReader) GetRemoteFileInfo(ctx context.Context, nodeID, path string) (*FileMetadata, error) {
	if r.sshClient == nil {
		return nil, fmt.Errorf("SSH client not configured")
	}

	// Check if file exists and get metadata using GNU stat (Linux HPC nodes)
	statCmd := fmt.Sprintf("stat -c '%%s %%Y' '%s'", escapePath(path))

	output, err := r.sshClient.ExecuteCommand(ctx, nodeID, statCmd)
	if err != nil {
		return &FileMetadata{
			Path:    path,
			Exists:  false,
			IsLocal: false,
			NodeID:  nodeID,
		}, nil
	}

	// Parse stat output: "size modtime"
	// Scan all lines to handle any SSH banners or warnings in combined output
	var size int64
	var modtime int64
	parsed := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var s, m int64
		if n, scanErr := fmt.Sscanf(line, "%d %d", &s, &m); n == 2 && scanErr == nil {
			size = s
			modtime = m
			parsed = true
			break
		}
	}
	if !parsed {
		// Non-empty output that didn't parse means the file likely doesn't exist
		// (stat exited 0 but output is unrecognizable) — treat as not found
		return &FileMetadata{
			Path:    path,
			Exists:  false,
			IsLocal: false,
			NodeID:  nodeID,
		}, nil
	}

	metadata := &FileMetadata{
		Path:     path,
		Size:     size,
		ModTime:  time.Unix(modtime, 0),
		Exists:   true,
		IsLocal:  false,
		NodeID:   nodeID,
		Encoding: "UTF-8", // Assume UTF-8
	}

	return metadata, nil
}

// HasSSHClient returns true if an SSH client is configured
func (r *RemoteFileReader) HasSSHClient() bool {
	return r.sshClient != nil
}

// escapePath escapes a file path for safe use in shell commands
func escapePath(path string) string {
	// Simple escape: replace single quotes with '\''
	return strings.ReplaceAll(path, "'", "'\\''")
}
