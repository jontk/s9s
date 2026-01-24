// Package views provides user interface components and widgets for displaying
// observability data within the S9S application. It includes specialized widgets
// for gauges, sparklines, heatmaps, and alerts with customizable styling and
// interactive features for comprehensive system monitoring visualization.
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

	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/alerts"
	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/logging"
	"github.com/jontk/s9s/plugins/observability/models"
	"github.com/jontk/s9s/plugins/observability/prometheus"
	"github.com/jontk/s9s/plugins/observability/views/widgets"
)

// ObservabilityView provides a comprehensive metrics dashboard
type ObservabilityView struct {
	app          *tview.Application
	client       *prometheus.CachedClient
	config       *config.Config
	queryBuilder *prometheus.QueryBuilder
	alertEngine  *alerts.Engine

	// Layout components
	root         *tview.Flex
	clusterPanel *tview.TextView
	nodeTable    *tview.Table
	jobTable     *tview.Table
	alertsPanel  *widgets.AlertsWidget
	cpuGauge     *widgets.GaugeWidget
	memoryGauge  *widgets.GaugeWidget

	// Data collectors
	nodeCollector *models.NodeMetricsCollector
	jobCollector  *models.JobMetricsCollector

	// Data
	nodeMetrics    map[string]*models.NodeMetrics
	jobMetrics     map[string]*models.JobMetrics
	clusterMetrics *models.AggregateNodeMetrics
	alerts         []models.Alert

	// State
	refreshInterval time.Duration
	stopChan        chan struct{}
	refreshTicker   *time.Ticker
	selectedNode    string
	selectedJob     string
	slurmClient     interface{} // SLURM client for job queries

	// Synchronization
	mu sync.RWMutex
}

