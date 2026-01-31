# Job Streaming Guide

s9s provides powerful real-time job log streaming capabilities that allow you to monitor job output as it's being written, similar to `tail -f` but with advanced features for SLURM environments.

## Overview

The streaming system supports:
- **Real-time log tailing**: Watch job stdout/stderr as it's written
- **Multi-stream monitoring**: Monitor multiple jobs simultaneously
- **SSH-based streaming**: Stream logs from compute nodes via SSH
- **File-based streaming**: Monitor log files on shared filesystems
- **Intelligent path resolution**: Automatically find job output files
- **Advanced filtering**: Filter log content in real-time
- **Buffer management**: Handle high-volume log streams efficiently

## Quick Start

### Basic Streaming
1. Open s9s: `s9s` or `s9s --mock`
2. Navigate to Jobs view
3. Select a running job
4. Press `o` to view job output
5. Press `s` to start streaming mode

### Stream Monitor
1. Press `Ctrl+s` to open Stream Monitor
2. Add streams using `+` key
3. Navigate between streams with arrow keys
4. Use `Ctrl+f` for filtering

## Streaming Sources

### 1. File-based Streaming

Monitor log files on shared storage:
```bash
# SLURM output files (default pattern)
/path/to/slurm/output/job_%j.out
/path/to/slurm/output/job_%j.err

# Custom paths
/shared/logs/job_123.log
/nfs/user/logs/my_job.out
```

**Configuration:**
```yaml
streaming:
  file_paths:
    base_dir: "/shared/slurm/logs"
    stdout_pattern: "job_%j.out"
    stderr_pattern: "job_%j.err"
  polling_interval: 100ms
  buffer_size: 64KB
```

### 2. SSH-based Streaming

Stream logs directly from compute nodes:
```bash
# Stream from specific node
ssh node001 tail -f /tmp/job_123.out

# Auto-detect job location
s9s stream --job 123 --auto-detect
```

**Configuration:**
```yaml
streaming:
  ssh:
    enabled: true
    timeout: 30s
    max_connections: 10
    commands:
      tail: "tail -f"
      grep: "grep"
```

### 3. SLURM API Streaming

Use SLURM's built-in streaming (if available):
```yaml
streaming:
  slurm_api:
    enabled: true
    endpoint: "/slurm/v0.0.43/job/{job_id}/output"
    chunk_size: 4KB
```

## Stream Monitor Interface

### Multi-Stream Layout

The Stream Monitor can display multiple job streams simultaneously:

```
┌─ Stream Monitor ─────────────────────────────────────┐
│ ┌─Job 123─┐ ┌─Job 124─┐ ┌─Job 125─┐ ┌─Job 126─┐     │
│ │[STDOUT] │ │[STDERR] │ │[STDOUT] │ │[STDOUT] │     │
│ │Running  │ │Running  │ │Running  │ │Running  │     │
│ │         │ │ERROR:   │ │Progress │ │Finished │     │
│ │Output   │ │Failed   │ │50%      │ │Success  │     │
│ │line 1   │ │to load  │ │Working  │ │100%     │     │
│ │Output   │ │module   │ │...      │ │Done.    │     │
│ │line 2   │ │         │ │         │ │         │     │
│ └─────────┘ └─────────┘ └─────────┘ └─────────┘     │
├─────────────────────────────────────────────────────┤
│ Streams: 4 | Active: 3 | Ctrl+s:Add | q:Quit       │
└─────────────────────────────────────────────────────┘
```

### Key Bindings

| Key | Action |
|-----|--------|
| `Ctrl+s` | Open Stream Monitor |
| `+` | Add new stream |
| `-` | Remove current stream |
| `←/→` | Navigate between streams |
| `↑/↓` | Scroll within stream |
| `f` | Filter content |
| `c` | Clear stream buffer |
| `p` | Pause/resume streaming |
| `s` | Save stream to file |
| `r` | Restart stream |
| `q` | Close Stream Monitor |

## Advanced Features

### Real-time Filtering

Apply filters to streaming content:

```bash
# Filter for errors
s9s stream --job 123 --filter "ERROR"

# Multiple filters
s9s stream --job 123 --filter "ERROR|WARN|FAIL"

# Exclude patterns
s9s stream --job 123 --exclude "DEBUG|TRACE"
```

**Filter Configuration:**
```yaml
streaming:
  filters:
    - name: "errors"
      pattern: "ERROR|FATAL|EXCEPTION"
      color: "red"
      highlight: true
    - name: "warnings"
      pattern: "WARN|WARNING"
      color: "yellow"
    - name: "progress"
      pattern: "Progress:|\\d+%"
      color: "green"
```

### Buffer Management

Handle high-volume streams efficiently:

```yaml
streaming:
  buffer:
    max_size: 10MB          # Maximum buffer size per stream
    max_lines: 10000        # Maximum lines to keep
    cleanup_threshold: 80   # Cleanup when buffer reaches 80%
    compression: true       # Compress old buffer data
```

### Stream Persistence

Save streaming sessions:

```yaml
streaming:
  persistence:
    enabled: true
    base_dir: "~/.s9s/streams"
    auto_save: true
    max_files: 100
```

## Configuration

### Complete Streaming Configuration

