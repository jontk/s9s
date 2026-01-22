package streaming

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ssh"
)

// FilteredStreamManager extends StreamManager with filtering capabilities
type FilteredStreamManager struct {
	*StreamManager
	filterManager *FilterManager
	searchers     map[string]*StreamSearcher // Per-stream searchers
	searchHistory *SearchHistory
	mu            sync.RWMutex
}

// NewFilteredStreamManager creates a new stream manager with filtering support
func NewFilteredStreamManager(client dao.SlurmClient, sshManager *ssh.SessionManager, config *SlurmConfig, configPath string) (*FilteredStreamManager, error) {
	// Create base stream manager
	baseManager, err := NewStreamManager(client, sshManager, config)
	if err != nil {
		return nil, err
	}

	fsm := &FilteredStreamManager{
		StreamManager: baseManager,
		filterManager: NewFilterManager(configPath),
		searchers:     make(map[string]*StreamSearcher),
		searchHistory: NewSearchHistory(20),
	}

	// Override the event emission to include filtering
	fsm.setupFilteredEventHandling()

	return fsm, nil
}

// setupFilteredEventHandling configures filtered event processing
func (fsm *FilteredStreamManager) setupFilteredEventHandling() {
	// We'll intercept events at the emission point
	// This requires modifying the base stream manager's emit methods
}

// StartFilteredStream starts a stream with filtering enabled
func (fsm *FilteredStreamManager) StartFilteredStream(jobID, outputType string) error {
	// Start regular stream
	if err := fsm.StartStream(jobID, outputType); err != nil {
		return err
	}

	// Create searcher for this stream
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	streamKey := fsm.makeStreamKey(jobID, outputType)
	if stream, exists := fsm.activeStreams[streamKey]; exists {
		fsm.searchers[streamKey] = NewStreamSearcher(stream.Buffer)
	}

	return nil
}

// SetQuickFilter sets a quick filter for all streams
func (fsm *FilteredStreamManager) SetQuickFilter(pattern string, filterType FilterType) error {
	return fsm.filterManager.QuickFilter(pattern, filterType)
}

// ClearFilters removes all active filters
func (fsm *FilteredStreamManager) ClearFilters() {
	fsm.filterManager.ClearFilters()
}

// LoadFilterPreset loads and activates a filter preset
func (fsm *FilteredStreamManager) LoadFilterPreset(presetID string) error {
	return fsm.filterManager.LoadPreset(presetID)
}

// GetFilterPresets returns available filter presets
func (fsm *FilteredStreamManager) GetFilterPresets() []*FilterPreset {
	return fsm.filterManager.GetPresets()
}

// SaveFilterPreset saves current filters as a preset
func (fsm *FilteredStreamManager) SaveFilterPreset(name, description, category string) (*FilterPreset, error) {
	return fsm.filterManager.SavePreset(name, description, category)
}

// Search performs a search on a specific stream
func (fsm *FilteredStreamManager) Search(jobID, outputType, query string, options SearchOptions) ([]*SearchResult, error) {
	fsm.mu.RLock()
	streamKey := fsm.makeStreamKey(jobID, outputType)
	searcher, exists := fsm.searchers[streamKey]
	fsm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active stream for job %s %s", jobID, outputType)
	}

	// Add to search history
	fsm.searchHistory.Add(query)

	return searcher.Search(query, options)
}

// SearchNext finds the next match in a stream
func (fsm *FilteredStreamManager) SearchNext(jobID, outputType string, currentLine int) (*SearchResult, error) {
	fsm.mu.RLock()
	streamKey := fsm.makeStreamKey(jobID, outputType)
	searcher, exists := fsm.searchers[streamKey]
	fsm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active stream for job %s %s", jobID, outputType)
	}

	return searcher.SearchNext(currentLine)
}

// SearchPrevious finds the previous match in a stream
func (fsm *FilteredStreamManager) SearchPrevious(jobID, outputType string, currentLine int) (*SearchResult, error) {
	fsm.mu.RLock()
	streamKey := fsm.makeStreamKey(jobID, outputType)
	searcher, exists := fsm.searchers[streamKey]
	fsm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active stream for job %s %s", jobID, outputType)
	}

	return searcher.SearchPrevious(currentLine)
}

// GetSearchHistory returns recent search queries
func (fsm *FilteredStreamManager) GetSearchHistory() []string {
	return fsm.searchHistory.Get()
}

// GetFilteredContent returns the filtered content for a stream
func (fsm *FilteredStreamManager) GetFilteredContent(jobID, outputType string, includeHighlights bool) ([]string, error) {
	fsm.mu.RLock()
	streamKey := fsm.makeStreamKey(jobID, outputType)
	stream, exists := fsm.activeStreams[streamKey]
	searcher := fsm.searchers[streamKey]
	fsm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active stream for job %s %s", jobID, outputType)
	}

	// Get all lines from buffer
	allLines := stream.Buffer.GetLines()
	filteredLines := make([]string, 0)

	// Process each line through filters
	for _, line := range allLines {
		timestamp := time.Now() // Would be better to store actual timestamps

		// Apply filters
		matched, highlights := fsm.filterManager.ApplyActiveFilters(line, timestamp)
		if !matched {
			continue
		}

		// Apply highlighting if requested
		if includeHighlights && len(highlights) > 0 {
			colors := make(map[string]string)
			for id := range highlights {
				colors[id] = "yellow" // Default color
			}
			line = HighlightLine(line, highlights, colors)
		}

		// Apply search highlighting if active
		if includeHighlights && searcher != nil {
			line = searcher.GetHighlightedLine(line, "cyan")
		}

		filteredLines = append(filteredLines, line)
	}

	return filteredLines, nil
}

