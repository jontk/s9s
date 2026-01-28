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

	// Note: InputCapture is NOT set here. Views handle all keyboard input in their OnKey methods.
	// The base Table also doesn't set InputCapture anymore to allow View.OnKey to work correctly.

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
	mst.mu.Lock()
	defer mst.mu.Unlock()

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

	// Adjust for header row and check bounds with Table mutex protection
	dataRow := row - 1
	mst.Table.mu.RLock()
	filteredDataLen := len(mst.filteredData)
	mst.Table.mu.RUnlock()

	if dataRow < 0 || dataRow >= filteredDataLen {
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

	// Prepare callback data with Table mutex protection
	var callRowToggle func()
	var callSelectionChange func()

	if mst.onRowToggle != nil {
		selected := mst.selectedRows[dataRow]
		mst.Table.mu.RLock()
		var data []string
		if dataRow < len(mst.filteredData) {
			data = mst.filteredData[dataRow]
		}
		mst.Table.mu.RUnlock()
		callRowToggle = func() { mst.onRowToggle(dataRow, selected, data) }
	}

	if mst.onSelectionChange != nil {
		mst.Table.mu.RLock()
		allSelected := mst.selectionCount == len(mst.filteredData)
		mst.Table.mu.RUnlock()
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

	mst.Table.mu.RLock()
	filteredDataLen := len(mst.filteredData)
	mst.Table.mu.RUnlock()

	for i := 0; i < filteredDataLen; i++ {
		mst.selectedRows[i] = true
	}
	mst.selectionCount = filteredDataLen
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

	selected := make([]int, 0, len(mst.selectedRows))
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
	mst.Table.mu.RLock()
	filteredData := mst.filteredData
	mst.Table.mu.RUnlock()

	for row := range mst.selectedRows {
		if row < len(filteredData) {
			return filteredData[row]
		}
	}

	// Fallback to current row
	return mst.Table.GetSelectedData()
}

// GetAllSelectedData returns the data for all selected rows
func (mst *MultiSelectTable) GetAllSelectedData() [][]string {
	mst.mu.RLock()
	defer mst.mu.RUnlock()

	mst.Table.mu.RLock()
	filteredData := mst.filteredData
	mst.Table.mu.RUnlock()

	var selectedData [][]string
	for row := range mst.selectedRows {
		if row < len(filteredData) {
			selectedData = append(selectedData, filteredData[row])
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

	// Handle multi-select mode shortcuts
	if multiSelectMode {
		if handled := mst.handleMultiSelectShortcuts(event, selectAllState); handled {
			return nil
		}
	}

	// Handle vim-style navigation
	if handled := mst.handleVimNavigation(event); handled {
		return nil
	}

	// Let the base table handle other input
	return mst.handleInput(event)
}

// handleMultiSelectShortcuts processes multi-select mode keyboard shortcuts
func (mst *MultiSelectTable) handleMultiSelectShortcuts(event *tcell.EventKey, selectAllState int) bool {
	switch event.Key() {
	case tcell.KeyCtrlA:
		mst.toggleSelectAll(selectAllState)
		return true
	case tcell.KeyDelete, tcell.KeyBackspace2:
		mst.ClearSelection()
		return true
	case tcell.KeyRune:
		return mst.handleMultiSelectRune(event.Rune(), selectAllState)
	}
	return false
}

// handleMultiSelectRune processes character shortcuts for multi-select mode
func (mst *MultiSelectTable) handleMultiSelectRune(r rune, selectAllState int) bool {
	switch r {
	case ' ':
		currentRow, _ := mst.GetSelection()
		mst.ToggleRow(currentRow)
		return true
	case 'a', 'A':
		mst.toggleSelectAll(selectAllState)
		return true
	case 'n', 'N':
		mst.ClearSelection()
		return true
	case 'i', 'I':
		mst.InvertSelection()
		return true
	}
	return false
}

// toggleSelectAll toggles between select all and clear all
func (mst *MultiSelectTable) toggleSelectAll(selectAllState int) {
	if selectAllState == 2 {
		mst.ClearSelection()
	} else {
		mst.SelectAll()
	}
}

// handleVimNavigation processes vim-style navigation keys
func (mst *MultiSelectTable) handleVimNavigation(event *tcell.EventKey) bool {
	if event.Key() != tcell.KeyRune {
		return false
	}

	switch event.Rune() {
	case 'j':
		mst.handleInput(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
		return true
	case 'k':
		mst.handleInput(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
		return true
	}
	return false
}

// InvertSelection inverts the current selection
func (mst *MultiSelectTable) InvertSelection() {
	mst.mu.Lock()
	defer mst.mu.Unlock()

	if !mst.multiSelectMode {
		return
	}

	mst.Table.mu.RLock()
	filteredDataLen := len(mst.filteredData)
	mst.Table.mu.RUnlock()

	newSelection := make(map[int]bool)
	for i := 0; i < filteredDataLen; i++ {
		if !mst.selectedRows[i] {
			newSelection[i] = true
		}
	}

	mst.selectedRows = newSelection
	mst.selectionCount = len(newSelection)
	mst.updateSelectAllStateUnsafe()
	mst.refreshDisplay()

	if mst.onSelectionChange != nil {
		allSelected := mst.selectionCount == filteredDataLen
		selCount := mst.selectionCount
		mst.mu.Unlock()
		mst.onSelectionChange(selCount, allSelected)
		mst.mu.Lock()
	}
}

/*
TODO(lint): Review unused code - func (*MultiSelectTable).updateSelectAllState is unused

updateSelectAllState updates the select all checkbox state
func (mst *MultiSelectTable) updateSelectAllState() {
	mst.mu.Lock()
	defer mst.mu.Unlock()
	mst.updateSelectAllStateUnsafe()
}
*/

// updateSelectAllStateUnsafe updates the select all checkbox state without locking
func (mst *MultiSelectTable) updateSelectAllStateUnsafe() {
	switch {
	case mst.selectionCount == 0:
		mst.selectAllState = 0 // None selected
	case mst.selectionCount == len(mst.filteredData):
		mst.selectAllState = 2 // All selected
	default:
		mst.selectAllState = 1 // Some selected
	}
}

// refreshDisplay updates the visual display with selection indicators
func (mst *MultiSelectTable) refreshDisplay() {
	// Acquire Table's mutex to protect tview.Table operations
	mst.Table.mu.Lock()
	defer mst.Table.mu.Unlock()

	mst.Table.Table.Clear()

	if len(mst.config.Columns) == 0 {
		return
	}

	// Render header row
	headerRow := mst.renderHeaderRow()

	// Render data rows
	mst.renderDataRows(headerRow)
}

// renderHeaderRow adds the header row to the table
func (mst *MultiSelectTable) renderHeaderRow() int {
	headerRow := 0
	if !mst.config.ShowHeader {
		return headerRow
	}

	for col, column := range mst.config.Columns {
		if column.Hidden {
			continue
		}

		var cellText string
		if col == 0 && mst.multiSelectMode && mst.showCheckboxes {
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

		mst.SetCell(headerRow, col, cell)
	}
	return 1
}

// renderDataRows adds all data rows to the table
func (mst *MultiSelectTable) renderDataRows(headerRow int) {
	filteredData := mst.filteredData

	for rowIndex, rowData := range filteredData {
		displayRow := headerRow + rowIndex
		mst.renderRow(displayRow, rowIndex, rowData)
	}
}

// renderRow renders a single data row with proper styling and checkboxes
func (mst *MultiSelectTable) renderRow(displayRow, rowIndex int, rowData []string) {
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
			checkboxIcon := mst.getCheckboxIcon(mst.selectedRows[rowIndex])
			cellText = fmt.Sprintf("%s %s", checkboxIcon, cellData)
		} else {
			cellText = cellData
		}

		cell := mst.createStyledCell(cellText, column, rowIndex)
		mst.SetCell(displayRow, col, cell)
	}
}

// createStyledCell creates a table cell with proper styling based on selection state
func (mst *MultiSelectTable) createStyledCell(cellText string, column Column, rowIndex int) *tview.TableCell {
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

	return cell
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

// SetData sets the table data and maintains selections when data changes.
func (mst *MultiSelectTable) SetData(data [][]string) {
	// Save current selections before data change
	selectedDataContent := mst.saveCurrentSelections()

	// Update base table data
	mst.Table.SetData(data)

	// Restore selections
	mst.mu.Lock()
	defer mst.mu.Unlock()

	mst.restoreSelections(selectedDataContent)
	mst.refreshDisplay()
}

// saveCurrentSelections saves the current selections by content for persistence
func (mst *MultiSelectTable) saveCurrentSelections() []string {
	mst.mu.Lock()
	defer mst.mu.Unlock()

	var selectedDataContent []string
	if !mst.multiSelectMode || len(mst.selectedRows) == 0 {
		return selectedDataContent
	}

	// Get filtered data through the accessor to respect Table's mutex
	filteredData := mst.GetFilteredData()
	for row := range mst.selectedRows {
		if row < len(filteredData) && len(filteredData[row]) > 0 {
			// Use first column (usually ID) as identifier
			selectedDataContent = append(selectedDataContent, filteredData[row][0])
		}
	}

	return selectedDataContent
}

// restoreSelections restores selections based on saved data content
func (mst *MultiSelectTable) restoreSelections(selectedDataContent []string) {
	if !mst.multiSelectMode || len(selectedDataContent) == 0 {
		return
	}

	newSelectedRows := make(map[int]bool)
	newSelectionCount := 0

	// Get updated filtered data through the accessor
	filteredData := mst.GetFilteredData()
	for rowIndex, rowData := range filteredData {
		if len(rowData) > 0 {
			if mst.isRowSelected(rowData[0], selectedDataContent) {
				newSelectedRows[rowIndex] = true
				newSelectionCount++
			}
		}
	}

	mst.selectedRows = newSelectedRows
	mst.selectionCount = newSelectionCount
	mst.updateSelectAllStateUnsafe()
}

// isRowSelected checks if a row ID is in the selected IDs list
func (mst *MultiSelectTable) isRowSelected(rowID string, selectedIDs []string) bool {
	for _, selectedID := range selectedIDs {
		if rowID == selectedID {
			return true
		}
	}
	return false
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