```yaml
streaming:
  # General settings
  enabled: true
  max_concurrent_streams: 20
  default_buffer_size: 1MB
  refresh_interval: 100ms

  # File-based streaming
  file_paths:
    base_dir: "/shared/slurm/logs"
    stdout_pattern: "slurm-%j.out"
    stderr_pattern: "slurm-%j.err"
    custom_patterns:
      - "/nfs/logs/job_%j/*.log"
      - "/tmp/slurm_%j_*.out"

  # SSH streaming
  ssh:
    enabled: true
    timeout: 30s
    max_connections: 5
    retry_attempts: 3
    key_file: "~/.ssh/id_rsa"

  # SLURM API streaming
  slurm_api:
    enabled: false
    chunk_size: 8KB
    poll_interval: 1s

  # Filters
  filters:
    - name: "errors"
      pattern: "(?i)error|fail|exception|fatal"
      action: "highlight"
      color: "red"
    - name: "warnings"
      pattern: "(?i)warn|warning"
      action: "highlight"
      color: "yellow"
    - name: "progress"
      pattern: "\\d+(\\.\\d+)?%|progress:"
      action: "highlight"
      color: "green"

  # Buffer management
  buffer:
    max_size_per_stream: 10MB
    max_total_size: 100MB
    max_lines: 50000
    cleanup_interval: 5m

  # Persistence
  persistence:
    enabled: true
    directory: "~/.s9s/stream_history"
    max_files: 50
    max_file_size: 100MB
    auto_cleanup_days: 7
```

## Troubleshooting

### Common Issues

#### Stream Not Starting

**Symptoms**: No output appears when streaming

**Causes**: File not found, permission issues, job not started

**Solutions**:
```bash
# Check if job output files exist
scontrol show job 123 | grep StdOut

# Verify file permissions
ls -la /path/to/job_output.out

# Check SSH connectivity
ssh node001 tail -f /tmp/job_123.out
```

#### High CPU/Memory Usage

**Symptoms**: s9s consuming excessive resources

**Causes**: Too many streams, large buffers, inefficient filtering

**Solutions**:
```yaml
streaming:
  max_concurrent_streams: 5    # Reduce concurrent streams
  buffer:
    max_size_per_stream: 1MB   # Smaller buffers
  refresh_interval: 500ms      # Slower refresh rate
```

#### SSH Connection Failures

**Symptoms**: "Connection refused" or timeout errors

**Causes**: SSH not configured, firewall, authentication issues

**Solutions**:
```bash
# Test SSH connectivity
ssh -o ConnectTimeout=10 node001 echo "test"

# Configure SSH keys
ssh-copy-id node001

# Update SSH config
cat >> ~/.ssh/config << 'EOF'
Host node*
    StrictHostKeyChecking no
    ConnectTimeout 30
    ServerAliveInterval 60
EOF
```

#### Missing Log Files

**Symptoms**: "File not found" errors

**Causes**: Incorrect paths, job not writing output, timing issues

**Solutions**:
```bash
# Check SLURM output configuration
scontrol show config | grep SlurmdLogFile

# Verify job script output redirection
cat job_script.sh | grep -E "(>|>>)"

# Use custom path patterns
s9s stream --job 123 --path "/custom/path/job_%j.log"
```

### Debug Mode

Enable detailed logging for troubleshooting:

```bash
# Enable streaming debug
export S9S_STREAMING_DEBUG=true
s9s --debug

# View streaming logs
tail -f ~/.s9s/debug.log | grep STREAM
```

## Performance Optimization

### For High-Volume Logs

```yaml
streaming:
  # Larger buffers for efficiency
  buffer:
    max_size_per_stream: 50MB
    max_lines: 100000

  # Batch updates
  refresh_interval: 200ms

  # Compression
  compression:
    enabled: true
    algorithm: "gzip"
    level: 6
```

### For Many Streams

```yaml
streaming:
  # Connection pooling
  ssh:
    connection_pooling: true
    max_idle_connections: 10

  # Efficient polling
  file_polling:
    use_inotify: true
    batch_size: 100

  # Resource limits
  max_concurrent_streams: 10
  memory_limit: 500MB
```

### For Slow Networks

```yaml
streaming:
  # Larger chunks
  ssh:
    chunk_size: 64KB

  # Compression
  compression:
    enabled: true

  # Slower refresh
  refresh_interval: 1s
```

## Integration Examples

### With Monitoring Tools

```bash
# Export streaming metrics
s9s stream --job 123 --export-metrics prometheus://localhost:9090

# Forward to syslog
s9s stream --job 123 --forward syslog://logserver:514
```

### With Alerting

```yaml
streaming:
  alerts:
    - pattern: "(?i)error|exception|fatal"
      action: "webhook"
      url: "https://alerts.example.com/webhook"
    - pattern: "(?i)completed successfully"
      action: "notification"
      type: "success"
```

### Scripting Integration

```bash
#!/bin/bash
# Start streaming and process output
s9s stream --job $JOB_ID --output-format json | \
jq -r '.content' | \
while IFS= read -r line; do
    if [[ $line == *"ERROR"* ]]; then
        echo "Alert: $line" | mail -s "Job Error" admin@example.com
    fi
done
```

## API Reference

### Streaming API

```go
// Start streaming a job
manager.StartStream(jobID, "stdout")

// Add filter
manager.AddFilter("errors", "ERROR|FAIL")

// Get stream data
events := manager.GetStream(jobID)
for event := range events {
    fmt.Printf("%s: %s\n", event.Timestamp, event.Content)
}
```

### Event Types

```go
type StreamEvent struct {
    JobID     string    `json:"job_id"`
    Type      string    `json:"type"`      // "stdout", "stderr"
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
    LineNum   int       `json:"line_num"`
    Source    string    `json:"source"`    // "file", "ssh", "api"
}
```

## Related Guides

- [Configuration Guide](../CONFIGURATION.md)
- [SSH Integration](./ssh-guide.md)
- [Performance Analysis](./performance-analysis.md)