// GetFilterStats returns filter statistics
func (fsm *FilteredStreamManager) GetFilterStats() map[string]FilterStats {
	return fsm.filterManager.GetFilterStats()
}

// EmitFilteredEvent emits an event after applying filters
func (fsm *FilteredStreamManager) EmitFilteredEvent(event StreamEvent) {
	// Extract lines from content
	lines := strings.Split(event.Content, "\n")
	filteredLines := make([]string, 0)
	filteredNewLines := make([]string, 0)

	// Apply filters to each line
	for _, line := range lines {
		matched, _ := fsm.filterManager.ApplyActiveFilters(line, event.Timestamp)
		if matched {
			filteredLines = append(filteredLines, line)
		}
	}

	// Apply filters to new lines
	for _, line := range event.NewLines {
		matched, _ := fsm.filterManager.ApplyActiveFilters(line, event.Timestamp)
		if matched {
			filteredNewLines = append(filteredNewLines, line)
		}
	}

	// Update event with filtered content
	event.Content = strings.Join(filteredLines, "\n")
	event.NewLines = filteredNewLines

	// Only emit if there's content after filtering
	if len(filteredLines) > 0 || event.EventType != StreamEventNewOutput {
		fsm.eventBus.Publish(event)
	}
}

// StopFilteredStream stops a stream and cleans up resources
func (fsm *FilteredStreamManager) StopFilteredStream(jobID, outputType string) error {
	streamKey := fsm.makeStreamKey(jobID, outputType)

	// Stop the stream
	if err := fsm.StopStream(jobID, outputType); err != nil {
		return err
	}

	// Clean up searcher
	fsm.mu.Lock()
	delete(fsm.searchers, streamKey)
	fsm.mu.Unlock()

	return nil
}

// CloseFiltered closes the filtered stream manager
func (fsm *FilteredStreamManager) CloseFiltered() error {
	// Clear all searchers
	fsm.mu.Lock()
	fsm.searchers = make(map[string]*StreamSearcher)
	fsm.mu.Unlock()

	// Close base manager
	return fsm.Close()
}

// GetActiveFilters returns the currently active filters
func (fsm *FilteredStreamManager) GetActiveFilters() []*StreamFilter {
	chain := fsm.filterManager.activeChain
	if chain == nil {
		return nil
	}
	return chain.Filters
}

// AddCustomFilter adds a custom filter
func (fsm *FilteredStreamManager) AddCustomFilter(filter *StreamFilter) error {
	if err := fsm.filterManager.AddFilter(filter); err != nil {
		return err
	}

	// Add to active chain
	if fsm.filterManager.activeChain == nil {
		fsm.filterManager.activeChain = &FilterChain{
			ID:      "custom",
			Name:    "Custom Filters",
			Mode:    ChainModeAND,
			Active:  true,
			Filters: []*StreamFilter{filter},
		}
	} else {
		fsm.filterManager.activeChain.Filters = append(fsm.filterManager.activeChain.Filters, filter)
	}

	return nil
}

// RemoveFilter removes a filter by ID
func (fsm *FilteredStreamManager) RemoveFilter(filterID string) error {
	return fsm.filterManager.RemoveFilter(filterID)
}

// SetFilterChainMode sets how multiple filters are combined (AND/OR)
func (fsm *FilteredStreamManager) SetFilterChainMode(mode ChainMode) {
	if fsm.filterManager.activeChain != nil {
		fsm.filterManager.activeChain.Mode = mode
	}
}

// StreamWithContext provides a filtered event channel for a specific stream
func (fsm *FilteredStreamManager) StreamWithContext(ctx context.Context, jobID, outputType string) (<-chan StreamEvent, error) {
	// Subscribe to the base stream
	eventChan := fsm.Subscribe(jobID, outputType)

	// Create filtered channel
	filteredChan := make(chan StreamEvent, 100)

	go func() {
		defer close(filteredChan)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventChan:
				if !ok {
					return
				}

				// Apply filters to the event
				if event.EventType == StreamEventNewOutput {
					filteredNewLines := make([]string, 0)
					for _, line := range event.NewLines {
						matched, _ := fsm.filterManager.ApplyActiveFilters(line, event.Timestamp)
						if matched {
							filteredNewLines = append(filteredNewLines, line)
						}
					}
					event.NewLines = filteredNewLines

					// Only send if there are matching lines
					if len(filteredNewLines) > 0 {
						select {
						case filteredChan <- event:
						case <-ctx.Done():
							return
						}
					}
				} else {
					// Always send non-content events
					select {
					case filteredChan <- event:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return filteredChan, nil
}
