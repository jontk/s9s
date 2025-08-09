package overlays

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	
	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/models"
	"github.com/jontk/s9s/plugins/observability/prometheus"
	"github.com/jontk/s9s/plugins/observability/views/widgets"
)

// NodesOverlay adds real-time metrics to the nodes view
type NodesOverlay struct {
	client       *prometheus.CachedClient
	collector    *models.NodeMetricsCollector
	queryBuilder *prometheus.QueryBuilder
	
	// Cache of node metrics
	metrics      map[string]*models.NodeMetrics
	sparklines   map[string]*widgets.TimeSeriesSparkline
	lastUpdate   time.Time
	updateMutex  sync.RWMutex
	
	// Configuration
	refreshInterval  time.Duration
	enabled          bool
	showSparklines   bool
	sparklinePoints  int
	nodeLabel        string
}

// NewNodesOverlay creates a new nodes overlay
func NewNodesOverlay(client *prometheus.CachedClient, nodeLabel string) *NodesOverlay {
	qb, _ := prometheus.NewQueryBuilder()
	
	return &NodesOverlay{
		client:           client,
		collector:        models.NewNodeMetricsCollector(nodeLabel),
		queryBuilder:     qb,
		metrics:          make(map[string]*models.NodeMetrics),
		sparklines:       make(map[string]*widgets.TimeSeriesSparkline),
		refreshInterval:  30 * time.Second,
		enabled:          true,
		showSparklines:   true,
		sparklinePoints:  20,
		nodeLabel:        nodeLabel,
	}
}

// GetID returns the overlay ID
func (o *NodesOverlay) GetID() string {
	return "nodes-metrics"
}

// GetName returns the overlay name
func (o *NodesOverlay) GetName() string {
	return "Node Metrics"
}

// GetTargetView returns the target view ID
func (o *NodesOverlay) GetTargetView() string {
	return "nodes"
}

// Initialize initializes the overlay
func (o *NodesOverlay) Initialize(ctx context.Context) error {
	// Start background metric collection
	go o.backgroundRefresh(ctx)
	return nil
}

// GetColumns returns additional columns to add to the nodes table
func (o *NodesOverlay) GetColumns() []plugin.OverlayColumn {
	columns := []plugin.OverlayColumn{
		{
			Name:     "CPU",
			Width:    12,
			Position: plugin.ColumnPositionAfter,
			After:    "State",
		},
		{
			Name:     "Memory",
			Width:    12,
			Position: plugin.ColumnPositionAfter,
			After:    "CPU",
		},
		{
			Name:     "Load",
			Width:    8,
			Position: plugin.ColumnPositionAfter,
			After:    "Memory",
		},
		{
			Name:     "Network I/O",
			Width:    16,
			Position: plugin.ColumnPositionAfter,
			After:    "Load",
		},
		{
			Name:     "Disk I/O",
			Width:    16,
			Position: plugin.ColumnPositionAfter,
			After:    "Network I/O",
		},
	}
	
	if o.showSparklines {
		columns = append(columns, plugin.OverlayColumn{
			Name:     "Trend",
			Width:    20,
			Position: plugin.ColumnPositionEnd,
		})
	}
	
	return columns
}

// GetCellData returns the data for a specific cell
func (o *NodesOverlay) GetCellData(column string, rowData map[string]interface{}) *plugin.OverlayCellData {
	// Extract node name from row data
	nodeName, ok := rowData["NodeName"].(string)
	if !ok {
		return nil
	}
	
	o.updateMutex.RLock()
	metrics, exists := o.metrics[nodeName]
	sparkline := o.sparklines[nodeName]
	o.updateMutex.RUnlock()
	
	if !exists {
		// Return placeholder if no metrics available
		return &plugin.OverlayCellData{
			Text:  "-",
			Color: tcell.ColorGray,
		}
	}
	
	switch column {
	case "CPU":
		cpu := metrics.Resources.CPU.Usage
		color := getUsageColor(cpu)
		
		// Create a mini bar
		bar := generateUsageBar(cpu, 8)
		
		return &plugin.OverlayCellData{
			Text:  fmt.Sprintf("%s %.1f%%", bar, cpu),
			Color: color,
		}
		
	case "Memory":
		mem := metrics.Resources.Memory
		usage := mem.Usage
		color := getUsageColor(usage)
		
		// Create a mini bar
		bar := generateUsageBar(usage, 8)
		
		return &plugin.OverlayCellData{
			Text:  fmt.Sprintf("%s %.1f%%", bar, usage),
			Color: color,
		}
		
	case "Load":
		load := metrics.Resources.CPU.Load1m
		loadPerCore := load / float64(metrics.Resources.CPU.Cores)
		
		color := tcell.ColorGreen
		if loadPerCore > 2.0 {
			color = tcell.ColorRed
		} else if loadPerCore > 1.0 {
			color = tcell.ColorYellow
		}
		
		return &plugin.OverlayCellData{
			Text:  fmt.Sprintf("%.2f", load),
			Color: color,
		}
		
	case "Network I/O":
		rx := metrics.Resources.Network.ReceiveBytesPerSec
		tx := metrics.Resources.Network.TransmitBytesPerSec
		
		return &plugin.OverlayCellData{
			Text:  fmt.Sprintf("↓%s ↑%s", formatBandwidth(rx), formatBandwidth(tx)),
			Color: tcell.ColorWhite,
		}
		
	case "Disk I/O":
		read := metrics.Resources.Disk.ReadBytesPerSec
		write := metrics.Resources.Disk.WriteBytesPerSec
		
		return &plugin.OverlayCellData{
			Text:  fmt.Sprintf("R:%s W:%s", formatBandwidth(read), formatBandwidth(write)),
			Color: tcell.ColorWhite,
		}
		
	case "Trend":
		if sparkline != nil && o.showSparklines {
			// This would need custom rendering support
			// For now, return a text representation
			return &plugin.OverlayCellData{
				Text:     generateTextSparkline(sparkline),
				Color:    tcell.ColorWhite,
				CustomRender: true, // Flag for custom rendering
			}
		}
		return nil
	}
	
	return nil
}

