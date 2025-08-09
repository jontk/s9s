package views

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	
	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/alerts"
	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/models"
	"github.com/jontk/s9s/plugins/observability/prometheus"
	"github.com/jontk/s9s/plugins/observability/views/widgets"
)

// ObservabilityView provides a comprehensive metrics dashboard
type ObservabilityView struct {
	app              *tview.Application
	client           *prometheus.CachedClient
	config           *config.Config
	queryBuilder     *prometheus.QueryBuilder
	alertEngine      *alerts.Engine
	
	// Layout components
	root             *tview.Flex
	clusterPanel     *tview.TextView
	nodeTable        *tview.Table
	jobTable         *tview.Table
	alertsPanel      *widgets.AlertsWidget
	cpuGauge         *widgets.GaugeWidget
	memoryGauge      *widgets.GaugeWidget
	
	// Data collectors
	nodeCollector    *models.NodeMetricsCollector
	jobCollector     *models.JobMetricsCollector
	
	// Data
	nodeMetrics      map[string]*models.NodeMetrics
	jobMetrics       map[string]*models.JobMetrics
	clusterMetrics   *models.AggregateNodeMetrics
	alerts           []models.Alert
	
	// State
	refreshInterval  time.Duration
	stopChan         chan struct{}
	refreshTicker    *time.Ticker
	selectedNode     string
	selectedJob      string
	
	// Synchronization
	mu               sync.RWMutex
}

// NewObservabilityView creates a new observability view
func NewObservabilityView(app *tview.Application, client *prometheus.CachedClient, cfg interface{}) *ObservabilityView {
	// Type assert config
	obsConfig, ok := cfg.(*config.Config)
	if !ok {
		// Use default config if type assertion fails
		obsConfig = config.DefaultConfig()
	}
	
	// Create query builder
	queryBuilder, _ := prometheus.NewQueryBuilder()
	
	// Create alert engine
	alertEngine := alerts.NewEngine(&obsConfig.Alerts, client)
	
	v := &ObservabilityView{
		app:             app,
		client:          client,
		config:          obsConfig,
		queryBuilder:    queryBuilder,
		alertEngine:     alertEngine,
		nodeCollector:   models.NewNodeMetricsCollector(obsConfig.Metrics.Node.NodeLabel),
		jobCollector:    models.NewJobMetricsCollector(obsConfig.Metrics.Job.CgroupPattern),
		nodeMetrics:     make(map[string]*models.NodeMetrics),
		jobMetrics:      make(map[string]*models.JobMetrics),
		refreshInterval: obsConfig.Display.RefreshInterval,
		stopChan:        make(chan struct{}),
	}
	
	// Set up alert callbacks
	v.setupAlertCallbacks()
	
	v.initializeLayout()
	return v
}

// initializeLayout sets up the view layout
func (v *ObservabilityView) initializeLayout() {
	// Create cluster overview panel
	v.clusterPanel = tview.NewTextView().
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle(" Cluster Overview ")
	
	// Create CPU and Memory gauges
	v.cpuGauge = widgets.NewGaugeWidget("CPU Usage", 0, 100, "%")
	v.memoryGauge = widgets.NewGaugeWidget("Memory Usage", 0, 100, "%")
	
	// Create gauges container
	gaugesContainer := tview.NewFlex().
		AddItem(v.cpuGauge.GetPrimitive(), 0, 1, false).
		AddItem(v.memoryGauge.GetPrimitive(), 0, 1, false)
	
	// Create top section with cluster info and gauges
	topSection := tview.NewFlex().
		AddItem(v.clusterPanel, 0, 2, false).
		AddItem(gaugesContainer, 0, 1, false)
	
	// Create node metrics table
	v.nodeTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetBorder(true).
		SetTitle(" Node Metrics ")
	
	// Set up node table headers
	headers := []string{"Node", "State", "CPU %", "Memory %", "Load", "Jobs", "Network", "Disk I/O"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false)
		v.nodeTable.SetCell(0, i, cell)
	}
	
	// Create job metrics table
	v.jobTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetBorder(true).
		SetTitle(" Job Metrics ")
	
	// Set up job table headers
	jobHeaders := []string{"Job ID", "User", "CPU %", "Memory", "CPU Limit", "Mem Limit", "Efficiency", "Status"}
	for i, header := range jobHeaders {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false)
		v.jobTable.SetCell(0, i, cell)
	}
	
	// Create alerts panel
	v.alertsPanel = widgets.NewAlertsWidget()
	
	// Create middle section with tables
	middleSection := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.nodeTable, 0, 1, true).
		AddItem(v.jobTable, 0, 1, false)
	
	// Create main layout
	v.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topSection, 8, 0, false).
		AddItem(middleSection, 0, 3, true).
		AddItem(v.alertsPanel.GetPrimitive(), 6, 0, false)
	
	// Set up keyboard shortcuts
	v.setupKeyboardShortcuts()
}

