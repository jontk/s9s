// Package layouts provides layout management and dashboard widget organization.
package layouts

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// LayoutManager manages dashboard layouts and widgets
type LayoutManager struct {
	mu             sync.RWMutex
	currentLayout  *Layout
	layouts        map[string]*Layout
	widgets        map[string]Widget
	container      *tview.Flex
	app            *tview.Application
	onLayoutChange []func(*Layout)
}

// Layout represents a dashboard layout configuration
type Layout struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Template    string            `json:"template"` // "standard", "compact", "monitoring", "admin"
	Grid        GridConfig        `json:"grid"`
	Widgets     []WidgetPlacement `json:"widgets"`
	Responsive  bool              `json:"responsive"`
	Created     int64             `json:"created"`
	Modified    int64             `json:"modified"`
}

// GridConfig defines the layout grid system
type GridConfig struct {
	Rows        int    `json:"rows"`
	Columns     int    `json:"columns"`
	GapSize     int    `json:"gap_size"`
	Orientation string `json:"orientation"` // "horizontal", "vertical", "grid"
}

// WidgetPlacement defines where a widget is positioned
type WidgetPlacement struct {
	WidgetID  string `json:"widget_id"`
	Row       int    `json:"row"`
	Column    int    `json:"column"`
	RowSpan   int    `json:"row_span"`
	ColSpan   int    `json:"col_span"`
	Width     int    `json:"width"`  // Percentage or fixed
	Height    int    `json:"height"` // Percentage or fixed
	Resizable bool   `json:"resizable"`
	Movable   bool   `json:"movable"`
	Visible   bool   `json:"visible"`
	Priority  int    `json:"priority"` // For responsive behavior
}

// Widget represents a dashboard widget
type Widget interface {
	ID() string
	Name() string
	Description() string
	Type() WidgetType
	Render() tview.Primitive
	Update() error
	Configure() error
	MinSize() (int, int)
	MaxSize() (int, int)
	OnResize(width, height int)
	OnFocus(focus bool)
}

// WidgetType defines widget categories
type WidgetType string

const (
	// WidgetTypeView is the widget type for view widgets like jobs and nodes.
	WidgetTypeView WidgetType = "view"
	// WidgetTypeMetrics is the widget type for resource usage charts.
	WidgetTypeMetrics WidgetType = "metrics"
)

// NewLayoutManager creates a new layout manager
func NewLayoutManager(app *tview.Application) *LayoutManager {
	lm := &LayoutManager{
		layouts:   make(map[string]*Layout),
		widgets:   make(map[string]Widget),
		app:       app,
		container: tview.NewFlex(),
	}

	// Initialize with default layouts
	lm.initializeDefaultLayouts()

	return lm
}

// AddLayout adds a new layout
func (lm *LayoutManager) AddLayout(layout *Layout) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if layout.ID == "" {
		return fmt.Errorf("layout ID cannot be empty")
	}

	// Validate layout
	if err := lm.validateLayout(layout); err != nil {
		return fmt.Errorf("invalid layout: %w", err)
	}

	lm.layouts[layout.ID] = layout
	return nil
}

// GetLayout retrieves a layout by ID
func (lm *LayoutManager) GetLayout(id string) (*Layout, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	layout, exists := lm.layouts[id]
	if !exists {
		return nil, fmt.Errorf("layout %s not found", id)
	}

	return layout, nil
}

// SetCurrentLayout switches to a different layout
func (lm *LayoutManager) SetCurrentLayout(id string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	layout, exists := lm.layouts[id]
	if !exists {
		return fmt.Errorf("layout %s not found", id)
	}

	// Store previous layout
	previousLayout := lm.currentLayout
	lm.currentLayout = layout

	// Apply the layout
	if err := lm.applyLayout(layout); err != nil {
		// Restore previous layout on error
		lm.currentLayout = previousLayout
		return fmt.Errorf("failed to apply layout: %w", err)
	}

	// Notify listeners
	for _, callback := range lm.onLayoutChange {
		go callback(layout)
	}

	return nil
}

// GetCurrentLayout returns the currently active layout
func (lm *LayoutManager) GetCurrentLayout() *Layout {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentLayout
}

