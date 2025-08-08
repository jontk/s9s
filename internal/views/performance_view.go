package views

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/performance"
	"github.com/jontk/s9s/internal/ui/widgets"
	"github.com/rivo/tview"
)

// PerformanceView provides a comprehensive performance monitoring interface
type PerformanceView struct {
	BaseView
	
	// Dependencies
	client    dao.SlurmClient
	profiler  *performance.Profiler
	optimizer *performance.Optimizer
	
	// UI Components
	app         *tview.Application
	pages       *tview.Pages
	container   *tview.Flex
	dashboard   *widgets.PerformanceDashboard
	controlBar  *tview.TextView
	
	// State
	monitoringEnabled bool
	autoRefresh       bool
	refreshInterval   time.Duration
}

// NewPerformanceView creates a new performance monitoring view
func NewPerformanceView(client dao.SlurmClient) *PerformanceView {
	pv := &PerformanceView{
		client:          client,
		monitoringEnabled: true,
		autoRefresh:     true,
		refreshInterval: 2 * time.Second,
	}
	
	pv.BaseView = BaseView{
		name:  "performance",
		title: "Performance Monitor",
	}
	
	// Initialize performance components
	pv.profiler = performance.NewProfiler()
	pv.optimizer = performance.NewOptimizer(pv.profiler)
	
	return pv
}

// Init initializes the performance view
func (pv *PerformanceView) Init(ctx context.Context) error {
	pv.ctx = ctx
	
	// Initialize dashboard
	pv.dashboard = widgets.NewPerformanceDashboard(pv.profiler, pv.optimizer)
	pv.dashboard.SetUpdateInterval(pv.refreshInterval)
	
	// Create control bar
	pv.controlBar = tview.NewTextView()
	pv.controlBar.SetDynamicColors(true)
	pv.controlBar.SetTextAlign(tview.AlignCenter)
	pv.updateControlBar()
	
	// Create main container
	pv.container = tview.NewFlex()
	pv.container.SetDirection(tview.FlexRow)
	pv.container.AddItem(pv.dashboard.GetContainer(), 0, 1, true)
	pv.container.AddItem(pv.controlBar, 2, 0, false)
	
	// Set up input handling
	pv.container.SetInputCapture(pv.handleInput)
	
	// Don't start monitoring automatically during initialization
	// Monitoring will be started when the view is first displayed
	// if pv.monitoringEnabled {
	//	if err := pv.dashboard.Start(); err != nil {
	//		log.Printf("Warning: Failed to start performance monitoring: %v", err)
	//	}
	// }
	
	return nil
}

// handleInput processes keyboard input for the performance view
func (pv *PerformanceView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF1:
		pv.showHelp()
		return nil
	case tcell.KeyF5:
		pv.refresh()
		return nil
	case tcell.KeyCtrlS:
		pv.toggleMonitoring()
		return nil
	case tcell.KeyCtrlR:
		pv.toggleAutoRefresh()
		return nil
	case tcell.KeyCtrlP:
		pv.showProfilerReport()
		return nil
	case tcell.KeyCtrlO:
		pv.showOptimizerSettings()
		return nil
	case tcell.KeyCtrlE:
		pv.exportMetrics()
		return nil
	}
	
	switch event.Rune() {
	case 's', 'S':
		pv.toggleMonitoring()
		return nil
	case 'r', 'R':
		pv.toggleAutoRefresh()
		return nil
	case 'p', 'P':
		pv.showProfilerReport()
		return nil
	case 'o', 'O':
		pv.showOptimizerSettings()
		return nil
	case 'e', 'E':
		pv.exportMetrics()
		return nil
	case 'h', 'H':
		pv.showHelp()
		return nil
	case '+':
		pv.increaseRefreshRate()
		return nil
	case '-':
		pv.decreaseRefreshRate()
		return nil
	}
	
	// Pass through to dashboard
	return event
}

// toggleMonitoring toggles performance monitoring on/off
func (pv *PerformanceView) toggleMonitoring() {
	pv.monitoringEnabled = !pv.monitoringEnabled
	
	if pv.monitoringEnabled {
		if err := pv.dashboard.Start(); err != nil {
			log.Printf("Error starting monitoring: %v", err)
		}
	} else {
		pv.dashboard.Stop()
	}
	
	pv.updateControlBar()
}