// setupKeyboardShortcuts configures keyboard navigation
func (v *ObservabilityView) setupKeyboardShortcuts() {
	// Global shortcuts for the view
	v.root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			// Cycle through focusable elements
			v.cycleFocus(false)
			return nil
		case tcell.KeyBacktab:
			// Reverse cycle
			v.cycleFocus(true)
			return nil
		case tcell.KeyCtrlR:
			// Force refresh
			go v.refresh(context.Background())
			return nil
		case tcell.KeyEsc, tcell.KeyCtrlC:
			// Return to previous view
			return nil
		}
		
		// Handle specific shortcuts
		switch event.Rune() {
		case 'n', 'N':
			// Focus node table
			v.app.SetFocus(v.nodeTable)
			return nil
		case 'j', 'J':
			// Focus job table
			v.app.SetFocus(v.jobTable)
			return nil
		case 'a', 'A':
			// Focus alerts
			v.app.SetFocus(v.alertsPanel.GetPrimitive())
			return nil
		case 'r', 'R':
			// Refresh data
			go v.refresh(context.Background())
			return nil
		case 'h', 'H', '?':
			// Show help
			v.showHelp()
			return nil
		}
		
		return event
	})
	
	// Node table selection handler
	v.nodeTable.SetSelectionChangedFunc(func(row, col int) {
		if row > 0 && row <= v.nodeTable.GetRowCount()-1 {
			cell := v.nodeTable.GetCell(row, 0)
			if cell != nil {
				v.selectedNode = cell.Text
			}
		}
	})
	
	// Job table selection handler
	v.jobTable.SetSelectionChangedFunc(func(row, col int) {
		if row > 0 && row <= v.jobTable.GetRowCount()-1 {
			cell := v.jobTable.GetCell(row, 0)
			if cell != nil {
				v.selectedJob = cell.Text
			}
		}
	})
}

// cycleFocus cycles through focusable elements
func (v *ObservabilityView) cycleFocus(reverse bool) {
	elements := []tview.Primitive{
		v.nodeTable,
		v.jobTable,
		v.alertsPanel.GetPrimitive(),
	}
	
	current := v.app.GetFocus()
	currentIndex := -1
	
	for i, elem := range elements {
		if elem == current {
			currentIndex = i
			break
		}
	}
	
	var nextIndex int
	if reverse {
		if currentIndex <= 0 {
			nextIndex = len(elements) - 1
		} else {
			nextIndex = currentIndex - 1
		}
	} else {
		nextIndex = (currentIndex + 1) % len(elements)
	}
	
	v.app.SetFocus(elements[nextIndex])
}

// GetPrimitive returns the root primitive for the view
func (v *ObservabilityView) GetPrimitive() tview.Primitive {
	return v.root
}

