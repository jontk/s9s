package components

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/ui/navigation"
	"github.com/rivo/tview"
)

// colorCodeRegex matches tview color codes like [red], [white], [#ff0000], etc.
var colorCodeRegex = regexp.MustCompile(`\[[^\]]*\]`)

// getDisplayWidth returns the actual display width of a string without color codes
func getDisplayWidth(text string) int {
	return len(colorCodeRegex.ReplaceAllString(text, ""))
}

// truncateWithColorCodes truncates text while preserving color codes
func truncateWithColorCodes(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	displayWidth := getDisplayWidth(text)
	if displayWidth <= maxWidth {
		return text
	}

	// Strip color codes for safe rune-level truncation
	stripped := colorCodeRegex.ReplaceAllString(text, "")
	if len([]rune(stripped)) <= maxWidth-3 {
		return text
	}

	// Truncate at rune level to avoid splitting UTF-8 sequences
	runes := []rune(stripped)
	if len(runes) > maxWidth-3 {
		return string(runes[:maxWidth-3]) + "..."
	}

	return stripped + "..."
}

// Column represents a table column definition
type Column struct {
	Name      string
	Width     int
	Alignment int // 0=left, 1=center, 2=right
	Sortable  bool
	Hidden    bool
}

// TableConfig holds table configuration
type TableConfig struct {
	Columns       []Column
	Selectable    bool
	Scrollable    bool
	FixedRows     int // Number of header rows
	ShowHeader    bool
	BorderColor   tcell.Color
	SelectedColor tcell.Color
	HeaderColor   tcell.Color
	EvenRowColor  tcell.Color
	OddRowColor   tcell.Color
}

// DefaultTableConfig returns default table configuration
func DefaultTableConfig() *TableConfig {
	return &TableConfig{
		Selectable:    true,
		Scrollable:    true,
		FixedRows:     1,
		ShowHeader:    true,
		BorderColor:   tcell.ColorWhite,
		SelectedColor: tcell.ColorYellow,
		HeaderColor:   tcell.ColorTeal,
		EvenRowColor:  tcell.ColorDefault,
		OddRowColor:   tcell.ColorDefault,
	}
}

// Table is a reusable table component with sorting and filtering
type Table struct {
	*tview.Table
	mu            sync.RWMutex // Protects data, filteredData, and table operations
	config        *TableConfig
	data          [][]string
	filteredData  [][]string
	sortColumn    int
	sortAscending bool
	filter        string
	onSelect      func(row, col int)
	onSort        func(col int, ascending bool)
	vimMode       *navigation.VimMode
}

// NewTable creates a new table component
func NewTable(config *TableConfig) *Table {
	if config == nil {
		config = DefaultTableConfig()
	}

	table := &Table{
		Table:         tview.NewTable(),
		config:        config,
		data:          [][]string{},
		filteredData:  [][]string{},
		sortColumn:    -1,
		sortAscending: true,
		vimMode:       navigation.NewVimMode(),
	}

	// Configure tview table
	table.Table.SetBorders(true).
		SetSelectable(config.Selectable, false).
		SetFixed(config.FixedRows, 0).
		SetBorderColor(config.BorderColor).
		SetBorderPadding(0, 0, 1, 1)

	if config.Selectable {
		table.SetSelectedStyle(tcell.StyleDefault.
			Background(config.SelectedColor).
			Foreground(tcell.ColorBlack))
	}

	// Note: InputCapture is NOT set here. Views handle all keyboard input in their OnKey methods.
	// Setting InputCapture on the table was intercepting events before they could reach the view's OnKey.
	// Instead, views should call table.handleInput manually if needed for navigation keys.

	return table
}

// SetColumns sets the table columns
func (t *Table) SetColumns(columns []Column) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.config.Columns = columns
	t.refresh()
}

// SetData sets the table data
func (t *Table) SetData(data [][]string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.data = data
	t.applyFilter()
	t.applySort()
	t.refresh()
}

// GetData returns the current table data
func (t *Table) GetData() [][]string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.data
}

// GetFilteredData returns the filtered table data
func (t *Table) GetFilteredData() [][]string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.filteredData
}

// SetFilter sets the filter string
func (t *Table) SetFilter(filter string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.filter = strings.ToLower(filter)
	t.applyFilter()
	t.applySort()
	t.refresh()
}

// GetFilter returns the current filter
func (t *Table) GetFilter() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.filter
}

// Sort sorts the table by the specified column
func (t *Table) Sort(column int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if column < 0 || column >= len(t.config.Columns) {
		return
	}

	if !t.config.Columns[column].Sortable {
		return
	}

	if t.sortColumn == column {
		t.sortAscending = !t.sortAscending
	} else {
		t.sortColumn = column
		t.sortAscending = true
	}

	t.applySort()
	t.refresh()

	if t.onSort != nil {
		t.onSort(column, t.sortAscending)
	}
}

