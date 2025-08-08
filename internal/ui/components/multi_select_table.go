package components

import (
	"fmt"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// MultiSelectTable extends the regular table with multi-select capabilities
type MultiSelectTable struct {
	*Table
	mu                sync.RWMutex // Protects all selection-related fields
	multiSelectMode   bool
	selectedRows      map[int]bool
	selectionCount    int
	onSelectionChange func(selectedCount int, allSelected bool)
	onRowToggle       func(row int, selected bool, data []string)
	showCheckboxes    bool
	selectAllState    int // 0=none, 1=some, 2=all
}

// NewMultiSelectTable creates a new multi-select table
func NewMultiSelectTable(config *TableConfig) *MultiSelectTable {
	if config == nil {
		config = DefaultTableConfig()
	}

	baseTable := NewTable(config)

	mst := &MultiSelectTable{
		Table:           baseTable,
		multiSelectMode: false,
		selectedRows:    make(map[int]bool),
		selectionCount:  0,
		showCheckboxes:  true,
		selectAllState:  0,
	}

	// Override input handling for multi-select
	mst.Table.Table.SetInputCapture(mst.handleMultiSelectInput)

	return mst
}

// SetMultiSelectMode enables or disables multi-select mode
func (mst *MultiSelectTable) SetMultiSelectMode(enabled bool) {
	mst.mu.Lock()
	defer mst.mu.Unlock()

	mst.multiSelectMode = enabled
	if !enabled {
		mst.clearSelectionUnsafe()
	}
	mst.refreshDisplay()
}

// IsMultiSelectMode returns whether multi-select mode is enabled
func (mst *MultiSelectTable) IsMultiSelectMode() bool {
	mst.mu.RLock()
	defer mst.mu.RUnlock()
	return mst.multiSelectMode
}

// SetShowCheckboxes controls whether to show selection checkboxes
func (mst *MultiSelectTable) SetShowCheckboxes(show bool) {
	mst.showCheckboxes = show
	mst.refreshDisplay()
}

// ToggleRow toggles the selection state of a row
func (mst *MultiSelectTable) ToggleRow(row int) {
	mst.mu.Lock()
	defer mst.mu.Unlock()

	if !mst.multiSelectMode {
		return
	}

	// Adjust for header row
	dataRow := row - 1
	if dataRow < 0 || dataRow >= len(mst.filteredData) {
		return
	}

	// Toggle selection
	if mst.selectedRows[dataRow] {
		delete(mst.selectedRows, dataRow)
		mst.selectionCount--
	} else {
		mst.selectedRows[dataRow] = true
		mst.selectionCount++
	}

	mst.updateSelectAllStateUnsafe()
	mst.refreshDisplay()

	// Notify callback (call outside of lock to avoid deadlock)
	var callRowToggle func()
	var callSelectionChange func()

	if mst.onRowToggle != nil {
		selected := mst.selectedRows[dataRow]
		data := mst.filteredData[dataRow]
		callRowToggle = func() { mst.onRowToggle(dataRow, selected, data) }
	}

	if mst.onSelectionChange != nil {
		allSelected := mst.selectionCount == len(mst.filteredData)
		selCount := mst.selectionCount
		callSelectionChange = func() { mst.onSelectionChange(selCount, allSelected) }
	}

	// Release lock before calling callbacks
	mst.mu.Unlock()

	if callRowToggle != nil {
		callRowToggle()
	}
	if callSelectionChange != nil {
		callSelectionChange()
	}

	// Re-acquire lock for defer cleanup
	mst.mu.Lock()
}

// SelectAll selects all visible rows
func (mst *MultiSelectTable) SelectAll() {
	mst.mu.Lock()
	defer mst.mu.Unlock()

	if !mst.multiSelectMode {
		return
	}

	for i := range mst.filteredData {
		mst.selectedRows[i] = true
	}
	mst.selectionCount = len(mst.filteredData)
	mst.selectAllState = 2
	mst.refreshDisplay()

	if mst.onSelectionChange != nil {
		selCount := mst.selectionCount
		mst.mu.Unlock()
		mst.onSelectionChange(selCount, true)
		mst.mu.Lock()
	}
}

// ClearSelection clears all selections
func (mst *MultiSelectTable) ClearSelection() {
	mst.mu.Lock()
	defer mst.mu.Unlock()
	mst.clearSelectionUnsafe()
}

// clearSelectionUnsafe clears all selections without locking (internal use)
func (mst *MultiSelectTable) clearSelectionUnsafe() {
	mst.selectedRows = make(map[int]bool)
	mst.selectionCount = 0
	mst.selectAllState = 0
	mst.refreshDisplay()

	if mst.onSelectionChange != nil {
		mst.mu.Unlock()
		mst.onSelectionChange(0, false)
		mst.mu.Lock()
	}
}

// GetSelectedRows returns the selected row indices
func (mst *MultiSelectTable) GetSelectedRows() []int {
	mst.mu.RLock()
	defer mst.mu.RUnlock()

	var selected []int
	for row := range mst.selectedRows {
		selected = append(selected, row)
	}
	return selected
}

// GetSelectedData returns the data for selected rows (multi-select mode) or current row (single mode)
func (mst *MultiSelectTable) GetSelectedData() []string {
	mst.mu.RLock()
	defer mst.mu.RUnlock()

	// For compatibility with existing code, return current row data when not in multi-select mode
	// or when no multi-selections are made
	if !mst.multiSelectMode || len(mst.selectedRows) == 0 {
		return mst.Table.GetSelectedData()
	}

	// In multi-select mode, return the first selected row for compatibility
	// (existing code expects []string, not [][]string)
	for row := range mst.selectedRows {
		if row < len(mst.filteredData) {
			return mst.filteredData[row]
		}
	}

	// Fallback to current row
	return mst.Table.GetSelectedData()
}

// GetAllSelectedData returns the data for all selected rows
func (mst *MultiSelectTable) GetAllSelectedData() [][]string {
	mst.mu.RLock()
	defer mst.mu.RUnlock()

	var selectedData [][]string
	for row := range mst.selectedRows {
		if row < len(mst.filteredData) {
			selectedData = append(selectedData, mst.filteredData[row])
		}
	}
	return selectedData
}

// GetSelectionCount returns the number of selected rows
func (mst *MultiSelectTable) GetSelectionCount() int {
	mst.mu.RLock()
	defer mst.mu.RUnlock()
	return mst.selectionCount
}

// IsRowSelected returns whether a row is selected
func (mst *MultiSelectTable) IsRowSelected(row int) bool {
	mst.mu.RLock()
	defer mst.mu.RUnlock()
	return mst.selectedRows[row]
}

// SetOnSelectionChange sets callback for selection changes
func (mst *MultiSelectTable) SetOnSelectionChange(callback func(selectedCount int, allSelected bool)) {
	mst.onSelectionChange = callback
}

// SetOnRowToggle sets callback for individual row toggle events
func (mst *MultiSelectTable) SetOnRowToggle(callback func(row int, selected bool, data []string)) {
	mst.onRowToggle = callback
}

// handleMultiSelectInput handles keyboard input for multi-select
func (mst *MultiSelectTable) handleMultiSelectInput(event *tcell.EventKey) *tcell.EventKey {
	// Check multi-select mode with read lock
	mst.mu.RLock()
	multiSelectMode := mst.multiSelectMode
	selectAllState := mst.selectAllState
	mst.mu.RUnlock()

	if multiSelectMode {
		switch event.Key() {
		case tcell.KeyCtrlA:
			// Select all
			if selectAllState == 2 {
				mst.ClearSelection()
			} else {
				mst.SelectAll()
			}
			return nil

		case tcell.KeyDelete, tcell.KeyBackspace2:
			// Clear selection
			mst.ClearSelection()
			return nil

		case tcell.KeyRune:
			switch event.Rune() {
			case ' ':
				// Toggle current row
				currentRow, _ := mst.Table.Table.GetSelection()
				mst.ToggleRow(currentRow)
				return nil
			case 'a', 'A':
				// Select all (alternative to Ctrl+A)
				if selectAllState == 2 {
					mst.ClearSelection()
				} else {
					mst.SelectAll()
				}
				return nil
			case 'n', 'N':
				// Clear selection (none)
				mst.ClearSelection()
				return nil
			case 'i', 'I':
				// Invert selection
				mst.InvertSelection()
				return nil
			}
		}
	}

	// Handle vim-style navigation
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case 'j':
			// Move down (like ArrowDown)
			return mst.Table.handleInput(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
		case 'k':
			// Move up (like ArrowUp)
			return mst.Table.handleInput(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
		}
	}

	// Let the base table handle other input
	return mst.Table.handleInput(event)
}

// InvertSelection inverts the current selection
func (mst *MultiSelectTable) InvertSelection() {
	mst.mu.Lock()
	defer mst.mu.Unlock()

	if !mst.multiSelectMode {
		return
	}

	newSelection := make(map[int]bool)
	for i := range mst.filteredData {
		if !mst.selectedRows[i] {
			newSelection[i] = true
		}
	}

	mst.selectedRows = newSelection
	mst.selectionCount = len(newSelection)
	mst.updateSelectAllStateUnsafe()
	mst.refreshDisplay()

	if mst.onSelectionChange != nil {
		allSelected := mst.selectionCount == len(mst.filteredData)
		selCount := mst.selectionCount
		mst.mu.Unlock()
		mst.onSelectionChange(selCount, allSelected)
		mst.mu.Lock()
	}
}

// updateSelectAllState updates the select all checkbox state
func (mst *MultiSelectTable) updateSelectAllState() {
	mst.mu.Lock()
	defer mst.mu.Unlock()
	mst.updateSelectAllStateUnsafe()
}

// updateSelectAllStateUnsafe updates the select all checkbox state without locking
func (mst *MultiSelectTable) updateSelectAllStateUnsafe() {
	if mst.selectionCount == 0 {
		mst.selectAllState = 0 // None selected
	} else if mst.selectionCount == len(mst.filteredData) {
		mst.selectAllState = 2 // All selected
	} else {
		mst.selectAllState = 1 // Some selected
	}
}

// refreshDisplay updates the visual display with selection indicators
func (mst *MultiSelectTable) refreshDisplay() {
	mst.Table.Table.Clear()

	if len(mst.config.Columns) == 0 {
		return
	}

	// Add header row with select-all checkbox if in multi-select mode
	headerRow := 0
	if mst.config.ShowHeader {
		for col, column := range mst.config.Columns {
			if column.Hidden {
				continue
			}

			var cellText string
			if col == 0 && mst.multiSelectMode && mst.showCheckboxes {
				// Add select-all checkbox in first column header
				selectAllIcon := mst.getSelectAllIcon()
				cellText = fmt.Sprintf("%s %s", selectAllIcon, column.Name)
			} else {
				cellText = column.Name
			}

			cell := tview.NewTableCell(cellText).
				SetTextColor(mst.config.HeaderColor).
				SetAlign(column.Alignment).
				SetSelectable(false).
				SetExpansion(1)

			mst.Table.Table.SetCell(headerRow, col, cell)
		}
		headerRow++
	}

	// Add data rows with selection indicators
	for rowIndex, rowData := range mst.filteredData {
		displayRow := headerRow + rowIndex

		for col, cellData := range rowData {
			if col >= len(mst.config.Columns) {
				break
			}

			column := mst.config.Columns[col]
			if column.Hidden {
				continue
			}

			var cellText string
			if col == 0 && mst.multiSelectMode && mst.showCheckboxes {
				// Add selection checkbox in first column
				checkboxIcon := mst.getCheckboxIcon(mst.selectedRows[rowIndex])
				cellText = fmt.Sprintf("%s %s", checkboxIcon, cellData)
			} else {
				cellText = cellData
			}

			// Apply row selection styling
			cell := tview.NewTableCell(cellText).
				SetAlign(column.Alignment).
				SetExpansion(1)

			if mst.selectedRows[rowIndex] {
				cell.SetBackgroundColor(tcell.ColorDarkBlue).
					SetTextColor(tcell.ColorWhite)
			} else {
				// Alternate row colors
				if rowIndex%2 == 0 {
					cell.SetBackgroundColor(mst.config.EvenRowColor)
				} else {
					cell.SetBackgroundColor(mst.config.OddRowColor)
				}
			}

			mst.Table.Table.SetCell(displayRow, col, cell)
		}
	}
}

// getCheckboxIcon returns the appropriate checkbox icon
func (mst *MultiSelectTable) getCheckboxIcon(selected bool) string {
	if selected {
		return "[green]☑[white]"
	}
	return "[gray]☐[white]"
}

// getSelectAllIcon returns the appropriate select-all icon
func (mst *MultiSelectTable) getSelectAllIcon() string {
	switch mst.selectAllState {
	case 0:
		return "[gray]☐[white]" // None selected
	case 1:
		return "[yellow]◐[white]" // Some selected
	case 2:
		return "[green]☑[white]" // All selected
	default:
		return "[gray]☐[white]"
	}
}

// Override SetData to maintain selections when data changes
func (mst *MultiSelectTable) SetData(data [][]string) {
	mst.mu.Lock()
	defer mst.mu.Unlock()

	// Store current selections by data content (for persistence across refreshes)
	var selectedDataContent []string
	if mst.multiSelectMode && len(mst.selectedRows) > 0 {
		for row := range mst.selectedRows {
			if row < len(mst.filteredData) && len(mst.filteredData[row]) > 0 {
				// Use first column (usually ID) as identifier
				selectedDataContent = append(selectedDataContent, mst.filteredData[row][0])
			}
		}
	}

	// Update base table data
	mst.Table.SetData(data)

	// Restore selections if multi-select mode is active
	if mst.multiSelectMode && len(selectedDataContent) > 0 {
		newSelectedRows := make(map[int]bool)
		newSelectionCount := 0

		for rowIndex, rowData := range mst.filteredData {
			if len(rowData) > 0 {
				for _, selectedID := range selectedDataContent {
					if rowData[0] == selectedID {
						newSelectedRows[rowIndex] = true
						newSelectionCount++
						break
					}
				}
			}
		}

		mst.selectedRows = newSelectedRows
		mst.selectionCount = newSelectionCount
		mst.updateSelectAllStateUnsafe()
	}

	mst.refreshDisplay()
}

// GetMultiSelectHints returns keyboard hints for multi-select mode
func (mst *MultiSelectTable) GetMultiSelectHints() []string {
	mst.mu.RLock()
	defer mst.mu.RUnlock()

	if !mst.multiSelectMode {
		return []string{}
	}

	return []string{
		"Space: Toggle Row",
		"Ctrl+A: Select All/None",
		"A: Select All",
		"N: Clear Selection",
		"I: Invert Selection",
		"Del: Clear Selection",
	}
}

// GetCurrentRowData returns the data for the currently highlighted row (maintains compatibility)
func (mst *MultiSelectTable) GetCurrentRowData() []string {
	return mst.Table.GetSelectedData()
}