// NewObservabilityView creates a new observability view
func NewObservabilityView(app *tview.Application, client *prometheus.CachedClient, cfg interface{}) *ObservabilityView {
	// Type assert config
	obsConfig, ok := cfg.(*config.Config)
	if !ok {
		// Use default config if type assertion fails
		logging.Warn("observability-view", "Config type assertion failed, using default config")
		obsConfig = config.DefaultConfig()
	}

	// Create query builder
	queryBuilder, err := prometheus.NewQueryBuilder()
	if err != nil {
		logging.Error("observability-view", "Failed to create query builder: %v", err)
		// Continue anyway, but log the error
	}
	if queryBuilder == nil {
		logging.Error("observability-view", "Query builder is nil after creation")
	} else {
		logging.Debug("observability-view", "Query builder created successfully")
	}

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
	v.clusterPanel = tview.NewTextView()
	v.clusterPanel.SetDynamicColors(true)
	v.clusterPanel.SetBorder(true)
	v.clusterPanel.SetTitle(" Cluster Overview ")

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
	v.nodeTable = tview.NewTable()
	v.nodeTable.SetBorders(false)
	v.nodeTable.SetSelectable(true, false)
	v.nodeTable.SetBorder(true)
	v.nodeTable.SetTitle(" Node Metrics ")

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
	v.jobTable = tview.NewTable()
	v.jobTable.SetBorders(false)
	v.jobTable.SetSelectable(true, false)
	v.jobTable.SetBorder(true)
	v.jobTable.SetTitle(" Job Metrics ")

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
			go func() { _ = v.refresh(context.Background()) }()
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
			go func() { _ = v.refresh(context.Background()) }()
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
	logging.Info("observability-view", "Starting observability view")

	// Start alert engine first
	logging.Debug("observability-view", "Starting alert engine")
	if err := v.alertEngine.Start(ctx); err != nil {
		logging.Error("observability-view", "Failed to start alert engine: %v", err)
		return fmt.Errorf("failed to start alert engine: %w", err)
	}

	// Initial refresh
	logging.Debug("observability-view", "Performing initial refresh")
	if err := v.refresh(ctx); err != nil {
		// Don't fail if initial refresh fails, just log and continue
		logging.Error("observability-view", "Initial refresh failed: %v", err)
		v.showError(fmt.Sprintf("Initial refresh failed: %v", err))
	}

	// Start refresh ticker
	logging.Debug("observability-view", "Starting refresh ticker with interval: %v", v.refreshInterval)
	v.refreshTicker = time.NewTicker(v.refreshInterval)

	// Start refresh goroutine
	go func() {
		logging.Debug("observability-view", "Refresh goroutine started")
		for {
			select {
			case <-v.refreshTicker.C:
				logging.Debug("observability-view", "Refresh ticker triggered")
				if err := v.refresh(ctx); err != nil {
					// Log error but continue
					logging.Error("observability-view", "Refresh error: %v", err)
					v.showError(fmt.Sprintf("Refresh error: %v", err))
				}
			case <-v.stopChan:
				logging.Debug("observability-view", "Stop signal received")
				return
			case <-ctx.Done():
				logging.Debug("observability-view", "Context cancelled")
				return
			}
		}
	}()

	logging.Info("observability-view", "Observability view started successfully")
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
	logging.Debug("observability-view", "Starting refresh cycle")
	v.mu.Lock()
	defer v.mu.Unlock()

	// Add panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			logging.Error("observability-view", "Panic recovered: %v", r)
			v.showError(fmt.Sprintf("Observability view panic recovered: %v", r))
		}
	}()

	// Fetch metrics from Prometheus with additional error handling
	logging.Debug("observability-view", "Fetching node metrics")
	func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("observability-view", "Node metrics fetch panic: %v", r)
				v.showError(fmt.Sprintf("Node metrics fetch panic: %v", r))
			}
		}()
		if err := v.fetchNodeMetrics(ctx); err != nil {
			logging.Error("observability-view", "Failed to fetch node metrics: %v", err)
			v.showError(fmt.Sprintf("Failed to fetch node metrics: %v", err))
		} else {
			logging.Debug("observability-view", "Node metrics fetched successfully")
		}
	}()

	logging.Debug("observability-view", "Fetching job metrics")
	func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("observability-view", "Job metrics fetch panic: %v", r)
				v.showError(fmt.Sprintf("Job metrics fetch panic: %v", r))
			}
		}()
		if err := v.fetchJobMetrics(ctx); err != nil {
			logging.Error("observability-view", "Failed to fetch job metrics: %v", err)
			v.showError(fmt.Sprintf("Failed to fetch job metrics: %v", err))
		} else {
			logging.Debug("observability-view", "Job metrics fetched successfully")
		}
	}()

	// Update cluster metrics from aggregated node data
	logging.Debug("observability-view", "Updating cluster metrics")
	v.updateClusterMetrics()

	// Update alert engine collectors with latest data
	logging.Debug("observability-view", "Updating alert engine collectors")
	v.alertEngine.SetNodeCollector(v.nodeCollector)
	v.alertEngine.SetJobCollector(v.jobCollector)

	// Update alerts
	logging.Debug("observability-view", "Updating alerts")
	v.updateAlerts()

	// Refresh UI
	logging.Debug("observability-view", "Refreshing UI - nodeMetrics count: %d, jobMetrics count: %d",
		len(v.nodeMetrics), len(v.jobMetrics))

	// Render immediately without queueing
	logging.Debug("observability-view", "Rendering UI components")
	v.renderClusterPanel()
	v.renderNodeTable()
	v.renderJobTable()
	v.renderAlerts()
	logging.Debug("observability-view", "UI rendering completed")

	// Force a redraw
	v.app.Draw()

	logging.Debug("observability-view", "Refresh cycle completed")
	return nil
}

