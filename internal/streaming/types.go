package streaming

import (
	"context"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ssh"
)

// StreamEventType represents the type of stream event
type StreamEventType string

const (
	StreamEventNewOutput   StreamEventType = "NEW_OUTPUT"
	StreamEventJobComplete StreamEventType = "JOB_COMPLETE"
	StreamEventError       StreamEventType = "ERROR"
	StreamEventFileRotated StreamEventType = "FILE_ROTATED"
	StreamEventStreamStart StreamEventType = "STREAM_START"
	StreamEventStreamStop  StreamEventType = "STREAM_STOP"
)

// StreamEvent represents an event in the streaming system
type StreamEvent struct {
	JobID      string          `json:"job_id"`
	OutputType string          `json:"output_type"` // "stdout" or "stderr"
	Content    string          `json:"content"`
	NewLines   []string        `json:"new_lines"` // New lines added since last event
	Timestamp  time.Time       `json:"timestamp"`
	EventType  StreamEventType `json:"event_type"`
	FileOffset int64           `json:"file_offset"` // Current file position
	Error      error           `json:"error,omitempty"`
}

// StreamManager manages real-time job output streaming
type StreamManager struct {
	client        dao.SlurmClient     // Uses SLURM API to get job metadata including file paths
	sshManager    *ssh.SessionManager // For remote file access
	fileWatcher   *fsnotify.Watcher   // For local file watching
	activeStreams map[string]*JobStream
	eventBus      *EventBus
	slurmConfig   *SlurmConfig // SLURM fallback paths and settings
	pathResolver  *PathResolver
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// JobStream represents an active streaming session for a job's output
type JobStream struct {
	JobID       string               // SLURM job ID
	OutputType  string               // "stdout" or "stderr"
	Buffer      *CircularBuffer      // Memory-efficient output storage
	FilePath    string               // Path to output file
	LastOffset  int64                // File offset for tailing
	IsActive    bool                 // Whether streaming is active
	IsRemote    bool                 // Local or remote file
	NodeID      string               // For remote files
	Subscribers []chan<- StreamEvent // Event subscribers
	FileWatcher *FileWatcher         // Individual file watcher
	LastUpdate  time.Time            // Last update timestamp
	// TODO(lint): Review unused code - field mu is unused
	// mu            sync.RWMutex         // Stream-specific mutex
}

// SlurmConfig contains SLURM-specific configuration for streaming
type SlurmConfig struct {
	// File paths (fallback when SLURM API doesn't provide paths)
	OutputDir    string `json:"output_dir"`    // Default: /var/spool/slurm/slurmd/job_output
	ErrorDir     string `json:"error_dir"`     // Usually same as OutputDir
	FilePattern  string `json:"file_pattern"`  // Default: "slurm-{jobid}.out"
	ErrorPattern string `json:"error_pattern"` // Default: "slurm-{jobid}.err"

	// Node configuration
	RemoteAccess bool     `json:"remote_access"` // Enable SSH for remote files
	LocalNodes   []string `json:"local_nodes"`   // Nodes accessible via local filesystem
	SSHUser      string   `json:"ssh_user"`      // Default SSH username
	SSHKeyPath   string   `json:"ssh_key_path"`  // SSH private key path

	// Performance tuning
	FileCheckInterval time.Duration `json:"file_check_interval"` // 1s
	MaxFileSize       int64         `json:"max_file_size"`       // 100MB
	BufferSize        int           `json:"buffer_size"`         // 10000 lines
}

// StreamConfig contains user-configurable streaming settings
type StreamConfig struct {
	MaxConcurrentStreams int           `json:"max_concurrent_streams"` // Default: 4
	BufferSize           int           `json:"buffer_size_lines"`      // Default: 10000
	PollInterval         time.Duration `json:"poll_interval"`          // Default: 2s
	MaxMemoryMB          int           `json:"max_memory_mb"`          // Default: 50
	AutoScroll           bool          `json:"auto_scroll_default"`    // Default: true
	ShowTimestamps       bool          `json:"show_timestamps"`        // Default: true
	ExportFormat         string        `json:"export_format"`          // Default: "txt"
}

// StreamingStats contains statistics about streaming operations
type StreamingStats struct {
	ActiveStreams int           `json:"active_streams"`
	TotalStreams  int           `json:"total_streams"`
	MemoryUsage   int64         `json:"memory_usage_bytes"`
	TotalEvents   int64         `json:"total_events"`
	ErrorCount    int64         `json:"error_count"`
	Uptime        time.Duration `json:"uptime"`
	LastError     error         `json:"last_error,omitempty"`
	LastErrorTime time.Time     `json:"last_error_time,omitempty"`
}

// FileWatcher handles individual file watching operations
type FileWatcher struct {
	FilePath   string
	IsRemote   bool
	NodeID     string
	LastOffset int64
	// TODO(lint): Review unused code - field mu is unused
	// mu          sync.RWMutex
}

// Default configuration values
const (
	DefaultOutputDir            = "/var/spool/slurm/slurmd/job_output"
	DefaultFilePattern          = "slurm-%s.out"
	DefaultErrorPattern         = "slurm-%s.err"
	DefaultBufferSize           = 10000
	DefaultMaxConcurrentStreams = 4
	DefaultPollInterval         = 2 * time.Second
	DefaultMaxMemoryMB          = 50
	DefaultFileCheckInterval    = 1 * time.Second
	DefaultMaxFileSize          = 100 * 1024 * 1024 // 100MB
)

// DefaultSlurmConfig returns default SLURM configuration
func DefaultSlurmConfig() *SlurmConfig {
	return &SlurmConfig{
		OutputDir:         DefaultOutputDir,
		ErrorDir:          DefaultOutputDir,
		FilePattern:       DefaultFilePattern,
		ErrorPattern:      DefaultErrorPattern,
		RemoteAccess:      true,
		LocalNodes:        []string{},
		SSHUser:           "",
		SSHKeyPath:        "",
		FileCheckInterval: DefaultFileCheckInterval,
		MaxFileSize:       DefaultMaxFileSize,
		BufferSize:        DefaultBufferSize,
	}
}

// DefaultStreamConfig returns default streaming configuration
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
		BufferSize:           DefaultBufferSize,
		PollInterval:         DefaultPollInterval,
		MaxMemoryMB:          DefaultMaxMemoryMB,
		AutoScroll:           true,
		ShowTimestamps:       true,
		ExportFormat:         "txt",
	}
}