// ListLayouts returns all available layouts
func (lm *LayoutManager) ListLayouts() map[string]*Layout {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	layouts := make(map[string]*Layout)
	for id, layout := range lm.layouts {
		layouts[id] = layout
	}
	return layouts
}

// RegisterWidget adds a widget to the available widgets
func (lm *LayoutManager) RegisterWidget(widget Widget) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if widget.ID() == "" {
		return fmt.Errorf("widget ID cannot be empty")
	}

	lm.widgets[widget.ID()] = widget
	return nil
}

// GetWidget retrieves a widget by ID
func (lm *LayoutManager) GetWidget(id string) (Widget, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.getWidgetLocked(id)
}

// getWidgetLocked retrieves a widget without acquiring the lock.
// Caller must hold lm.mu (read or write).
func (lm *LayoutManager) getWidgetLocked(id string) (Widget, error) {
	widget, exists := lm.widgets[id]
	if !exists {
		return nil, fmt.Errorf("widget %s not found", id)
	}

	return widget, nil
}

// GetContainer returns the main layout container
func (lm *LayoutManager) GetContainer() *tview.Flex {
	return lm.container
}

// OnLayoutChange registers a callback for layout changes
func (lm *LayoutManager) OnLayoutChange(callback func(*Layout)) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.onLayoutChange = append(lm.onLayoutChange, callback)
}

// ResizeLayout handles terminal resize events
func (lm *LayoutManager) ResizeLayout() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.currentLayout == nil {
		return nil
	}

	if lm.currentLayout.Responsive {
		return lm.applyResponsiveLayout(lm.currentLayout)
	}

	return nil
}

// ExportLayout exports a layout to JSON
func (lm *LayoutManager) ExportLayout(id string) ([]byte, error) {
	layout, err := lm.GetLayout(id)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(layout, "", "  ")
}

// ImportLayout imports a layout from JSON
func (lm *LayoutManager) ImportLayout(data []byte) error {
	var layout Layout
	if err := json.Unmarshal(data, &layout); err != nil {
		return fmt.Errorf("failed to parse layout: %w", err)
	}

	return lm.AddLayout(&layout)
}

// validateLayout validates a layout configuration
func (lm *LayoutManager) validateLayout(layout *Layout) error {
	if err := lm.validateLayoutName(layout); err != nil {
		return err
	}

	if err := lm.validateGridDimensions(layout); err != nil {
		return err
	}

	return lm.validateWidgetPlacements(layout)
}

// validateLayoutName validates the layout name is not empty
func (lm *LayoutManager) validateLayoutName(layout *Layout) error {
	if layout.Name == "" {
		return fmt.Errorf("layout name cannot be empty")
	}
	return nil
}

// validateGridDimensions validates grid dimensions are positive
func (lm *LayoutManager) validateGridDimensions(layout *Layout) error {
	if layout.Grid.Rows <= 0 || layout.Grid.Columns <= 0 {
		return fmt.Errorf("grid dimensions must be positive")
	}
	return nil
}

// validateWidgetPlacements validates all widget placements in the layout
func (lm *LayoutManager) validateWidgetPlacements(layout *Layout) error {
	for _, widget := range layout.Widgets {
		if err := lm.validateWidgetPlacement(layout, &widget); err != nil {
			return err
		}
	}
	return nil
}

// validateWidgetPlacement validates a single widget placement
func (lm *LayoutManager) validateWidgetPlacement(layout *Layout, widget *WidgetPlacement) error {
	if widget.WidgetID == "" {
		return fmt.Errorf("widget ID cannot be empty")
	}

	if widget.Row < 0 || widget.Column < 0 {
		return fmt.Errorf("widget position cannot be negative")
	}

	if widget.Row >= layout.Grid.Rows || widget.Column >= layout.Grid.Columns {
		return fmt.Errorf("widget position exceeds grid boundaries")
	}

	if widget.RowSpan <= 0 || widget.ColSpan <= 0 {
		return fmt.Errorf("widget span must be positive")
	}

	return nil
}

// applyLayout applies a layout to the container
func (lm *LayoutManager) applyLayout(layout *Layout) error {
	// Clear existing layout
	lm.container.Clear()

	// Create grid based on layout configuration
	switch layout.Grid.Orientation {
	case "horizontal":
		return lm.applyHorizontalLayout(layout)
	case "vertical":
		return lm.applyVerticalLayout(layout)
	case "grid":
		return lm.applyGridLayout(layout)
	default:
		return lm.applyGridLayout(layout) // Default to grid
	}
}

