package streaming

import (
	"fmt"
	"regexp"
	"sync"
)

// SearchResult represents a search match in the stream
type SearchResult struct {
	LineNumber int      `json:"line_number"`
	Line       string   `json:"line"`
	Matches    []Match  `json:"matches"`
	Context    []string `json:"context,omitempty"` // Lines before/after for context
}

// Match represents a single match within a line
type Match struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
}

// SearchOptions configures search behavior
type SearchOptions struct {
	CaseSensitive bool `json:"case_sensitive"`
	WholeWord     bool `json:"whole_word"`
	UseRegex      bool `json:"use_regex"`
	ContextLines  int  `json:"context_lines"` // Number of lines before/after to include
	MaxResults    int  `json:"max_results"`   // Limit number of results
	Reverse       bool `json:"reverse"`       // Search from bottom to top
}

// StreamSearcher handles searching within stream buffers
type StreamSearcher struct {
	buffer       *CircularBuffer
	results      []*SearchResult
	currentQuery string
	options      SearchOptions
	regex        *regexp.Regexp
	mu           sync.RWMutex
}

// NewStreamSearcher creates a new stream searcher
func NewStreamSearcher(buffer *CircularBuffer) *StreamSearcher {
	return &StreamSearcher{
		buffer:  buffer,
		results: make([]*SearchResult, 0),
		options: SearchOptions{
			CaseSensitive: false,
			WholeWord:     false,
			UseRegex:      false,
			ContextLines:  2,
			MaxResults:    100,
			Reverse:       false,
		},
	}
}

// Search performs a search on the buffer content
func (ss *StreamSearcher) Search(query string, options SearchOptions) ([]*SearchResult, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.currentQuery = query
	ss.options = options
	ss.results = make([]*SearchResult, 0)

	// Prepare search pattern
	if err := ss.preparePattern(query); err != nil {
		return nil, err
	}

	// Get all lines from buffer
	lines := ss.buffer.GetLines()
	if len(lines) == 0 {
		return ss.results, nil
	}

	// Search through lines
	if options.Reverse {
		for i := len(lines) - 1; i >= 0; i-- {
			ss.searchLine(i, lines[i], lines)
			if len(ss.results) >= options.MaxResults {
				break
			}
		}
	} else {
		for i, line := range lines {
			ss.searchLine(i, line, lines)
			if len(ss.results) >= options.MaxResults {
				break
			}
		}
	}

	return ss.results, nil
}

// SearchNext finds the next occurrence after the given line number
func (ss *StreamSearcher) SearchNext(startLine int) (*SearchResult, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if ss.currentQuery == "" {
		return nil, fmt.Errorf("no active search")
	}

	lines := ss.buffer.GetLines()
	for i := startLine + 1; i < len(lines); i++ {
		if result := ss.searchSingleLine(i, lines[i], lines); result != nil {
			return result, nil
		}
	}

	return nil, fmt.Errorf("no more matches found")
}

// SearchPrevious finds the previous occurrence before the given line number
func (ss *StreamSearcher) SearchPrevious(startLine int) (*SearchResult, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if ss.currentQuery == "" {
		return nil, fmt.Errorf("no active search")
	}

	lines := ss.buffer.GetLines()
	for i := startLine - 1; i >= 0; i-- {
		if result := ss.searchSingleLine(i, lines[i], lines); result != nil {
			return result, nil
		}
	}

	return nil, fmt.Errorf("no more matches found")
}

// GetHighlightedLine returns a line with search matches highlighted
func (ss *StreamSearcher) GetHighlightedLine(line string, highlightColor string) string {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if ss.currentQuery == "" {
		return line
	}

	matches := ss.findMatches(line)
	if len(matches) == 0 {
		return line
	}

	// Build highlighted line
	highlighted := ""
	lastEnd := 0

	for _, match := range matches {
		// Add text before match
		highlighted += line[lastEnd:match.Start]
		// Add highlighted match
		highlighted += fmt.Sprintf("[%s]%s[white]", highlightColor, line[match.Start:match.End])
		lastEnd = match.End
	}

	// Add remaining text
	highlighted += line[lastEnd:]

	return highlighted
}

// preparePattern prepares the search pattern based on options
func (ss *StreamSearcher) preparePattern(query string) error {
	pattern := query

	if !ss.options.UseRegex {
		// Escape regex special characters
		pattern = regexp.QuoteMeta(pattern)
	}

	if ss.options.WholeWord {
		pattern = `\b` + pattern + `\b`
	}

	flags := ""
	if !ss.options.CaseSensitive {
		flags = "(?i)"
	}

	fullPattern := flags + pattern

	regex, err := regexp.Compile(fullPattern)
	if err != nil {
		return fmt.Errorf("invalid search pattern: %w", err)
	}

	ss.regex = regex
	return nil
}

