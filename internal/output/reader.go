// Package output provides functionality for reading SLURM job output files.
// It supports both local filesystem access and remote file retrieval via SSH,
// with efficient handling of large files through streaming and pagination.
package output

import (
	"context"
	"os"
	"time"
)

// ReadOptions configures how output is read
type ReadOptions struct {
	MaxBytes      int64         // Maximum bytes to read (0 = unlimited)
	MaxLines      int           // Maximum lines to read (0 = unlimited)
	TailMode      bool          // Read from end instead of beginning
	Offset        int64         // Byte offset to start reading
	Timeout       time.Duration // Timeout for remote operations
	CacheEnabled  bool          // Enable local caching
	ForceRefresh  bool          // Bypass cache
}

// DefaultReadOptions returns sensible defaults for read operations
func DefaultReadOptions() ReadOptions {
	return ReadOptions{
		MaxBytes:     10 * 1024 * 1024, // 10MB default
		MaxLines:     10000,             // 10k lines default
		TailMode:     false,
		Offset:       0,
		Timeout:      30 * time.Second,
		CacheEnabled: true,
		ForceRefresh: false,
	}
}

// FileMetadata contains file information
type FileMetadata struct {
	Path        string
	Size        int64
	ModTime     time.Time
	Exists      bool
	IsLocal     bool
	NodeID      string
	Permissions os.FileMode
	Encoding    string
}

// OutputContent represents read output
type OutputContent struct {
	Content     string
	Metadata    *FileMetadata
	Truncated   bool
	TruncatedAt int64
	LinesRead   int
	BytesRead   int64
	Source      string // "local" or "remote:node042"
}

// Reader interface for job output
type Reader interface {
	// ReadJobOutput reads complete job output
	ReadJobOutput(ctx context.Context, jobID, outputType string) (*OutputContent, error)

	// ReadPartial reads output with options
	ReadPartial(ctx context.Context, jobID, outputType string, opts ReadOptions) (*OutputContent, error)

	// GetMetadata returns file metadata without reading content
	GetMetadata(ctx context.Context, jobID, outputType string) (*FileMetadata, error)
}