// applyHorizontalLayout creates a horizontal layout
func (lm *LayoutManager) applyHorizontalLayout(layout *Layout) error {
	// Create horizontal flex container
	for _, placement := range layout.Widgets {
		if !placement.Visible {
			continue
		}

		widget, err := lm.getWidgetLocked(placement.WidgetID)
		if err != nil {
			continue // Skip missing widgets
		}

		// Add widget with proportional width
		lm.container.AddItem(widget.Render(), 0, placement.Width, false)
	}

	return nil
}

// applyVerticalLayout creates a vertical layout
func (lm *LayoutManager) applyVerticalLayout(layout *Layout) error {
	// Set container direction to vertical
	lm.container.SetDirection(tview.FlexRow)

	for _, placement := range layout.Widgets {
		if !placement.Visible {
			continue
		}

		widget, err := lm.getWidgetLocked(placement.WidgetID)
		if err != nil {
			continue // Skip missing widgets
		}

		// Add widget with proportional height
		lm.container.AddItem(widget.Render(), 0, placement.Height, false)
	}

	return nil
}

// applyGridLayout builds a nested flex tree that properly handles row and column spanning.
//
// Algorithm:
//  1. Find "row bands" — groups of consecutive rows linked by row-spanning widgets.
//  2. For each band, build a horizontal flex. Within each column position, if
//     multiple widgets are stacked vertically, wrap them in a vertical flex.
//  3. Stack all bands vertically in the container.
func (lm *LayoutManager) applyGridLayout(layout *Layout) error {
	lm.container.SetDirection(tview.FlexRow)

	// Collect visible, resolvable placements
	var placements []WidgetPlacement
	for _, p := range layout.Widgets {
		if !p.Visible {
			continue
		}
		if _, err := lm.getWidgetLocked(p.WidgetID); err != nil {
			continue
		}
		placements = append(placements, p)
	}

	// Find row bands — connected groups of rows linked by spanning widgets
	bands := lm.findRowBands(layout.Grid.Rows, placements)

	for _, band := range bands {
		bandFlex := lm.buildBandFlex(band, placements)
		if bandFlex != nil {
			lm.container.AddItem(bandFlex, 0, 1, false)
		}
	}

	return nil
}

// rowBand represents a contiguous group of rows that share spanning widgets.
type rowBand struct {
	startRow int
	endRow   int // exclusive
}

// findRowBands groups rows into bands connected by row-spanning widgets.
func (lm *LayoutManager) findRowBands(numRows int, placements []WidgetPlacement) []rowBand {
	// For each row, find the furthest row any widget starting there reaches
	reach := make([]int, numRows)
	for i := range reach {
		reach[i] = i + 1
	}
	for _, p := range placements {
		end := p.Row + p.RowSpan
		if end > reach[p.Row] {
			reach[p.Row] = end
		}
	}

	// Merge overlapping reaches into bands
	var bands []rowBand
	i := 0
	for i < numRows {
		start := i
		end := reach[i]
		// Extend band while rows overlap
		for j := start + 1; j < end && j < numRows; j++ {
			if reach[j] > end {
				end = reach[j]
			}
		}
		bands = append(bands, rowBand{startRow: start, endRow: end})
		i = end
	}
	return bands
}

// buildBandFlex creates a horizontal flex for a row band. Widgets that span the
// full band height sit alongside vertical stacks of smaller widgets.
func (lm *LayoutManager) buildBandFlex(band rowBand, placements []WidgetPlacement) *tview.Flex {
	// Collect placements that start within this band
	var bandPlacements []WidgetPlacement
	for _, p := range placements {
		if p.Row >= band.startRow && p.Row < band.endRow {
			bandPlacements = append(bandPlacements, p)
		}
	}
	if len(bandPlacements) == 0 {
		return nil
	}

	colGroups, cols := lm.groupByColumn(bandPlacements)

	hFlex := tview.NewFlex()
	for _, col := range cols {
		group := colGroups[col]
		width := group[0].Width
		hFlex.AddItem(lm.buildColumnItem(group), 0, width, false)
	}

	return hFlex
}