// toggleAutoRefresh toggles automatic refresh
func (pv *PerformanceView) toggleAutoRefresh() {
	pv.autoRefresh = !pv.autoRefresh
	pv.updateControlBar()
}

// increaseRefreshRate increases the refresh rate (decreases interval)
func (pv *PerformanceView) increaseRefreshRate() {
	if pv.refreshInterval > 500*time.Millisecond {
		pv.refreshInterval -= 500 * time.Millisecond
		pv.dashboard.SetUpdateInterval(pv.refreshInterval)
		pv.updateControlBar()
	}
}

// decreaseRefreshRate decreases the refresh rate (increases interval)
func (pv *PerformanceView) decreaseRefreshRate() {
	if pv.refreshInterval < 10*time.Second {
		pv.refreshInterval += 500 * time.Millisecond
		pv.dashboard.SetUpdateInterval(pv.refreshInterval)
		pv.updateControlBar()
	}
}

// showProfilerReport displays detailed profiler information
func (pv *PerformanceView) showProfilerReport() {
	if pv.profiler == nil {
		pv.showModal("Error", "Profiler not available")
		return
	}
	
	// Get operation stats and memory info
	stats := pv.profiler.GetOperationStats()
	memStats := pv.profiler.CaptureMemoryStats()
	
	// Create detailed report content
	content := fmt.Sprintf("Performance Profiler Report\n\n")
	content += fmt.Sprintf("Report Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("Total Operations: %d\n", len(stats))
	content += fmt.Sprintf("Memory Usage: %d MB / %d MB\n", 
		memStats.HeapInuse/1024/1024, 
		memStats.Sys/1024/1024)
	
	if len(stats) > 0 {
		content += "\nOperation Statistics:\n"
		for name, stat := range stats {
			content += fmt.Sprintf("• %s: %d ops, avg %v, total %v\n", 
				name, stat.Count, stat.AverageTime, stat.TotalTime)
		}
	}
	
	pv.showModal("Profiler Report", content)
}

// showOptimizerSettings displays optimizer configuration
func (pv *PerformanceView) showOptimizerSettings() {
	if pv.optimizer == nil {
		pv.showModal("Error", "Optimizer not available")
		return
	}
	
	content := "Performance Optimizer Settings\n\n"
	content += "Available Optimizations:\n"
	content += "• Memory optimization (Ctrl+M)\n"
	content += "• Performance tuning (Ctrl+P)\n"
	content += "• Garbage collection tuning\n"
	content += "• Connection pooling\n\n"
	content += "Auto-optimization: " 
	if pv.dashboard.IsRunning() {
		content += "Enabled\n"
	} else {
		content += "Disabled\n"
	}
	
	content += "\nPress O to toggle auto-optimization"
	
	pv.showModal("Optimizer Settings", content)
}

// exportMetrics shows the export dialog for performance metrics
func (pv *PerformanceView) exportMetrics() {
	if pv.profiler == nil {
		pv.showModal("Error", "No profiler data to export")
		return
	}
	
	// Create export dialog
	exportDialog := widgets.NewPerformanceExportDialog(pv.profiler, pv.optimizer)
	
	// Set export handler
	exportDialog.SetExportHandler(func(format export.ExportFormat, path string) {
		// Close dialog
		if pv.pages != nil {
			pv.pages.RemovePage("export-dialog")
		}
		
		// Show success message
		pv.app.QueueUpdateDraw(func() {
			pv.controlBar.SetText("[green]Performance report exported successfully[white]")
		})
		
		// Restore normal control bar after a few seconds
		go func() {
			time.Sleep(3 * time.Second)
			pv.app.QueueUpdateDraw(func() {
				pv.updateControlBar()
			})
		}()
	})
	
	// Set cancel handler
	exportDialog.SetCancelHandler(func() {
		if pv.pages != nil {
			pv.pages.RemovePage("export-dialog")
		}
	})
	
	// Show dialog
	if pv.pages != nil {
		pv.pages.AddPage("export-dialog", exportDialog, true, true)
	}
}

// showHelp displays help information
func (pv *PerformanceView) showHelp() {
	content := "Performance Monitor Help\n\n"
	content += "MONITORING CONTROLS:\n"
	content += "S, Ctrl+S    - Toggle monitoring on/off\n"
	content += "R, Ctrl+R    - Toggle auto-refresh\n"
	content += "F5           - Manual refresh\n"
	content += "+/-          - Increase/decrease refresh rate\n\n"
	
	content += "PROFILER & OPTIMIZER:\n"
	content += "P, Ctrl+P    - Show profiler report\n"
	content += "O, Ctrl+O    - Show optimizer settings\n"
	content += "E, Ctrl+E    - Export metrics\n\n"
	
	content += "DASHBOARD CONTROLS:\n"
	content += "A            - Toggle alerts\n"
	content += "C            - Clear history\n"
	content += "O            - Toggle auto-optimize\n\n"
	
	content += "NAVIGATION:\n"
	content += "F1, H        - Show this help\n"
	content += "Esc, Q       - Return to main menu\n\n"
	
	content += "The dashboard shows real-time CPU, memory, network,\n"
	content += "and operation metrics with color-coded alerts."
	
	pv.showModal("Help", content)
}

// updateControlBar updates the control bar with current status
func (pv *PerformanceView) updateControlBar() {
	status := ""
	
	// Monitoring status
	if pv.monitoringEnabled {
		status += "[green]●[white] Monitoring: ON  "
	} else {
		status += "[red]●[white] Monitoring: OFF  "
	}
	
	// Auto-refresh status
	if pv.autoRefresh {
		status += "[green]●[white] Auto-refresh: ON  "
	} else {
		status += "[gray]●[white] Auto-refresh: OFF  "
	}
	
	// Refresh interval
	status += fmt.Sprintf("Interval: %v  ", pv.refreshInterval)
	
	// Controls hint
	status += "\n[gray]S:Start/Stop R:Refresh +:Faster -:Slower P:Profiler O:Optimizer E:Export H:Help[white]"
	
	pv.controlBar.SetText(status)
}

// showModal displays a modal dialog
func (pv *PerformanceView) showModal(title, content string) {
	modal := tview.NewModal()
	modal.SetText(content)
	modal.SetTitle(title)
	modal.AddButtons([]string{"Close"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if pv.pages != nil {
			pv.pages.RemovePage("modal")
		}
	})
	
	if pv.pages != nil {
		pv.pages.AddPage("modal", modal, true, true)
	}
}

// BaseView interface implementation

// Render returns the main container
func (pv *PerformanceView) Render() tview.Primitive {
	return pv.container
}

// OnFocus is called when the view gains focus
func (pv *PerformanceView) OnFocus() error {
	// Start monitoring when the view becomes active
	if pv.monitoringEnabled && pv.dashboard != nil {
		if err := pv.dashboard.Start(); err != nil {
			log.Printf("Warning: Failed to start performance monitoring: %v", err)
		}
	}
	return nil
}

// OnLoseFocus is called when the view loses focus
func (pv *PerformanceView) OnLoseFocus() error {
	// Stop monitoring when view loses focus to prevent rendering conflicts
	if pv.dashboard != nil {
		pv.dashboard.Stop()
	}
	return nil
}

// Update refreshes the view data
func (pv *PerformanceView) Update() error {
	if pv.autoRefresh {
		pv.dashboard.Start()
	}
	return nil
}

// Refresh manually refreshes the view
func (pv *PerformanceView) Refresh() error {
	pv.dashboard.Start()
	return nil
}

// refresh is the internal refresh method
func (pv *PerformanceView) refresh() {
	pv.Refresh()
}

// OnKey handles key events
func (pv *PerformanceView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	return pv.handleInput(event)
}

// Hints returns keyboard shortcuts
func (pv *PerformanceView) Hints() []string {
	return []string{
		"S: Toggle monitoring",
		"R: Toggle auto-refresh",
		"F5: Refresh",
		"P: Profiler report",
		"O: Optimizer settings",
		"E: Export metrics",
		"H: Help",
		"+/-: Adjust refresh rate",
	}
}

// Stop stops the performance monitoring
func (pv *PerformanceView) Stop() error {
	if pv.dashboard != nil {
		pv.dashboard.Stop()
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