// GetSelectedRow returns the currently selected row index
func (t *Table) GetSelectedRow() int {
	row, _ := t.GetSelection()
	if row < t.config.FixedRows {
		return -1
	}
	return row - t.config.FixedRows
}

// GetSelectedData returns the data for the currently selected row
func (t *Table) GetSelectedData() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	row := t.GetSelectedRow()
	if row < 0 || row >= len(t.filteredData) {
		return nil
	}
	return t.filteredData[row]
}

// SetOnSelect sets the selection callback
func (t *Table) SetOnSelect(fn func(row, col int)) {
	t.onSelect = fn
	t.SetSelectedFunc(func(row, col int) {
		if row >= t.config.FixedRows && t.onSelect != nil {
			t.onSelect(row-t.config.FixedRows, col)
		}
	})
}

// SetOnSort sets the sort callback
func (t *Table) SetOnSort(fn func(col int, ascending bool)) {
	t.onSort = fn
}

// Clear clears the table
func (t *Table) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.data = [][]string{}
	t.filteredData = [][]string{}
	t.Table.Clear()
}

// refresh updates the table display
func (t *Table) refresh() {
	t.Table.Clear()
	t.renderHeader()
	t.renderDataRows()
}

// renderHeader renders the table header row
func (t *Table) renderHeader() {
	if !t.config.ShowHeader || len(t.config.Columns) == 0 {
		return
	}

	for col, column := range t.config.Columns {
		if column.Hidden {
			continue
		}

		header := t.getHeaderText(col, column)
		cell := tview.NewTableCell(header).
			SetTextColor(t.config.HeaderColor).
			SetAlign(column.Alignment).
			SetSelectable(false).
			SetExpansion(1)

		t.SetCell(0, col, cell)
	}
}

// getHeaderText returns the header text with sort indicator if applicable
func (t *Table) getHeaderText(col int, column Column) string {
	header := column.Name
	if column.Sortable && col == t.sortColumn {
		if t.sortAscending {
			header += " ▲"
		} else {
			header += " ▼"
		}
	}
	return header
}

// renderDataRows renders all data rows
func (t *Table) renderDataRows() {
	for rowIdx, rowData := range t.filteredData {
		displayRow := rowIdx + t.config.FixedRows
		t.renderRow(displayRow, rowIdx, rowData)
	}
}

// renderRow renders a single data row
func (t *Table) renderRow(displayRow, rowIdx int, rowData []string) {
	for colIdx, cellData := range rowData {
		if colIdx >= len(t.config.Columns) || t.config.Columns[colIdx].Hidden {
			continue
		}

		// Truncate text if necessary, accounting for color codes
		maxWidth := t.config.Columns[colIdx].Width
		if maxWidth > 0 {
			cellData = truncateWithColorCodes(cellData, maxWidth)
		}

		cell := tview.NewTableCell(cellData).
			SetAlign(t.config.Columns[colIdx].Alignment).
			SetExpansion(1)

		t.applyRowColoring(cell, rowIdx)
		t.SetCell(displayRow, colIdx, cell)
	}
}

// applyRowColoring applies alternating row colors
func (t *Table) applyRowColoring(cell *tview.TableCell, rowIdx int) {
	if rowIdx%2 == 0 && t.config.EvenRowColor != tcell.ColorDefault {
		cell.SetTextColor(t.config.EvenRowColor)
	} else if rowIdx%2 == 1 && t.config.OddRowColor != tcell.ColorDefault {
		cell.SetTextColor(t.config.OddRowColor)
	}
}

// applyFilter applies the current filter to the data
func (t *Table) applyFilter() {
	if t.filter == "" {
		t.filteredData = make([][]string, len(t.data))
		copy(t.filteredData, t.data)
		return
	}

	t.filteredData = [][]string{}
	for _, row := range t.data {
		match := false
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), t.filter) {
				match = true
				break
			}
		}
		if match {
			t.filteredData = append(t.filteredData, row)
		}
	}
}

// applySort sorts the filtered data
func (t *Table) applySort() {
	if t.sortColumn < 0 || t.sortColumn >= len(t.config.Columns) {
		return
	}

	sort.Slice(t.filteredData, func(i, j int) bool {
		if t.sortColumn >= len(t.filteredData[i]) || t.sortColumn >= len(t.filteredData[j]) {
			return false
		}

		a := t.filteredData[i][t.sortColumn]
		b := t.filteredData[j][t.sortColumn]

		if t.sortAscending {
			return a < b
		}
		return a > b
	})
}

// handleInput handles keyboard input
func (t *Table) handleInput(event *tcell.EventKey) *tcell.EventKey {
	// Process vim mode first
	event = t.processVimMode(event)
	if event == nil {
		return nil
	}

	// Handle column sorting with number keys
	if t.handleNumberKeySorting(event) {
		return nil
	}

	// Handle navigation keys
	if t.handleNavigationKeys(event) {
		return nil
	}

	// DEBUG: Log keys that are being returned
	if keyName, ok := tcell.KeyNames[event.Key()]; ok {
		if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyF2 {
			fmt.Fprintf(os.Stderr, "DEBUG Table.handleInput: Returning unhandled key: %s (%d)\n", keyName, event.Key())
		}
	}

	return event
}

