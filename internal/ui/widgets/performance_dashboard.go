package widgets

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/performance"
	"github.com/rivo/tview"
)

// PerformanceDashboard provides a real-time performance monitoring widget
type PerformanceDashboard struct {
	mu        sync.RWMutex
	container *tview.Flex

	// Performance components
	profiler  *performance.Profiler
	optimizer *performance.Optimizer

	// UI Widgets
	cpuChart     *tview.TextView
	memoryChart  *tview.TextView
	networkChart *tview.TextView
	opsChart     *tview.TextView
	metricsTable *tview.Table
	alertsPanel  *tview.TextView

	// Data
	cpuHistory     []float64
	memoryHistory  []float64
	networkHistory []float64
	opsHistory     []float64
	maxHistory     int

	// State
	running        bool
	updateInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc

	// Configuration
	showAlerts   bool
	autoOptimize bool
	thresholds   PerformanceThresholds
}

// PerformanceThresholds defines alerting thresholds
type PerformanceThresholds struct {
	CPUWarning      float64
	CPUCritical     float64
	MemoryWarning   float64
	MemoryCritical  float64
	NetworkWarning  float64
	NetworkCritical float64
	OpsWarning      float64
	OpsCritical     float64
}

// NewPerformanceDashboard creates a new performance dashboard
func NewPerformanceDashboard(profiler *performance.Profiler, optimizer *performance.Optimizer) *PerformanceDashboard {
	ctx, cancel := context.WithCancel(context.Background())

	pd := &PerformanceDashboard{
		profiler:       profiler,
		optimizer:      optimizer,
		maxHistory:     50,
		updateInterval: 1 * time.Second,
		ctx:            ctx,
		cancel:         cancel,
		showAlerts:     true,
		autoOptimize:   true,
		thresholds: PerformanceThresholds{
			CPUWarning:      70.0,
			CPUCritical:     90.0,
			MemoryWarning:   80.0,
			MemoryCritical:  95.0,
			NetworkWarning:  1000.0, // MB/s
			NetworkCritical: 2000.0,
			OpsWarning:      1000.0, // ops/sec
			OpsCritical:     5000.0,
		},
	}

	pd.initializeHistory()
	pd.initializeUI()

	return pd
}

// initializeHistory sets up data history arrays
func (pd *PerformanceDashboard) initializeHistory() {
	pd.cpuHistory = make([]float64, 0, pd.maxHistory)
	pd.memoryHistory = make([]float64, 0, pd.maxHistory)
	pd.networkHistory = make([]float64, 0, pd.maxHistory)
	pd.opsHistory = make([]float64, 0, pd.maxHistory)
}

