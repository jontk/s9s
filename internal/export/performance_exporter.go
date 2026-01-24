package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/fileperms"
	"github.com/jontk/s9s/internal/mathutil"
	"github.com/jontk/s9s/internal/performance"
	"github.com/jontk/s9s/internal/security"
)

// PerformanceExporter handles exporting performance reports to various formats
type PerformanceExporter struct {
	defaultPath string
}

// NewPerformanceExporter creates a new performance exporter
func NewPerformanceExporter(defaultPath string) *PerformanceExporter {
	if defaultPath == "" {
		homeDir, _ := os.UserHomeDir()
		defaultPath = filepath.Join(homeDir, "slurm_exports", "performance")
	}

	// Ensure export directory exists
	_ = os.MkdirAll(defaultPath, fileperms.DirUserOnly)

	return &PerformanceExporter{
		defaultPath: defaultPath,
	}
}

// PerformanceReportData represents performance metrics for export
type PerformanceReportData struct {
	GeneratedAt      time.Time                      `json:"generated_at"`
	ReportPeriod     string                         `json:"report_period"`
	SystemMetrics    SystemMetrics                  `json:"system_metrics"`
	OperationStats   []performance.OperationSummary `json:"operation_stats"`
	OptimizationTips []performance.Recommendation   `json:"optimization_tips"`
}

// SystemMetrics represents system-level performance metrics
type SystemMetrics struct {
	CPUUsage       float64       `json:"cpu_usage"`
	MemoryUsage    float64       `json:"memory_usage"`
	MemoryUsed     uint64        `json:"memory_used"`
	MemoryTotal    uint64        `json:"memory_total"`
	GoroutineCount int           `json:"goroutine_count"`
	ResponseTime   time.Duration `json:"response_time"`
}

// ExportPerformanceReport exports a performance report in the specified format
func (pe *PerformanceExporter) ExportPerformanceReport(profiler *performance.Profiler, optimizer *performance.Optimizer, format ExportFormat, customPath string) (*ExportResult, error) {
	result := &ExportResult{
		Format:    format,
		Timestamp: time.Now(),
	}

	// Collect performance data
	memStats := profiler.CaptureMemoryStats()
	opStatsMap := profiler.GetOperationStats()

	// Convert operation stats map to slice
	opStats := make([]performance.OperationSummary, 0, len(opStatsMap))
	for _, stat := range opStatsMap {
		opStats = append(opStats, stat)
	}

	// Calculate system metrics
	systemMetrics := SystemMetrics{
		CPUUsage:       calculateCPUUsage(opStats),
		MemoryUsage:    float64(memStats.HeapInuse) / float64(memStats.Sys) * 100.0,
		MemoryUsed:     memStats.HeapInuse,
		MemoryTotal:    memStats.Sys,
		GoroutineCount: runtime.NumGoroutine(),
		ResponseTime:   calculateAvgResponseTime(opStats),
	}

	// Get optimization recommendations
	recommendations := optimizer.Analyze()

	data := PerformanceReportData{
		GeneratedAt:      time.Now(),
		ReportPeriod:     "Last 24 hours", // This could be configurable
		SystemMetrics:    systemMetrics,
		OperationStats:   opStats,
		OptimizationTips: recommendations,
	}

	// Generate filename
	filename := pe.generateFilename(format)

	// Determine output path
	var outputPath string
	if customPath != "" {
		outputPath = customPath
	} else {
		outputPath = filepath.Join(pe.defaultPath, filename)
	}

	// Validate output path is within safe directory
	// Allow writes within defaultPath or user's home directory
	homeDir, _ := os.UserHomeDir()
	validPath, validationErr := security.ValidatePathWithinBase(outputPath, pe.defaultPath)
	if validationErr != nil && homeDir != "" {
		// Try validating against home directory as fallback
		validPath, validationErr = security.ValidatePathWithinBase(outputPath, homeDir)
	}
	if validationErr != nil {
		result.Error = fmt.Errorf("invalid export path %q: %w", outputPath, validationErr)
		return result, result.Error
	}
	outputPath = validPath

	result.FilePath = outputPath

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, fileperms.DirUserOnly); err != nil{
		result.Error = fmt.Errorf("failed to create directory %s: %w", dir, err)
		return result, result.Error
	}

	// Export based on format
	var err error
	switch format {
	case FormatText:
		err = pe.exportText(data, outputPath)
	case FormatJSON:
		err = pe.exportJSON(data, outputPath)
	case FormatCSV:
		err = pe.exportCSV(data, outputPath)
	case FormatMarkdown:
		err = pe.exportMarkdown(data, outputPath)
	case FormatHTML:
		err = pe.exportHTML(data, outputPath)
	default:
		err = fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		result.Error = err
		return result, err
	}

	// Get file size
	if stat, err := os.Stat(outputPath); err == nil {
		result.Size = stat.Size()
	}

	result.Success = true
	return result, nil
}

