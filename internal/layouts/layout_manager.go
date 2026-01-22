package layouts

import (
	"encoding/json"
	"fmt"
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
	WidgetTypeView       WidgetType = "view"       // Jobs, Nodes, etc.
	WidgetTypeMetrics    WidgetType = "metrics"    // Resource usage charts
	WidgetTypeAlerts     WidgetType = "alerts"     // System alerts
	WidgetTypeStatus     WidgetType = "status"     // Cluster status
	WidgetTypeTerminal   WidgetType = "terminal"   // Command terminal
	WidgetTypeQuickStart WidgetType = "quickstart" // Quick action buttons
	WidgetTypeClock      WidgetType = "clock"      // Time display
	WidgetTypeLogs       WidgetType = "logs"       // Log viewer
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
	if layout.Name == "" {
		return fmt.Errorf("layout name cannot be empty")
	}

	if layout.Grid.Rows <= 0 || layout.Grid.Columns <= 0 {
		return fmt.Errorf("grid dimensions must be positive")
	}

	// Validate widget placements
	for _, widget := range layout.Widgets {
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

		widget, err := lm.GetWidget(placement.WidgetID)
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

		widget, err := lm.GetWidget(placement.WidgetID)
		if err != nil {
			continue // Skip missing widgets
		}

		// Add widget with proportional height
		lm.container.AddItem(widget.Render(), placement.Height, 0, false)
	}

	return nil
}

