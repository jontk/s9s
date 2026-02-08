package views

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/rivo/tview"
)

// PerformanceView provides cluster performance monitoring
type PerformanceView struct {
	BaseView

	// Dependencies
	client dao.SlurmClient

	// UI Components
	app       *tview.Application
	pages     *tview.Pages
	container *tview.Flex
	statsGrid *tview.Flex

	// Metric displays
	jobsBox    *tview.TextView
	nodesBox   *tview.TextView
	resourceBox *tview.TextView
	controlBar *tview.TextView

	// State
	autoRefresh     bool
	refreshInterval time.Duration
	refreshTimer    *time.Timer
	metrics         *dao.ClusterMetrics
}

// NewPerformanceView creates a new cluster performance monitoring view
func NewPerformanceView(client dao.SlurmClient) *PerformanceView {
	pv := &PerformanceView{
		client:          client,
		autoRefresh:     true,
		refreshInterval: 5 * time.Second,
	}

	pv.BaseView = BaseView{
		name:  "performance",
		title: "Cluster Performance",
	}

	return pv
}

// Init initializes the performance view
func (pv *PerformanceView) Init(ctx context.Context) error {
	pv.ctx = ctx

	// Create metric boxes
	pv.jobsBox = pv.createMetricBox("Jobs")
	pv.nodesBox = pv.createMetricBox("Nodes")
	pv.resourceBox = pv.createMetricBox("Resources")

	// Create stats grid
	pv.statsGrid = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(pv.jobsBox, 0, 1, false).
		AddItem(pv.nodesBox, 0, 1, false).
		AddItem(pv.resourceBox, 0, 1, false)

	// Create control bar
	pv.controlBar = tview.NewTextView()
	pv.controlBar.SetDynamicColors(true)
	pv.controlBar.SetTextAlign(tview.AlignCenter)
	pv.updateControlBar()

	// Create main container
	pv.container = tview.NewFlex()
	pv.container.SetDirection(tview.FlexRow)
	pv.container.SetBorder(true)
	pv.container.SetTitle(" 📊 Cluster Performance ")
	pv.container.SetTitleAlign(tview.AlignCenter)
	pv.container.AddItem(pv.statsGrid, 0, 1, false)
	pv.container.AddItem(pv.controlBar, 2, 0, false)

	// Set up input handling
	pv.container.SetInputCapture(pv.handleInput)

	// Load initial data
	_ = pv.Refresh()

	return nil
}

// createMetricBox creates a bordered box for displaying metrics
func (pv *PerformanceView) createMetricBox(title string) *tview.TextView {
	box := tview.NewTextView()
	box.SetBorder(true)
	box.SetTitle(fmt.Sprintf(" %s ", title))
	box.SetDynamicColors(true)
	box.SetTextAlign(tview.AlignCenter)
	return box
}

// handleInput processes keyboard input
func (pv *PerformanceView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'r', 'R':
		pv.toggleAutoRefresh()
		return nil
	}

	switch event.Key() {
	case tcell.KeyF5:
		_ = pv.Refresh()
		return nil
	}

	return event
}

// toggleAutoRefresh toggles automatic refresh
func (pv *PerformanceView) toggleAutoRefresh() {
	pv.autoRefresh = !pv.autoRefresh
	pv.updateControlBar()

	if pv.autoRefresh {
		pv.scheduleRefresh()
	} else if pv.refreshTimer != nil {
		pv.refreshTimer.Stop()
	}
}

// updateControlBar updates the control bar with current status
func (pv *PerformanceView) updateControlBar() {
	status := ""

	// Auto-refresh status
	if pv.autoRefresh {
		status += "[green]●[white] Auto-refresh: ON  "
	} else {
		status += "[gray]●[white] Auto-refresh: OFF  "
	}

	// Refresh interval
	status += fmt.Sprintf("Interval: %v  ", pv.refreshInterval)

	// Last update
	if pv.metrics != nil {
		status += fmt.Sprintf("Last update: %s  ", pv.metrics.LastUpdated.Format("15:04:05"))
	}

	// Controls hint
	status += "\n[gray]R:Toggle Auto-refresh F5:Refresh[white]"

	pv.controlBar.SetText(status)
}

// Refresh manually refreshes the view
func (pv *PerformanceView) Refresh() error {
	metrics, err := pv.client.Info().GetStats()
	if err != nil {
		return err
	}

	pv.metrics = metrics
	pv.updateDisplay()
	pv.updateControlBar()

	if pv.autoRefresh {
		pv.scheduleRefresh()
	}

	return nil
}

