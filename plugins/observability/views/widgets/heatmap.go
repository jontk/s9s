package widgets

import (
	"fmt"
	"math"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// HeatmapWidget displays a grid of values as a heatmap
type HeatmapWidget struct {
	*tview.Box

	title      string
	data       map[string]map[string]float64 // [row][col]value
	rows       []string                      // Row labels
	cols       []string                      // Column labels
	min        float64
	max        float64
	autoScale  bool

	// Display options
	showValues bool
	cellWidth  int
	cellHeight int
	colorFunc  func(float64, float64, float64) tcell.Color

	// Selection
	selectedRow int
	selectedCol int
	selectable  bool
}

// NewHeatmapWidget creates a new heatmap widget
func NewHeatmapWidget(title string) *HeatmapWidget {
	h := &HeatmapWidget{
		Box:        tview.NewBox(),
		title:      title,
		data:       make(map[string]map[string]float64),
		rows:       []string{},
		cols:       []string{},
		autoScale:  true,
		showValues: true,
		cellWidth:  8,
		cellHeight: 1,
		colorFunc:  defaultHeatmapColorFunc,
		selectedRow: -1,
		selectedCol: -1,
		selectable: true,
	}

	h.SetBorder(true).SetTitle(title)
	return h
}

// SetData sets the heatmap data
func (h *HeatmapWidget) SetData(data map[string]map[string]float64) {
	h.data = data

	// Extract unique rows and columns
	rowMap := make(map[string]bool)
	colMap := make(map[string]bool)

	for row, cols := range data {
		rowMap[row] = true
		for col := range cols {
			colMap[col] = true
		}
	}

	// Convert to sorted slices
	h.rows = make([]string, 0, len(rowMap))
	for row := range rowMap {
		h.rows = append(h.rows, row)
	}
	sort.Strings(h.rows)

	h.cols = make([]string, 0, len(colMap))
	for col := range colMap {
		h.cols = append(h.cols, col)
	}
	sort.Strings(h.cols)

	// Update scale if auto-scaling
	if h.autoScale {
		h.updateScale()
	}
}

// SetScale sets manual scale
func (h *HeatmapWidget) SetScale(min, max float64) {
	h.min = min
	h.max = max
	h.autoScale = false
}

// SetCellSize sets the cell dimensions
func (h *HeatmapWidget) SetCellSize(width, height int) {
	h.cellWidth = width
	h.cellHeight = height
}

// GetPrimitive returns the primitive for this widget
func (h *HeatmapWidget) GetPrimitive() tview.Primitive {
	return h
}

// updateScale calculates min/max from data
func (h *HeatmapWidget) updateScale() {
	if len(h.data) == 0 {
		return
	}

	first := true
	for _, cols := range h.data {
		for _, value := range cols {
			if first {
				h.min = value
				h.max = value
				first = false
			} else {
				if value < h.min {
					h.min = value
				}
				if value > h.max {
					h.max = value
				}
			}
		}
	}

	// Add some padding
	if h.max == h.min {
		h.max = h.min + 1
	}
}

// Draw draws the heatmap
func (h *HeatmapWidget) Draw(screen tcell.Screen) {
	h.Box.DrawForSubclass(screen, h)

	x, y, width, height := h.GetInnerRect()
	if width <= 0 || height <= 0 || len(h.rows) == 0 || len(h.cols) == 0 {
		return
	}

	// Calculate layout
	maxRowLabelWidth := 0
	for _, row := range h.rows {
		if len(row) > maxRowLabelWidth {
			maxRowLabelWidth = len(row)
		}
	}
	maxRowLabelWidth++ // Add padding

	// Check if we have enough space
	availableWidth := width - maxRowLabelWidth
	availableCols := availableWidth / h.cellWidth
	colsToShow := len(h.cols)
	if colsToShow > availableCols {
		colsToShow = availableCols
	}

	availableHeight := height - 1 // Reserve one line for column headers
	rowsToShow := len(h.rows)
	if rowsToShow > availableHeight/h.cellHeight {
		rowsToShow = availableHeight / h.cellHeight
	}

	// Draw column headers
	headerY := y
	for i := 0; i < colsToShow; i++ {
		col := h.cols[i]
		cellX := x + maxRowLabelWidth + i*h.cellWidth

		// Truncate column label if needed
		label := col
		if len(label) > h.cellWidth-1 {
			label = label[:h.cellWidth-1]
		}

		// Center the label
		labelX := cellX + (h.cellWidth-len(label))/2
		for j, ch := range label {
			screen.SetContent(labelX+j, headerY, ch, nil,
				tcell.StyleDefault.Foreground(tcell.ColorYellow))
		}
	}

	// Draw rows
	for rowIdx := 0; rowIdx < rowsToShow; rowIdx++ {
		row := h.rows[rowIdx]
		rowY := y + 1 + rowIdx*h.cellHeight

		// Draw row label
		label := row
		if len(label) > maxRowLabelWidth-1 {
			label = label[:maxRowLabelWidth-1]
		}
		for i, ch := range label {
			screen.SetContent(x+i, rowY, ch, nil,
				tcell.StyleDefault.Foreground(tcell.ColorYellow))
		}

		// Draw cells
		for colIdx := 0; colIdx < colsToShow; colIdx++ {
			col := h.cols[colIdx]
			cellX := x + maxRowLabelWidth + colIdx*h.cellWidth

			// Get value
			value := 0.0
			if rowData, ok := h.data[row]; ok {
				if v, ok := rowData[col]; ok {
					value = v
				}
			}

			// Draw cell
			h.drawCell(screen, cellX, rowY, h.cellWidth, h.cellHeight, value,
				rowIdx == h.selectedRow && colIdx == h.selectedCol)
		}
	}
}

// drawCell draws a single heatmap cell
func (h *HeatmapWidget) drawCell(screen tcell.Screen, x, y, width, height int, value float64, selected bool) {
	// Get color
	color := h.colorFunc(value, h.min, h.max)

	// Fill cell with color
	style := tcell.StyleDefault.Background(color)
	if selected && h.selectable {
		style = style.Reverse(true)
	}

	// Determine text color based on background
	textColor := tcell.ColorBlack
	if color == tcell.ColorBlack || color == tcell.ColorDarkBlue || color == tcell.ColorDarkRed {
		textColor = tcell.ColorWhite
	}
	style = style.Foreground(textColor)

	// Fill cell
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			screen.SetContent(x+dx, y+dy, ' ', nil, style)
		}
	}

	// Draw value if enabled
	if h.showValues && width > 4 {
		valueStr := formatHeatmapValue(value)
		if len(valueStr) > width-2 {
			valueStr = valueStr[:width-2]
		}

		// Center the value
		valueX := x + (width-len(valueStr))/2
		valueY := y + height/2

		for i, ch := range valueStr {
			screen.SetContent(valueX+i, valueY, ch, nil, style)
		}
	}
}