// fetchNodeMetrics fetches node metrics from Prometheus
func (v *ObservabilityView) fetchNodeMetrics(ctx context.Context) error {
	logging.Debug("observability-view", "fetchNodeMetrics starting")

	// Add extra safety checks
	if v.client == nil {
		logging.Error("observability-view", "Prometheus client not initialized")
		return fmt.Errorf("prometheus client not initialized")
	}
	if v.queryBuilder == nil {
		logging.Error("observability-view", "Query builder not initialized")
		return fmt.Errorf("query builder not initialized")
	}

	// Get list of nodes - in a real implementation, this would come from SLURM
	// For now, we'll query Prometheus for all nodes with error handling
	logging.Debug("observability-view", "Getting node list")
	var nodes []string
	func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("observability-view", "getNodeList panicked: %v", r)
				// If getNodeList panics, use empty list
				nodes = []string{}
			}
		}()
		nodes = v.getNodeList(ctx)
	}()

	logging.Info("observability-view", "Processing %d nodes", len(nodes))

	for _, nodeName := range nodes {
		// Get the full instance name (with port) for Prometheus queries
		instanceName := nodeName
		if fullInstance, exists := nodeInstanceMap[nodeName]; exists {
			instanceName = fullInstance
			logging.Debug("observability-view", "Using instance name %s for node %s", instanceName, nodeName)
		}

		// Build queries for this node with error handling
		queries, err := func() (map[string]string, error) {
			var err error
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("query building panic: %v", r)
				}
			}()
			result, queryErr := v.queryBuilder.GetNodeQueries(instanceName, v.config.Metrics.Node.NodeLabel)
			if queryErr != nil {
				err = queryErr
			}
			return result, err
		}()

		if err != nil {
			logging.Error("observability-view", "Failed to build queries for node %s: %v", nodeName, err)
			continue
		}

		logging.Debug("observability-view", "Built %d queries for node %s", len(queries), nodeName)
		for qName, qStr := range queries {
			// Log first 100 chars of query to avoid too much output
			if len(qStr) > 100 {
				logging.Debug("observability-view", "Query %s: %s...", qName, qStr[:100])
			} else {
				logging.Debug("observability-view", "Query %s: %s", qName, qStr)
			}
		}

		// Execute queries in batch with error handling
		results, err := func() (map[string]*prometheus.QueryResult, error) {
			var err error
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("batch query panic: %v", r)
				}
			}()
			result, queryErr := v.client.BatchQuery(ctx, queries, time.Now())
			if queryErr != nil {
				err = queryErr
			}
			return result, err
		}()

		if err != nil {
			logging.Error("observability-view", "Failed to execute queries for node %s: %v", nodeName, err)
			continue
		}

		logging.Debug("observability-view", "Batch query returned %d results for node %s", len(results), nodeName)

		// Convert results to TimeSeries
		metrics := make(map[string]*models.TimeSeries)
		for queryName, result := range results {
			if len(result.Data.Result) > 0 {
				logging.Debug("observability-view", "Query %s returned %d results for node %s",
					queryName, len(result.Data.Result), nodeName)
				// Convert first result to TimeSeries
				ts := v.convertToTimeSeries(queryName, result)
				if ts != nil {
					metrics[queryName] = ts
					logging.Debug("observability-view", "Converted %s to TimeSeries with %d values",
						queryName, len(ts.Values))
				} else {
					logging.Warn("observability-view", "Failed to convert %s to TimeSeries", queryName)
				}
			} else {
				logging.Debug("observability-view", "Query %s returned no results for node %s",
					queryName, nodeName)
			}
		}

		// Log the metrics being sent to the collector
		logging.Info("observability-view", "Updating node %s with %d metrics", nodeName, len(metrics))

		// Update node collector
		v.nodeCollector.UpdateFromPrometheus(nodeName, metrics)

		// Check if the node was actually stored
		if storedNode, exists := v.nodeCollector.GetNode(nodeName); exists {
			logging.Info("observability-view", "Node %s successfully stored: CPU=%.2f%%, Memory=%.2f%%",
				nodeName, storedNode.Resources.CPU.Usage, storedNode.Resources.Memory.Usage)
		} else {
			logging.Error("observability-view", "Node %s was not stored in collector!", nodeName)
		}
	}

	// Update local cache
	v.nodeMetrics = v.nodeCollector.GetAllNodes()

	logging.Info("observability-view", "fetchNodeMetrics completed: %d nodes in cache", len(v.nodeMetrics))
	for nodeName, metrics := range v.nodeMetrics {
		logging.Debug("observability-view", "Node %s metrics: CPU=%.2f%%, Memory=%.2f%%, State=%s",
			nodeName, metrics.Resources.CPU.Usage, metrics.Resources.Memory.Usage, metrics.NodeState)
	}

	return nil
}

