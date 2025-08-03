package components

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestMultiSelectTable_Creation(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	if table == nil {
		t.Fatal("NewMultiSelectTable returned nil")
	}

	if table.IsMultiSelectMode() {
		t.Error("Expected multi-select mode to be disabled by default")
	}

	if table.GetSelectionCount() != 0 {
		t.Error("Expected selection count to be 0 initially")
	}
}

func TestMultiSelectTable_ToggleMode(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	// Test enabling multi-select mode
	table.SetMultiSelectMode(true)
	if !table.IsMultiSelectMode() {
		t.Error("Expected multi-select mode to be enabled")
	}

	// Test disabling multi-select mode
	table.SetMultiSelectMode(false)
	if table.IsMultiSelectMode() {
		t.Error("Expected multi-select mode to be disabled")
	}
}

func TestMultiSelectTable_Selection(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	// Set up test data
	testData := [][]string{
		{"job1", "user1", "running", "node1"},
		{"job2", "user2", "pending", "node2"},
		{"job3", "user1", "completed", "node3"},
	}
	table.SetData(testData)

	// Enable multi-select mode
	table.SetMultiSelectMode(true)

	// Test row selection
	table.ToggleRow(1) // Select first data row (header is row 0)
	if table.GetSelectionCount() != 1 {
		t.Errorf("Expected selection count 1, got %d", table.GetSelectionCount())
	}

	if !table.IsRowSelected(0) { // Data row 0 (display row 1)
		t.Error("Expected row 0 to be selected")
	}

	// Test selecting another row
	table.ToggleRow(2) // Select second data row
	if table.GetSelectionCount() != 2 {
		t.Errorf("Expected selection count 2, got %d", table.GetSelectionCount())
	}

	// Test deselecting a row
	table.ToggleRow(1) // Deselect first data row
	if table.GetSelectionCount() != 1 {
		t.Errorf("Expected selection count 1, got %d", table.GetSelectionCount())
	}

	if table.IsRowSelected(0) {
		t.Error("Expected row 0 to be deselected")
	}
}

func TestMultiSelectTable_SelectAll(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	// Set up test data
	testData := [][]string{
		{"job1", "user1", "running", "node1"},
		{"job2", "user2", "pending", "node2"},
		{"job3", "user1", "completed", "node3"},
	}
	table.SetData(testData)
	table.SetMultiSelectMode(true)

	// Test select all
	table.SelectAll()
	if table.GetSelectionCount() != 3 {
		t.Errorf("Expected selection count 3, got %d", table.GetSelectionCount())
	}

	// Verify all rows are selected
	for i := 0; i < 3; i++ {
		if !table.IsRowSelected(i) {
			t.Errorf("Expected row %d to be selected", i)
		}
	}
}

func TestMultiSelectTable_ClearSelection(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	// Set up test data and select some rows
	testData := [][]string{
		{"job1", "user1", "running", "node1"},
		{"job2", "user2", "pending", "node2"},
	}
	table.SetData(testData)
	table.SetMultiSelectMode(true)
	table.SelectAll()

	// Test clear selection
	table.ClearSelection()
	if table.GetSelectionCount() != 0 {
		t.Errorf("Expected selection count 0, got %d", table.GetSelectionCount())
	}

	// Verify no rows are selected
	for i := 0; i < 2; i++ {
		if table.IsRowSelected(i) {
			t.Errorf("Expected row %d to be deselected", i)
		}
	}
}

func TestMultiSelectTable_GetSelectedData(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	// Set up test data
	testData := [][]string{
		{"job1", "user1", "running", "node1"},
		{"job2", "user2", "pending", "node2"},
		{"job3", "user1", "completed", "node3"},
	}
	table.SetData(testData)

	// Test single-row behavior (compatibility mode)
	table.SetMultiSelectMode(false)
	// Simulate selecting row 1 (second row)
	table.Table.Select(2, 0) // Row 2 in display (row 1 in data)
	
	selectedData := table.GetSelectedData()
	if selectedData == nil {
		t.Fatal("GetSelectedData returned nil")
	}
	
	if len(selectedData) != 4 {
		t.Errorf("Expected 4 columns, got %d", len(selectedData))
	}

	// Test multi-row behavior
	table.SetMultiSelectMode(true)
	table.ToggleRow(1) // Select first data row
	table.ToggleRow(3) // Select third data row

	allSelectedData := table.GetAllSelectedData()
	if len(allSelectedData) != 2 {
		t.Errorf("Expected 2 selected rows, got %d", len(allSelectedData))
	}
}

func TestMultiSelectTable_InvertSelection(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	// Set up test data
	testData := [][]string{
		{"job1", "user1", "running", "node1"},
		{"job2", "user2", "pending", "node2"},
		{"job3", "user1", "completed", "node3"},
	}
	table.SetData(testData)
	table.SetMultiSelectMode(true)

	// Select first row
	table.ToggleRow(1)
	if table.GetSelectionCount() != 1 {
		t.Error("Expected 1 row selected initially")
	}

	// Invert selection
	table.InvertSelection()
	if table.GetSelectionCount() != 2 {
		t.Errorf("Expected 2 rows selected after invert, got %d", table.GetSelectionCount())
	}

	// First row should be deselected, others selected
	if table.IsRowSelected(0) {
		t.Error("Expected row 0 to be deselected after invert")
	}
	if !table.IsRowSelected(1) || !table.IsRowSelected(2) {
		t.Error("Expected rows 1 and 2 to be selected after invert")
	}
}

func TestMultiSelectTable_KeyboardShortcuts(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	// Set up test data
	testData := [][]string{
		{"job1", "user1", "running", "node1"},
		{"job2", "user2", "pending", "node2"},
	}
	table.SetData(testData)
	table.SetMultiSelectMode(true)

	// Test space key for toggle
	spaceEvent := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	table.Table.Select(1, 0) // Select first data row for display
	result := table.handleMultiSelectInput(spaceEvent)
	
	if result != nil {
		t.Error("Expected space key to be handled (return nil)")
	}

	// Test Ctrl+A for select all
	ctrlAEvent := tcell.NewEventKey(tcell.KeyCtrlA, 0, tcell.ModNone)
	result = table.handleMultiSelectInput(ctrlAEvent)
	
	if result != nil {
		t.Error("Expected Ctrl+A to be handled (return nil)")
	}
	
	if table.GetSelectionCount() != 2 {
		t.Errorf("Expected all rows selected after Ctrl+A, got %d", table.GetSelectionCount())
	}
}

func TestMultiSelectTable_GetHints(t *testing.T) {
	config := DefaultTableConfig()
	table := NewMultiSelectTable(config)

	// Test hints when multi-select is disabled
	hints := table.GetMultiSelectHints()
	if len(hints) != 0 {
		t.Error("Expected no hints when multi-select is disabled")
	}

	// Test hints when multi-select is enabled
	table.SetMultiSelectMode(true)
	hints = table.GetMultiSelectHints()
	
	if len(hints) == 0 {
		t.Error("Expected hints when multi-select is enabled")
	}

	// Check for expected hint content
	hintText := ""
	for _, hint := range hints {
		hintText += hint + " "
	}

	expectedHints := []string{"Space", "Ctrl+A", "Select All", "Clear", "Invert"}
	for _, expected := range expectedHints {
		if !strings.Contains(hintText, expected) {
			t.Errorf("Expected hint text to contain '%s'", expected)
		}
	}
}