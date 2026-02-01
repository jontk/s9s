package output

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// LocalFileReader handles reading files from the local filesystem
type LocalFileReader struct {
	maxMemoryBuffer int64 // Maximum memory for buffering (default 100MB)
}

// NewLocalFileReader creates a new local file reader
func NewLocalFileReader() *LocalFileReader {
	return &LocalFileReader{
		maxMemoryBuffer: 100 * 1024 * 1024, // 100MB
	}
}

// ReadFile reads a complete local file
func (r *LocalFileReader) ReadFile(ctx context.Context, path string) (string, error) {
	// Check context first
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// Get file info first to check size
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", path)
		}
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// If file is too large, return error suggesting partial read
	if info.Size() > r.maxMemoryBuffer {
		return "", fmt.Errorf("file too large (%d bytes, max %d): use TailFile or HeadFile instead",
			info.Size(), r.maxMemoryBuffer)
	}

	// Read the file
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }() // Close error irrelevant for read operations

	// Read with context cancellation check
	contentBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Check if content is valid UTF-8
	if !utf8.Valid(contentBytes) {
		return "", fmt.Errorf("file contains non-UTF-8 content (binary file?)")
	}

	return string(contentBytes), nil
}

// TailFile reads last N lines efficiently
func (r *LocalFileReader) TailFile(ctx context.Context, path string, lines int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", path)
		}
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }() // Close error irrelevant for read operations

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// For small files, just read everything
	if stat.Size() < 1024*1024 { // < 1MB
		content, err := io.ReadAll(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return r.getLastNLines(string(content), lines), nil
	}

	// For large files, read from the end
	// Estimate: 100 bytes per line average
	estimatedBytes := int64(lines * 100)
	startPos := stat.Size() - estimatedBytes
	if startPos < 0 {
		startPos = 0
	}

	_, err = file.Seek(startPos, 0)
	if err != nil {
		return "", fmt.Errorf("failed to seek in file: %w", err)
	}

	// Read from estimated position
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return r.getLastNLines(string(content), lines), nil
}

// HeadFile reads first N lines
func (r *LocalFileReader) HeadFile(ctx context.Context, path string, lines int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", path)
		}
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }() // Close error irrelevant for read operations

	scanner := bufio.NewScanner(file)
	var result strings.Builder
	lineCount := 0

	for scanner.Scan() && lineCount < lines {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		result.WriteString(scanner.Text())
		result.WriteString("\n")
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return result.String(), nil
}

// ReadRange reads specific byte range
func (r *LocalFileReader) ReadRange(ctx context.Context, path string, offset, length int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", path)
		}
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }() // Close error irrelevant for read operations

	// Seek to offset
	_, err = file.Seek(offset, 0)
	if err != nil {
		return "", fmt.Errorf("failed to seek to offset: %w", err)
	}

	// Read specified length
	buffer := make([]byte, length)
	n, err := io.ReadFull(file, buffer)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("failed to read range: %w", err)
	}

	return string(buffer[:n]), nil
}

// GetFileInfo returns file metadata
func (r *LocalFileReader) GetFileInfo(path string) (*FileMetadata, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileMetadata{
				Path:   path,
				Exists: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	metadata := &FileMetadata{
		Path:        path,
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		Exists:      true,
		IsLocal:     true,
		Permissions: info.Mode(),
		Encoding:    "UTF-8", // Assume UTF-8, could be detected
	}

	return metadata, nil
}

// getLastNLines returns the last N lines from a string
func (r *LocalFileReader) getLastNLines(content string, n int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= n {
		return content
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