// fetchJobMetrics fetches job metrics from Prometheus
func (v *ObservabilityView) fetchJobMetrics(ctx context.Context) error {
	// Get list of running jobs from Prometheus discovery
	jobs := v.getJobList(ctx)
	logging.Info("observability-view", "Fetching metrics for %d jobs: %v", len(jobs), jobs)

	// Try to get job details from SLURM if client is available
	jobDetails := v.getJobDetailsFromSlurm(ctx, jobs)

	for _, jobID := range jobs {
		// Build queries for this job
		queries, err := v.queryBuilder.GetJobQueries(jobID)
		if err != nil {
			logging.Error("observability-view", "Failed to build queries for job %s: %v", jobID, err)
			continue
		}

		// Log the actual queries being made
		for queryName, query := range queries {
			logging.Debug("observability-view", "Job %s query %s: %s", jobID, queryName, query)
		}

		// Execute queries in batch
		results, err := v.client.BatchQuery(ctx, queries, time.Now())
		if err != nil {
			logging.Error("observability-view", "Failed to execute queries for job %s: %v", jobID, err)
			continue
		}

		// Convert results to TimeSeries
		metrics := make(map[string]*models.TimeSeries)
		foundMetrics := 0
		for queryName, result := range results {
			if len(result.Data.Result) > 0 {
				logging.Debug("observability-view", "Job %s query %s returned %d results", jobID, queryName, len(result.Data.Result))

				// Log the actual result for debugging
				if vec, err := result.GetVector(); err == nil && len(vec) > 0 {
					for i, sample := range vec {
						if i < 2 { // Log first 2 samples
							logging.Debug("observability-view", "Job %s query %s sample %d: value=%f, metric=%v",
								jobID, queryName, i, sample.Value.Value(), sample.Metric)
						}
					}
				}

				ts := v.convertToTimeSeries(queryName, result)
				if ts != nil {
					metrics[queryName] = ts
					foundMetrics++
					logging.Debug("observability-view", "Job %s query %s successfully converted to TimeSeries", jobID, queryName)
				} else {
					logging.Warn("observability-view", "Job %s query %s failed to convert to TimeSeries", jobID, queryName)
				}
			} else {
				logging.Debug("observability-view", "Job %s query %s returned no results", jobID, queryName)
			}
		}

		logging.Info("observability-view", "Job %s: found %d metrics from %d queries", jobID, foundMetrics, len(queries))

		// Update job collector with Prometheus metrics
		v.jobCollector.UpdateFromPrometheus(jobID, metrics)

		// Update with SLURM info if available
		if details, ok := jobDetails[jobID]; ok {
			v.jobCollector.UpdateJobInfo(jobID, details)
		}
	}

	// Update local cache
	v.jobMetrics = v.jobCollector.GetAllJobs()

	return nil
}

// convertToTimeSeries converts Prometheus query result to TimeSeries
func (v *ObservabilityView) convertToTimeSeries(name string, result *prometheus.QueryResult) *models.TimeSeries {
	if result == nil {
		return nil
	}

	// Try to get vector data first
	vector, err := result.GetVector()
	if err == nil && len(vector) > 0 {
		// Use first sample
		sample := vector[0]

		ts := &models.TimeSeries{
			Name:   name,
			Labels: sample.Metric,
			Values: []models.MetricValue{},
			Type:   models.MetricTypeCustom,
		}

		// Add the current value
		ts.Values = append(ts.Values, models.MetricValue{
			Timestamp: sample.Timestamp,
			Value:     sample.Value.Value(),
			Labels:    sample.Metric,
		})

		return ts
	}

	// Try scalar as fallback
	scalarValue, timestamp, err := result.GetScalar()
	if err == nil {
		ts := &models.TimeSeries{
			Name:   name,
			Labels: map[string]string{},
			Values: []models.MetricValue{},
			Type:   models.MetricTypeCustom,
		}

		ts.Values = append(ts.Values, models.MetricValue{
			Timestamp: timestamp,
			Value:     scalarValue,
			Labels:    map[string]string{},
		})

		return ts
	}

	return nil
}