// searchLine searches for matches in a single line
func (ss *StreamSearcher) searchLine(lineNum int, line string, allLines []string) {
	result := ss.searchSingleLine(lineNum, line, allLines)
	if result != nil {
		ss.results = append(ss.results, result)
	}
}

// searchSingleLine searches for matches in a single line and returns result
func (ss *StreamSearcher) searchSingleLine(lineNum int, line string, allLines []string) *SearchResult {
	matches := ss.findMatches(line)
	if len(matches) == 0 {
		return nil
	}

	result := &SearchResult{
		LineNumber: lineNum + 1, // 1-based line numbers
		Line:       line,
		Matches:    matches,
	}

	// Add context lines if requested
	if ss.options.ContextLines > 0 {
		result.Context = ss.getContext(lineNum, allLines)
	}

	return result
}

// findMatches finds all matches in a line
func (ss *StreamSearcher) findMatches(line string) []Match {
	if ss.regex == nil {
		return nil
	}

	allMatches := ss.regex.FindAllStringIndex(line, -1)
	if len(allMatches) == 0 {
		return nil
	}

	matches := make([]Match, len(allMatches))
	for i, loc := range allMatches {
		matches[i] = Match{
			Start: loc[0],
			End:   loc[1],
			Text:  line[loc[0]:loc[1]],
		}
	}

	return matches
}

// getContext returns context lines around a match
func (ss *StreamSearcher) getContext(lineNum int, allLines []string) []string {
	context := make([]string, 0)

	// Lines before
	start := lineNum - ss.options.ContextLines
	if start < 0 {
		start = 0
	}

	// Lines after
	end := lineNum + ss.options.ContextLines + 1
	if end > len(allLines) {
		end = len(allLines)
	}

	// Build context
	for i := start; i < end; i++ {
		if i != lineNum {
			context = append(context, allLines[i])
		}
	}

	return context
}

// GetStats returns search statistics
func (ss *StreamSearcher) GetStats() (totalMatches int, matchedLines int) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	totalMatches = 0
	matchedLines = len(ss.results)

	for _, result := range ss.results {
		totalMatches += len(result.Matches)
	}

	return totalMatches, matchedLines
}

// Clear clears search results and pattern
func (ss *StreamSearcher) Clear() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.currentQuery = ""
	ss.regex = nil
	ss.results = make([]*SearchResult, 0)
}

// HighlightLine applies multiple highlights to a line (for both search and filters)
func HighlightLine(line string, highlights map[string][]int, colors map[string]string) string {
	if len(highlights) == 0 {
		return line
	}

	// Collect all highlight ranges
	type highlightRange struct {
		start    int
		end      int
		color    string
		priority int
	}

	ranges := make([]highlightRange, 0)
	priority := 0

	for id, indices := range highlights {
		color := colors[id]
		if color == "" {
			color = "yellow"
		}

		for i := 0; i < len(indices); i += 2 {
			if i+1 < len(indices) {
				ranges = append(ranges, highlightRange{
					start:    indices[i],
					end:      indices[i+1],
					color:    color,
					priority: priority,
				})
			}
		}
		priority++
	}

	// Sort ranges by start position
	for i := 0; i < len(ranges); i++ {
		for j := i + 1; j < len(ranges); j++ {
			if ranges[i].start > ranges[j].start {
				ranges[i], ranges[j] = ranges[j], ranges[i]
			}
		}
	}

	// Build highlighted line
	highlighted := ""
	lastEnd := 0

	for _, r := range ranges {
		// Skip overlapping ranges (keep higher priority)
		if r.start < lastEnd {
			continue
		}

		// Add text before highlight
		highlighted += line[lastEnd:r.start]
		// Add highlighted text
		highlighted += fmt.Sprintf("[%s]%s[white]", r.color, line[r.start:r.end])
		lastEnd = r.end
	}

	// Add remaining text
	highlighted += line[lastEnd:]

	return highlighted
}

// SearchHistory tracks recent searches
type SearchHistory struct {
	queries []string
	maxSize int
	mu      sync.RWMutex
}

// NewSearchHistory creates a new search history
func NewSearchHistory(maxSize int) *SearchHistory {
	return &SearchHistory{
		queries: make([]string, 0),
		maxSize: maxSize,
	}
}

// Add adds a query to history
func (sh *SearchHistory) Add(query string) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Don't add duplicates
	for i, q := range sh.queries {
		if q == query {
			// Move to front
			sh.queries = append([]string{query}, append(sh.queries[:i], sh.queries[i+1:]...)...)
			return
		}
	}

	// Add to front
	sh.queries = append([]string{query}, sh.queries...)

	// Trim to max size
	if len(sh.queries) > sh.maxSize {
		sh.queries = sh.queries[:sh.maxSize]
	}
}

// Get returns the search history
func (sh *SearchHistory) Get() []string {
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	result := make([]string, len(sh.queries))
	copy(result, sh.queries)
	return result
}
