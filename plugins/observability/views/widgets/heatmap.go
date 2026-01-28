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

	title     string
	data      map[string]map[string]float64 // [row][col]value
	rows      []string                      // Row labels
	cols      []string                      // Column labels
	min       float64
	max       float64
	autoScale bool

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
		Box:         tview.NewBox(),
		title:       title,
		data:        make(map[string]map[string]float64),
		rows:        []string{},
		cols:        []string{},
		autoScale:   true,
		showValues:  true,
		cellWidth:   8,
		cellHeight:  1,
		colorFunc:   defaultHeatmapColorFunc,
		selectedRow: -1,
		selectedCol: -1,
		selectable:  true,
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
func (h *HeatmapWidget) SetScale(minVal, maxVal float64) {
	h.min = minVal
	h.max = maxVal
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
	h.DrawForSubclass(screen, h)

	x, y, width, height := h.GetInnerRect()
	if width <= 0 || height <= 0 || len(h.rows) == 0 || len(h.cols) == 0 {
		return
	}

	// Calculate layout parameters
	layout := h.calculateHeatmapLayout(x, y, width, height)

	// Draw column headers
	h.drawColumnHeaders(screen, layout)

	// Draw rows with cells
	h.drawRowsWithCells(screen, layout)
}

type heatmapLayout struct {
	x                int
	y                int
	maxRowLabelWidth int
	colsToShow       int
	rowsToShow       int
	availableWidth   int
	availableHeight  int
}

func (h *HeatmapWidget) calculateHeatmapLayout(x, y, width, height int) *heatmapLayout {
	// Calculate max row label width
	maxRowLabelWidth := 0
	for _, row := range h.rows {
		if len(row) > maxRowLabelWidth {
			maxRowLabelWidth = len(row)
		}
	}
	maxRowLabelWidth++ // Add padding

	// Calculate columns to show
	availableWidth := width - maxRowLabelWidth
	availableCols := availableWidth / h.cellWidth
	colsToShow := len(h.cols)
	if colsToShow > availableCols {
		colsToShow = availableCols
	}

	// Calculate rows to show
	availableHeight := height - 1 // Reserve one line for column headers
	rowsToShow := len(h.rows)
	if rowsToShow > availableHeight/h.cellHeight {
		rowsToShow = availableHeight / h.cellHeight
	}

	return &heatmapLayout{
		x:                x,
		y:                y,
		maxRowLabelWidth: maxRowLabelWidth,
		colsToShow:       colsToShow,
		rowsToShow:       rowsToShow,
		availableWidth:   availableWidth,
		availableHeight:  availableHeight,
	}
}

func (h *HeatmapWidget) drawColumnHeaders(screen tcell.Screen, layout *heatmapLayout) {
	headerY := layout.y
	for i := 0; i < layout.colsToShow; i++ {
		col := h.cols[i]
		cellX := layout.x + layout.maxRowLabelWidth + i*h.cellWidth

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
}

func (h *HeatmapWidget) drawRowsWithCells(screen tcell.Screen, layout *heatmapLayout) {
	for rowIdx := 0; rowIdx < layout.rowsToShow; rowIdx++ {
		row := h.rows[rowIdx]
		rowY := layout.y + 1 + rowIdx*h.cellHeight

		h.drawRowLabel(screen, layout, row, rowY)
		h.drawRowCells(screen, layout, rowIdx, row, rowY)
	}
}

func (h *HeatmapWidget) drawRowLabel(screen tcell.Screen, layout *heatmapLayout, row string, rowY int) {
	label := row
	if len(label) > layout.maxRowLabelWidth-1 {
		label = label[:layout.maxRowLabelWidth-1]
	}
	for i, ch := range label {
		screen.SetContent(layout.x+i, rowY, ch, nil,
			tcell.StyleDefault.Foreground(tcell.ColorYellow))
	}
}

func (h *HeatmapWidget) drawRowCells(screen tcell.Screen, layout *heatmapLayout, rowIdx int, row string, rowY int) {
	for colIdx := 0; colIdx < layout.colsToShow; colIdx++ {
		col := h.cols[colIdx]
		cellX := layout.x + layout.maxRowLabelWidth + colIdx*h.cellWidth

		// Get value
		value := h.getCellValue(row, col)

		// Draw cell
		h.drawCell(screen, cellX, rowY, h.cellWidth, h.cellHeight, value,
			rowIdx == h.selectedRow && colIdx == h.selectedCol)
	}
}

func (h *HeatmapWidget) getCellValue(row, col string) float64 {
	if rowData, ok := h.data[row]; ok {
		if v, ok := rowData[col]; ok {
			return v
		}
	}
	return 0.0
}

// drawCell draws a single heatmap cell
func (h *HeatmapWidget) drawCell(screen tcell.Screen, x, y, width, height int, value float64, selected bool) {
	// Get color and style
	color := h.colorFunc(value, h.min, h.max)
	style := h.buildCellStyle(color, selected)

	// Fill background
	h.fillCellBackground(screen, x, y, width, height, style)

	// Draw value if enabled
	if h.showValues && width > 4 {
		h.drawCellValue(screen, x, y, width, height, value, style)
	}
}

func (h *HeatmapWidget) buildCellStyle(color tcell.Color, selected bool) tcell.Style {
	style := tcell.StyleDefault.Background(color)

	if selected && h.selectable {
		style = style.Reverse(true)
	}

	textColor := h.getTextColorForBackground(color)
	return style.Foreground(textColor)
}

func (h *HeatmapWidget) getTextColorForBackground(color tcell.Color) tcell.Color {
	// Use white text on dark backgrounds for better contrast
	switch color {
	case tcell.ColorBlack, tcell.ColorDarkBlue, tcell.ColorDarkRed:
		return tcell.ColorWhite
	default:
		return tcell.ColorBlack
	}
}

func (h *HeatmapWidget) fillCellBackground(screen tcell.Screen, x, y, width, height int, style tcell.Style) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			screen.SetContent(x+dx, y+dy, ' ', nil, style)
		}
	}
}

