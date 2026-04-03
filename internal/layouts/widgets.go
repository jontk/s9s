package layouts

import (
	"fmt"
	"time"

	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/views"
	"github.com/rivo/tview"
)

// BaseWidget provides common widget functionality
type BaseWidget struct {
	id          string
	name        string
	description string
	widgetType  WidgetType
	primitive   tview.Primitive
	minWidth    int
	minHeight   int
	maxWidth    int
	maxHeight   int
	focused     bool
	updateFunc  func() error
	configFunc  func() error
}

// ID returns the widget ID
func (w *BaseWidget) ID() string {
	return w.id
}

// Name returns the widget name
func (w *BaseWidget) Name() string {
	return w.name
}

// Description returns the widget description
func (w *BaseWidget) Description() string {
	return w.description
}

// Type returns the widget type
func (w *BaseWidget) Type() WidgetType {
	return w.widgetType
}

// Render returns the widget's tview primitive
func (w *BaseWidget) Render() tview.Primitive {
	return w.primitive
}

// Update calls the widget's update function
func (w *BaseWidget) Update() error {
	if w.updateFunc != nil {
		return w.updateFunc()
	}
	return nil
}

// Configure calls the widget's configuration function
func (w *BaseWidget) Configure() error {
	if w.configFunc != nil {
		return w.configFunc()
	}
	return nil
}

// MinSize returns the minimum widget size
func (w *BaseWidget) MinSize() (int, int) {
	return w.minWidth, w.minHeight
}

// MaxSize returns the maximum widget size
func (w *BaseWidget) MaxSize() (int, int) {
	return w.maxWidth, w.maxHeight
}

// OnResize handles widget resizing
func (w *BaseWidget) OnResize(_, _ int) {
	// Default implementation - can be overridden
}

// OnFocus handles widget focus changes
func (w *BaseWidget) OnFocus(focus bool) {
	w.focused = focus
}

// ViewWidget wraps a view as a widget
type ViewWidget struct {
	*BaseWidget
	view views.View
}

// NewViewWidget creates a new view widget
func NewViewWidget(id, name string, view views.View) *ViewWidget {
	widget := &ViewWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        name,
			description: fmt.Sprintf("%s view widget", name),
			widgetType:  WidgetTypeView,
			primitive:   view.Render(),
			minWidth:    20,
			minHeight:   10,
			maxWidth:    0, // No max
			maxHeight:   0, // No max
		},
		view: view,
	}

	widget.updateFunc = func() error {
		return view.Refresh()
	}

	return widget
}

// MetricsWidget displays system metrics
type MetricsWidget struct {
	*BaseWidget
	client      dao.SlurmClient
	textView    *tview.TextView
	updateTimer *time.Ticker
}

// NewMetricsWidget creates a new metrics widget
func NewMetricsWidget(id string, client dao.SlurmClient) *MetricsWidget {
	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetBorder(true)
	textView.SetTitle(" Cluster Metrics ")
	textView.SetTitleAlign(tview.AlignCenter)

	widget := &MetricsWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        "Cluster Metrics",
			description: "Real-time cluster resource metrics",
			widgetType:  WidgetTypeMetrics,
			primitive:   textView,
			minWidth:    25,
			minHeight:   8,
			maxWidth:    60,
			maxHeight:   20,
		},
		client:   client,
		textView: textView,
	}

	widget.updateFunc = widget.updateMetrics
	widget.startAutoUpdate()

	return widget
}

// updateMetrics refreshes the metrics display
func (w *MetricsWidget) updateMetrics() error {
	if w.client.Info() == nil {
		w.textView.SetText("[red]No cluster info available[white]")
		return nil
	}

	stats, err := w.client.Info().GetStats()
	if err != nil {
		w.textView.SetText(fmt.Sprintf("[red]Error: %v[white]", err))
		return err
	}

	content := fmt.Sprintf(`[yellow]CPU Usage:[white] %.1f%%
[yellow]Memory:[white] %.1f%%
[yellow]Jobs:[white] %d running, %d pending
[yellow]Nodes:[white] %d total, %d active

[green]Updated:[white] %s`,
		stats.CPUUsage,
		stats.MemoryUsage,
		stats.RunningJobs, stats.PendingJobs,
		stats.TotalNodes, stats.ActiveNodes,
		time.Now().Format("15:04:05"))

	w.textView.SetText(content)
	return nil
}

// startAutoUpdate starts automatic metric updates
func (w *MetricsWidget) startAutoUpdate() {
	w.updateTimer = time.NewTicker(5 * time.Second)
	go func() {
		for range w.updateTimer.C {
			_ = w.updateMetrics()
		}
	}()
}
