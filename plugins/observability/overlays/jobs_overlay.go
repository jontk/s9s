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
func (o *JobsOverlay) GetColumns() []plugin.OverlayColumn {
	return []plugin.OverlayColumn{
		{
			Name:     "CPU%",
			Width:    8,
			Position: plugin.ColumnPositionAfter,
			After:    "State",
		},
		{
			Name:     "Memory",
			Width:    10,
			Position: plugin.ColumnPositionAfter,
			After:    "CPU%",
		},
		{
			Name:     "Efficiency",
			Width:    10,
			Position: plugin.ColumnPositionAfter,
			After:    "Memory",
		},
	}
}

// GetCellData returns the data for a specific cell
func (o *JobsOverlay) GetCellData(column string, rowData map[string]interface{}) *plugin.OverlayCellData {
	// Extract job ID from row data
	jobID, ok := rowData["JobID"].(string)
	if !ok {
		return nil
	}
	
	o.updateMutex.RLock()
	metrics, exists := o.metrics[jobID]
	o.updateMutex.RUnlock()
	
	if !exists {
		// Return placeholder if no metrics available
		return &plugin.OverlayCellData{
			Text:  "-",
			Color: tcell.ColorGray,
		}
	}
	
	switch column {
	case "CPU%":
		cpu := metrics.Resources.CPU.Usage
		color := tcell.ColorGreen
		if cpu > 90 {
			color = tcell.ColorRed
		} else if cpu > 75 {
			color = tcell.ColorYellow
		}
		
		return &plugin.OverlayCellData{
			Text:  fmt.Sprintf("%.1f%%", cpu),
			Color: color,
		}
		
	case "Memory":
		mem := metrics.Resources.Memory.Used
		return &plugin.OverlayCellData{
			Text:  models.FormatValue(float64(mem), "bytes"),
			Color: tcell.ColorWhite,
		}
		
	case "Efficiency":
		eff := metrics.Efficiency.OverallEfficiency
		color := tcell.ColorGreen
		if eff < 50 {
			color = tcell.ColorYellow
		}
		if eff < 20 {
			color = tcell.ColorRed
		}
		
		// Add sparkline or bar
		bar := generateEfficiencyBar(eff)
		
		return &plugin.OverlayCellData{
			Text:  fmt.Sprintf("%s %.0f%%", bar, eff),
			Color: color,
		}
	}
	
	return nil
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
	
	newMetrics := make(map[string]*models.JobMetrics)
	
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
	if result == nil || result.Data.Result == nil || len(result.Data.Result) == 0 {
		return nil
	}
	
	firstResult := result.Data.Result[0]
	
	ts := &models.TimeSeries{
		Name:   name,
		Labels: firstResult.Metric,
		Values: []models.MetricValue{},
		Type:   models.MetricTypeCustom,
	}
	
	if len(firstResult.Value) >= 2 {
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