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
)

// JobsOverlay adds real-time metrics to the jobs view
type JobsOverlay struct {
	client       *prometheus.CachedClient
	collector    *models.JobMetricsCollector
	queryBuilder *prometheus.QueryBuilder

	// Cache of job metrics
	metrics      map[string]*models.JobMetrics
	lastUpdate   time.Time
	updateMutex  sync.RWMutex

	// Configuration
	refreshInterval time.Duration
	enabled         bool
}

// NewJobsOverlay creates a new jobs overlay
func NewJobsOverlay(client *prometheus.CachedClient, cgroupPattern string) *JobsOverlay {
	qb, _ := prometheus.NewQueryBuilder()

	return &JobsOverlay{
		client:          client,
		collector:       models.NewJobMetricsCollector(cgroupPattern),
		queryBuilder:    qb,
		metrics:         make(map[string]*models.JobMetrics),
		refreshInterval: 30 * time.Second,
		enabled:         true,
	}
}

// GetID returns the overlay ID
func (o *JobsOverlay) GetID() string {
	return "jobs-metrics"
}

// GetName returns the overlay name
func (o *JobsOverlay) GetName() string {
	return "Job Metrics"
}

// GetTargetView returns the target view ID
func (o *JobsOverlay) GetTargetView() string {
	return "jobs"
}

// Initialize initializes the overlay
func (o *JobsOverlay) Initialize(ctx context.Context) error {
	// Start background metric collection
	go o.backgroundRefresh(ctx)
	return nil
}

// GetColumns returns additional columns to add to the jobs table
func (o *JobsOverlay) GetColumns() []plugin.ColumnDefinition {
	return []plugin.ColumnDefinition{
		{
			ID:       "cpu_pct",
			Name:     "CPU%",
			Width:    8,
			Priority: 10,
			Align:    "right",
		},
		{
			ID:       "memory",
			Name:     "Memory",
			Width:    10,
			Priority: 20,
			Align:    "right",
		},
		{
			ID:       "efficiency",
			Name:     "Efficiency",
			Width:    10,
			Priority: 30,
			Align:    "right",
		},
	}
}

// GetCellData returns data for a specific cell
func (o *JobsOverlay) GetCellData(ctx context.Context, viewID string, rowID interface{}, columnID string) (string, error) {
	// Extract job ID from row ID
	jobID, ok := rowID.(string)
	if !ok {
		return "-", nil
	}

	o.updateMutex.RLock()
	metrics, exists := o.metrics[jobID]
	o.updateMutex.RUnlock()

	if !exists {
		return "-", nil
	}

	switch columnID {
	case "cpu_pct":
		return fmt.Sprintf("%.1f%%", metrics.Resources.CPU.Usage), nil

	case "memory":
		return models.FormatValue(float64(metrics.Resources.Memory.Used), "bytes"), nil

	case "efficiency":
		eff := metrics.Efficiency.OverallEfficiency
		bar := generateEfficiencyBar(eff)
		return fmt.Sprintf("%s %.0f%%", bar, eff), nil
	}

	return "", fmt.Errorf("unknown column: %s", columnID)
}

// GetCellStyle returns styling for a specific cell
func (o *JobsOverlay) GetCellStyle(ctx context.Context, viewID string, rowID interface{}, columnID string) plugin.CellStyle {
	// Extract job ID from row ID
	jobID, ok := rowID.(string)
	if !ok {
		return plugin.CellStyle{Foreground: "gray"}
	}

	o.updateMutex.RLock()
	metrics, exists := o.metrics[jobID]
	o.updateMutex.RUnlock()

	if !exists {
		return plugin.CellStyle{Foreground: "gray"}
	}

	switch columnID {
	case "cpu_pct":
		cpu := metrics.Resources.CPU.Usage
		if cpu > 90 {
			return plugin.CellStyle{Foreground: "red"}
		} else if cpu > 75 {
			return plugin.CellStyle{Foreground: "yellow"}
		}
		return plugin.CellStyle{Foreground: "green"}

	case "memory":
		return plugin.CellStyle{Foreground: "white"}

	case "efficiency":
		eff := metrics.Efficiency.OverallEfficiency
		if eff < 20 {
			return plugin.CellStyle{Foreground: "red"}
		} else if eff < 50 {
			return plugin.CellStyle{Foreground: "yellow"}
		}
		return plugin.CellStyle{Foreground: "green"}
	}

	return plugin.CellStyle{Foreground: "white"}
}

// ShouldRefresh indicates if the overlay needs refresh
func (o *JobsOverlay) ShouldRefresh() bool {
	return time.Since(o.lastUpdate) > o.refreshInterval
}

