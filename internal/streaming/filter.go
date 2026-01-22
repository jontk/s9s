package streaming

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// FilterType represents the type of filter to apply
type FilterType string

const (
	FilterTypeKeyword   FilterType = "keyword"
	FilterTypeRegex     FilterType = "regex"
	FilterTypeTimeRange FilterType = "timerange"
	FilterTypeLogLevel  FilterType = "loglevel"
	FilterTypeInvert    FilterType = "invert"
)

// LogLevel represents common log levels for filtering
type LogLevel string

const (
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelError   LogLevel = "ERROR"
	LogLevelFatal   LogLevel = "FATAL"
)

// StreamFilter represents a filter that can be applied to streaming output
type StreamFilter struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           FilterType     `json:"type"`
	Enabled        bool           `json:"enabled"`
	Pattern        string         `json:"pattern"`    // For keyword/regex filters
	Regex          *regexp.Regexp `json:"-"`          // Compiled regex
	TimeStart      time.Time      `json:"time_start"` // For time range filters
	TimeEnd        time.Time      `json:"time_end"`
	LogLevels      []LogLevel     `json:"log_levels"` // For log level filters
	Invert         bool           `json:"invert"`     // Invert the filter results
	CaseSensitive  bool           `json:"case_sensitive"`
	Highlight      bool           `json:"highlight"` // Highlight matches
	HighlightColor string         `json:"highlight_color"`
	Stats          FilterStats    `json:"stats"`
}

// FilterStats tracks filter performance and matches
type FilterStats struct {
	MatchCount     int64     `json:"match_count"`
	ProcessedLines int64     `json:"processed_lines"`
	LastMatch      time.Time `json:"last_match"`
	Created        time.Time `json:"created"`
}

// FilterChain represents a collection of filters applied in sequence
type FilterChain struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Filters     []*StreamFilter `json:"filters"`
	Mode        ChainMode       `json:"mode"` // AND or OR
	Active      bool            `json:"active"`
}

// ChainMode determines how multiple filters are combined
type ChainMode string

const (
	ChainModeAND ChainMode = "AND" // All filters must match
	ChainModeOR  ChainMode = "OR"  // Any filter must match
)

// FilterPreset represents a saved filter configuration
type FilterPreset struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Filters     []*StreamFilter `json:"filters"`
	Tags        []string        `json:"tags"`
	Created     time.Time       `json:"created"`
	LastUsed    time.Time       `json:"last_used"`
	UseCount    int             `json:"use_count"`
}

// NewStreamFilter creates a new stream filter
func NewStreamFilter(filterType FilterType, pattern string) (*StreamFilter, error) {
	filter := &StreamFilter{
		ID:             GenerateFilterID(),
		Type:           filterType,
		Enabled:        true,
		Pattern:        pattern,
		CaseSensitive:  false,
		Highlight:      true,
		HighlightColor: "yellow",
		Stats: FilterStats{
			Created: time.Now(),
		},
	}

	// Compile regex if needed
	if filterType == FilterTypeRegex {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		filter.Regex = regex
	}

	return filter, nil
}

// Match checks if a line matches the filter criteria
func (f *StreamFilter) Match(line string, timestamp time.Time) (bool, []int) {
	if !f.Enabled {
		return false, nil
	}

	var matched bool
	var indices []int

	switch f.Type {
	case FilterTypeKeyword:
		matched, indices = f.matchKeyword(line)
	case FilterTypeRegex:
		matched, indices = f.matchRegex(line)
	case FilterTypeTimeRange:
		matched = f.matchTimeRange(timestamp)
	case FilterTypeLogLevel:
		matched = f.matchLogLevel(line)
	}

	// Apply invert if needed
	if f.Invert {
		matched = !matched
		indices = nil // No highlighting for inverted matches
	}

	// Update stats
	if matched {
		f.Stats.MatchCount++
		f.Stats.LastMatch = time.Now()
	}
	f.Stats.ProcessedLines++

	return matched, indices
}

// matchKeyword performs keyword matching
func (f *StreamFilter) matchKeyword(line string) (bool, []int) {
	searchLine := line
	searchPattern := f.Pattern

	if !f.CaseSensitive {
		searchLine = strings.ToLower(line)
		searchPattern = strings.ToLower(f.Pattern)
	}

	index := strings.Index(searchLine, searchPattern)
	if index >= 0 {
		// Return start and end indices for highlighting
		return true, []int{index, index + len(f.Pattern)}
	}

	return false, nil
}