// initializeUI sets up the dashboard UI components
func (pd *PerformanceDashboard) initializeUI() {
	// Create main container
	pd.container = tview.NewFlex()
	pd.container.SetDirection(tview.FlexRow)
	pd.container.SetBorder(true)
	pd.container.SetTitle(" 📊 Performance Dashboard ")
	pd.container.SetTitleAlign(tview.AlignCenter)

	// Create top row with charts
	chartsRow := tview.NewFlex()
	chartsRow.SetDirection(tview.FlexColumn)

	// CPU Chart
	pd.cpuChart = tview.NewTextView()
	pd.cpuChart.SetBorder(true)
	pd.cpuChart.SetTitle(" CPU Usage % ")
	pd.cpuChart.SetDynamicColors(true)
	pd.cpuChart.SetWrap(false)

	// Memory Chart
	pd.memoryChart = tview.NewTextView()
	pd.memoryChart.SetBorder(true)
	pd.memoryChart.SetTitle(" Memory Usage % ")
	pd.memoryChart.SetDynamicColors(true)
	pd.memoryChart.SetWrap(false)

	// Network Chart
	pd.networkChart = tview.NewTextView()
	pd.networkChart.SetBorder(true)
	pd.networkChart.SetTitle(" Network MB/s ")
	pd.networkChart.SetDynamicColors(true)
	pd.networkChart.SetWrap(false)

	// Operations Chart
	pd.opsChart = tview.NewTextView()
	pd.opsChart.SetBorder(true)
	pd.opsChart.SetTitle(" Operations/sec ")
	pd.opsChart.SetDynamicColors(true)
	pd.opsChart.SetWrap(false)

	// Add charts to top row
	chartsRow.AddItem(pd.cpuChart, 0, 1, false)
	chartsRow.AddItem(pd.memoryChart, 0, 1, false)
	chartsRow.AddItem(pd.networkChart, 0, 1, false)
	chartsRow.AddItem(pd.opsChart, 0, 1, false)

	// Create bottom row with metrics and alerts
	bottomRow := tview.NewFlex()
	bottomRow.SetDirection(tview.FlexColumn)

	// Metrics Table
	pd.metricsTable = tview.NewTable()
	pd.metricsTable.SetBorder(true)
	pd.metricsTable.SetTitle(" 📈 Detailed Metrics ")
	pd.metricsTable.SetSelectable(true, false)
	pd.setupMetricsTable()

	// Alerts Panel
	pd.alertsPanel = tview.NewTextView()
	pd.alertsPanel.SetBorder(true)
	pd.alertsPanel.SetTitle(" 🚨 Alerts & Recommendations ")
	pd.alertsPanel.SetDynamicColors(true)
	pd.alertsPanel.SetScrollable(true)
	pd.alertsPanel.SetWrap(true)

	// Add to bottom row
	bottomRow.AddItem(pd.metricsTable, 0, 2, false)
	bottomRow.AddItem(pd.alertsPanel, 0, 1, false)

	// Add rows to main container
	pd.container.AddItem(chartsRow, 0, 2, false)
	pd.container.AddItem(bottomRow, 0, 1, false)

	// Set up input handling
	pd.container.SetInputCapture(pd.handleInput)
}

// setupMetricsTable initializes the metrics table headers
func (pd *PerformanceDashboard) setupMetricsTable() {
	headers := []string{"Metric", "Current", "Average", "Peak", "Status"}
	for i, header := range headers {
		cell := tview.NewTableCell(header)
		cell.SetTextColor(tcell.ColorYellow)
		cell.SetAlign(tview.AlignCenter)
		cell.SetSelectable(false)
		pd.metricsTable.SetCell(0, i, cell)
	}
}

// handleInput processes keyboard input for the dashboard
func (pd *PerformanceDashboard) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF5:
		pd.refresh()
		return nil
	case tcell.KeyCtrlR:
		pd.reset()
		return nil
	case tcell.KeyCtrlO:
		pd.toggleAutoOptimize()
		return nil
	case tcell.KeyCtrlA:
		pd.toggleAlerts()
		return nil
	}

	switch event.Rune() {
	case 'r', 'R':
		pd.refresh()
		return nil
	case 'o', 'O':
		pd.toggleAutoOptimize()
		return nil
	case 'a', 'A':
		pd.toggleAlerts()
		return nil
	case 'c', 'C':
		pd.clearHistory()
		return nil
	}

	return event
}

// Start begins real-time monitoring
func (pd *PerformanceDashboard) Start() error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if pd.running {
		return fmt.Errorf("dashboard already running")
	}

	pd.running = true
	go pd.updateLoop()

	return nil
}

// Stop stops real-time monitoring
func (pd *PerformanceDashboard) Stop() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if !pd.running {
		return
	}

	pd.running = false
	pd.cancel()
}

// updateLoop runs the main update loop
func (pd *PerformanceDashboard) updateLoop() {
	pd.mu.RLock()
	interval := pd.updateInterval
	pd.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-pd.ctx.Done():
			return
		case <-ticker.C:
			pd.updateMetrics()
		}
	}
}