// Start begins the view refresh loop and alert engine
func (v *ObservabilityView) Start(ctx context.Context) error {
	// Start alert engine first
	if err := v.alertEngine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start alert engine: %w", err)
	}
	
	// Initial refresh
	if err := v.refresh(ctx); err != nil {
		// Don't fail if initial refresh fails, just log and continue
		v.showError(fmt.Sprintf("Initial refresh failed: %v", err))
	}
	
	// Start refresh ticker
	v.refreshTicker = time.NewTicker(v.refreshInterval)
	
	// Start refresh goroutine
	go func() {
		for {
			select {
			case <-v.refreshTicker.C:
				if err := v.refresh(ctx); err != nil {
					// Log error but continue
					v.showError(fmt.Sprintf("Refresh error: %v", err))
				}
			case <-v.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	
	return nil
}

// Stop stops the view refresh loop and alert engine
func (v *ObservabilityView) Stop(ctx context.Context) error {
	close(v.stopChan)
	if v.refreshTicker != nil {
		v.refreshTicker.Stop()
	}
	
	// Stop alert engine
	if err := v.alertEngine.Stop(); err != nil {
		// Log error but don't fail the stop operation
		v.showError(fmt.Sprintf("Error stopping alert engine: %v", err))
	}
	
	return nil
}

// refresh updates all metrics
func (v *ObservabilityView) refresh(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	// Fetch metrics from Prometheus
	if err := v.fetchNodeMetrics(ctx); err != nil {
		// Log error but continue with other metrics
		v.showError(fmt.Sprintf("Failed to fetch node metrics: %v", err))
	}
	
	if err := v.fetchJobMetrics(ctx); err != nil {
		// Log error but continue
		v.showError(fmt.Sprintf("Failed to fetch job metrics: %v", err))
	}
	
	// Update cluster metrics from aggregated node data
	v.updateClusterMetrics()
	
	// Update alert engine collectors with latest data
	v.alertEngine.SetNodeCollector(v.nodeCollector)
	v.alertEngine.SetJobCollector(v.jobCollector)
	
	// Update alerts
	v.updateAlerts()
	
	// Refresh UI
	v.app.QueueUpdateDraw(func() {
		v.renderClusterPanel()
		v.renderNodeTable()
		v.renderJobTable()
		v.renderAlerts()
	})
	
	return nil
}

// fetchNodeMetrics fetches node metrics from Prometheus
func (v *ObservabilityView) fetchNodeMetrics(ctx context.Context) error {
	// Get list of nodes - in a real implementation, this would come from SLURM
	// For now, we'll query Prometheus for all nodes
	nodes := v.getNodeList(ctx)
	
	for _, nodeName := range nodes {
		// Build queries for this node
		queries, err := v.queryBuilder.GetNodeQueries(nodeName, v.config.Metrics.Node.NodeLabel)
		if err != nil {
			continue
		}
		
		// Execute queries in batch
		results, err := v.client.BatchQuery(ctx, queries, time.Now())
		if err != nil {
			continue
		}
		
		// Convert results to TimeSeries
		metrics := make(map[string]*models.TimeSeries)
		for queryName, result := range results {
			if result.Data.Result != nil && len(result.Data.Result) > 0 {
				// Convert first result to TimeSeries
				ts := v.convertToTimeSeries(queryName, result)
				if ts != nil {
					metrics[queryName] = ts
				}
			}
		}
		
		// Update node collector
		v.nodeCollector.UpdateFromPrometheus(nodeName, metrics)
	}
	
	// Update local cache
	v.nodeMetrics = v.nodeCollector.GetAllNodes()
	
	return nil
}

// fetchJobMetrics fetches job metrics from Prometheus
func (v *ObservabilityView) fetchJobMetrics(ctx context.Context) error {
	// Get list of running jobs - in a real implementation, this would come from SLURM
	// For now, we'll use a placeholder list
	jobs := v.getJobList(ctx)
	
	for _, jobID := range jobs {
		// Build queries for this job
		queries, err := v.queryBuilder.GetJobQueries(jobID)
		if err != nil {
			continue
		}
		
		// Execute queries in batch
		results, err := v.client.BatchQuery(ctx, queries, time.Now())
		if err != nil {
			continue
		}
		
		// Convert results to TimeSeries
		metrics := make(map[string]*models.TimeSeries)
		for queryName, result := range results {
			if result.Data.Result != nil && len(result.Data.Result) > 0 {
				ts := v.convertToTimeSeries(queryName, result)
				if ts != nil {
					metrics[queryName] = ts
				}
			}
		}
		
		// Update job collector
		v.jobCollector.UpdateFromPrometheus(jobID, metrics)
	}
	
	// Update local cache
	v.jobMetrics = v.jobCollector.GetAllJobs()
	
	return nil
}

// convertToTimeSeries converts Prometheus query result to TimeSeries
func (v *ObservabilityView) convertToTimeSeries(name string, result *prometheus.QueryResult) *models.TimeSeries {
	if result == nil || result.Data.Result == nil || len(result.Data.Result) == 0 {
		return nil
	}
	
	// Get first result (we're assuming single-value queries for now)
	firstResult := result.Data.Result[0]
	
	ts := &models.TimeSeries{
		Name:   name,
		Labels: firstResult.Metric,
		Values: []models.MetricValue{},
		Type:   models.MetricTypeCustom,
	}
	
	// Add the current value
	if len(firstResult.Value) >= 2 {
		// Convert timestamp and value
		timestamp := int64(firstResult.Value[0].(float64))
		valueStr, ok := firstResult.Value[1].(string)
		if ok {
			if value, err := parseFloat(valueStr); err == nil {
				ts.Values = append(ts.Values, models.MetricValue{
					Timestamp: time.Unix(timestamp, 0),
					Value:     value,
					Labels:    firstResult.Metric,
				})
			}
		}
	}
	
	return ts
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// getNodeList returns list of nodes to query
func (v *ObservabilityView) getNodeList(ctx context.Context) []string {
	// In a real implementation, this would query SLURM for node list
	// For now, we'll try to discover nodes from Prometheus
	
	// Query for all nodes with node_exporter up
	query := `up{job="node-exporter"}`
	result, err := v.client.Query(ctx, query, time.Now())
	if err != nil || result.Data.Result == nil {
		// Return empty list or default nodes
		return []string{"node001", "node002", "node003", "node004"}
	}
	
	nodes := []string{}
	for _, r := range result.Data.Result {
		if instance, ok := r.Metric[v.config.Metrics.Node.NodeLabel]; ok {
			nodes = append(nodes, instance)
		}
	}
	
	return nodes
}

// getJobList returns list of jobs to query
func (v *ObservabilityView) getJobList(ctx context.Context) []string {
	// In a real implementation, this would query SLURM for running jobs
	// For now, return a placeholder list
	return []string{"12345", "12346", "12347"}
}

// updateClusterMetrics updates cluster-wide metrics from node data
func (v *ObservabilityView) updateClusterMetrics() {
	// Get aggregate metrics from node collector
	v.clusterMetrics = v.nodeCollector.GetAggregateMetrics()
	
	// Update gauges
	if v.clusterMetrics != nil {
		v.cpuGauge.SetValue(v.clusterMetrics.AverageCPUUsage)
		v.memoryGauge.SetValue(v.clusterMetrics.MemoryUsagePercent)
	}
}


// setupAlertCallbacks configures alert engine callbacks
func (v *ObservabilityView) setupAlertCallbacks() {
	// Set up alert callback - when new alert fires
	v.alertEngine.SetAlertCallback(func(alert alerts.Alert) {
		v.app.QueueUpdateDraw(func() {
			// Convert engine alert to model alert
			modelAlert := v.convertToModelAlert(alert)
			
			// Update alerts list
			v.mu.Lock()
			v.alerts = append([]models.Alert{modelAlert}, v.alerts...)
			// Limit to most recent 50 alerts
			if len(v.alerts) > 50 {
				v.alerts = v.alerts[:50]
			}
			v.mu.Unlock()
			
			// Update alerts panel
			v.alertsPanel.SetAlerts(v.alerts)
			
			// Show notification if configured
			if v.config.Alerts.ShowNotifications {
				v.showNotification(fmt.Sprintf("[%s] %s", alert.Severity, alert.Message))
			}
		})
	})
	
	// Set up resolved callback - when alert resolves
	v.alertEngine.SetResolvedCallback(func(alert alerts.Alert) {
		v.app.QueueUpdateDraw(func() {
			// Update alert status in the list
			v.mu.Lock()
			for i := range v.alerts {
				if v.alerts[i].Name == alert.RuleName && v.alerts[i].Source == alert.Source {
					v.alerts[i].State = "resolved"
					v.alerts[i].ResolvedAt = alert.ResolvedAt
					break
				}
			}
			v.mu.Unlock()
			
			// Update alerts panel
			v.alertsPanel.SetAlerts(v.alerts)
		})
	})
}

// convertToModelAlert converts an engine alert to a model alert
func (v *ObservabilityView) convertToModelAlert(alert alerts.Alert) models.Alert {
	return models.Alert{
		Name:       alert.RuleName,
		Severity:   alert.Severity,
		State:      string(alert.State),
		Message:    alert.Message,
		Source:     alert.Source,
		Value:      alert.Value,
		Threshold:  alert.Threshold,
		Timestamp:  alert.FirstSeen,
		ResolvedAt: alert.ResolvedAt,
		Labels:     alert.Labels,
	}
}

// updateAlerts fetches and updates active alerts from the alert engine
func (v *ObservabilityView) updateAlerts() {
	// Get active alerts from engine
	activeAlerts := v.alertEngine.GetActiveAlerts()
	
	// Convert to model alerts
	v.mu.Lock()
	newAlerts := make([]models.Alert, 0, len(activeAlerts))
	for _, alert := range activeAlerts {
		newAlerts = append(newAlerts, v.convertToModelAlert(alert))
	}
	
	// Merge with existing alerts (to preserve resolved alerts)
	// Keep resolved alerts for some time
	cutoff := time.Now().Add(-30 * time.Minute)
	for _, existingAlert := range v.alerts {
		if existingAlert.State == "resolved" && existingAlert.ResolvedAt.After(cutoff) {
			// Keep recently resolved alerts
			found := false
			for _, newAlert := range newAlerts {
				if newAlert.Name == existingAlert.Name && newAlert.Source == existingAlert.Source {
					found = true
					break
				}
			}
			if !found {
				newAlerts = append(newAlerts, existingAlert)
			}
		}
	}
	
	// Sort by timestamp (most recent first)
	sort.Slice(newAlerts, func(i, j int) bool {
		return newAlerts[i].Timestamp.After(newAlerts[j].Timestamp)
	})
	
	v.alerts = newAlerts
	v.mu.Unlock()
	
	v.alertsPanel.SetAlerts(v.alerts)
}

// renderClusterPanel updates the cluster overview panel
func (v *ObservabilityView) renderClusterPanel() {
	if v.clusterMetrics == nil {
		return
	}
	
	// Get node and job summaries
	nodeSummary := v.nodeCollector.GetNodesSummary()
	jobSummary := v.jobCollector.GetJobsSummary()
	
	activeNodes := v.clusterMetrics.ActiveNodes
	downNodes := nodeSummary["down"] + nodeSummary["drain"]
	runningJobs := jobSummary["RUNNING"] + jobSummary["R"]
	pendingJobs := jobSummary["PENDING"] + jobSummary["PD"]
	
	text := fmt.Sprintf(
		`[yellow]Total Nodes:[white] %d active, %d down
[yellow]Total CPUs:[white] %d cores
[yellow]Load Average:[white] %.1f per core
[yellow]Memory:[white] %s / %s (%.1f%%)
[yellow]Jobs Running:[white] %d
[yellow]Jobs Pending:[white] %d`,
		activeNodes, downNodes,
		v.clusterMetrics.TotalCPUCores,
		v.clusterMetrics.AverageLoadPerCore,
		models.FormatValue(float64(v.clusterMetrics.UsedMemory), "bytes"),
		models.FormatValue(float64(v.clusterMetrics.TotalMemory), "bytes"),
		v.clusterMetrics.MemoryUsagePercent,
		runningJobs, pendingJobs,
	)
	
	v.clusterPanel.SetText(text)
}

// renderNodeTable updates the node metrics table
func (v *ObservabilityView) renderNodeTable() {
	// Clear existing rows (except header)
	for i := v.nodeTable.GetRowCount() - 1; i > 0; i-- {
		v.nodeTable.RemoveRow(i)
	}
	
	row := 1
	for _, node := range v.nodeMetrics {
		metrics := &node.Resources
		
		cpuColor := models.GetColorForUsage(metrics.CPU.Usage)
		memColor := models.GetColorForUsage(metrics.Memory.Usage)
		
		// Determine state color
		stateColor := tcell.ColorGreen
		state := node.NodeState
		if state == "" {
			state = "IDLE"
		}
		switch state {
		case "down", "DOWN":
			stateColor = tcell.ColorRed
		case "drain", "DRAIN":
			stateColor = tcell.ColorYellow
		case "alloc", "ALLOC", "ALLOCATED":
			stateColor = tcell.ColorBlue
		}
		
		v.nodeTable.SetCell(row, 0, tview.NewTableCell(node.NodeName))
		v.nodeTable.SetCell(row, 1, tview.NewTableCell(state).SetTextColor(stateColor))
		v.nodeTable.SetCell(row, 2, tview.NewTableCell(
			fmt.Sprintf("%.1f%%", metrics.CPU.Usage)).SetTextColor(getColor(cpuColor)))
		v.nodeTable.SetCell(row, 3, tview.NewTableCell(
			fmt.Sprintf("%.1f%%", metrics.Memory.Usage)).SetTextColor(getColor(memColor)))
		v.nodeTable.SetCell(row, 4, tview.NewTableCell(
			fmt.Sprintf("%.1f", metrics.CPU.Load1m)))
		v.nodeTable.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%d", node.JobCount)))
		v.nodeTable.SetCell(row, 6, tview.NewTableCell(
			fmt.Sprintf("↓%s ↑%s", 
				models.FormatValue(metrics.Network.ReceiveBytesPerSec, "bytes/sec"),
				models.FormatValue(metrics.Network.TransmitBytesPerSec, "bytes/sec"))))
		v.nodeTable.SetCell(row, 7, tview.NewTableCell(
			fmt.Sprintf("R:%s W:%s",
				models.FormatValue(metrics.Disk.ReadBytesPerSec, "bytes/sec"),
				models.FormatValue(metrics.Disk.WriteBytesPerSec, "bytes/sec"))))
		
		row++
	}
}

// renderJobTable updates the job metrics table
func (v *ObservabilityView) renderJobTable() {
	// Clear existing rows (except header)
	for i := v.jobTable.GetRowCount() - 1; i > 0; i-- {
		v.jobTable.RemoveRow(i)
	}
	
	row := 1
	for _, job := range v.jobMetrics {
		// Skip non-running jobs
		if job.State != "RUNNING" && job.State != "R" {
			continue
		}
		
		metrics := &job.Resources
		
		// Determine efficiency color
		effColor := tcell.ColorGreen
		if job.Efficiency.OverallEfficiency < 50 {
			effColor = tcell.ColorYellow
		}
		if job.Efficiency.OverallEfficiency < 20 {
			effColor = tcell.ColorRed
		}
		
		// Determine state color
		stateColor := tcell.ColorGreen
		if job.State == "PENDING" || job.State == "PD" {
			stateColor = tcell.ColorYellow
		}
		
		v.jobTable.SetCell(row, 0, tview.NewTableCell(job.JobID))
		v.jobTable.SetCell(row, 1, tview.NewTableCell(job.User))
		v.jobTable.SetCell(row, 2, tview.NewTableCell(
			fmt.Sprintf("%.1f%%", metrics.CPU.Usage)).
			SetTextColor(getColor(models.GetColorForUsage(metrics.CPU.Usage))))
		v.jobTable.SetCell(row, 3, tview.NewTableCell(
			models.FormatValue(float64(metrics.Memory.Used), "bytes")))
		v.jobTable.SetCell(row, 4, tview.NewTableCell(
			fmt.Sprintf("%d", job.AllocatedCPUs)))
		v.jobTable.SetCell(row, 5, tview.NewTableCell(
			models.FormatValue(float64(job.AllocatedMem), "bytes")))
		v.jobTable.SetCell(row, 6, tview.NewTableCell(
			fmt.Sprintf("%.1f%%", job.Efficiency.OverallEfficiency)).SetTextColor(effColor))
		v.jobTable.SetCell(row, 7, tview.NewTableCell(job.State).
			SetTextColor(stateColor))
		
		row++
	}
}

// renderAlerts updates the alerts panel
func (v *ObservabilityView) renderAlerts() {
	// Alerts are handled by the AlertsWidget
}

// showHelp displays help information
func (v *ObservabilityView) showHelp() {
	helpText := `Observability View - Keyboard Shortcuts:

Navigation:
  Tab/Shift+Tab : Cycle through panels
  n/N           : Focus node table
  j/J           : Focus job table  
  a/A           : Focus alerts panel
  ↑/↓           : Navigate in tables
  
Actions:
  r/R/Ctrl+R    : Refresh metrics
  Enter         : View details for selected item
  h/H/?         : Show this help
  Esc/Ctrl+C    : Exit view
  
Display:
  s/S           : Toggle sparklines
  c/C           : Cycle color schemes`
	
	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			v.app.SetRoot(v.root, true)
		})
	
	v.app.SetRoot(modal, true)
}

// showError displays an error message
func (v *ObservabilityView) showError(message string) {
	// TODO: Implement error display
	// For now, could log or show in a status bar
}

// showNotification displays a notification message
func (v *ObservabilityView) showNotification(message string) {
	// TODO: Implement notification display
	// For now, could show a temporary toast or status message
	// This could be integrated with the alerts panel or a separate notification system
}

// getColor converts string color to tcell color
func getColor(color string) tcell.Color {
	switch color {
	case "red":
		return tcell.ColorRed
	case "yellow":
		return tcell.ColorYellow
	case "orange":
		return tcell.ColorOrange
	case "green":
		return tcell.ColorGreen
	default:
		return tcell.ColorWhite
	}
}

// Interface implementations

// GetID returns the view ID
func (v *ObservabilityView) GetID() string {
	return "observability"
}

// GetName returns the view name
func (v *ObservabilityView) GetName() string {
	return "Observability"
}

// GetDescription returns the view description
func (v *ObservabilityView) GetDescription() string {
	return "Real-time metrics and monitoring dashboard"
}

// Initialize initializes the view
func (v *ObservabilityView) Initialize(ctx context.Context) error {
	return v.Start(ctx)
}

// Cleanup cleans up the view
func (v *ObservabilityView) Cleanup(ctx context.Context) error {
	return v.Stop(ctx)
}

// HandleEvent handles view events
func (v *ObservabilityView) HandleEvent(event plugin.ViewEvent) error {
	switch event.Type {
	case "refresh":
		return v.refresh(context.Background())
	case "node_selected":
		if node, ok := event.Data["node"].(string); ok {
			v.selectedNode = node
		}
	case "job_selected":
		if job, ok := event.Data["job"].(string); ok {
			v.selectedJob = job
		}
	}
	return nil
}