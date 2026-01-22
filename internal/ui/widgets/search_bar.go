package widgets

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/streaming"
	"github.com/rivo/tview"
)

// SearchBar provides a search interface for stream content
type SearchBar struct {
	container     *tview.Flex
	searchInput   *tview.InputField
	searchOptions *tview.Form
	resultsView   *tview.TextView
	historyList   *tview.List

	streamManager  *streaming.FilteredStreamManager
	currentJobID   string
	currentOutput  string
	currentResults []*streaming.SearchResult
	currentIndex   int

	onSearchResult func(result *streaming.SearchResult)
	onHighlight    func(line int)

	// UI state
	showOptions bool
	showHistory bool
	options     streaming.SearchOptions
}

// NewSearchBar creates a new search bar widget
func NewSearchBar(streamManager *streaming.FilteredStreamManager) *SearchBar {
	sb := &SearchBar{
		streamManager: streamManager,
		currentIndex:  -1,
		options: streaming.SearchOptions{
			CaseSensitive: false,
			WholeWord:     false,
			UseRegex:      false,
			ContextLines:  2,
			MaxResults:    100,
			Reverse:       false,
		},
	}

	sb.buildUI()
	return sb
}

// buildUI creates the search bar interface
func (sb *SearchBar) buildUI() {
	// Create main container
	sb.container = tview.NewFlex().SetDirection(tview.FlexRow)

	// Search input row
	searchRow := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Search input
	sb.searchInput = tview.NewInputField()
	sb.searchInput.SetLabel("Search: ")
	sb.searchInput.SetFieldBackgroundColor(tcell.ColorBlack)
	sb.searchInput.SetPlaceholder("Enter search pattern...")
	sb.searchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			sb.performSearch()
		case tcell.KeyEsc:
			sb.clearSearch()
		case tcell.KeyTab:
			sb.searchNext()
		case tcell.KeyBacktab:
			sb.searchPrevious()
		}
	})

	// Search navigation buttons
	prevBtn := tview.NewButton("◀ Prev")
	prevBtn.SetSelectedFunc(sb.searchPrevious)

	nextBtn := tview.NewButton("Next ▶")
	nextBtn.SetSelectedFunc(sb.searchNext)

	optionsBtn := tview.NewButton("Options")
	optionsBtn.SetSelectedFunc(sb.toggleOptions)

	historyBtn := tview.NewButton("History")
	historyBtn.SetSelectedFunc(sb.toggleHistory)

	clearBtn := tview.NewButton("Clear")
	clearBtn.SetSelectedFunc(sb.clearSearch)

	// Add to search row
	searchRow.AddItem(sb.searchInput, 0, 1, true)
	searchRow.AddItem(prevBtn, 10, 0, false)
	searchRow.AddItem(nextBtn, 10, 0, false)
	searchRow.AddItem(optionsBtn, 12, 0, false)
	searchRow.AddItem(historyBtn, 12, 0, false)
	searchRow.AddItem(clearBtn, 10, 0, false)

	// Results view
	sb.resultsView = tview.NewTextView()
	sb.resultsView.SetDynamicColors(true)
	sb.resultsView.SetBorder(true)
	sb.resultsView.SetTitle(" Search Results ")
	sb.updateResultsView()

	// Search options form (hidden by default)
	sb.searchOptions = tview.NewForm()
	sb.searchOptions.SetBorder(true)
	sb.searchOptions.SetTitle(" Search Options ")
	sb.searchOptions.SetFieldBackgroundColor(tcell.ColorBlack)

	sb.searchOptions.AddCheckbox("Case sensitive", sb.options.CaseSensitive, func(checked bool) {
		sb.options.CaseSensitive = checked
	})
	sb.searchOptions.AddCheckbox("Whole word", sb.options.WholeWord, func(checked bool) {
		sb.options.WholeWord = checked
	})
	sb.searchOptions.AddCheckbox("Use regex", sb.options.UseRegex, func(checked bool) {
		sb.options.UseRegex = checked
	})
	sb.searchOptions.AddCheckbox("Search reverse", sb.options.Reverse, func(checked bool) {
		sb.options.Reverse = checked
	})
	sb.searchOptions.AddInputField("Context lines", fmt.Sprintf("%d", sb.options.ContextLines), 10,
		func(textToCheck string, lastChar rune) bool {
			return lastChar >= '0' && lastChar <= '9'
		}, func(text string) {
			var lines int
			_, _ = fmt.Sscanf(text, "%d", &lines)
			sb.options.ContextLines = lines
		})

	// Search history list (hidden by default)
	sb.historyList = tview.NewList()
	sb.historyList.SetBorder(true)
	sb.historyList.SetTitle(" Search History ")
	sb.historyList.ShowSecondaryText(false)

	// Build layout
	sb.container.AddItem(searchRow, 3, 0, true)
	sb.container.AddItem(sb.resultsView, 3, 0, false)
}