/*
TODO(lint): Review unused code - func parseFloat is unused

parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
*/

// nodeInstanceMap stores mapping from node name to full instance name with port
var nodeInstanceMap = make(map[string]string)

// getNodeList returns list of nodes to query
func (v *ObservabilityView) getNodeList(ctx context.Context) []string {
	// In a real implementation, this would query SLURM for node list
	// For now, we'll try to discover nodes from Prometheus

	// Query for all nodes with node_exporter up
	query := `up{job="node-exporter"}`
	logging.Debug("observability-view", "Querying for nodes: %s", query)
	result, err := v.client.Query(ctx, query, time.Now())
	if err != nil {
		logging.Error("observability-view", "Node discovery query failed: %v", err)
		// Return default nodes for testing
		defaultNodes := []string{"node001", "node002", "node003", "node004"}
		logging.Debug("observability-view", "Using default nodes: %v", defaultNodes)
		return defaultNodes
	}

	if result.Data.Result == nil {
		logging.Warn("observability-view", "No result data from node discovery query")
		defaultNodes := []string{"node001", "node002", "node003", "node004"}
		return defaultNodes
	}

	nodes := []string{}
	// Clear the mapping for fresh discovery
	nodeInstanceMap = make(map[string]string)

	vector, err := result.GetVector()
	if err == nil {
		logging.Info("observability-view", "Found %d potential nodes in Prometheus", len(vector))
		for _, sample := range vector {
			if instance, ok := sample.Metric[v.config.Metrics.Node.NodeLabel]; ok {
				// Strip port from instance name (e.g., "rocky9.ar.jontk.com:9100" -> "rocky9.ar.jontk.com")
				nodeName := instance
				if strings.Contains(instance, ":") {
					parts := strings.Split(instance, ":")
					nodeName = parts[0]
				}
				// Store mapping from node name to full instance
				nodeInstanceMap[nodeName] = instance
				nodes = append(nodes, nodeName)
				logging.Debug("observability-view", "Added node: %s (maps to instance: %s)", nodeName, instance)
			}
		}
	} else {
		logging.Error("observability-view", "Failed to parse vector from node discovery: %v", err)
	}

	logging.Info("observability-view", "Final node list: %v", nodes)
	logging.Debug("observability-view", "Node instance mapping: %v", nodeInstanceMap)
	return nodes
}

// SetSlurmClient sets the SLURM client for job queries
func (v *ObservabilityView) SetSlurmClient(client interface{}) {
	v.slurmClient = client
	logging.Info("observability-view", "SLURM client set for job queries")
}

// getJobList returns list of jobs to query
func (v *ObservabilityView) getJobList(ctx context.Context) []string {
	// Try to discover jobs from Prometheus metrics first
	jobs := v.discoverJobsFromPrometheus(ctx)
	if len(jobs) > 0 {
		logging.Info("observability-view", "Discovered %d jobs from Prometheus", len(jobs))
		return jobs
	}

	// Fallback to SLURM client if available
	if v.slurmClient != nil {
		// Use reflection to call GetJobs method if it exists
		// This is a temporary solution - ideally we'd have a proper interface
		logging.Debug("observability-view", "SLURM client available, querying for jobs")
		// TODO: Implement proper SLURM job query
	}

	// Return empty list if no jobs found
	logging.Debug("observability-view", "No jobs discovered")
	return []string{}
}

