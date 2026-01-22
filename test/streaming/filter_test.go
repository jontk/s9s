package streaming

import (
	"fmt"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/streaming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamFilter(t *testing.T) {
	t.Run("KeywordFilter", func(t *testing.T) {
		filter, err := streaming.NewStreamFilter(streaming.FilterTypeKeyword, "error")
		require.NoError(t, err)
		require.NotNil(t, filter)

		// Test matching
		matched, indices := filter.Match("This is an error message", time.Now())
		assert.True(t, matched)
		assert.Equal(t, []int{11, 16}, indices) // "error" at position 11-16

		// Test non-matching
		matched, _ = filter.Match("This is a normal message", time.Now())
		assert.False(t, matched)

		// Test case insensitive
		matched, _ = filter.Match("This is an ERROR message", time.Now())
		assert.True(t, matched)
	})

	t.Run("RegexFilter", func(t *testing.T) {
		filter, err := streaming.NewStreamFilter(streaming.FilterTypeRegex, `\berror\b|\bfailed\b`)
		require.NoError(t, err)

		// Test matching
		matched, _ := filter.Match("Operation failed with error", time.Now())
		assert.True(t, matched)

		matched, _ = filter.Match("The task has failed", time.Now())
		assert.True(t, matched)

		// Test non-matching
		matched, _ = filter.Match("Operation succeeded", time.Now())
		assert.False(t, matched)
	})

	t.Run("LogLevelFilter", func(t *testing.T) {
		filter, err := streaming.NewStreamFilter(streaming.FilterTypeLogLevel, "")
		require.NoError(t, err)

		filter.LogLevels = []streaming.LogLevel{
			streaming.LogLevelError,
			streaming.LogLevelWarning,
		}

		// Test matching
		matched, _ := filter.Match("[ERROR] Database connection failed", time.Now())
		assert.True(t, matched)

		matched, _ = filter.Match("WARNING: Low memory", time.Now())
		assert.True(t, matched)

		// Test non-matching
		matched, _ = filter.Match("[INFO] Server started", time.Now())
		assert.False(t, matched)
	})

	t.Run("TimeRangeFilter", func(t *testing.T) {
		filter, err := streaming.NewStreamFilter(streaming.FilterTypeTimeRange, "")
		require.NoError(t, err)

		now := time.Now()
		filter.TimeStart = now.Add(-1 * time.Hour)
		filter.TimeEnd = now.Add(1 * time.Hour)

		// Test matching
		matched, _ := filter.Match("Current time message", now)
		assert.True(t, matched)

		// Test non-matching
		matched, _ = filter.Match("Old message", now.Add(-2*time.Hour))
		assert.False(t, matched)
	})

	t.Run("InvertFilter", func(t *testing.T) {
		filter, err := streaming.NewStreamFilter(streaming.FilterTypeKeyword, "debug")
		require.NoError(t, err)

		filter.Invert = true

		// Test inverted matching
		matched, _ := filter.Match("This is a debug message", time.Now())
		assert.False(t, matched) // Inverted

		matched, _ = filter.Match("This is an info message", time.Now())
		assert.True(t, matched) // Inverted
	})
}

func TestFilterChain(t *testing.T) {
	t.Run("ANDMode", func(t *testing.T) {
		// Create filters
		filter1, _ := streaming.NewStreamFilter(streaming.FilterTypeKeyword, "error")
		filter2, _ := streaming.NewStreamFilter(streaming.FilterTypeKeyword, "database")

		chain := &streaming.FilterChain{
			ID:      "test-chain",
			Name:    "Test Chain",
			Mode:    streaming.ChainModeAND,
			Active:  true,
			Filters: []*streaming.StreamFilter{filter1, filter2},
		}

		// Test matching - both conditions must match
		matched, _ := chain.Apply("Database error occurred", time.Now())
		assert.True(t, matched)

		// Test non-matching - only one condition matches
		matched, _ = chain.Apply("Network error occurred", time.Now())
		assert.False(t, matched)
	})

	t.Run("ORMode", func(t *testing.T) {
		// Create filters
		filter1, _ := streaming.NewStreamFilter(streaming.FilterTypeKeyword, "error")
		filter2, _ := streaming.NewStreamFilter(streaming.FilterTypeKeyword, "warning")

		chain := &streaming.FilterChain{
			ID:      "test-chain",
			Name:    "Test Chain",
			Mode:    streaming.ChainModeOR,
			Active:  true,
			Filters: []*streaming.StreamFilter{filter1, filter2},
		}

		// Test matching - either condition matches
		matched, _ := chain.Apply("This is an error", time.Now())
		assert.True(t, matched)

		matched, _ = chain.Apply("This is a warning", time.Now())
		assert.True(t, matched)

		// Test non-matching - neither condition matches
		matched, _ = chain.Apply("This is info", time.Now())
		assert.False(t, matched)
	})

	t.Run("InactiveChain", func(t *testing.T) {
		filter, _ := streaming.NewStreamFilter(streaming.FilterTypeKeyword, "error")

		chain := &streaming.FilterChain{
			ID:      "test-chain",
			Name:    "Test Chain",
			Mode:    streaming.ChainModeAND,
			Active:  false, // Inactive
			Filters: []*streaming.StreamFilter{filter},
		}

		// Should always return true when inactive
		matched, _ := chain.Apply("This is an error", time.Now())
		assert.True(t, matched)
	})
}

func TestFilterManager(t *testing.T) {
	// Create temp directory for presets
	tempDir := t.TempDir()

	fm := streaming.NewFilterManager(tempDir)
	require.NotNil(t, fm)

	t.Run("AddAndGetFilter", func(t *testing.T) {
		filter, err := streaming.NewStreamFilter(streaming.FilterTypeKeyword, "test")
		require.NoError(t, err)

		err = fm.AddFilter(filter)
		assert.NoError(t, err)

		retrieved, err := fm.GetFilter(filter.ID)
		assert.NoError(t, err)
		assert.Equal(t, filter.ID, retrieved.ID)
	})

	t.Run("QuickFilter", func(t *testing.T) {
		err := fm.QuickFilter("error", streaming.FilterTypeKeyword)
		assert.NoError(t, err)

		// Check that filter is active
		matched, _ := fm.ApplyActiveFilters("This is an error", time.Now())
		assert.True(t, matched)

		matched, _ = fm.ApplyActiveFilters("This is normal", time.Now())
		assert.False(t, matched)
	})

	t.Run("ClearFilters", func(t *testing.T) {
		// Set a filter
		_ = fm.QuickFilter("test", streaming.FilterTypeKeyword)

		// Clear filters
		fm.ClearFilters()

		// All lines should pass now
		matched, _ := fm.ApplyActiveFilters("test message", time.Now())
		assert.True(t, matched)
	})

	t.Run("SaveAndLoadPreset", func(t *testing.T) {
		// Create a filter setup
		_ = fm.QuickFilter("error|warning", streaming.FilterTypeRegex)

		// Save as preset
		preset, err := fm.SavePreset("Error Filter", "Filters errors and warnings", "Debugging")
		assert.NoError(t, err)
		assert.NotNil(t, preset)

		// Clear filters
		fm.ClearFilters()

		// Load preset
		err = fm.LoadPreset(preset.ID)
		assert.NoError(t, err)

		// Verify filter is active
		matched, _ := fm.ApplyActiveFilters("This is an error", time.Now())
		assert.True(t, matched)
	})

	t.Run("GetPresets", func(t *testing.T) {
		presets := fm.GetPresets()
		assert.NotEmpty(t, presets)

		// Should include common presets
		found := false
		for _, preset := range presets {
			if preset.ID == "errors_only" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should include common presets")
	})
}

func TestStreamSearcher(t *testing.T) {
	// Create a buffer with test content
	buffer := streaming.NewCircularBuffer(100)
	buffer.Append([]string{
		"Line 1: This is a test",
		"Line 2: Error occurred here",
		"Line 3: Another test line",
		"Line 4: Warning message",
		"Line 5: Error again",
	})

	searcher := streaming.NewStreamSearcher(buffer)

	t.Run("BasicSearch", func(t *testing.T) {
		options := streaming.SearchOptions{
			CaseSensitive: false,
			WholeWord:     false,
			UseRegex:      false,
			ContextLines:  0,
			MaxResults:    100,
			Reverse:       false,
		}

		results, err := searcher.Search("error", options)
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// Check first result
		assert.Equal(t, 2, results[0].LineNumber)
		assert.Contains(t, results[0].Line, "Error occurred")

		// Check second result
		assert.Equal(t, 5, results[1].LineNumber)
		assert.Contains(t, results[1].Line, "Error again")
	})

	t.Run("RegexSearch", func(t *testing.T) {
		options := streaming.SearchOptions{
			CaseSensitive: false,
			UseRegex:      true,
			ContextLines:  0,
			MaxResults:    100,
		}

		results, err := searcher.Search(`(error|warning)`, options)
		assert.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("WholeWordSearch", func(t *testing.T) {
		options := streaming.SearchOptions{
			CaseSensitive: false,
			WholeWord:     true,
			UseRegex:      false,
			ContextLines:  0,
			MaxResults:    100,
		}

		results, err := searcher.Search("test", options)
		assert.NoError(t, err)
		assert.Len(t, results, 2) // Should match "test" but not "tests"
	})

	t.Run("SearchWithContext", func(t *testing.T) {
		options := streaming.SearchOptions{
			CaseSensitive: false,
			UseRegex:      false,
			ContextLines:  1,
			MaxResults:    100,
		}

		results, err := searcher.Search("Error occurred", options)
		assert.NoError(t, err)
		assert.Len(t, results, 1)

		// Should have context lines
		assert.NotEmpty(t, results[0].Context)
	})

	t.Run("HighlightedLine", func(t *testing.T) {
		_, _ = searcher.Search("error", streaming.SearchOptions{})

		highlighted := searcher.GetHighlightedLine("This is an error message", "yellow")
		assert.Contains(t, highlighted, "[yellow]error[white]")
	})
}

func TestSearchHistory(t *testing.T) {
	history := streaming.NewSearchHistory(5)

	// Add queries
	history.Add("test1")
	history.Add("test2")
	history.Add("test3")

	// Get history
	queries := history.Get()
	assert.Len(t, queries, 3)
	assert.Equal(t, "test3", queries[0]) // Most recent first

	// Add duplicate
	history.Add("test1")
	queries = history.Get()
	assert.Len(t, queries, 3)
	assert.Equal(t, "test1", queries[0]) // Moved to front

	// Test max size
	history.Add("test4")
	history.Add("test5")
	history.Add("test6") // Should push out oldest

	queries = history.Get()
	assert.Len(t, queries, 5) // Max size
}

func BenchmarkFiltering(b *testing.B) {
	filter, _ := streaming.NewStreamFilter(streaming.FilterTypeRegex, `\b(error|warning|critical)\b`)

	testLines := []string{
		"[INFO] Application started successfully",
		"[ERROR] Database connection failed",
		"[WARNING] Memory usage is high",
		"[DEBUG] Processing request",
		"[CRITICAL] System failure detected",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, line := range testLines {
			filter.Match(line, time.Now())
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	buffer := streaming.NewCircularBuffer(1000)

	// Add many lines
	lines := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		if i%10 == 0 {
			lines[i] = fmt.Sprintf("Line %d: Error occurred", i)
		} else {
			lines[i] = fmt.Sprintf("Line %d: Normal operation", i)
		}
	}
	buffer.Append(lines)

	searcher := streaming.NewStreamSearcher(buffer)
	options := streaming.SearchOptions{
		UseRegex:   false,
		MaxResults: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = searcher.Search("error", options)
	}
}