// GetRowEnhancement returns row-level enhancements
func (o *NodesOverlay) GetRowEnhancement(rowData map[string]interface{}) *plugin.OverlayRowEnhancement {
	nodeName, ok := rowData["NodeName"].(string)
	if !ok {
		return nil
	}
	
	o.updateMutex.RLock()
	metrics, exists := o.metrics[nodeName]
	o.updateMutex.RUnlock()
	
	if !exists {
		return nil
	}
	
	// Check node health
	health := metrics.GetHealthStatus()
	
	switch health {
	case "critical":
		return &plugin.OverlayRowEnhancement{
			BackgroundColor: tcell.ColorDarkRed,
			Bold:            true,
		}
	case "warning":
		return &plugin.OverlayRowEnhancement{
			BackgroundColor: tcell.ColorDarkGoldenrod,
		}
	case "unhealthy":
		return &plugin.OverlayRowEnhancement{
			TextColor: tcell.ColorGray,
			Italic:    true,
		}
	}
	
	return nil
}

// GetTooltip returns tooltip text for a cell
func (o *NodesOverlay) GetTooltip(column string, rowData map[string]interface{}) string {
	nodeName, ok := rowData["NodeName"].(string)
	if !ok {
		return ""
	}
	
	o.updateMutex.RLock()
	metrics, exists := o.metrics[nodeName]
	o.updateMutex.RUnlock()
	
	if !exists {
		return "No metrics available"
	}
	
	switch column {
	case "CPU":
		return fmt.Sprintf("CPU Usage: %.1f%% (%d cores)\nLoad: %.2f, %.2f, %.2f\nSystem: %.1f%%, User: %.1f%%",
			metrics.Resources.CPU.Usage,
			metrics.Resources.CPU.Cores,
			metrics.Resources.CPU.Load1m,
			metrics.Resources.CPU.Load5m,
			metrics.Resources.CPU.Load15m,
			metrics.Resources.CPU.System,
			metrics.Resources.CPU.User)
			
	case "Memory":
		return fmt.Sprintf("Memory: %s / %s (%.1f%%)\nAvailable: %s\nCache: %s, Buffer: %s",
			models.FormatValue(float64(metrics.Resources.Memory.Used), "bytes"),
			models.FormatValue(float64(metrics.Resources.Memory.Total), "bytes"),
			metrics.Resources.Memory.Usage,
			models.FormatValue(float64(metrics.Resources.Memory.Available), "bytes"),
			models.FormatValue(float64(metrics.Resources.Memory.Cache), "bytes"),
			models.FormatValue(float64(metrics.Resources.Memory.Buffer), "bytes"))
			
	case "Network I/O":
		return fmt.Sprintf("Network Receive: %s/s\nNetwork Transmit: %s/s\nErrors: Rx %d, Tx %d",
			models.FormatValue(metrics.Resources.Network.ReceiveBytesPerSec, "bytes"),
			models.FormatValue(metrics.Resources.Network.TransmitBytesPerSec, "bytes"),
			metrics.Resources.Network.ReceiveErrors,
			metrics.Resources.Network.TransmitErrors)
			
	case "Disk I/O":
		return fmt.Sprintf("Disk Read: %s/s (%.0f ops/s)\nDisk Write: %s/s (%.0f ops/s)\nI/O Utilization: %.1f%%",
			models.FormatValue(metrics.Resources.Disk.ReadBytesPerSec, "bytes"),
			metrics.Resources.Disk.ReadOpsPerSec,
			models.FormatValue(metrics.Resources.Disk.WriteBytesPerSec, "bytes"),
			metrics.Resources.Disk.WriteOpsPerSec,
			metrics.Resources.Disk.IOUtilization)
	}
	
	return ""
}

// HandleEvent handles view events
func (o *NodesOverlay) HandleEvent(event plugin.OverlayEvent) error {
	switch event.Type {
	case "refresh":
		// Force refresh metrics
		return o.refreshMetrics(context.Background())
	case "toggle":
		// Toggle overlay on/off
		o.enabled = !o.enabled
	case "toggle_sparklines":
		// Toggle sparklines
		o.showSparklines = !o.showSparklines
	}
	return nil
}