// updateMetrics collects and updates all performance metrics
func (pd *PerformanceDashboard) updateMetrics() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if pd.profiler == nil {
		return
	}

	// Get current metrics
	stats := pd.profiler.GetOperationStats()
	memStats := pd.profiler.CaptureMemoryStats()

	// Update CPU metrics
	cpuUsage := pd.calculateCPUUsage(stats)
	pd.addToHistory(&pd.cpuHistory, cpuUsage)
	pd.updateCPUChart()

	// Update Memory metrics
	memUsage := pd.calculateMemoryUsage(memStats)
	pd.addToHistory(&pd.memoryHistory, memUsage)
	pd.updateMemoryChart()

	// Update Network metrics
	netUsage := pd.calculateNetworkUsage(stats)
	pd.addToHistory(&pd.networkHistory, netUsage)
	pd.updateNetworkChart()

	// Update Operations metrics
	opsRate := pd.calculateOpsRate(stats)
	pd.addToHistory(&pd.opsHistory, opsRate)
	pd.updateOpsChart()

	// Update detailed metrics table
	pd.updateMetricsTable(cpuUsage, memUsage, netUsage, opsRate)

	// Check for alerts and recommendations
	if pd.showAlerts {
		pd.updateAlerts(cpuUsage, memUsage, netUsage, opsRate)
	}

	// Auto-optimize if enabled
	if pd.autoOptimize && pd.optimizer != nil {
		pd.performAutoOptimization(cpuUsage, memUsage, netUsage, opsRate)
	}
}

// addToHistory adds a value to a history slice, maintaining max size
func (pd *PerformanceDashboard) addToHistory(history *[]float64, value float64) {
	*history = append(*history, value)
	if len(*history) > pd.maxHistory {
		*history = (*history)[1:]
	}
}

// calculateCPUUsage extracts CPU usage from operation stats
func (pd *PerformanceDashboard) calculateCPUUsage(stats map[string]performance.OperationSummary) float64 {
	if len(stats) == 0 {
		return 0.0
	}

	// Calculate CPU usage based on operation timings
	totalTime := time.Duration(0)
	for _, op := range stats {
		totalTime += op.AverageTime * time.Duration(op.Count)
	}

	// Estimate CPU usage percentage
	windowDuration := pd.updateInterval
	if windowDuration == 0 {
		windowDuration = 1 * time.Second
	}

	cpuUsage := float64(totalTime.Nanoseconds()) / float64(windowDuration.Nanoseconds()) * 100.0
	if cpuUsage > 100.0 {
		cpuUsage = 100.0
	}

	return cpuUsage
}

// calculateMemoryUsage extracts memory usage from memory stats
func (pd *PerformanceDashboard) calculateMemoryUsage(memStats runtime.MemStats) float64 {
	// Calculate memory usage percentage based on heap
	if memStats.Sys > 0 {
		return float64(memStats.HeapInuse) / float64(memStats.Sys) * 100.0
	}

	return 0.0
}

// calculateNetworkUsage estimates network usage from operation stats
func (pd *PerformanceDashboard) calculateNetworkUsage(stats map[string]performance.OperationSummary) float64 {
	if stats == nil {
		return 0.0
	}

	// Estimate network usage based on operations
	networkOps := int64(0)
	for name, op := range stats {
		if strings.Contains(strings.ToLower(name), "ssh") ||
			strings.Contains(strings.ToLower(name), "api") ||
			strings.Contains(strings.ToLower(name), "network") {
			networkOps += op.Count
		}
	}

	// Convert to MB/s estimate
	return float64(networkOps) * 0.1 // Rough estimate
}

// calculateOpsRate calculates operations per second
func (pd *PerformanceDashboard) calculateOpsRate(stats map[string]performance.OperationSummary) float64 {
	if stats == nil {
		return 0.0
	}

	// Calculate ops/sec based on total operations
	totalOps := int64(0)
	for _, op := range stats {
		totalOps += op.Count
	}

	windowSeconds := pd.updateInterval.Seconds()
	if windowSeconds == 0 {
		windowSeconds = 1.0
	}

	return float64(totalOps) / windowSeconds
}