// getJobDetailsFromSlurm fetches job details from SLURM for the given job IDs
func (v *ObservabilityView) getJobDetailsFromSlurm(ctx context.Context, jobIDs []string) map[string]models.JobInfo {
	details := make(map[string]models.JobInfo)

	logging.Debug("observability-view", "getJobDetailsFromSlurm called with jobs: %v", jobIDs)

	if v.slurmClient == nil {
		logging.Debug("observability-view", "No SLURM client available for job details")
		return details
	}

	// Try to use the SLURM client to get job information
	logging.Debug("observability-view", "SLURM client type: %T", v.slurmClient)

	// Try different type assertions for the SLURM client
	var slurmClient dao.SlurmClient
	var ok bool

	// First try direct interface
	if slurmClient, ok = v.slurmClient.(dao.SlurmClient); ok {
		logging.Debug("observability-view", "SLURM client implements dao.SlurmClient interface")
	} else if adapter, ok := v.slurmClient.(*dao.SlurmAdapter); ok {
		// If it's a SlurmAdapter, it should implement SlurmClient
		slurmClient = adapter
		logging.Debug("observability-view", "SLURM client is SlurmAdapter, casting to interface")
	} else {
		logging.Debug("observability-view", "SLURM client does not support job listing interface")
		return details
	}

	if client := slurmClient; client != nil {
		logging.Debug("observability-view", "Querying SLURM for job details")

		// Get job manager from SLURM client
		jobMgr := client.Jobs()
		if jobMgr == nil {
			logging.Error("observability-view", "SLURM client returned nil job manager")
			return details
		}

		// Query for each job ID
		for _, jobID := range jobIDs {
			job, err := jobMgr.Get(jobID)
			if err != nil {
				logging.Debug("observability-view", "Failed to get job %s from SLURM: %v", jobID, err)
				continue
			}

			if job != nil {
				// Convert SLURM job to our JobInfo structure
				info := models.JobInfo{
					JobName:  job.Name,
					User:     job.User,
					State:    job.State,
					NodeList: strings.Split(job.NodeList, ","),
				}

				// Parse start time if available
				if job.StartTime != nil {
					info.StartTime = *job.StartTime
				}

				// Log the raw job data to understand what we're getting
				logging.Debug("observability-view", "SLURM job data: ID=%s, Name=%s, User=%s, State=%s, NodeCount=%d, Command=%s",
					job.ID, job.Name, job.User, job.State, job.NodeCount, job.Command)

				// For now, use NodeCount as CPU allocation until we can get actual CPU data from SLURM
				// This is a limitation of the current dao.Job struct not having NumCPUs field
				if job.NodeCount > 0 {
					// Use 1 CPU per node as a more conservative estimate
					// TODO: Update dao.Job struct to include NumCPUs from SLURM API
					info.AllocatedCPUs = job.NodeCount // Assume 1 CPU per node for now
				}

				details[jobID] = info

				logging.Debug("observability-view", "Got SLURM details for job %s: user=%s, state=%s, nodes=%d",
					jobID, info.User, info.State, job.NodeCount)
			}
		}

		logging.Info("observability-view", "Retrieved SLURM details for %d jobs", len(details))
	} else {
		logging.Debug("observability-view", "SLURM client does not support job listing interface")
	}

	return details
}

