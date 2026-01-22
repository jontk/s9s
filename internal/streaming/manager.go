package streaming

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ssh"
)

// NewStreamManager creates a new stream manager
func NewStreamManager(client dao.SlurmClient, sshManager *ssh.SessionManager, config *SlurmConfig) (*StreamManager, error) {
	if config == nil {
		config = DefaultSlurmConfig()
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sm := &StreamManager{
		client:        client,
		sshManager:    sshManager,
		fileWatcher:   watcher,
		activeStreams: make(map[string]*JobStream),
		eventBus:      NewEventBus(),
		slurmConfig:   config,
		pathResolver:  NewPathResolver(client, config),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start the file watcher goroutine
	go sm.watchFileEvents()

	return sm, nil
}

// StartStream begins watching job output file
func (sm *StreamManager) StartStream(jobID, outputType string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	streamKey := sm.makeStreamKey(jobID, outputType)

	// Check if stream already exists
	if stream, exists := sm.activeStreams[streamKey]; exists {
		if stream.IsActive {
			return fmt.Errorf("stream already active for job %s %s", jobID, outputType)
		}
	}

	// Resolve file path using SLURM API
	filePath, isRemote, nodeID, err := sm.pathResolver.ResolveOutputPath(jobID, outputType)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Validate the path
	if err := sm.pathResolver.ValidateOutputPath(filePath, isRemote, nodeID); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Create new job stream
	stream := &JobStream{
		JobID:       jobID,
		OutputType:  outputType,
		Buffer:      NewCircularBuffer(sm.slurmConfig.BufferSize),
		FilePath:    filePath,
		LastOffset:  0,
		IsActive:    true,
		IsRemote:    isRemote,
		NodeID:      nodeID,
		Subscribers: make([]chan<- StreamEvent, 0),
		LastUpdate:  GetCurrentTime(),
	}

	// Store the stream
	sm.activeStreams[streamKey] = stream

	// Start watching the file
	if err := sm.startFileWatching(stream); err != nil {
		delete(sm.activeStreams, streamKey)
		return fmt.Errorf("failed to start file watching: %w", err)
	}

	// Publish stream start event
	sm.eventBus.Publish(StreamEvent{
		JobID:      jobID,
		OutputType: outputType,
		EventType:  StreamEventStreamStart,
		Timestamp:  GetCurrentTime(),
	})

	return nil
}

// StopStream stops watching job output file
func (sm *StreamManager) StopStream(jobID, outputType string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	streamKey := sm.makeStreamKey(jobID, outputType)

	stream, exists := sm.activeStreams[streamKey]
	if !exists {
		return fmt.Errorf("no active stream for job %s %s", jobID, outputType)
	}

	// Stop the stream
	stream.IsActive = false

	// Remove file watcher if local
	if !stream.IsRemote {
		_ = sm.fileWatcher.Remove(stream.FilePath)
	}

	// Publish stream stop event
	sm.eventBus.Publish(StreamEvent{
		JobID:      jobID,
		OutputType: outputType,
		EventType:  StreamEventStreamStop,
		Timestamp:  GetCurrentTime(),
	})

	// Clean up after a delay to allow final events to be processed
	go func() {
		time.Sleep(1 * time.Second)
		sm.mu.Lock()
		delete(sm.activeStreams, streamKey)
		sm.mu.Unlock()

		// Unsubscribe all subscribers
		sm.eventBus.UnsubscribeAll(jobID, outputType)
	}()

	return nil
}

// Subscribe adds a subscriber for stream events
func (sm *StreamManager) Subscribe(jobID, outputType string) <-chan StreamEvent {
	ch := make(chan StreamEvent, 100) // Buffered channel
	sm.eventBus.Subscribe(jobID, outputType, ch)
	return ch
}

// Unsubscribe removes a subscriber
func (sm *StreamManager) Unsubscribe(jobID, outputType string, ch <-chan StreamEvent) {
	// Note: This is a limitation of Go's type system
	// In practice, the channel would be stored internally and managed properly
	// For now, we'll use UnsubscribeAll as a workaround
	sm.eventBus.UnsubscribeAll(jobID, outputType)
}

// GetBuffer returns the current buffer contents for a stream
func (sm *StreamManager) GetBuffer(jobID, outputType string) ([]string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	streamKey := sm.makeStreamKey(jobID, outputType)
	stream, exists := sm.activeStreams[streamKey]
	if !exists {
		return nil, fmt.Errorf("no stream found for job %s %s", jobID, outputType)
	}

	return stream.Buffer.GetLines(), nil
}

// GetStats returns streaming statistics
func (sm *StreamManager) GetStats() StreamingStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var memoryUsage int64
	activeCount := 0

	for _, stream := range sm.activeStreams {
		if stream.IsActive {
			activeCount++
		}
		memoryUsage += stream.Buffer.EstimateMemoryUsage()
	}

	return StreamingStats{
		ActiveStreams: activeCount,
		TotalStreams:  len(sm.activeStreams),
		MemoryUsage:   memoryUsage,
		// Other stats would be tracked over time
	}
}

// Close shuts down the stream manager
func (sm *StreamManager) Close() error {
	sm.cancel()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Stop all streams
	for _, stream := range sm.activeStreams {
		stream.IsActive = false
	}

	// Close file watcher
	if sm.fileWatcher != nil {
		_ = sm.fileWatcher.Close()
	}

	// Clear event bus
	sm.eventBus.Clear()

	return nil
}

// startFileWatching begins monitoring a file for changes
func (sm *StreamManager) startFileWatching(stream *JobStream) error {
	if stream.IsRemote {
		return sm.startRemoteFileWatching(stream)
	}
	return sm.startLocalFileWatching(stream)
}

// startLocalFileWatching uses fsnotify for local file monitoring
func (sm *StreamManager) startLocalFileWatching(stream *JobStream) error {
	// Read existing content first
	content, offset, err := sm.readFileFromOffset(stream.FilePath, stream.LastOffset)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing content: %w", err)
	}

	if len(content) > 0 {
		sm.emitNewContent(stream, content, offset)
		stream.LastOffset = offset
	}

	// Add file to watcher
	err = sm.fileWatcher.Add(stream.FilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to watch file %s: %w", stream.FilePath, err)
	}

	return nil
}

