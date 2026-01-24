package widgets

import (
	"fmt"
	"math"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SparklineWidget displays a time series as a sparkline chart
type SparklineWidget struct {
	*tview.Box

	title     string
	values    []float64
	maxPoints int
	min       float64
	max       float64
	autoScale bool

	// Display options
	showValue  bool
	showMinMax bool
	unit       string
	colorFunc  func(float64) tcell.Color
}

// NewSparklineWidget creates a new sparkline widget
func NewSparklineWidget(title string, maxPoints int) *SparklineWidget {
	s := &SparklineWidget{
		Box:        tview.NewBox(),
		title:      title,
		values:     make([]float64, 0, maxPoints),
		maxPoints:  maxPoints,
		autoScale:  true,
		showValue:  true,
		showMinMax: true,
		colorFunc:  defaultSparklineColorFunc,
	}

	s.SetBorder(true).SetTitle(title)
	return s
}

// AddValue adds a new value to the sparkline
func (s *SparklineWidget) AddValue(value float64) {
	s.values = append(s.values, value)

	// Keep only the most recent values
	if len(s.values) > s.maxPoints {
		s.values = s.values[len(s.values)-s.maxPoints:]
	}

	// Update min/max if auto-scaling
	if s.autoScale && len(s.values) > 0 {
		s.updateScale()
	}
}

// SetValues sets all values at once
func (s *SparklineWidget) SetValues(values []float64) {
	if len(values) > s.maxPoints {
		s.values = values[len(values)-s.maxPoints:]
	} else {
		s.values = values
	}

	if s.autoScale && len(s.values) > 0 {
		s.updateScale()
	}
}

// SetScale sets manual scale
func (s *SparklineWidget) SetScale(minVal, maxVal float64) {
	s.min = minVal
	s.max = maxVal
	s.autoScale = false
}

// SetUnit sets the unit for display
func (s *SparklineWidget) SetUnit(unit string) {
	s.unit = unit
}

// GetPrimitive returns the primitive for this widget
func (s *SparklineWidget) GetPrimitive() tview.Primitive {
	return s
}

// updateScale updates the min/max scale based on values
func (s *SparklineWidget) updateScale() {
	if len(s.values) == 0 {
		return
	}

	s.min = s.values[0]
	s.max = s.values[0]

	for _, v := range s.values {
		if v < s.min {
			s.min = v
		}
		if v > s.max {
			s.max = v
		}
	}

	// Add some padding to the scale
	dataRange := s.max - s.min
	if dataRange < 0.001 {
		// Avoid division by zero for constant values
		s.min--
		s.max++
	} else {
		padding := dataRange * 0.1
		s.min -= padding
		s.max += padding
	}
}

// Draw draws the sparkline
func (s *SparklineWidget) Draw(screen tcell.Screen) {
	s.DrawForSubclass(screen, s)

	x, y, width, height := s.GetInnerRect()
	if width <= 0 || height <= 0 || len(s.values) == 0 {
		return
	}

	// Reserve space for value display
	chartHeight := height
	valueY := y + height - 1
	if s.showValue && height > 1 {
		chartHeight = height - 1
	}

	// Draw the sparkline
	s.drawSparkline(screen, x, y, width, chartHeight)

	// Draw current value and min/max
	if s.showValue {
		s.drawValueLine(screen, x, valueY, width)
	}
}

// drawSparkline draws the actual sparkline chart
func (s *SparklineWidget) drawSparkline(screen tcell.Screen, x, y, width, height int) {
	if len(s.values) == 0 || height <= 0 {
		return
	}

	// Calculate how many values to display
	valuesToShow := len(s.values)
	if valuesToShow > width {
		valuesToShow = width
	}

	// Start position for values
	startIdx := len(s.values) - valuesToShow

	// Sparkline characters (from lowest to highest)
	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Calculate scale
	dataRange := s.max - s.min
	if dataRange < 0.001 {
		dataRange = 1 // Avoid division by zero
	}

	// Draw each point
	for i := 0; i < valuesToShow; i++ {
		value := s.values[startIdx+i]

		// Normalize value to 0-1
		normalized := (value - s.min) / dataRange
		normalized = math.Max(0, math.Min(1, normalized))

		// Get color
		color := s.colorFunc(value)

		if height == 1 {
			// Single line sparkline
			charIdx := int(normalized * float64(len(sparkChars)-1))
			screen.SetContent(x+i, y, sparkChars[charIdx], nil,
				tcell.StyleDefault.Foreground(color))
		} else {
			// Multi-line sparkline
			barHeight := int(normalized * float64(height))
			if barHeight == 0 && normalized > 0 {
				barHeight = 1
			}

			// Draw vertical bar
			for j := 0; j < height; j++ {
				if j < barHeight {
					// Fill from bottom up
					screen.SetContent(x+i, y+height-1-j, '█', nil,
						tcell.StyleDefault.Foreground(color))
				} else if j == barHeight && barHeight < height {
					// Top of bar with partial block
					partialIdx := int((normalized*float64(height) - float64(barHeight)) * 8)
					if partialIdx > 0 && partialIdx < len(sparkChars) {
						screen.SetContent(x+i, y+height-1-j, sparkChars[partialIdx-1], nil,
							tcell.StyleDefault.Foreground(color))
					}
				}
			}
		}
	}
}

// drawValueLine draws the current value and min/max
func (s *SparklineWidget) drawValueLine(screen tcell.Screen, x, y, width int) {
	if len(s.values) == 0 {
		return
	}

	current := s.values[len(s.values)-1]

	parts := []string{}

	// Current value
	if s.unit != "" {
		parts = append(parts, fmt.Sprintf("%.1f%s", current, s.unit))
	} else {
		parts = append(parts, fmt.Sprintf("%.1f", current))
	}

	// Min/Max
	if s.showMinMax {
		parts = append(parts, fmt.Sprintf("min:%.1f max:%.1f", s.min, s.max))
	}

	text := strings.Join(parts, " ")

	// Center the text
	if len(text) < width {
		startX := x + (width-len(text))/2
		for i, ch := range text {
			screen.SetContent(startX+i, y, ch, nil, tcell.StyleDefault)
		}
	} else {
		// Truncate if too long
		for i := 0; i < width && i < len(text); i++ {
			screen.SetContent(x+i, y, rune(text[i]), nil, tcell.StyleDefault)
		}
	}
}

// defaultSparklineColorFunc returns colors based on value
func defaultSparklineColorFunc(_ float64) tcell.Color {
	// This should be customized based on the metric type
	return tcell.ColorWhite
}

// SparklineGroup manages multiple sparklines
type SparklineGroup struct {
	*tview.Flex
	sparklines map[string]*SparklineWidget
}

// NewSparklineGroup creates a new sparkline group
func NewSparklineGroup(direction int) *SparklineGroup {
	return &SparklineGroup{
		Flex:       tview.NewFlex().SetDirection(direction),
		sparklines: make(map[string]*SparklineWidget),
	}
}

// AddSparkline adds a sparkline to the group
func (sg *SparklineGroup) AddSparkline(id, title string, maxPoints int) *SparklineWidget {
	sparkline := NewSparklineWidget(title, maxPoints)
	sg.sparklines[id] = sparkline
	sg.AddItem(sparkline, 4, 1, false) // Height of 4 for border + chart + value
	return sparkline
}

// UpdateValue updates a specific sparkline with a new value
func (sg *SparklineGroup) UpdateValue(id string, value float64) {
	if sparkline, ok := sg.sparklines[id]; ok {
		sparkline.AddValue(value)
	}
}

// UpdateAll updates all sparklines with new values
func (sg *SparklineGroup) UpdateAll(values map[string]float64) {
	for id, value := range values {
		sg.UpdateValue(id, value)
	}
}

// TimeSeriesSparkline is a specialized sparkline for time series data
type TimeSeriesSparkline struct {
	*SparklineWidget
	timestamps []int64 // Unix timestamps
	timeWindow int64   // Time window in seconds
}

// NewTimeSeriesSparkline creates a new time series sparkline
func NewTimeSeriesSparkline(title string, timeWindow int64) *TimeSeriesSparkline {
	maxPoints := 60 // Default to 60 points (e.g., 1 minute of per-second data)

	return &TimeSeriesSparkline{
		SparklineWidget: NewSparklineWidget(title, maxPoints),
		timestamps:      make([]int64, 0, maxPoints),
		timeWindow:      timeWindow,
	}
}

// AddTimedValue adds a value with timestamp
func (ts *TimeSeriesSparkline) AddTimedValue(value float64, timestamp int64) {
	ts.timestamps = append(ts.timestamps, timestamp)
	ts.AddValue(value)

	// Keep timestamps in sync with values
	if len(ts.timestamps) > ts.maxPoints {
		ts.timestamps = ts.timestamps[len(ts.timestamps)-ts.maxPoints:]
	}

	// Remove old values outside time window
	if ts.timeWindow > 0 && len(ts.timestamps) > 0 {
		cutoff := timestamp - ts.timeWindow
		firstValid := 0

		for i, t := range ts.timestamps {
			if t >= cutoff {
				firstValid = i
				break
			}
		}

		if firstValid > 0 {
			ts.timestamps = ts.timestamps[firstValid:]
			ts.values = ts.values[firstValid:]
		}
	}
}