// updateCPUChart updates the CPU usage chart
func (pd *PerformanceDashboard) updateCPUChart() {
	chart := pd.generateAsciiChart(pd.cpuHistory, "CPU", "%", pd.thresholds.CPUWarning, pd.thresholds.CPUCritical)
	pd.cpuChart.SetText(chart)
}

// updateMemoryChart updates the memory usage chart
func (pd *PerformanceDashboard) updateMemoryChart() {
	chart := pd.generateAsciiChart(pd.memoryHistory, "Memory", "%", pd.thresholds.MemoryWarning, pd.thresholds.MemoryCritical)
	pd.memoryChart.SetText(chart)
}

// updateNetworkChart updates the network usage chart
func (pd *PerformanceDashboard) updateNetworkChart() {
	chart := pd.generateAsciiChart(pd.networkHistory, "Network", "MB/s", pd.thresholds.NetworkWarning, pd.thresholds.NetworkCritical)
	pd.networkChart.SetText(chart)
}

// updateOpsChart updates the operations rate chart
func (pd *PerformanceDashboard) updateOpsChart() {
	chart := pd.generateAsciiChart(pd.opsHistory, "Ops", "/sec", pd.thresholds.OpsWarning, pd.thresholds.OpsCritical)
	pd.opsChart.SetText(chart)
}

// generateAsciiChart creates a simple ASCII chart from data
func (pd *PerformanceDashboard) generateAsciiChart(data []float64, name, unit string, warningThreshold, criticalThreshold float64) string {
	if len(data) == 0 {
		return fmt.Sprintf("[gray]No %s data[white]", name)
	}

	// Get current and summary values
	current := data[len(data)-1]
	avg := pd.calculateAverage(data)
	max := pd.calculateMax(data)

	// Determine color based on thresholds
	color := "green"
	if current >= criticalThreshold {
		color = "red"
	} else if current >= warningThreshold {
		color = "yellow"
	}

	// Create simple bar chart
	chart := fmt.Sprintf("[%s]Current: %.1f%s[white]\n", color, current, unit)
	chart += fmt.Sprintf("Average: %.1f%s\n", avg, unit)
	chart += fmt.Sprintf("Peak: %.1f%s\n\n", max, unit)

	// Add simple trend line
	trend := pd.generateTrendLine(data, 20)
	chart += trend

	return chart
}

// generateTrendLine creates a simple trend visualization
func (pd *PerformanceDashboard) generateTrendLine(data []float64, width int) string {
	if len(data) < 2 {
		return "[gray]Insufficient data[white]"
	}

	max := pd.calculateMax(data)
	if max == 0 {
		max = 1
	}

	line := ""
	step := len(data) / width
	if step < 1 {
		step = 1
	}

	for i := 0; i < len(data); i += step {
		value := data[i]
		height := int((value / max) * 8)

		switch height {
		case 0:
			line += "_"
		case 1:
			line += "▁"
		case 2:
			line += "▂"
		case 3:
			line += "▃"
		case 4:
			line += "▄"
		case 5:
			line += "▅"
		case 6:
			line += "▆"
		case 7:
			line += "▇"
		default:
			line += "█"
		}

		if len(line) >= width {
			break
		}
	}

	return line
}

// calculateAverage calculates the average of a slice
func (pd *PerformanceDashboard) calculateAverage(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}

	return sum / float64(len(data))
}

// calculateMax finds the maximum value in a slice
func (pd *PerformanceDashboard) calculateMax(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	max := data[0]
	for _, v := range data {
		if v > max {
			max = v
		}
	}

	return max
}