// formatHeatmapValue formats a value for display in a cell
func formatHeatmapValue(value float64) string {
	if value >= 100 {
		return fmt.Sprintf("%.0f", value)
	} else if value >= 10 {
		return fmt.Sprintf("%.1f", value)
	} else {
		return fmt.Sprintf("%.2f", value)
	}
}

// defaultHeatmapColorFunc returns colors based on value range
func defaultHeatmapColorFunc(value, min, max float64) tcell.Color {
	// Normalize to 0-1
	normalized := (value - min) / (max - min)
	if math.IsNaN(normalized) || math.IsInf(normalized, 0) {
		normalized = 0
	}
	normalized = math.Max(0, math.Min(1, normalized))

	// Color gradient from blue (cold) to red (hot)
	if normalized < 0.25 {
		return tcell.ColorBlue
	} else if normalized < 0.5 {
		return tcell.ColorGreen
	} else if normalized < 0.75 {
		return tcell.ColorYellow
	} else if normalized < 0.9 {
		return tcell.ColorOrange
	} else {
		return tcell.ColorRed
	}
}

// InputHandler handles input events
func (h *HeatmapWidget) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return h.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if !h.selectable {
			return
		}

		switch event.Key() {
		case tcell.KeyUp:
			if h.selectedRow > 0 {
				h.selectedRow--
			}
		case tcell.KeyDown:
			if h.selectedRow < len(h.rows)-1 {
				h.selectedRow++
			}
		case tcell.KeyLeft:
			if h.selectedCol > 0 {
				h.selectedCol--
			}
		case tcell.KeyRight:
			if h.selectedCol < len(h.cols)-1 {
				h.selectedCol++
			}
		case tcell.KeyEnter:
			// Could trigger a callback here
		}
	})
}

// GetSelectedCell returns the currently selected cell
func (h *HeatmapWidget) GetSelectedCell() (row, col string, value float64, ok bool) {
	if h.selectedRow < 0 || h.selectedRow >= len(h.rows) ||
		h.selectedCol < 0 || h.selectedCol >= len(h.cols) {
		return "", "", 0, false
	}

	row = h.rows[h.selectedRow]
	col = h.cols[h.selectedCol]

	if rowData, ok := h.data[row]; ok {
		if v, ok := rowData[col]; ok {
			return row, col, v, true
		}
	}

	return row, col, 0, false
}

// NodeUtilizationHeatmap is a specialized heatmap for node utilization
type NodeUtilizationHeatmap struct {
	*HeatmapWidget
}

// NewNodeUtilizationHeatmap creates a heatmap optimized for node utilization display
func NewNodeUtilizationHeatmap() *NodeUtilizationHeatmap {
	h := &NodeUtilizationHeatmap{
		HeatmapWidget: NewHeatmapWidget("Node Utilization"),
	}

	// Configure for percentage display
	h.SetScale(0, 100)
	h.SetCellSize(6, 1)

	// Custom color function for utilization
	h.colorFunc = func(value, min, max float64) tcell.Color {
		if value >= 90 {
			return tcell.ColorRed
		} else if value >= 75 {
			return tcell.ColorOrange
		} else if value >= 50 {
			return tcell.ColorYellow
		} else if value >= 25 {
			return tcell.ColorGreen
		} else {
			return tcell.ColorBlue
		}
	}

	return h
}

// SetNodeMetrics sets metrics for nodes grouped by some criteria
func (h *NodeUtilizationHeatmap) SetNodeMetrics(metrics map[string]map[string]float64) {
	h.SetData(metrics)
}