// processVimMode processes the event through vim mode
func (t *Table) processVimMode(event *tcell.EventKey) *tcell.EventKey {
	vimEvent := t.vimMode.HandleKey(event)
	if vimEvent == nil {
		return nil
	}
	if vimEvent != event {
		return vimEvent
	}
	return event
}

// handleNumberKeySorting handles column sorting with number keys
func (t *Table) handleNumberKeySorting(event *tcell.EventKey) bool {
	if event.Key() < tcell.KeyRune || event.Rune() < '1' || event.Rune() > '9' {
		return false
	}
	if t.vimMode.IsWaitingForKey() {
		return false
	}

	col := int(event.Rune() - '1')
	if col < len(t.config.Columns) {
		t.Sort(col)
		return true
	}
	return false
}

// handleNavigationKeys handles navigation key events
func (t *Table) handleNavigationKeys(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyPgUp:
		t.handlePageUp()
		return true
	case tcell.KeyPgDn:
		t.handlePageDown()
		return true
	case tcell.KeyHome:
		t.handleHome()
		return true
	case tcell.KeyEnd:
		t.handleEnd()
		return true
	}
	return false
}

// handlePageUp handles page up navigation
func (t *Table) handlePageUp() {
	row, col := t.GetSelection()
	newRow := row - 10
	if newRow < t.config.FixedRows {
		newRow = t.config.FixedRows
	}
	t.Select(newRow, col)
}

// handlePageDown handles page down navigation
func (t *Table) handlePageDown() {
	row, col := t.GetSelection()
	newRow := row + 10
	maxRow := t.GetRowCount() - 1
	if newRow > maxRow {
		newRow = maxRow
	}
	t.Select(newRow, col)
}

// handleHome handles home key navigation
func (t *Table) handleHome() {
	_, col := t.GetSelection()
	t.Select(t.config.FixedRows, col)
}

// handleEnd handles end key navigation
func (t *Table) handleEnd() {
	_, col := t.GetSelection()
	t.Select(t.GetRowCount()-1, col)
}

// Draw overrides the base Table Draw to add mutex protection
func (t *Table) Draw(screen tcell.Screen) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	t.Table.Draw(screen)
}

// TableBuilder provides a fluent interface for building tables
type TableBuilder struct {
	config *TableConfig
}

// NewTableBuilder creates a new table builder
func NewTableBuilder() *TableBuilder {
	return &TableBuilder{
		config: DefaultTableConfig(),
	}
}

// WithColumns sets the columns
func (tb *TableBuilder) WithColumns(columns ...Column) *TableBuilder {
	tb.config.Columns = columns
	return tb
}

// WithSelectable sets whether the table is selectable
func (tb *TableBuilder) WithSelectable(selectable bool) *TableBuilder {
	tb.config.Selectable = selectable
	return tb
}

// WithHeader sets whether to show the header
func (tb *TableBuilder) WithHeader(show bool) *TableBuilder {
	tb.config.ShowHeader = show
	return tb
}

// WithColors sets the table colors
func (tb *TableBuilder) WithColors(selected, header, border tcell.Color) *TableBuilder {
	tb.config.SelectedColor = selected
	tb.config.HeaderColor = header
	tb.config.BorderColor = border
	return tb
}

// Build creates the table
func (tb *TableBuilder) Build() *Table {
	return NewTable(tb.config)
}

// ColumnBuilder provides a fluent interface for building columns
type ColumnBuilder struct {
	column Column
}

// NewColumn creates a new column builder
func NewColumn(name string) *ColumnBuilder {
	return &ColumnBuilder{
		column: Column{
			Name:      name,
			Width:     0,
			Alignment: tview.AlignLeft,
			Sortable:  true,
			Hidden:    false,
		},
	}
}

// Width sets the column width
func (cb *ColumnBuilder) Width(width int) *ColumnBuilder {
	cb.column.Width = width
	return cb
}

// Align sets the column alignment
func (cb *ColumnBuilder) Align(alignment int) *ColumnBuilder {
	cb.column.Alignment = alignment
	return cb
}

// Sortable sets whether the column is sortable
func (cb *ColumnBuilder) Sortable(sortable bool) *ColumnBuilder {
	cb.column.Sortable = sortable
	return cb
}

// Hidden sets whether the column is hidden
func (cb *ColumnBuilder) Hidden(hidden bool) *ColumnBuilder {
	cb.column.Hidden = hidden
	return cb
}

// Build returns the column
func (cb *ColumnBuilder) Build() Column {
	return cb.column
}