// matchRegex performs regex matching
func (f *StreamFilter) matchRegex(line string) (bool, []int) {
	if f.Regex == nil {
		return false, nil
	}

	loc := f.Regex.FindStringIndex(line)
	if loc != nil {
		return true, loc
	}

	return false, nil
}

// matchTimeRange checks if the timestamp falls within the time range
func (f *StreamFilter) matchTimeRange(timestamp time.Time) bool {
	if timestamp.IsZero() {
		return false
	}

	afterStart := f.TimeStart.IsZero() || timestamp.After(f.TimeStart) || timestamp.Equal(f.TimeStart)
	beforeEnd := f.TimeEnd.IsZero() || timestamp.Before(f.TimeEnd) || timestamp.Equal(f.TimeEnd)

	return afterStart && beforeEnd
}

// matchLogLevel checks if the line contains one of the specified log levels
func (f *StreamFilter) matchLogLevel(line string) bool {
	if len(f.LogLevels) == 0 {
		return false
	}

	upperLine := strings.ToUpper(line)
	for _, level := range f.LogLevels {
		if strings.Contains(upperLine, string(level)) {
			return true
		}
	}

	return false
}

// ApplyChain applies a filter chain to a line
func (fc *FilterChain) Apply(line string, timestamp time.Time) (bool, map[string][]int) {
	if !fc.Active || len(fc.Filters) == 0 {
		return true, nil
	}

	matches := make(map[string][]int)
	matchCount := 0

	for _, filter := range fc.Filters {
		matched, indices := filter.Match(line, timestamp)
		if matched {
			matchCount++
			if indices != nil && filter.Highlight {
				matches[filter.ID] = indices
			}
		}

		// Short-circuit evaluation
		if fc.Mode == ChainModeOR && matched {
			return true, matches
		}
		if fc.Mode == ChainModeAND && !matched {
			return false, nil
		}
	}

	// For AND mode, all filters must match
	if fc.Mode == ChainModeAND {
		return matchCount == len(fc.Filters), matches
	}

	// For OR mode, at least one filter must match
	return matchCount > 0, matches
}

// GenerateFilterID generates a unique filter ID
func GenerateFilterID() string {
	return fmt.Sprintf("filter_%d", time.Now().UnixNano())
}

// Common filter presets
func GetCommonPresets() []*FilterPreset {
	return []*FilterPreset{
		{
			ID:          "errors_only",
			Name:        "Errors Only",
			Description: "Show only error messages",
			Category:    "Log Levels",
			Filters: []*StreamFilter{
				{
					Type:      FilterTypeLogLevel,
					LogLevels: []LogLevel{LogLevelError, LogLevelFatal},
					Enabled:   true,
				},
			},
			Tags: []string{"errors", "debugging"},
		},
		{
			ID:          "warnings_errors",
			Name:        "Warnings & Errors",
			Description: "Show warnings and errors",
			Category:    "Log Levels",
			Filters: []*StreamFilter{
				{
					Type:      FilterTypeLogLevel,
					LogLevels: []LogLevel{LogLevelWarning, LogLevelError, LogLevelFatal},
					Enabled:   true,
				},
			},
			Tags: []string{"warnings", "errors", "debugging"},
		},
		{
			ID:          "performance",
			Name:        "Performance Issues",
			Description: "Find performance-related messages",
			Category:    "Performance",
			Filters: []*StreamFilter{
				{
					Type:    FilterTypeRegex,
					Pattern: `(slow|performance|latency|timeout|delay|hang)`,
					Enabled: true,
				},
			},
			Tags: []string{"performance", "optimization"},
		},
		{
			ID:          "memory_issues",
			Name:        "Memory Issues",
			Description: "Find memory-related problems",
			Category:    "Resources",
			Filters: []*StreamFilter{
				{
					Type:    FilterTypeRegex,
					Pattern: `(memory|heap|OOM|out of memory|allocation failed)`,
					Enabled: true,
				},
			},
			Tags: []string{"memory", "resources", "debugging"},
		},
	}
}