// exportText exports performance report as plain text
func (pe *PerformanceExporter) exportText(data PerformanceReportData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write header
	if _, err := fmt.Fprintf(file, "S9s Performance Report\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "=====================\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Generated: %s\n", data.GeneratedAt.Format("2006-01-02 15:04:05")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Period: %s\n\n", data.ReportPeriod); err != nil {
		return err
	}

	// System Metrics
	if _, err := fmt.Fprintf(file, "System Metrics\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "--------------\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "CPU Usage: %.2f%%\n", data.SystemMetrics.CPUUsage); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Memory Usage: %.2f%% (%s / %s)\n",
		data.SystemMetrics.MemoryUsage,
		formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryUsed)),
		formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryTotal))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Goroutines: %d\n", data.SystemMetrics.GoroutineCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Response Time (avg): %v\n\n", data.SystemMetrics.ResponseTime); err != nil {
		return err
	}

	// Operation Statistics
	if _, err := fmt.Fprintf(file, "Operation Statistics\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "-------------------\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "%-30s %10s %15s %15s\n", "Operation", "Count", "Avg Time", "Total Time"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "%s\n", strings.Repeat("-", 80)); err != nil {
		return err
	}

	for _, op := range data.OperationStats {
		if _, err := fmt.Fprintf(file, "%-30s %10d %15v %15v\n",
			op.Name,
			op.Count,
			op.AverageTime,
			op.TotalTime); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(file, "\n"); err != nil {
		return err
	}

	// Optimization Tips
	if len(data.OptimizationTips) > 0 {
		if _, err := fmt.Fprintf(file, "Optimization Recommendations\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(file, "---------------------------\n"); err != nil {
			return err
		}
		for i, tip := range data.OptimizationTips {
			if _, err := fmt.Fprintf(file, "%d. %s\n", i+1, tip.Suggestion); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(file, "   Impact: %s | Category: %s\n\n", tip.Impact, tip.Category); err != nil {
				return err
			}
		}
	}

	return nil
}

// exportJSON exports performance report as JSON
func (pe *PerformanceExporter) exportJSON(data PerformanceReportData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// exportCSV exports performance report as CSV
func (pe *PerformanceExporter) exportCSV(data PerformanceReportData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write metadata
	if err := writer.Write([]string{"Report Type", "Performance Report"}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Generated", data.GeneratedAt.Format("2006-01-02 15:04:05")}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Period", data.ReportPeriod}); err != nil {
		return err
	}
	if err := writer.Write([]string{}); err != nil { // Empty line
		return err
	}

	// System Metrics
	if err := writer.Write([]string{"System Metrics"}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Metric", "Value"}); err != nil {
		return err
	}
	if err := writer.Write([]string{"CPU Usage", fmt.Sprintf("%.2f%%", data.SystemMetrics.CPUUsage)}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Memory Usage", fmt.Sprintf("%.2f%%", data.SystemMetrics.MemoryUsage)}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Memory Used", formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryUsed))}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Memory Total", formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryTotal))}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Goroutines", fmt.Sprintf("%d", data.SystemMetrics.GoroutineCount)}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Response Time", data.SystemMetrics.ResponseTime.String()}); err != nil {
		return err
	}
	if err := writer.Write([]string{}); err != nil { // Empty line
		return err
	}

	// Operation Statistics
	if err := writer.Write([]string{"Operation Statistics"}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Operation", "Count", "Average Time", "Total Time"}); err != nil {
		return err
	}
	for _, op := range data.OperationStats {
		if err := writer.Write([]string{
			op.Name,
			fmt.Sprintf("%d", op.Count),
			op.AverageTime.String(),
			op.TotalTime.String(),
		}); err != nil {
			return err
		}
	}

	return nil
}

