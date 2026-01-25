package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	texttemplate "text/template"
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

// Text template for performance report
const textReportTemplate = `S9s Performance Report
=====================

Generated: {{.GeneratedAt}}
Period: {{.ReportPeriod}}

System Metrics
--------------
CPU Usage: {{printf "%.2f" .SystemMetrics.CPUUsage}}%
Memory Usage: {{printf "%.2f" .SystemMetrics.MemoryUsage}}% ({{.MemoryUsedFormatted}} / {{.MemoryTotalFormatted}})
Goroutines: {{.SystemMetrics.GoroutineCount}}
Response Time (avg): {{.SystemMetrics.ResponseTime}}

Operation Statistics
-------------------
{{printf "%-30s %10s %15s %15s" "Operation" "Count" "Avg Time" "Total Time"}}
{{.Separator}}
{{range .OperationStats}}{{printf "%-30s %10d %15v %15v" .Name .Count .AverageTime .TotalTime}}
{{end}}
{{if .OptimizationTips}}
Optimization Recommendations
---------------------------
{{range $i, $tip := .OptimizationTips}}{{add $i 1}}. {{$tip.Suggestion}}
   Impact: {{$tip.Impact}} | Category: {{$tip.Category}}

{{end}}{{end}}`

// Markdown template for performance report
const markdownReportTemplate = `# S9s Performance Report

**Generated:** {{.GeneratedAt}}
**Period:** {{.ReportPeriod}}

## System Metrics

| Metric | Value |
|--------|-------|
| CPU Usage | {{printf "%.2f" .SystemMetrics.CPUUsage}}% |
| Memory Usage | {{printf "%.2f" .SystemMetrics.MemoryUsage}}% |
| Memory Used | {{.MemoryUsedFormatted}} |
| Memory Total | {{.MemoryTotalFormatted}} |
| Goroutines | {{.SystemMetrics.GoroutineCount}} |
| Avg Response Time | {{.SystemMetrics.ResponseTime}} |

## Operation Statistics

| Operation | Count | Avg Time | Total Time |
|-----------|-------|----------|------------|
{{range .OperationStats}}| {{.Name}} | {{.Count}} | {{.AverageTime}} | {{.TotalTime}} |
{{end}}
{{if .OptimizationTips}}## Optimization Recommendations

{{range $i, $tip := .OptimizationTips}}### {{add $i 1}}. {{$tip.Suggestion}}

- **Impact:** {{$tip.Impact}}
- **Category:** {{$tip.Category}}

{{end}}{{end}}`

// TemplateData wraps PerformanceReportData with additional formatting
type TemplateData struct {
	*PerformanceReportData
	MemoryUsedFormatted   string
	MemoryTotalFormatted  string
	Separator             string
	GeneratedAt           string
}

// ExportPerformanceReport exports a performance report in the specified format
func (pe *PerformanceExporter) ExportPerformanceReport(profiler *performance.Profiler, optimizer *performance.Optimizer, format ExportFormat, customPath string) (*ExportResult, error) {
	result := &ExportResult{
		Format:    format,
		Timestamp: time.Now(),
	}

	// Collect and build report data
	data := pe.buildPerformanceReportData(profiler, optimizer)

	// Determine and validate output path
	outputPath, err := pe.determineAndValidateOutputPath(format, customPath)
	if err != nil {
		result.Error = err
		return result, err
	}
	result.FilePath = outputPath

	// Ensure directory exists
	if err := pe.ensureExportDirectory(outputPath); err != nil {
		result.Error = err
		return result, err
	}

	// Export based on format
	if err := pe.exportByFormat(&data, outputPath, format); err != nil {
		result.Error = err
		return result, err
	}

	// Get file size and mark success
	if stat, err := os.Stat(outputPath); err == nil {
		result.Size = stat.Size()
	}

	result.Success = true
	return result, nil
}

func (pe *PerformanceExporter) buildPerformanceReportData(profiler *performance.Profiler, optimizer *performance.Optimizer) PerformanceReportData {
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

	return PerformanceReportData{
		GeneratedAt:      time.Now(),
		ReportPeriod:     "Last 24 hours",
		SystemMetrics:    systemMetrics,
		OperationStats:   opStats,
		OptimizationTips: recommendations,
	}
}

func (pe *PerformanceExporter) determineAndValidateOutputPath(format ExportFormat, customPath string) (string, error) {
	// Determine base path
	var outputPath string
	if customPath != "" {
		outputPath = customPath
	} else {
		outputPath = filepath.Join(pe.defaultPath, pe.generateFilename(format))
	}

	// Validate output path is within safe directory
	homeDir, _ := os.UserHomeDir()
	validPath, validationErr := security.ValidatePathWithinBase(outputPath, pe.defaultPath)
	if validationErr != nil && homeDir != "" {
		// Try validating against home directory as fallback
		validPath, validationErr = security.ValidatePathWithinBase(outputPath, homeDir)
	}
	if validationErr != nil {
		return "", fmt.Errorf("invalid export path %q: %w", outputPath, validationErr)
	}

	return validPath, nil
}