// updateMetricsTable updates the detailed metrics table
func (pd *PerformanceDashboard) updateMetricsTable(cpuUsage, memUsage, netUsage, opsRate float64) {
	metrics := []struct {
		name     string
		current  float64
		history  []float64
		unit     string
		warning  float64
		critical float64
	}{
		{"CPU", cpuUsage, pd.cpuHistory, "%", pd.thresholds.CPUWarning, pd.thresholds.CPUCritical},
		{"Memory", memUsage, pd.memoryHistory, "%", pd.thresholds.MemoryWarning, pd.thresholds.MemoryCritical},
		{"Network", netUsage, pd.networkHistory, "MB/s", pd.thresholds.NetworkWarning, pd.thresholds.NetworkCritical},
		{"Operations", opsRate, pd.opsHistory, "/sec", pd.thresholds.OpsWarning, pd.thresholds.OpsCritical},
	}

	for i, metric := range metrics {
		row := i + 1

		// Metric name
		pd.metricsTable.SetCell(row, 0, tview.NewTableCell(metric.name))

		// Current value
		color := tcell.ColorGreen
		if metric.current >= metric.critical {
			color = tcell.ColorRed
		} else if metric.current >= metric.warning {
			color = tcell.ColorYellow
		}

		currentCell := tview.NewTableCell(fmt.Sprintf("%.1f%s", metric.current, metric.unit))
		currentCell.SetTextColor(color)
		pd.metricsTable.SetCell(row, 1, currentCell)

		// Average
		avg := pd.calculateAverage(metric.history)
		pd.metricsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.1f%s", avg, metric.unit)))

		// Peak
		peak := pd.calculateMax(metric.history)
		pd.metricsTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%.1f%s", peak, metric.unit)))

		// Status
		status := "OK"
		statusColor := tcell.ColorGreen
		if metric.current >= metric.critical {
			status = "CRITICAL"
			statusColor = tcell.ColorRed
		} else if metric.current >= metric.warning {
			status = "WARNING"
			statusColor = tcell.ColorYellow
		}

		statusCell := tview.NewTableCell(status)
		statusCell.SetTextColor(statusColor)
		pd.metricsTable.SetCell(row, 4, statusCell)
	}
}

// updateAlerts checks thresholds and updates the alerts panel
func (pd *PerformanceDashboard) updateAlerts(cpuUsage, memUsage, netUsage, opsRate float64) {
	alerts := []string{}
	recommendations := []string{}

	// Check CPU alerts
	if cpuUsage >= pd.thresholds.CPUCritical {
		alerts = append(alerts, fmt.Sprintf("[red]🚨 CRITICAL: CPU usage at %.1f%%[white]", cpuUsage))
		recommendations = append(recommendations, "[yellow]• Consider enabling CPU optimization[white]")
		recommendations = append(recommendations, "[yellow]• Reduce concurrent operations[white]")
	} else if cpuUsage >= pd.thresholds.CPUWarning {
		alerts = append(alerts, fmt.Sprintf("[yellow]⚠️  WARNING: CPU usage at %.1f%%[white]", cpuUsage))
	}

	// Check Memory alerts
	if memUsage >= pd.thresholds.MemoryCritical {
		alerts = append(alerts, fmt.Sprintf("[red]🚨 CRITICAL: Memory usage at %.1f%%[white]", memUsage))
		recommendations = append(recommendations, "[yellow]• Enable garbage collection optimization[white]")
		recommendations = append(recommendations, "[yellow]• Clear unnecessary caches[white]")
	} else if memUsage >= pd.thresholds.MemoryWarning {
		alerts = append(alerts, fmt.Sprintf("[yellow]⚠️  WARNING: Memory usage at %.1f%%[white]", memUsage))
	}

	// Check Network alerts
	if netUsage >= pd.thresholds.NetworkCritical {
		alerts = append(alerts, fmt.Sprintf("[red]🚨 CRITICAL: Network usage at %.1f MB/s[white]", netUsage))
		recommendations = append(recommendations, "[yellow]• Consider connection pooling[white]")
	} else if netUsage >= pd.thresholds.NetworkWarning {
		alerts = append(alerts, fmt.Sprintf("[yellow]⚠️  WARNING: High network usage %.1f MB/s[white]", netUsage))
	}

	// Check Operations alerts
	if opsRate >= pd.thresholds.OpsCritical {
		alerts = append(alerts, fmt.Sprintf("[red]🚨 CRITICAL: High operation rate %.1f/sec[white]", opsRate))
		recommendations = append(recommendations, "[yellow]• Enable operation batching[white]")
	} else if opsRate >= pd.thresholds.OpsWarning {
		alerts = append(alerts, fmt.Sprintf("[yellow]⚠️  WARNING: High operation rate %.1f/sec[white]", opsRate))
	}

	// Build alerts text
	var alertText string
	if len(alerts) == 0 {
		alertText = "[green]✅ All systems operating normally[white]\n\n"
	} else {
		alertText = "🚨 ACTIVE ALERTS:\n"
		for _, alert := range alerts {
			alertText += alert + "\n"
		}
		alertText += "\n"
	}

	// Add recommendations
	if len(recommendations) > 0 {
		alertText += "💡 RECOMMENDATIONS:\n"
		for _, rec := range recommendations {
			alertText += rec + "\n"
		}
		alertText += "\n"
	}

	// Add controls help
	alertText += "[gray]CONTROLS:[white]\n"
	alertText += "[gray]F5/R: Refresh • Ctrl+O/O: Toggle auto-optimize[white]\n"
	alertText += "[gray]Ctrl+A/A: Toggle alerts • C: Clear history[white]"

	pd.alertsPanel.SetText(alertText)
}