// SetStream sets the current stream to search
func (sb *SearchBar) SetStream(jobID, outputType string) {
	sb.currentJobID = jobID
	sb.currentOutput = outputType
	sb.clearSearch()
}

// performSearch executes the search
func (sb *SearchBar) performSearch() {
	query := sb.searchInput.GetText()
	if query == "" {
		return
	}

	// Perform search
	results, err := sb.streamManager.Search(
		sb.currentJobID,
		sb.currentOutput,
		query,
		sb.options,
	)

	if err != nil {
		sb.resultsView.SetText(fmt.Sprintf("[red]Error: %v[white]", err))
		return
	}

	sb.currentResults = results
	sb.currentIndex = -1

	// Update display
	sb.updateResultsView()

	// Navigate to first result
	if len(results) > 0 {
		sb.navigateToResult(0)
	}
}

// searchNext finds the next match
func (sb *SearchBar) searchNext() {
	if len(sb.currentResults) == 0 {
		return
	}

	newIndex := sb.currentIndex + 1
	if newIndex >= len(sb.currentResults) {
		newIndex = 0 // Wrap around
	}

	sb.navigateToResult(newIndex)
}

// searchPrevious finds the previous match
func (sb *SearchBar) searchPrevious() {
	if len(sb.currentResults) == 0 {
		return
	}

	newIndex := sb.currentIndex - 1
	if newIndex < 0 {
		newIndex = len(sb.currentResults) - 1 // Wrap around
	}

	sb.navigateToResult(newIndex)
}

// navigateToResult navigates to a specific search result
func (sb *SearchBar) navigateToResult(index int) {
	if index < 0 || index >= len(sb.currentResults) {
		return
	}

	sb.currentIndex = index
	result := sb.currentResults[index]

	// Update results view
	sb.updateResultsView()

	// Trigger callbacks
	if sb.onSearchResult != nil {
		sb.onSearchResult(result)
	}

	if sb.onHighlight != nil {
		sb.onHighlight(result.LineNumber)
	}
}

// clearSearch clears the current search
func (sb *SearchBar) clearSearch() {
	sb.searchInput.SetText("")
	sb.currentResults = nil
	sb.currentIndex = -1
	sb.updateResultsView()

	// Clear search in stream manager
	if sb.currentJobID != "" {
		// Would need to add a clear search method to the stream manager
	}
}

// toggleOptions shows/hides search options
func (sb *SearchBar) toggleOptions() {
	sb.showOptions = !sb.showOptions
	sb.updateLayout()
}

// toggleHistory shows/hides search history
func (sb *SearchBar) toggleHistory() {
	sb.showHistory = !sb.showHistory

	if sb.showHistory {
		sb.loadHistory()
	}

	sb.updateLayout()
}

// loadHistory loads search history
func (sb *SearchBar) loadHistory() {
	sb.historyList.Clear()

	history := sb.streamManager.GetSearchHistory()
	for _, query := range history {
		q := query // Capture for closure
		sb.historyList.AddItem(query, "", 0, func() {
			sb.searchInput.SetText(q)
			sb.performSearch()
			sb.showHistory = false
			sb.updateLayout()
		})
	}
}