func (h *HeatmapWidget) drawCellValue(screen tcell.Screen, x, y, width, height int, value float64, style tcell.Style) {
	valueStr := formatHeatmapValue(value)

	// Truncate if too long
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

// formatHeatmapValue formats a value for display in a cell
func formatHeatmapValue(value float64) string {
	switch {
	case value >= 100:
		return fmt.Sprintf("%.0f", value)
	case value >= 10:
		return fmt.Sprintf("%.1f", value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// defaultHeatmapColorFunc returns colors based on value range
func defaultHeatmapColorFunc(value, minVal, maxVal float64) tcell.Color {
	// Normalize to 0-1
	normalized := (value - minVal) / (maxVal - minVal)
	if math.IsNaN(normalized) || math.IsInf(normalized, 0) {
		normalized = 0
	}
	normalized = math.Max(0, math.Min(1, normalized))

	// Color gradient from blue (cold) to red (hot)
	switch {
	case normalized < 0.25:
		return tcell.ColorBlue
	case normalized < 0.5:
		return tcell.ColorGreen
	case normalized < 0.75:
		return tcell.ColorYellow
	case normalized < 0.9:
		return tcell.ColorOrange
	default:
		return tcell.ColorRed
	}
}

// InputHandler handles input events
func (h *HeatmapWidget) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return h.WrapInputHandler(func(event *tcell.EventKey, _ func(p tview.Primitive)) {
		if !h.selectable {
			return
		}

		keyHandlers := map[tcell.Key]func(){
			tcell.KeyUp:    h.handleKeyUp,
			tcell.KeyDown:  h.handleKeyDown,
			tcell.KeyLeft:  h.handleKeyLeft,
			tcell.KeyRight: h.handleKeyRight,
			tcell.KeyEnter: h.handleKeyEnter,
		}

		if handler, ok := keyHandlers[event.Key()]; ok {
			handler()
		}
	})
}

// handleKeyUp moves selection up
func (h *HeatmapWidget) handleKeyUp() {
	if h.selectedRow > 0 {
		h.selectedRow--
	}
}

// handleKeyDown moves selection down
func (h *HeatmapWidget) handleKeyDown() {
	if h.selectedRow < len(h.rows)-1 {
		h.selectedRow++
	}
}

// handleKeyLeft moves selection left
func (h *HeatmapWidget) handleKeyLeft() {
	if h.selectedCol > 0 {
		h.selectedCol--
	}
}

// handleKeyRight moves selection right
func (h *HeatmapWidget) handleKeyRight() {
	if h.selectedCol < len(h.cols)-1 {
		h.selectedCol++
	}
}

// handleKeyEnter handles Enter key press
func (h *HeatmapWidget) handleKeyEnter() {
	// Could trigger a callback here
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
	h.colorFunc = func(value, _, _ float64) tcell.Color {
		switch {
		case value >= 90:
			return tcell.ColorRed
		case value >= 75:
			return tcell.ColorOrange
		case value >= 50:
			return tcell.ColorYellow
		case value >= 25:
			return tcell.ColorGreen
		default:
			return tcell.ColorBlue
		}
	}

	return h
}

// SetNodeMetrics sets metrics for nodes grouped by some criteria
func (h *NodeUtilizationHeatmap) SetNodeMetrics(metrics map[string]map[string]float64) {
	h.SetData(metrics)
}
