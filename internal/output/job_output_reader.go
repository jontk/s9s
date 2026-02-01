package output

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PathResolver defines the interface for resolving job output paths
type PathResolver interface {
	ResolveOutputPath(jobID, outputType string) (path string, isRemote bool, nodeID string, err error)
}

// JobOutputReader orchestrates reading job output files
type JobOutputReader struct {
	pathResolver PathResolver
	localReader  *LocalFileReader
	remoteReader *RemoteFileReader
	cache        *OutputCache
}

// NewJobOutputReader creates a new job output reader
func NewJobOutputReader(pathResolver PathResolver, sshClient SSHExecutor) *JobOutputReader {
	return &JobOutputReader{
		pathResolver: pathResolver,
		localReader:  NewLocalFileReader(),
		remoteReader: NewRemoteFileReader(sshClient),
		cache:        NewOutputCache(5*time.Minute, 100), // 5 min TTL, 100 entries max
	}
}

// SetCache sets a custom cache (useful for testing or custom cache configurations)
func (r *JobOutputReader) SetCache(cache *OutputCache) {
	r.cache = cache
}

// ClearCache clears the output cache
func (r *JobOutputReader) ClearCache() {
	if r.cache != nil {
		r.cache.Clear()
	}
}

// ReadJobOutput reads complete job output
func (r *JobOutputReader) ReadJobOutput(ctx context.Context, jobID, outputType string) (*OutputContent, error) {
	return r.ReadPartial(ctx, jobID, outputType, DefaultReadOptions())
}

// ReadPartial reads output with options
func (r *JobOutputReader) ReadPartial(ctx context.Context, jobID, outputType string, opts ReadOptions) (*OutputContent, error) {
	// Check cache if enabled and not forcing refresh
	if opts.CacheEnabled && !opts.ForceRefresh && r.cache != nil {
		cacheKey := GenerateCacheKey(jobID, outputType)
		if cached, found := r.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Resolve file path using PathResolver
	filePath, isRemote, nodeID, err := r.pathResolver.ResolveOutputPath(jobID, outputType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve output path: %w", err)
	}

	var content string
	var metadata *FileMetadata
	var source string

	if isRemote {
		// Read from remote node via SSH
		source = fmt.Sprintf("remote:%s", nodeID)

		// Get metadata first
		metadata, err = r.remoteReader.GetRemoteFileInfo(ctx, nodeID, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get remote file info: %w", err)
		}

		if !metadata.Exists {
			return nil, fmt.Errorf("output file not found on remote node %s: %s", nodeID, filePath)
		}

		// Check file size and determine read strategy
		if metadata.Size > opts.MaxBytes && opts.MaxBytes > 0 {
			// File is too large, use tail mode
			if opts.TailMode {
				content, err = r.remoteReader.TailRemoteFile(ctx, nodeID, filePath, opts.MaxLines)
			} else {
				content, err = r.remoteReader.HeadRemoteFile(ctx, nodeID, filePath, opts.MaxLines)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read remote file: %w", err)
			}
		} else {
			// Read full file
			content, err = r.remoteReader.ReadRemoteFile(ctx, nodeID, filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read remote file: %w", err)
			}
		}
	} else {
		// Read from local filesystem
		source = "local"

		// Get metadata first
		metadata, err = r.localReader.GetFileInfo(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get local file info: %w", err)
		}

		if !metadata.Exists {
			return nil, fmt.Errorf("output file not found: %s (job may not have started yet)", filePath)
		}

		// Check file size and determine read strategy
		if metadata.Size > opts.MaxBytes && opts.MaxBytes > 0 {
			// File is too large, use tail or head mode
			if opts.TailMode {
				content, err = r.localReader.TailFile(ctx, filePath, opts.MaxLines)
			} else {
				content, err = r.localReader.HeadFile(ctx, filePath, opts.MaxLines)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read local file: %w", err)
			}
		} else {
			// Read full file
			content, err = r.localReader.ReadFile(ctx, filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read local file: %w", err)
			}
		}
	}

	// Calculate lines read
	linesRead := strings.Count(content, "\n")
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		linesRead++ // Count last line if it doesn't end with newline
	}

	// Determine if content was truncated
	truncated := false
	if opts.MaxBytes > 0 && metadata.Size > opts.MaxBytes {
		truncated = true
	}
	if opts.MaxLines > 0 && linesRead >= opts.MaxLines {
		truncated = true
	}

	result := &OutputContent{
		Content:     content,
		Metadata:    metadata,
		Truncated:   truncated,
		TruncatedAt: int64(len(content)),
		LinesRead:   linesRead,
		BytesRead:   int64(len(content)),
		Source:      source,
	}

	// Store in cache if enabled
	if opts.CacheEnabled && r.cache != nil {
		cacheKey := GenerateCacheKey(jobID, outputType)
		r.cache.Set(cacheKey, result, 0) // Use default TTL
	}

	return result, nil
}

// GetMetadata returns file metadata without reading content
func (r *JobOutputReader) GetMetadata(ctx context.Context, jobID, outputType string) (*FileMetadata, error) {
	// Resolve file path using PathResolver
	filePath, isRemote, nodeID, err := r.pathResolver.ResolveOutputPath(jobID, outputType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve output path: %w", err)
	}

	if isRemote {
		// Get remote file metadata
		return r.remoteReader.GetRemoteFileInfo(ctx, nodeID, filePath)
	}

	// Get local file metadata
	return r.localReader.GetFileInfo(filePath)
}