// discoverJobsFromPrometheus discovers running jobs from Prometheus metrics
func (v *ObservabilityView) discoverJobsFromPrometheus(ctx context.Context) []string {
	// Query for all container metrics that match the SLURM job pattern
	// This will find all jobs that have metrics in cAdvisor/cgroup-exporter
	query := `container_cpu_usage_seconds_total{id=~"/system.slice/.*slurmstepd.scope/job_.*"}`

	logging.Debug("observability-view", "Discovering jobs with query: %s", query)

	result, err := v.client.Query(ctx, query, time.Now())
	if err != nil {
		logging.Error("observability-view", "Job discovery query failed: %v", err)
		return []string{}
	}

	if result.Data.Result == nil {
		logging.Debug("observability-view", "No job metrics found in Prometheus")
		return []string{}
	}

	// Extract unique job IDs from the metric labels
	jobMap := make(map[string]bool)
	vector, err := result.GetVector()
	if err == nil {
		for _, sample := range vector {
			if id, ok := sample.Metric["id"]; ok {
				// Extract job ID from the cgroup path
				// Format: /system.slice/rocky9.ar.jontk.com_slurmstepd.scope/job_222
				parts := strings.Split(id, "/")
				if len(parts) > 0 {
					lastPart := parts[len(parts)-1]
					if strings.HasPrefix(lastPart, "job_") {
						jobID := strings.TrimPrefix(lastPart, "job_")
						jobMap[jobID] = true
						logging.Debug("observability-view", "Discovered job %s from metric label: %s", jobID, id)
					}
				}
			}
		}
	}

	// Convert map to slice
	jobs := make([]string, 0, len(jobMap))
	for jobID := range jobMap {
		jobs = append(jobs, jobID)
	}

	// Sort for consistent ordering
	sort.Strings(jobs)

	logging.Info("observability-view", "Discovered jobs from Prometheus: %v", jobs)
	return jobs
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
	// Note: Mutex is already held by the caller (refresh function)
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
	// Mutex unlock removed - already handled by caller

	v.alertsPanel.SetAlerts(v.alerts)
}