// scheduleRefresh schedules the next auto-refresh
func (pv *PerformanceView) scheduleRefresh() {
	if pv.refreshTimer != nil {
		pv.refreshTimer.Stop()
	}

	pv.refreshTimer = time.AfterFunc(pv.refreshInterval, func() {
		if pv.app != nil {
			pv.app.QueueUpdateDraw(func() {
				_ = pv.Refresh()
			})
		}
	})
}

// updateDisplay updates all metric displays
func (pv *PerformanceView) updateDisplay() {
	if pv.metrics == nil {
		return
	}

	// Jobs metrics
	jobsText := fmt.Sprintf("\n[yellow]Total:[white] %d\n\n", pv.metrics.TotalJobs)
	jobsText += fmt.Sprintf("[green]Running:[white] %d\n", pv.metrics.RunningJobs)
	jobsText += fmt.Sprintf("[blue]Pending:[white] %d\n", pv.metrics.PendingJobs)
	pv.jobsBox.SetText(jobsText)

	// Nodes metrics
	nodesText := fmt.Sprintf("\n[yellow]Total:[white] %d\n\n", pv.metrics.TotalNodes)
	nodesText += fmt.Sprintf("[green]Active:[white] %d\n", pv.metrics.ActiveNodes)
	nodesText += fmt.Sprintf("[blue]Idle:[white] %d\n", pv.metrics.IdleNodes)
	nodesText += fmt.Sprintf("[red]Down:[white] %d\n", pv.metrics.DownNodes)
	pv.nodesBox.SetText(nodesText)

	// Resource metrics
	cpuColor := pv.getUsageColor(pv.metrics.CPUUsage)
	memColor := pv.getUsageColor(pv.metrics.MemoryUsage)

	resourceText := "\n[yellow]Cluster Utilization[white]\n\n"
	resourceText += fmt.Sprintf("[%s]CPU:[white] %.1f%%\n", cpuColor, pv.metrics.CPUUsage)
	resourceText += fmt.Sprintf("[%s]Memory:[white] %.1f%%\n", memColor, pv.metrics.MemoryUsage)
	resourceText += fmt.Sprintf("\n%s\n", pv.getUsageBar(pv.metrics.CPUUsage, "CPU"))
	resourceText += fmt.Sprintf("%s\n", pv.getUsageBar(pv.metrics.MemoryUsage, "Mem"))
	pv.resourceBox.SetText(resourceText)
}

// getUsageColor returns a color based on usage percentage
func (pv *PerformanceView) getUsageColor(usage float64) string {
	if usage >= 90 {
		return "red"
	} else if usage >= 75 {
		return "yellow"
	}
	return "green"
}

// getUsageBar creates a visual bar for usage percentage
func (pv *PerformanceView) getUsageBar(usage float64, label string) string {
	barWidth := 20
	filled := int(usage / 100.0 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	color := pv.getUsageColor(usage)
	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	return fmt.Sprintf("[%s]%s %s[white]", color, label, bar)
}

// BaseView interface implementation

// Render returns the main container
func (pv *PerformanceView) Render() tview.Primitive {
	return pv.container
}

// OnFocus is called when the view gains focus
func (pv *PerformanceView) OnFocus() error {
	_ = pv.Refresh()
	if pv.autoRefresh {
		pv.scheduleRefresh()
	}
	return nil
}

// OnLoseFocus is called when the view loses focus
func (pv *PerformanceView) OnLoseFocus() error {
	if pv.refreshTimer != nil {
		pv.refreshTimer.Stop()
	}
	return nil
}

// Update refreshes the view data
func (pv *PerformanceView) Update() error {
	return pv.Refresh()
}

// OnKey handles key events
func (pv *PerformanceView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	return pv.handleInput(event)
}

// Hints returns keyboard shortcuts
func (pv *PerformanceView) Hints() []string {
	return []string{
		"R: Toggle auto-refresh",
		"F5: Manual refresh",
	}
}

// Stop stops the performance monitoring
func (pv *PerformanceView) Stop() error {
	if pv.refreshTimer != nil {
		pv.refreshTimer.Stop()
	}
	return nil
}

// SetApp sets the tview application reference
func (pv *PerformanceView) SetApp(app *tview.Application) {
	pv.app = app
	pv.BaseView.SetApp(app)
}

// SetPages sets the pages reference for modal handling
func (pv *PerformanceView) SetPages(pages *tview.Pages) {
	pv.pages = pages
}