// performAutoOptimization applies automatic optimizations based on metrics
func (pd *PerformanceDashboard) performAutoOptimization(cpuUsage, memUsage, netUsage, opsRate float64) {
	if pd.optimizer == nil {
		return
	}

	// Apply optimizations based on current metrics
	if cpuUsage >= pd.thresholds.CPUWarning {
		pd.optimizer.TuneForInteractive()
		log.Printf("Auto-optimization: Applied interactive tuning due to high CPU usage")
	}

	if memUsage >= pd.thresholds.MemoryWarning {
		pd.optimizer.EnableAutoTune(true)
		log.Printf("Auto-optimization: Enabled auto-tuning due to high memory usage")
	}

	if opsRate >= pd.thresholds.OpsWarning {
		pd.optimizer.TuneForBatchOperations()
		log.Printf("Auto-optimization: Applied batch tuning due to high operation rate")
	}
}

// Helper methods for dashboard control

// refresh manually refreshes the dashboard
func (pd *PerformanceDashboard) refresh() {
	go pd.updateMetrics()
}

// reset clears all data and resets the dashboard
func (pd *PerformanceDashboard) reset() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.initializeHistory()
	pd.cpuChart.Clear()
	pd.memoryChart.Clear()
	pd.networkChart.Clear()
	pd.opsChart.Clear()
	pd.setupMetricsTable()
	pd.alertsPanel.Clear()
}

// toggleAutoOptimize toggles automatic optimization
func (pd *PerformanceDashboard) toggleAutoOptimize() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.autoOptimize = !pd.autoOptimize
	status := "disabled"
	if pd.autoOptimize {
		status = "enabled"
	}
	log.Printf("Auto-optimization %s", status)
}

// toggleAlerts toggles alert monitoring
func (pd *PerformanceDashboard) toggleAlerts() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.showAlerts = !pd.showAlerts
	if !pd.showAlerts {
		pd.alertsPanel.SetText("[gray]Alerts disabled[white]")
	}
}

// clearHistory clears all performance history
func (pd *PerformanceDashboard) clearHistory() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.initializeHistory()
}

// GetContainer returns the main dashboard container
func (pd *PerformanceDashboard) GetContainer() tview.Primitive {
	return pd.container
}

// SetUpdateInterval sets the dashboard update frequency
func (pd *PerformanceDashboard) SetUpdateInterval(interval time.Duration) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.updateInterval = interval
}

// SetThresholds updates the alerting thresholds
func (pd *PerformanceDashboard) SetThresholds(thresholds PerformanceThresholds) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.thresholds = thresholds
}

// IsRunning returns whether the dashboard is currently running
func (pd *PerformanceDashboard) IsRunning() bool {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	return pd.running
}