// applyGridLayout creates a complex grid layout
func (lm *LayoutManager) applyGridLayout(layout *Layout) error {
	// Create a grid of flex containers
	rows := make([]*tview.Flex, layout.Grid.Rows)

	// Initialize rows
	for i := 0; i < layout.Grid.Rows; i++ {
		rows[i] = tview.NewFlex()
	}

	// Track column usage per row
	colUsage := make([][]bool, layout.Grid.Rows)
	for i := range colUsage {
		colUsage[i] = make([]bool, layout.Grid.Columns)
	}

	// Place widgets in grid
	for _, placement := range layout.Widgets {
		if !placement.Visible {
			continue
		}

		widget, err := lm.GetWidget(placement.WidgetID)
		if err != nil {
			continue // Skip missing widgets
		}

		// Validate placement bounds
		if placement.Row >= layout.Grid.Rows || placement.Column >= layout.Grid.Columns {
			continue
		}

		// Check for overlaps
		canPlace := true
		for r := placement.Row; r < placement.Row+placement.RowSpan && r < layout.Grid.Rows; r++ {
			for c := placement.Column; c < placement.Column+placement.ColSpan && c < layout.Grid.Columns; c++ {
				if colUsage[r][c] {
					canPlace = false
					break
				}
			}
			if !canPlace {
				break
			}
		}

		if !canPlace {
			continue // Skip overlapping widgets
		}

		// Mark cells as used
		for r := placement.Row; r < placement.Row+placement.RowSpan && r < layout.Grid.Rows; r++ {
			for c := placement.Column; c < placement.Column+placement.ColSpan && c < layout.Grid.Columns; c++ {
				colUsage[r][c] = true
			}
		}

		// Add widget to appropriate row
		if placement.ColSpan == 1 {
			rows[placement.Row].AddItem(widget.Render(), 0, placement.Width, false)
		} else {
			// For multi-column spans, create a wrapper
			wrapper := tview.NewFlex()
			wrapper.AddItem(widget.Render(), 0, 1, false)
			rows[placement.Row].AddItem(wrapper, 0, placement.Width, false)
		}
	}

	// Add rows to main container
	lm.container.SetDirection(tview.FlexRow)
	for _, row := range rows {
		if row.GetItemCount() > 0 {
			lm.container.AddItem(row, 0, 1, false)
		}
	}

	return nil
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

// initializeDefaultLayouts creates built-in layout templates
func (lm *LayoutManager) initializeDefaultLayouts() {
	// Standard Layout
	standard := &Layout{
		ID:          "standard",
		Name:        "Standard",
		Description: "Default layout with main view and side panels",
		Template:    "standard",
		Grid: GridConfig{
			Rows:        3,
			Columns:     3,
			GapSize:     1,
			Orientation: "grid",
		},
		Widgets: []WidgetPlacement{
			{WidgetID: "main-view", Row: 0, Column: 0, RowSpan: 2, ColSpan: 2, Width: 70, Height: 80, Visible: true, Priority: 10},
			{WidgetID: "metrics", Row: 0, Column: 2, RowSpan: 1, ColSpan: 1, Width: 30, Height: 40, Visible: true, Priority: 8},
			{WidgetID: "alerts", Row: 1, Column: 2, RowSpan: 1, ColSpan: 1, Width: 30, Height: 40, Visible: true, Priority: 7},
			{WidgetID: "status", Row: 2, Column: 0, RowSpan: 1, ColSpan: 3, Width: 100, Height: 20, Visible: true, Priority: 9},
		},
		Responsive: true,
		Created:    0,
		Modified:   0,
	}

	// Compact Layout
	compact := &Layout{
		ID:          "compact",
		Name:        "Compact",
		Description: "Minimal layout for small terminals",
		Template:    "compact",
		Grid: GridConfig{
			Rows:        2,
			Columns:     1,
			GapSize:     0,
			Orientation: "vertical",
		},
		Widgets: []WidgetPlacement{
			{WidgetID: "main-view", Row: 0, Column: 0, RowSpan: 1, ColSpan: 1, Width: 100, Height: 85, Visible: true, Priority: 10},
			{WidgetID: "status", Row: 1, Column: 0, RowSpan: 1, ColSpan: 1, Width: 100, Height: 15, Visible: true, Priority: 9},
		},
		Responsive: true,
		Created:    0,
		Modified:   0,
	}

	// Monitoring Layout
	monitoring := &Layout{
		ID:          "monitoring",
		Name:        "Monitoring",
		Description: "Focus on metrics and system health",
		Template:    "monitoring",
		Grid: GridConfig{
			Rows:        2,
			Columns:     2,
			GapSize:     1,
			Orientation: "grid",
		},
		Widgets: []WidgetPlacement{
			{WidgetID: "metrics", Row: 0, Column: 0, RowSpan: 1, ColSpan: 1, Width: 50, Height: 50, Visible: true, Priority: 10},
			{WidgetID: "health", Row: 0, Column: 1, RowSpan: 1, ColSpan: 1, Width: 50, Height: 50, Visible: true, Priority: 10},
			{WidgetID: "alerts", Row: 1, Column: 0, RowSpan: 1, ColSpan: 1, Width: 50, Height: 50, Visible: true, Priority: 9},
			{WidgetID: "logs", Row: 1, Column: 1, RowSpan: 1, ColSpan: 1, Width: 50, Height: 50, Visible: true, Priority: 8},
		},
		Responsive: true,
		Created:    0,
		Modified:   0,
	}

	// Admin Layout
	admin := &Layout{
		ID:          "admin",
		Name:        "Administrator",
		Description: "Comprehensive view for system administrators",
		Template:    "admin",
		Grid: GridConfig{
			Rows:        3,
			Columns:     4,
			GapSize:     1,
			Orientation: "grid",
		},
		Widgets: []WidgetPlacement{
			{WidgetID: "main-view", Row: 0, Column: 0, RowSpan: 2, ColSpan: 2, Width: 50, Height: 70, Visible: true, Priority: 10},
			{WidgetID: "metrics", Row: 0, Column: 2, RowSpan: 1, ColSpan: 1, Width: 25, Height: 35, Visible: true, Priority: 9},
			{WidgetID: "health", Row: 0, Column: 3, RowSpan: 1, ColSpan: 1, Width: 25, Height: 35, Visible: true, Priority: 9},
			{WidgetID: "alerts", Row: 1, Column: 2, RowSpan: 1, ColSpan: 1, Width: 25, Height: 35, Visible: true, Priority: 8},
			{WidgetID: "quickstart", Row: 1, Column: 3, RowSpan: 1, ColSpan: 1, Width: 25, Height: 35, Visible: true, Priority: 7},
			{WidgetID: "status", Row: 2, Column: 0, RowSpan: 1, ColSpan: 2, Width: 50, Height: 30, Visible: true, Priority: 9},
			{WidgetID: "terminal", Row: 2, Column: 2, RowSpan: 1, ColSpan: 2, Width: 50, Height: 30, Visible: true, Priority: 6},
		},
		Responsive: true,
		Created:    0,
		Modified:   0,
	}

	// Add layouts
	lm.layouts["standard"] = standard
	lm.layouts["compact"] = compact
	lm.layouts["monitoring"] = monitoring
	lm.layouts["admin"] = admin

	// Set default layout
	lm.currentLayout = standard
}