func (pe *PerformanceExporter) ensureExportDirectory(outputPath string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, fileperms.DirUserOnly); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}

func (pe *PerformanceExporter) exportByFormat(data *PerformanceReportData, outputPath string, format ExportFormat) error {
	switch format {
	case FormatText:
		return pe.exportText(data, outputPath)
	case FormatJSON:
		return pe.exportJSON(data, outputPath)
	case FormatCSV:
		return pe.exportCSV(data, outputPath)
	case FormatMarkdown:
		return pe.exportMarkdown(data, outputPath)
	case FormatHTML:
		return pe.exportHTML(data, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportText exports performance report as plain text
func (pe *PerformanceExporter) exportText(data *PerformanceReportData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Prepare template data
	templateData := TemplateData{
		PerformanceReportData: data,
		MemoryUsedFormatted:   formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryUsed)),
		MemoryTotalFormatted:  formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryTotal)),
		Separator:             strings.Repeat("-", 80),
		GeneratedAt:           data.GeneratedAt.Format("2006-01-02 15:04:05"),
	}

	// Parse and execute template
	tmpl, err := texttemplate.New("textReport").Funcs(texttemplate.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(textReportTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl.Execute(file, templateData)
}

// exportJSON exports performance report as JSON
func (pe *PerformanceExporter) exportJSON(data *PerformanceReportData, outputPath string) error {
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
func (pe *PerformanceExporter) exportCSV(data *PerformanceReportData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write sections in order
	if err := pe.writeCSVMetadata(writer, data); err != nil {
		return err
	}
	if err := pe.writeCSVSystemMetrics(writer, data); err != nil {
		return err
	}
	if err := pe.writeCSVOperationStats(writer, data); err != nil {
		return err
	}

	return nil
}

// writeCSVMetadata writes metadata section to CSV
func (pe *PerformanceExporter) writeCSVMetadata(writer *csv.Writer, data *PerformanceReportData) error {
	rows := [][]string{
		{"Report Type", "Performance Report"},
		{"Generated", data.GeneratedAt.Format("2006-01-02 15:04:05")},
		{"Period", data.ReportPeriod},
		{}, // Empty line
	}

	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// writeCSVSystemMetrics writes system metrics section to CSV
func (pe *PerformanceExporter) writeCSVSystemMetrics(writer *csv.Writer, data *PerformanceReportData) error {
	rows := [][]string{
		{"System Metrics"},
		{"Metric", "Value"},
		{"CPU Usage", fmt.Sprintf("%.2f%%", data.SystemMetrics.CPUUsage)},
		{"Memory Usage", fmt.Sprintf("%.2f%%", data.SystemMetrics.MemoryUsage)},
		{"Memory Used", formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryUsed))},
		{"Memory Total", formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryTotal))},
		{"Goroutines", fmt.Sprintf("%d", data.SystemMetrics.GoroutineCount)},
		{"Response Time", data.SystemMetrics.ResponseTime.String()},
		{}, // Empty line
	}

	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// writeCSVOperationStats writes operation statistics section to CSV
func (pe *PerformanceExporter) writeCSVOperationStats(writer *csv.Writer, data *PerformanceReportData) error {
	if err := writer.Write([]string{"Operation Statistics"}); err != nil {
		return err
	}
	if err := writer.Write([]string{"Operation", "Count", "Average Time", "Total Time"}); err != nil {
		return err
	}

	for _, op := range data.OperationStats {
		row := []string{
			op.Name,
			fmt.Sprintf("%d", op.Count),
			op.AverageTime.String(),
			op.TotalTime.String(),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportMarkdown exports performance report as Markdown
func (pe *PerformanceExporter) exportMarkdown(data *PerformanceReportData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Prepare template data
	templateData := TemplateData{
		PerformanceReportData: data,
		MemoryUsedFormatted:   formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryUsed)),
		MemoryTotalFormatted:  formatBytes(mathutil.Uint64ToInt64(data.SystemMetrics.MemoryTotal)),
		GeneratedAt:           data.GeneratedAt.Format("2006-01-02 15:04:05"),
	}

	// Parse and execute template
	tmpl, err := texttemplate.New("markdownReport").Funcs(texttemplate.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(markdownReportTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl.Execute(file, templateData)
}

// exportHTML exports performance report as HTML
func (pe *PerformanceExporter) exportHTML(data *PerformanceReportData, outputPath string) error {
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
	funcMap := htmltemplate.FuncMap{
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

	tmpl, err := htmltemplate.New("report").Funcs(funcMap).Parse(htmlTemplate)
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
func (pe *PerformanceExporter) ExportComparison(_, _ *PerformanceReportData, _ ExportFormat, _ string) (*ExportResult, error) {
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