// startRemoteFileWatching uses SSH tail for remote file monitoring
func (sm *StreamManager) startRemoteFileWatching(stream *JobStream) error {
	if sm.sshManager == nil {
		return fmt.Errorf("SSH manager not available for remote streaming")
	}

	// For now, return an error as SSH connection method needs to be implemented
	// TODO: Implement proper SSH connection handling
	return fmt.Errorf("remote file streaming not yet implemented - SSH connection interface needs updating")
}

/*
TODO(lint): Review unused code - func (*StreamManager).remoteFileTailer is unused

remoteFileTailer handles SSH-based file tailing
func (sm *StreamManager) remoteFileTailer(stream *JobStream, sshConn interface{}) {
	defer func() {
		stream.mu.Lock()
		stream.IsActive = false
		stream.mu.Unlock()
	}()

	// TODO: Implement SSH session creation and command execution
	// This is a placeholder for proper SSH implementation
	sm.emitError(stream, fmt.Errorf("SSH session creation not implemented"))
	return
}
*/

/*
TODO(lint): Review unused code - func (*StreamManager).streamRemoteOutput is unused

streamRemoteOutput processes output from remote tail command
func (sm *StreamManager) streamRemoteOutput(stream *JobStream, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	var buffer strings.Builder

	for scanner.Scan() {
		stream.mu.RLock()
		isActive := stream.IsActive
		stream.mu.RUnlock()

		if !isActive {
			break
		}

		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteString("\n")

		// Emit content periodically or when buffer gets large
		if buffer.Len() > 1024 {
			content := buffer.String()
			buffer.Reset()

			sm.emitNewContent(stream, content, stream.LastOffset+int64(len(content)))
			stream.LastOffset += int64(len(content))
		}
	}

	// Emit any remaining content
	if buffer.Len() > 0 {
		content := buffer.String()
		sm.emitNewContent(stream, content, stream.LastOffset+int64(len(content)))
	}
}
*/

// watchFileEvents processes fsnotify events
func (sm *StreamManager) watchFileEvents() {
	for {
		select {
		case event, ok := <-sm.fileWatcher.Events:
			if !ok {
				return
			}
			sm.handleFileEvent(event)

		case err, ok := <-sm.fileWatcher.Errors:
			if !ok {
				return
			}
			sm.handleFileError(err)

		case <-sm.ctx.Done():
			return
		}
	}
}