// GetRowEnhancement returns row-level enhancements
func (o *JobsOverlay) GetRowEnhancement(rowData map[string]interface{}) *plugin.OverlayRowEnhancement {
	jobID, ok := rowData["JobID"].(string)
	if !ok {
		return nil
	}

	o.updateMutex.RLock()
	metrics, exists := o.metrics[jobID]
	o.updateMutex.RUnlock()

	if !exists || metrics.Efficiency.OverallEfficiency >= 50 {
		return nil
	}

	// Highlight inefficient jobs
	if metrics.Efficiency.OverallEfficiency < 20 {
		return &plugin.OverlayRowEnhancement{
			BackgroundColor: tcell.ColorDarkRed,
			Bold:            true,
		}
	}

	return nil
}

// GetTooltip returns tooltip text for a cell
func (o *JobsOverlay) GetTooltip(column string, rowData map[string]interface{}) string {
	jobID, ok := rowData["JobID"].(string)
	if !ok {
		return ""
	}

	o.updateMutex.RLock()
	metrics, exists := o.metrics[jobID]
	o.updateMutex.RUnlock()

	if !exists {
		return "No metrics available"
	}

	switch column {
	case "CPU%":
		return fmt.Sprintf("CPU: %.1f%% of %d cores\nThrottled: %.1f%%",
			metrics.Resources.CPU.Usage,
			metrics.AllocatedCPUs,
			metrics.Resources.CPU.Throttled)

	case "Memory":
		return fmt.Sprintf("Memory: %s of %s\nCache: %s",
			models.FormatValue(float64(metrics.Resources.Memory.Used), "bytes"),
			models.FormatValue(float64(metrics.AllocatedMem), "bytes"),
			models.FormatValue(float64(metrics.Resources.Memory.Cache), "bytes"))

	case "Efficiency":
		return fmt.Sprintf("CPU Efficiency: %.1f%%\nMemory Efficiency: %.1f%%\nWasted: %.1f CPU cores, %s memory",
			metrics.Efficiency.CPUEfficiency,
			metrics.Efficiency.MemEfficiency,
			metrics.Efficiency.CPUWasted,
			models.FormatValue(float64(metrics.Efficiency.MemWasted), "bytes"))
	}

	return ""
}

// HandleEvent handles view events
func (o *JobsOverlay) HandleEvent(event plugin.OverlayEvent) error {
	switch event.Type {
	case "refresh":
		// Force refresh metrics
		return o.refreshMetrics(context.Background())
	case "toggle":
		// Toggle overlay on/off
		o.enabled = !o.enabled
	}
	return nil
}

// Stop stops the overlay
func (o *JobsOverlay) Stop(ctx context.Context) error {
	// Cleanup will be handled by context cancellation
	return nil
}

// backgroundRefresh runs periodic metric updates
func (o *JobsOverlay) backgroundRefresh(ctx context.Context) {
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
func (o *JobsOverlay) refreshMetrics(ctx context.Context) error {
	if !o.enabled {
		return nil
	}

	// In a real implementation, this would get job list from SLURM
	// For now, we'll discover jobs from existing metrics
	jobIDs := o.getActiveJobIDs()

	for _, jobID := range jobIDs {
		// Build queries for this job
		queries, err := o.queryBuilder.GetJobQueries(jobID)
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
		o.collector.UpdateFromPrometheus(jobID, metrics)
	}

	// Get all metrics from collector
	allMetrics := o.collector.GetAllJobs()

	// Update cache
	o.updateMutex.Lock()
	o.metrics = allMetrics
	o.lastUpdate = time.Now()
	o.updateMutex.Unlock()

	return nil
}

// getColorString converts a tcell color to a string
func getColorString(color tcell.Color) string {
	switch color {
	case tcell.ColorRed:
		return "red"
	case tcell.ColorYellow:
		return "yellow"
	case tcell.ColorGreen:
		return "green"
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

// getActiveJobIDs returns list of active job IDs
func (o *JobsOverlay) getActiveJobIDs() []string {
	// In a real implementation, this would query SLURM
	// For now, return a sample list
	return []string{"12345", "12346", "12347", "12348", "12349"}
}

// generateEfficiencyBar creates a visual bar for efficiency
func generateEfficiencyBar(efficiency float64) string {
	if efficiency < 0 {
		efficiency = 0
	}
	if efficiency > 100 {
		efficiency = 100
	}

	filled := int(efficiency / 10)
	bar := ""

	for i := 0; i < 10; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return bar
}

// convertToTimeSeries converts Prometheus result to TimeSeries
func convertToTimeSeries(name string, result *prometheus.QueryResult) *models.TimeSeries {
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

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}