// updateLayout updates the container layout
func (sb *SearchBar) updateLayout() {
	sb.container.Clear()

	// Always add search row and results
	searchRow := sb.container.GetItem(0)
	if searchRow != nil {
		sb.container.AddItem(searchRow, 3, 0, true)
	}
	sb.container.AddItem(sb.resultsView, 3, 0, false)

	// Add options if visible
	if sb.showOptions {
		sb.container.AddItem(sb.searchOptions, 8, 0, false)
	}

	// Add history if visible
	if sb.showHistory {
		sb.container.AddItem(sb.historyList, 0, 1, false)
	}
}

// updateResultsView updates the search results display
func (sb *SearchBar) updateResultsView() {
	if sb.currentResults == nil || len(sb.currentResults) == 0 {
		sb.resultsView.SetText("[gray]No search results[white]")
		return
	}

	var text strings.Builder

	// Summary
	totalMatches := 0
	for _, result := range sb.currentResults {
		totalMatches += len(result.Matches)
	}

	if sb.currentIndex >= 0 {
		text.WriteString(fmt.Sprintf("Match [yellow]%d[white] of [green]%d[white] ",
			sb.currentIndex+1, len(sb.currentResults)))
	}

	text.WriteString(fmt.Sprintf("([cyan]%d[white] total matches in [cyan]%d[white] lines)\n",
		totalMatches, len(sb.currentResults)))

	// Current match preview
	if sb.currentIndex >= 0 && sb.currentIndex < len(sb.currentResults) {
		result := sb.currentResults[sb.currentIndex]
		text.WriteString(fmt.Sprintf("\nLine %d: ", result.LineNumber))

		// Highlight matches in preview
		line := result.Line
		if len(result.Matches) > 0 {
			highlighted := ""
			lastEnd := 0

			for _, match := range result.Matches {
				highlighted += line[lastEnd:match.Start]
				highlighted += fmt.Sprintf("[yellow::b]%s[white::-]", match.Text)
				lastEnd = match.End
			}
			highlighted += line[lastEnd:]

			text.WriteString(highlighted)
		} else {
			text.WriteString(line)
		}
	}

	sb.resultsView.SetText(text.String())
}

// SetOnSearchResult sets the callback for search results
func (sb *SearchBar) SetOnSearchResult(callback func(result *streaming.SearchResult)) {
	sb.onSearchResult = callback
}

// SetOnHighlight sets the callback for line highlighting
func (sb *SearchBar) SetOnHighlight(callback func(line int)) {
	sb.onHighlight = callback
}

// GetContainer returns the main container
func (sb *SearchBar) GetContainer() tview.Primitive {
	return sb.container
}

// HandleInput processes keyboard shortcuts
func (sb *SearchBar) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlF:
		// Focus search input
		// Focus search input - handled by application
		return nil
	case tcell.KeyF3:
		// Search next
		sb.searchNext()
		return nil
	case tcell.KeyCtrlG:
		// Search next (vim-style)
		sb.searchNext()
		return nil
	case tcell.KeyF15: // Shift+F3 equivalent
		// Search previous (vim-style)
		sb.searchPrevious()
		return nil
	}

	switch event.Rune() {
	case '/':
		// Focus search input (vim-style)
		// Focus search input - handled by application
		return nil
	case 'n':
		// Next match (vim-style)
		sb.searchNext()
		return nil
	case 'N':
		// Previous match (vim-style)
		sb.searchPrevious()
		return nil
	}

	return event
}

// IsActive returns whether search is active
func (sb *SearchBar) IsActive() bool {
	return len(sb.currentResults) > 0
}

// GetCurrentMatch returns the current match index and total
func (sb *SearchBar) GetCurrentMatch() (current, total int) {
	if sb.currentIndex < 0 {
		return 0, len(sb.currentResults)
	}
	return sb.currentIndex + 1, len(sb.currentResults)
}
