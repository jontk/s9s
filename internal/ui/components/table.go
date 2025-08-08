package components

import (
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/ui/navigation"
	"github.com/rivo/tview"
)

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
		table.Table.SetSelectedStyle(tcell.StyleDefault.
			Background(config.SelectedColor).
			Foreground(tcell.ColorBlack))
	}

	// Set up input handling
	table.Table.SetInputCapture(table.handleInput)

	return table
}

// SetColumns sets the table columns
func (t *Table) SetColumns(columns []Column) {
	t.config.Columns = columns
	t.refresh()
}

// SetData sets the table data
func (t *Table) SetData(data [][]string) {
	t.data = data
	t.applyFilter()
	t.applySort()
	t.refresh()
}

// GetData returns the current table data
func (t *Table) GetData() [][]string {
	return t.data
}

// GetFilteredData returns the filtered table data
func (t *Table) GetFilteredData() [][]string {
	return t.filteredData
}

// SetFilter sets the filter string
func (t *Table) SetFilter(filter string) {
	t.filter = strings.ToLower(filter)
	t.applyFilter()
	t.applySort()
	t.refresh()
}

// GetFilter returns the current filter
func (t *Table) GetFilter() string {
	return t.filter
}

// Sort sorts the table by the specified column
func (t *Table) Sort(column int) {
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
	row, _ := t.Table.GetSelection()
	if row < t.config.FixedRows {
		return -1
	}
	return row - t.config.FixedRows
}

// GetSelectedData returns the data for the currently selected row
func (t *Table) GetSelectedData() []string {
	row := t.GetSelectedRow()
	if row < 0 || row >= len(t.filteredData) {
		return nil
	}
	return t.filteredData[row]
}

// SetOnSelect sets the selection callback
func (t *Table) SetOnSelect(fn func(row, col int)) {
	t.onSelect = fn
	t.Table.SetSelectedFunc(func(row, col int) {
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
	t.data = [][]string{}
	t.filteredData = [][]string{}
	t.Table.Clear()
}

// refresh updates the table display
func (t *Table) refresh() {
	t.Table.Clear()

	// Add header row if configured
	if t.config.ShowHeader && len(t.config.Columns) > 0 {
		for col, column := range t.config.Columns {
			if column.Hidden {
				continue
			}

			header := column.Name
			if column.Sortable && col == t.sortColumn {
				if t.sortAscending {
					header += " ▲"
				} else {
					header += " ▼"
				}
			}

			cell := tview.NewTableCell(header).
				SetTextColor(t.config.HeaderColor).
				SetAlign(column.Alignment).
				SetSelectable(false).
				SetExpansion(1)

			t.Table.SetCell(0, col, cell)
		}
	}

	// Add data rows
	for rowIdx, rowData := range t.filteredData {
		displayRow := rowIdx + t.config.FixedRows

		for colIdx, cellData := range rowData {
			if colIdx >= len(t.config.Columns) || t.config.Columns[colIdx].Hidden {
				continue
			}

			// Truncate text if necessary
			maxWidth := t.config.Columns[colIdx].Width
			if maxWidth > 0 && len(cellData) > maxWidth {
				cellData = cellData[:maxWidth-3] + "..."
			}

			cell := tview.NewTableCell(cellData).
				SetAlign(t.config.Columns[colIdx].Alignment).
				SetExpansion(1)

			// Apply row coloring
			if rowIdx%2 == 0 && t.config.EvenRowColor != tcell.ColorDefault {
				cell.SetTextColor(t.config.EvenRowColor)
			} else if rowIdx%2 == 1 && t.config.OddRowColor != tcell.ColorDefault {
				cell.SetTextColor(t.config.OddRowColor)
			}

			t.Table.SetCell(displayRow, colIdx, cell)
		}
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
	// Let vim mode handle the event first
	vimEvent := t.vimMode.HandleKey(event)
	if vimEvent == nil {
		// Vim mode consumed the event
		return nil
	}
	if vimEvent != event {
		// Vim mode translated the event
		event = vimEvent
	}

	// Handle column sorting with number keys (but skip if vim mode has repeat count)
	if event.Key() >= tcell.KeyRune && event.Rune() >= '1' && event.Rune() <= '9' && !t.vimMode.IsWaitingForKey() {
		col := int(event.Rune() - '1')
		if col < len(t.config.Columns) {
			t.Sort(col)
			return nil
		}
	}

	// Handle page navigation
	switch event.Key() {
	case tcell.KeyPgUp:
		row, col := t.Table.GetSelection()
		newRow := row - 10
		if newRow < t.config.FixedRows {
			newRow = t.config.FixedRows
		}
		t.Table.Select(newRow, col)
		return nil

	case tcell.KeyPgDn:
		row, col := t.Table.GetSelection()
		newRow := row + 10
		maxRow := t.Table.GetRowCount() - 1
		if newRow > maxRow {
			newRow = maxRow
		}
		t.Table.Select(newRow, col)
		return nil

	case tcell.KeyHome:
		_, col := t.Table.GetSelection()
		t.Table.Select(t.config.FixedRows, col)
		return nil

	case tcell.KeyEnd:
		_, col := t.Table.GetSelection()
		t.Table.Select(t.Table.GetRowCount()-1, col)
		return nil

	}

	return event
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