// exportMarkdown exports performance report as Markdown
func (pe *PerformanceExporter) exportMarkdown(data PerformanceReportData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write Markdown content
	if _, err := fmt.Fprintf(file, "# S9s Performance Report\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "**Generated:** %s  \n", data.GeneratedAt.Format("2006-01-02 15:04:05")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "**Period:** %s\n\n", data.ReportPeriod); err != nil {
		return err
	}

	// System Metrics
	if _, err := fmt.Fprintf(file, "## System Metrics\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "| Metric | Value |\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "|--------|-------|\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "| CPU Usage | %.2f%% |\n", data.SystemMetrics.CPUUsage); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "| Memory Usage | %.2f%% |\n", data.SystemMetrics.MemoryUsage); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "| Memory Used | %s |\n", formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryUsed))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "| Memory Total | %s |\n", formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryTotal))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "| Goroutines | %d |\n", data.SystemMetrics.GoroutineCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "| Avg Response Time | %v |\n\n", data.SystemMetrics.ResponseTime); err != nil {
		return err
	}

	// Operation Statistics
	if _, err := fmt.Fprintf(file, "## Operation Statistics\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "| Operation | Count | Avg Time | Total Time |\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "|-----------|-------|----------|------------|\n"); err != nil {
		return err
	}
	for _, op := range data.OperationStats {
		if _, err := fmt.Fprintf(file, "| %s | %d | %v | %v |\n",
			op.Name, op.Count, op.AverageTime, op.TotalTime); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(file, "\n"); err != nil {
		return err
	}

	// Optimization Tips
	if len(data.OptimizationTips) > 0 {
		if _, err := fmt.Fprintf(file, "## Optimization Recommendations\n\n"); err != nil {
			return err
		}
		for i, tip := range data.OptimizationTips {
			if _, err := fmt.Fprintf(file, "### %d. %s\n\n", i+1, tip.Suggestion); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(file, "- **Impact:** %s\n", tip.Impact); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(file, "- **Category:** %s\n\n", tip.Category); err != nil {
				return err
			}
		}
	}

	return nil
}

// exportHTML exports performance report as HTML
func (pe *PerformanceExporter) exportHTML(data PerformanceReportData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// HTML template
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>S9s Performance Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { background-color: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        table { border-collapse: collapse; width: 100%; margin-top: 10px; }
        th, td { text-align: left; padding: 12px; border-bottom: 1px solid #ddd; }
        th { background-color: #4CAF50; color: white; }
        tr:hover { background-color: #f5f5f5; }
        .metric-card { background-color: #f9f9f9; padding: 15px; margin: 10px 0; border-radius: 4px; }
        .tip { background-color: #fff3cd; padding: 15px; margin: 10px 0; border-radius: 4px; border-left: 4px solid #ffc107; }
        .impact-high { color: #d32f2f; font-weight: bold; }
        .impact-medium { color: #f57c00; font-weight: bold; }
        .impact-low { color: #388e3c; }
        .chart { margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>S9s Performance Report</h1>
        <p><strong>Generated:</strong> {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p><strong>Period:</strong> {{.ReportPeriod}}</p>

        <h2>System Metrics</h2>
        <div class="metric-card">
            <table>
                <tr><th>Metric</th><th>Value</th></tr>
                <tr><td>CPU Usage</td><td>{{printf "%.2f%%" .SystemMetrics.CPUUsage}}</td></tr>
                <tr><td>Memory Usage</td><td>{{printf "%.2f%%" .SystemMetrics.MemoryUsage}}</td></tr>
                <tr><td>Memory Used</td><td>{{.SystemMetrics.MemoryUsed | formatBytes}}</td></tr>
                <tr><td>Memory Total</td><td>{{.SystemMetrics.MemoryTotal | formatBytes}}</td></tr>
                <tr><td>Goroutines</td><td>{{.SystemMetrics.GoroutineCount}}</td></tr>
                <tr><td>Average Response Time</td><td>{{.SystemMetrics.ResponseTime}}</td></tr>
            </table>
        </div>

        <h2>Operation Statistics</h2>
        <table>
            <tr>
                <th>Operation</th>
                <th>Count</th>
                <th>Average Time</th>
                <th>Total Time</th>
                <th>Errors</th>
            </tr>
            {{range .OperationStats}}
            <tr>
                <td>{{.Name}}</td>
                <td>{{.Count}}</td>
                <td>{{.AverageTime}}</td>
                <td>{{.TotalTime}}</td>
                <td>0</td>
            </tr>
            {{end}}
        </table>


        {{if .OptimizationTips}}
        <h2>Optimization Recommendations</h2>
        {{range $i, $tip := .OptimizationTips}}
        <div class="tip">
            <h3>{{add $i 1}}. {{$tip.Suggestion}}</h3>
            <p><span class="impact-{{lower $tip.Impact}}">Impact: {{$tip.Impact}}</span> | Category: {{$tip.Category}}</p>
        </div>
        {{end}}
        {{end}}
    </div>
</body>
</html>`

	// Parse and execute template
	funcMap := template.FuncMap{
		"formatBytes": func(b interface{}) string {
			switch v := b.(type) {
			case int64:
				return formatBytes(v)
			case uint64:
				return formatBytes(mathutil.Uint64ToInt64(v))
			default:
				return "0 B"
			}
		},
		"add":   func(a, b int) int { return a + b },
		"lower": strings.ToLower,
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return nil
}

// generateFilename creates a standardized filename for export
func (pe *PerformanceExporter) generateFilename(format ExportFormat) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("performance_report_%s.%s", timestamp, string(format))
}

// GetDefaultPath returns the default export path
func (pe *PerformanceExporter) GetDefaultPath() string {
	return pe.defaultPath
}

// SetDefaultPath sets the default export path
func (pe *PerformanceExporter) SetDefaultPath(path string) {
	pe.defaultPath = path
	_ = os.MkdirAll(path, fileperms.DirUserOnly)
}

// BatchExportReports exports multiple performance reports in different formats
func (pe *PerformanceExporter) BatchExportReports(profiler *performance.Profiler, optimizer *performance.Optimizer, formats []ExportFormat) ([]*ExportResult, error) {
	results := make([]*ExportResult, 0, len(formats))

	for _, format := range formats {
		result, err := pe.ExportPerformanceReport(profiler, optimizer, format, "")
		if err != nil {
			result.Error = err
		}
		results = append(results, result)
	}

	return results, nil
}

// ExportComparison exports a comparison between two performance snapshots
func (pe *PerformanceExporter) ExportComparison(before, after PerformanceReportData, format ExportFormat, customPath string) (*ExportResult, error) {
	// This would implement comparison logic between two snapshots
	// For now, we'll just return an error indicating it's not implemented
	return nil, fmt.Errorf("comparison export not yet implemented")
}

// calculateCPUUsage calculates CPU usage from operation stats
func calculateCPUUsage(opStats []performance.OperationSummary) float64 {
	if len(opStats) == 0 {
		return 0.0
	}

	// Calculate CPU usage based on operation timings
	totalTime := time.Duration(0)
	for _, op := range opStats {
		totalTime += op.TotalTime
	}

	// Estimate CPU usage percentage (simplified)
	// This is a rough estimate based on operations per second
	cpuUsage := float64(totalTime.Nanoseconds()) / float64(time.Second.Nanoseconds()) * 100.0
	if cpuUsage > 100.0 {
		cpuUsage = 100.0
	}

	return cpuUsage
}

// calculateAvgResponseTime calculates average response time from operation stats
func calculateAvgResponseTime(opStats []performance.OperationSummary) time.Duration {
	if len(opStats) == 0 {
		return 0
	}

	totalTime := time.Duration(0)
	totalCount := int64(0)

	for _, op := range opStats {
		totalTime += op.TotalTime
		totalCount += op.Count
	}

	if totalCount == 0 {
		return 0
	}

	return totalTime / time.Duration(totalCount)
}