// renderClusterPanel updates the cluster overview panel
func (v *ObservabilityView) renderClusterPanel() {
	logging.Debug("observability-view", "renderClusterPanel: clusterMetrics available=%v", v.clusterMetrics != nil)
	if v.clusterMetrics == nil {
		v.clusterPanel.SetText("[yellow]Cluster Overview:[white] Loading cluster data...")
		return
	}

	// Get node and job summaries
	nodeSummary := v.nodeCollector.GetNodesSummary()
	jobSummary := v.jobCollector.GetJobsSummary()
	logging.Debug("observability-view", "Node summary: %+v", nodeSummary)
	logging.Debug("observability-view", "Job summary: %+v", jobSummary)

	activeNodes := v.clusterMetrics.ActiveNodes
	downNodes := nodeSummary["down"] + nodeSummary["drain"]
	runningJobs := jobSummary["RUNNING"] + jobSummary["R"]
	pendingJobs := jobSummary["PENDING"] + jobSummary["PD"]

	logging.Debug("observability-view", "Cluster metrics - ActiveNodes:%d, TotalCPUs:%d, Memory:%.1f%%",
		activeNodes, v.clusterMetrics.TotalCPUCores, v.clusterMetrics.MemoryUsagePercent)

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
	logging.Debug("observability-view", "renderNodeTable: nodeMetrics count=%d", len(v.nodeMetrics))
	// Clear existing rows (except header)
	for i := v.nodeTable.GetRowCount() - 1; i > 0; i-- {
		v.nodeTable.RemoveRow(i)
	}

	if len(v.nodeMetrics) == 0 {
		// Add empty row with debug message
		v.nodeTable.SetCell(1, 0, tview.NewTableCell("No nodes detected"))
		v.nodeTable.SetCell(1, 1, tview.NewTableCell("Check Prometheus"))
		logging.Warn("observability-view", "No node metrics available")
		return
	}

	row := 1
	for nodeName, node := range v.nodeMetrics {
		logging.Debug("observability-view", "Processing node: %s, state: %s, CPU: %.1f%%",
			nodeName, node.NodeState, node.Resources.CPU.Usage)

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
	logging.Debug("observability-view", "renderJobTable: jobMetrics count=%d", len(v.jobMetrics))
	// Clear existing rows (except header)
	for i := v.jobTable.GetRowCount() - 1; i > 0; i-- {
		v.jobTable.RemoveRow(i)
	}

	if len(v.jobMetrics) == 0 {
		// Add empty row with debug message
		v.jobTable.SetCell(1, 0, tview.NewTableCell("No jobs detected"))
		v.jobTable.SetCell(1, 1, tview.NewTableCell("Check SLURM/cgroup"))
		logging.Warn("observability-view", "No job metrics available")
		return
	}

	row := 1
	runningJobs := 0
	for jobID, job := range v.jobMetrics {
		logging.Debug("observability-view", "Processing job: %s, state: %s, CPU: %.1f%%",
			jobID, job.State, job.Resources.CPU.Usage)

		// Skip non-running jobs only if state is explicitly set
		// If state is empty, assume the job is running (since we're getting metrics for it)
		if job.State != "" && job.State != "RUNNING" && job.State != "R" {
			continue
		}
		runningJobs++

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
		// Use cgroup limits if SLURM allocation info is not available
		cpuLimit := job.AllocatedCPUs
		memLimit := job.AllocatedMem

		// If SLURM info is not available (0), try to use cgroup limits
		if cpuLimit == 0 && metrics.CPU.Limit > 0 {
			cpuLimit = int(metrics.CPU.Limit)
		}
		if memLimit == 0 && metrics.Memory.Limit > 0 {
			memLimit = metrics.Memory.Limit
		}

		v.jobTable.SetCell(row, 4, tview.NewTableCell(
			fmt.Sprintf("%d", cpuLimit)))
		v.jobTable.SetCell(row, 5, tview.NewTableCell(
			models.FormatValue(float64(memLimit), "bytes")))
		v.jobTable.SetCell(row, 6, tview.NewTableCell(
			fmt.Sprintf("%.1f%%", job.Efficiency.OverallEfficiency)).SetTextColor(effColor))
		v.jobTable.SetCell(row, 7, tview.NewTableCell(job.State).
			SetTextColor(stateColor))

		row++
	}
	logging.Info("observability-view", "Total running jobs displayed: %d", runningJobs)
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
	logging.Info("observability-view", "Initialize called")
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

// GetHelp returns help text for this view
func (v *ObservabilityView) GetHelp() string {
	return `Observability View - Keyboard Shortcuts:

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
}

// HandleKey processes keyboard input
func (v *ObservabilityView) HandleKey(event *tcell.EventKey) bool {
	// Handle global shortcuts
	switch event.Key() {
	case tcell.KeyTab:
		// Cycle through focusable elements
		v.cycleFocus(false)
		return true
	case tcell.KeyBacktab:
		// Reverse cycle
		v.cycleFocus(true)
		return true
	case tcell.KeyCtrlR:
		// Force refresh
		go func() { _ = v.refresh(context.Background()) }()
		return true
	case tcell.KeyEsc, tcell.KeyCtrlC:
		// Return to previous view
		return false
	}

	// Handle specific shortcuts
	switch event.Rune() {
	case 'n', 'N':
		// Focus node table
		v.app.SetFocus(v.nodeTable)
		return true
	case 'j', 'J':
		// Focus job table
		v.app.SetFocus(v.jobTable)
		return true
	case 'a', 'A':
		// Focus alerts
		v.app.SetFocus(v.alertsPanel.GetPrimitive())
		return true
	case 'r', 'R':
		// Refresh data
		go func() { _ = v.refresh(context.Background()) }()
		return true
	case 'h', 'H', '?':
		// Show help
		v.showHelp()
		return true
	}

	return false
}

// SetFocus sets focus to this view
func (v *ObservabilityView) SetFocus(app *tview.Application) {
	app.SetFocus(v.nodeTable) // Default focus to node table
}

// Update refreshes the view data
func (v *ObservabilityView) Update(ctx context.Context) error {
	logging.Debug("observability-view", "Update called")

	// If the view hasn't been started, start it first
	if v.refreshTicker == nil {
		logging.Info("observability-view", "View not started, starting now")
		if err := v.Start(ctx); err != nil {
			logging.Error("observability-view", "Failed to start view: %v", err)
			return err
		}
	}

	return v.refresh(ctx)
}