// groupByColumn groups placements by starting column and returns sorted unique columns.
func (lm *LayoutManager) groupByColumn(placements []WidgetPlacement) (map[int][]WidgetPlacement, []int) {
	colGroups := make(map[int][]WidgetPlacement)
	var cols []int
	seen := make(map[int]bool)

	for _, p := range placements {
		colGroups[p.Column] = append(colGroups[p.Column], p)
		if !seen[p.Column] {
			seen[p.Column] = true
			cols = append(cols, p.Column)
		}
	}

	sort.Ints(cols)

	return colGroups, cols
}

// buildColumnItem builds a primitive for a column group — a single widget or
// a vertical stack if multiple widgets share the column.
func (lm *LayoutManager) buildColumnItem(group []WidgetPlacement) tview.Primitive {
	if len(group) == 1 {
		// Error can be safely ignored: the caller (applyGridLayout) already
		// verified that every widget ID in group resolves successfully, and
		// we still hold lm.mu so the map cannot change.
		w, _ := lm.getWidgetLocked(group[0].WidgetID)
		return w.Render()
	}

	// Sort by row within the column
	sort.Slice(group, func(i, j int) bool {
		return group[i].Row < group[j].Row
	})

	vFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, p := range group {
		// Error ignored for the same reason as above: widgets were pre-validated
		// in applyGridLayout and the lock is still held.
		w, _ := lm.getWidgetLocked(p.WidgetID)
		vFlex.AddItem(w.Render(), 0, 1, false)
	}
	return vFlex
}

// applyResponsiveLayout applies responsive behavior
func (lm *LayoutManager) applyResponsiveLayout(layout *Layout) error {
	// Get terminal size (use a reasonable default if unavailable)
	width := 100 // Default width

	// Try to get actual terminal size if available
	if screen := tcell.NewSimulationScreen(""); screen != nil {
		w, _ := screen.Size()
		if w > 0 {
			width = w
		}
	}

	// Adjust layout based on terminal size
	if width < 80 {
		// Compact mode for narrow terminals
		return lm.applyCompactLayout(layout)
	} else if width > 120 {
		// Wide mode for large terminals
		return lm.applyWideLayout(layout)
	}

	// Normal layout
	return lm.applyLayout(layout)
}

// applyCompactLayout creates a compact layout for small terminals
func (lm *LayoutManager) applyCompactLayout(layout *Layout) error {
	// Use vertical layout for compact mode
	compactLayout := *layout
	compactLayout.Grid.Orientation = "vertical"

	// Hide lower priority widgets
	visibleWidgets := []WidgetPlacement{}
	for _, widget := range layout.Widgets {
		if widget.Priority >= 5 { // Only show high-priority widgets
			visibleWidgets = append(visibleWidgets, widget)
		}
	}
	compactLayout.Widgets = visibleWidgets

	return lm.applyVerticalLayout(&compactLayout)
}

// applyWideLayout creates an extended layout for wide terminals
func (lm *LayoutManager) applyWideLayout(layout *Layout) error {
	// Use grid layout with more columns for wide mode
	wideLayout := *layout
	wideLayout.Grid.Columns = layout.Grid.Columns + 2

	return lm.applyGridLayout(&wideLayout)
}

// initializeDefaultLayouts creates built-in layout templates.
// The dashboard's own 6-panel view is the default (no layout applied).
// Layouts offer alternative arrangements of standalone widgets.
func (lm *LayoutManager) initializeDefaultLayouts() {
	// Monitoring Layout — metrics and health side by side, full height
	monitoring := &Layout{
		ID:          "monitoring",
		Name:        "Monitoring",
		Description: "Live metrics and health checks side by side",
		Template:    "monitoring",
		Grid: GridConfig{
			Rows:        1,
			Columns:     2,
			GapSize:     1,
			Orientation: "grid",
		},
		Widgets: []WidgetPlacement{
			{WidgetID: "metrics", Row: 0, Column: 0, RowSpan: 1, ColSpan: 1, Width: 40, Height: 100, Visible: true, Priority: 10},
			{WidgetID: "health", Row: 0, Column: 1, RowSpan: 1, ColSpan: 1, Width: 60, Height: 100, Visible: true, Priority: 10},
		},
		Responsive: true,
	}

	lm.layouts["monitoring"] = monitoring
	// No default layout — dashboard shows its built-in panels
}