// handleFileEvent processes a file system event
func (sm *StreamManager) handleFileEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Write != fsnotify.Write {
		return
	}

	sm.mu.RLock()
	var relevantStream *JobStream
	for _, stream := range sm.activeStreams {
		if stream.FilePath == event.Name && stream.IsActive && !stream.IsRemote {
			relevantStream = stream
			break
		}
	}
	sm.mu.RUnlock()

	if relevantStream == nil {
		return
	}

	// Read new content
	content, offset, err := sm.readFileFromOffset(relevantStream.FilePath, relevantStream.LastOffset)
	if err != nil {
		sm.emitError(relevantStream, err)
		return
	}

	if len(content) > 0 {
		sm.emitNewContent(relevantStream, content, offset)
		relevantStream.LastOffset = offset
		relevantStream.LastUpdate = GetCurrentTime()
	}
}

// handleFileError processes file watcher errors
func (sm *StreamManager) handleFileError(err error) {
	// Log error and potentially notify relevant streams
	// In a production system, you'd want proper logging here
}

// readFileFromOffset reads file content from a specific offset
func (sm *StreamManager) readFileFromOffset(filePath string, offset int64) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", offset, err
	}
	defer func() { _ = file.Close() }()

	// Seek to offset
	currentOffset, err := file.Seek(offset, 0)
	if err != nil {
		return "", offset, err
	}

	// Read remaining content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", currentOffset, err
	}

	newOffset := currentOffset + int64(len(content))
	return string(content), newOffset, nil
}

// emitNewContent processes new content and emits events
func (sm *StreamManager) emitNewContent(stream *JobStream, content string, newOffset int64) {
	if content == "" {
		return
	}

	// Add to buffer
	stream.Buffer.AppendString(content)

	// Split into lines for event
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Emit event
	event := StreamEvent{
		JobID:      stream.JobID,
		OutputType: stream.OutputType,
		Content:    content,
		NewLines:   lines,
		Timestamp:  GetCurrentTime(),
		EventType:  StreamEventNewOutput,
		FileOffset: newOffset,
	}

	sm.eventBus.Publish(event)
}

// emitError emits an error event for a stream
func (sm *StreamManager) emitError(stream *JobStream, err error) {
	event := StreamEvent{
		JobID:      stream.JobID,
		OutputType: stream.OutputType,
		EventType:  StreamEventError,
		Error:      err,
		Timestamp:  GetCurrentTime(),
	}

	sm.eventBus.Publish(event)
}

// makeStreamKey creates a unique key for a job stream
func (sm *StreamManager) makeStreamKey(jobID, outputType string) string {
	return jobID + ":" + outputType
}

// IsStreamActive returns true if a stream is currently active
func (sm *StreamManager) IsStreamActive(jobID, outputType string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	streamKey := sm.makeStreamKey(jobID, outputType)
	stream, exists := sm.activeStreams[streamKey]
	return exists && stream.IsActive
}

// GetActiveStreams returns information about all active streams
func (sm *StreamManager) GetActiveStreams() []StreamInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var streams []StreamInfo
	for _, stream := range sm.activeStreams {
		if stream.IsActive {
			stats := stream.Buffer.GetStats()
			streams = append(streams, StreamInfo{
				JobID:       stream.JobID,
				OutputType:  stream.OutputType,
				FilePath:    stream.FilePath,
				IsRemote:    stream.IsRemote,
				NodeID:      stream.NodeID,
				BufferSize:  stats.CurrentSize,
				BufferUsage: stats.UsagePercent,
				LastUpdate:  stream.LastUpdate,
				Subscribers: sm.eventBus.GetSubscriberCount(stream.JobID, stream.OutputType),
			})
		}
	}

	return streams
}

// StreamInfo contains information about an active stream
type StreamInfo struct {
	JobID       string    `json:"job_id"`
	OutputType  string    `json:"output_type"`
	FilePath    string    `json:"file_path"`
	IsRemote    bool      `json:"is_remote"`
	NodeID      string    `json:"node_id"`
	BufferSize  int       `json:"buffer_size"`
	BufferUsage float64   `json:"buffer_usage_percent"`
	LastUpdate  time.Time `json:"last_update"`
	Subscribers int       `json:"subscribers"`
}
