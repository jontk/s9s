package overlays

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"

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
func (o *NodesOverlay) GetColumns() []plugin.ColumnDefinition {
	columns := []plugin.ColumnDefinition{
		{
			ID:       "cpu",
			Name:     "CPU",
			Width:    12,
			Priority: 10,
			Align:    "right",
		},
		{
			ID:       "memory",
			Name:     "Memory",
			Width:    12,
			Priority: 20,
			Align:    "right",
		},
		{
			ID:       "load",
			Name:     "Load",
			Width:    8,
			Priority: 30,
			Align:    "right",
		},
		{
			ID:       "network_io",
			Name:     "Network I/O",
			Width:    16,
			Priority: 40,
			Align:    "right",
		},
		{
			ID:       "disk_io",
			Name:     "Disk I/O",
			Width:    16,
			Priority: 50,
			Align:    "right",
		},
	}

	if o.showSparklines {
		columns = append(columns, plugin.ColumnDefinition{
			ID:       "trend",
			Name:     "Trend",
			Width:    20,
			Priority: 60,
			Align:    "center",
		})
	}

	return columns
}

// GetCellData returns data for a specific cell
func (o *NodesOverlay) GetCellData(ctx context.Context, viewID string, rowID interface{}, columnID string) (string, error) {
	// Extract node name from row ID
	nodeName, ok := rowID.(string)
	if !ok {
		return "-", nil
	}

	o.updateMutex.RLock()
	metrics, exists := o.metrics[nodeName]
	sparkline := o.sparklines[nodeName]
	o.updateMutex.RUnlock()

	if !exists {
		return "-", nil
	}

	switch columnID {
	case "cpu":
		cpu := metrics.Resources.CPU.Usage
		bar := generateUsageBar(cpu, 8)
		return fmt.Sprintf("%s %.1f%%", bar, cpu), nil

	case "memory":
		mem := metrics.Resources.Memory
		usage := mem.Usage
		bar := generateUsageBar(usage, 8)
		return fmt.Sprintf("%s %.1f%%", bar, usage), nil

	case "load":
		load := metrics.Resources.CPU.Load1m
		return fmt.Sprintf("%.2f", load), nil

	case "network_io":
		rx := metrics.Resources.Network.ReceiveBytesPerSec
		tx := metrics.Resources.Network.TransmitBytesPerSec
		return fmt.Sprintf("↓%s ↑%s", formatBandwidth(rx), formatBandwidth(tx)), nil

	case "disk_io":
		read := metrics.Resources.Disk.ReadBytesPerSec
		write := metrics.Resources.Disk.WriteBytesPerSec
		return fmt.Sprintf("R:%s W:%s", formatBandwidth(read), formatBandwidth(write)), nil

	case "trend":
		if sparkline != nil && o.showSparklines {
			return generateTextSparkline(sparkline), nil
		}
		return "", nil
	}

	return "", fmt.Errorf("unknown column: %s", columnID)
}

// GetCellStyle returns styling for a specific cell
func (o *NodesOverlay) GetCellStyle(ctx context.Context, viewID string, rowID interface{}, columnID string) plugin.CellStyle {
	// Extract node name from row ID
	nodeName, ok := rowID.(string)
	if !ok {
		return plugin.CellStyle{Foreground: "gray"}
	}

	o.updateMutex.RLock()
	metrics, exists := o.metrics[nodeName]
	o.updateMutex.RUnlock()

	if !exists {
		return plugin.CellStyle{Foreground: "gray"}
	}

	switch columnID {
	case "cpu":
		cpu := metrics.Resources.CPU.Usage
		return plugin.CellStyle{Foreground: colorToString(getUsageColor(cpu))}

	case "memory":
		usage := metrics.Resources.Memory.Usage
		return plugin.CellStyle{Foreground: colorToString(getUsageColor(usage))}

	case "load":
		load := metrics.Resources.CPU.Load1m
		loadPerCore := load / float64(metrics.Resources.CPU.Cores)

		if loadPerCore > 2.0 {
			return plugin.CellStyle{Foreground: "red"}
		} else if loadPerCore > 1.0 {
			return plugin.CellStyle{Foreground: "yellow"}
		}
		return plugin.CellStyle{Foreground: "green"}

	case "network_io", "disk_io", "trend":
		return plugin.CellStyle{Foreground: "white"}
	}

	return plugin.CellStyle{Foreground: "white"}
}

// ShouldRefresh indicates if the overlay needs refresh
func (o *NodesOverlay) ShouldRefresh() bool {
	return time.Since(o.lastUpdate) > o.refreshInterval
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
			RowStyle: plugin.CellStyle{
				Foreground: "gray",
				Italic:     true,
			},
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
	vector, err := result.GetVector()
	if err == nil {
		for _, sample := range vector {
			if instance, ok := sample.Metric[o.nodeLabel]; ok {
				nodes = append(nodes, instance)
			}
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

// colorToString converts a tcell color to a string
func colorToString(color tcell.Color) string {
	switch color {
	case tcell.ColorRed:
		return "red"
	case tcell.ColorYellow:
		return "yellow"
	case tcell.ColorGreen:
		return "green"
	case tcell.ColorOrange:
		return "orange"
	case tcell.ColorBlue:
		return "blue"
	case tcell.ColorWhite:
		return "white"
	case tcell.ColorGray:
		return "gray"
	default:
		return "white"
	}
}

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
	// Simplified text sparkline - in reality, you'd want proper sparkline rendering
	// For now, just return a placeholder since we can't access internal fields
	return "▂▃▅▇▆▄▂▁"
}