// Stop stops the overlay
func (o *NodesOverlay) Stop(ctx context.Context) error {
	// Cleanup will be handled by context cancellation
	return nil
}

// backgroundRefresh runs periodic metric updates
func (o *NodesOverlay) backgroundRefresh(ctx context.Context) {
	ticker := time.NewTicker(o.refreshInterval)
	defer ticker.Stop()
	
	// Initial refresh
	o.refreshMetrics(ctx)
	
	for {
		select {
		case <-ticker.C:
			o.refreshMetrics(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// refreshMetrics fetches latest metrics from Prometheus
func (o *NodesOverlay) refreshMetrics(ctx context.Context) error {
	if !o.enabled {
		return nil
	}
	
	// Get list of nodes
	nodes := o.getNodeList(ctx)
	
	for _, nodeName := range nodes {
		// Build queries for this node
		queries, err := o.queryBuilder.GetNodeQueries(nodeName, o.nodeLabel)
		if err != nil {
			continue
		}
		
		// Execute queries
		results, err := o.client.BatchQuery(ctx, queries, time.Now())
		if err != nil {
			continue
		}
		
		// Convert to time series
		metrics := make(map[string]*models.TimeSeries)
		for queryName, result := range results {
			if result.Data.Result != nil && len(result.Data.Result) > 0 {
				ts := convertToTimeSeries(queryName, result)
				if ts != nil {
					metrics[queryName] = ts
				}
			}
		}
		
		// Update collector
		o.collector.UpdateFromPrometheus(nodeName, metrics)
		
		// Update sparkline
		o.updateSparkline(nodeName, metrics)
	}
	
	// Get all metrics from collector
	allMetrics := o.collector.GetAllNodes()
	
	// Update cache
	o.updateMutex.Lock()
	o.metrics = allMetrics
	o.lastUpdate = time.Now()
	o.updateMutex.Unlock()
	
	return nil
}

// getNodeList returns list of nodes to query
func (o *NodesOverlay) getNodeList(ctx context.Context) []string {
	// Query for all nodes with node_exporter up
	query := `up{job="node-exporter"}`
	result, err := o.client.Query(ctx, query, time.Now())
	if err != nil || result.Data.Result == nil {
		// Return empty list
		return []string{}
	}
	
	nodes := []string{}
	for _, r := range result.Data.Result {
		if instance, ok := r.Metric[o.nodeLabel]; ok {
			nodes = append(nodes, instance)
		}
	}
	
	return nodes
}

// updateSparkline updates sparkline data for a node
func (o *NodesOverlay) updateSparkline(nodeName string, metrics map[string]*models.TimeSeries) {
	o.updateMutex.Lock()
	defer o.updateMutex.Unlock()
	
	// Get or create sparkline
	sparkline, exists := o.sparklines[nodeName]
	if !exists {
		sparkline = widgets.NewTimeSeriesSparkline(fmt.Sprintf("%s CPU", nodeName), 300) // 5 minutes
		o.sparklines[nodeName] = sparkline
	}
	
	// Add CPU usage value
	if cpuMetrics, ok := metrics["node_cpu_usage"]; ok && cpuMetrics.Latest() != nil {
		sparkline.AddTimedValue(cpuMetrics.Latest().Value, time.Now().Unix())
	}
}

// Helper functions

// getUsageColor returns a color based on usage percentage
func getUsageColor(usage float64) tcell.Color {
	switch {
	case usage >= 90:
		return tcell.ColorRed
	case usage >= 75:
		return tcell.ColorYellow
	case usage >= 50:
		return tcell.ColorOrange
	default:
		return tcell.ColorGreen
	}
}

// generateUsageBar creates a visual bar for usage percentage
func generateUsageBar(usage float64, width int) string {
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	
	filled := int(usage * float64(width) / 100)
	bar := ""
	
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	
	return bar
}

// formatBandwidth formats bandwidth values
func formatBandwidth(bytesPerSec float64) string {
	// Use shorter format for overlay
	value := models.FormatValue(bytesPerSec, "bytes/sec")
	// Remove "/s" suffix since we already indicate it's a rate
	if len(value) > 2 && value[len(value)-2:] == "/s" {
		value = value[:len(value)-2]
	}
	return value
}

// generateTextSparkline creates a text representation of sparkline
func generateTextSparkline(sparkline *widgets.TimeSeriesSparkline) string {
	// This is simplified - in reality, you'd want proper sparkline rendering
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	
	// Get recent values
	values := sparkline.SparklineWidget.values
	if len(values) == 0 {
		return ""
	}
	
	// Take last N values
	numPoints := 10
	if len(values) < numPoints {
		numPoints = len(values)
	}
	
	startIdx := len(values) - numPoints
	result := ""
	
	for i := startIdx; i < len(values); i++ {
		val := values[i]
		normalized := (val - sparkline.SparklineWidget.min) / (sparkline.SparklineWidget.max - sparkline.SparklineWidget.min)
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}
		
		charIdx := int(normalized * float64(len(chars)-1))
		result += string(chars[charIdx])
	}
	
	return result
}