package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/fileperms"
	"github.com/jontk/s9s/internal/security"
)

// Format represents different output export formats
type Format string

const (
	// FormatText is the text export format.
	FormatText Format = "txt"
	// FormatJSON is the JSON export format.
	FormatJSON Format = "json"
	// FormatCSV is the CSV export format.
	FormatCSV Format = "csv"
	// FormatMarkdown is the Markdown export format.
	FormatMarkdown Format = "md"
	// FormatHTML is the HTML export format.
	FormatHTML Format = "html"
)

type ExportFormat = Format

// JobOutputExporter handles exporting job output to various formats
type JobOutputExporter struct {
	defaultPath string
}

// NewJobOutputExporter creates a new job output exporter
func NewJobOutputExporter(defaultPath string) *JobOutputExporter {
	if defaultPath == "" {
		homeDir, _ := os.UserHomeDir()
		defaultPath = filepath.Join(homeDir, "slurm_exports")
	}

	// Ensure export directory exists
	_ = os.MkdirAll(defaultPath, fileperms.DirUserOnly)

	return &JobOutputExporter{
		defaultPath: defaultPath,
	}
}

// JobOutputData represents the structure of job output data
type JobOutputData struct {
	JobID       string    `json:"job_id"`
	JobName     string    `json:"job_name"`
	OutputType  string    `json:"output_type"` // "stdout" or "stderr"
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
	ExportedBy  string    `json:"exported_by"`
	ExportTime  time.Time `json:"export_time"`
	ContentSize int       `json:"content_size_bytes"`
}

// Result contains information about the export operation
type Result struct {
	FilePath  string
	Format    Format
	Size      int64
	Success   bool
	Error     error
	Timestamp time.Time
}

type ExportResult = Result

// ExportJobOutput exports job output to a file in the specified format
func (e *JobOutputExporter) ExportJobOutput(jobID, jobName, outputType, content string) (string, error) {
	// Create JobOutputData for internal use
	data := JobOutputData{
		JobID:       jobID,
		JobName:     jobName,
		OutputType:  outputType,
		Content:     content,
		Timestamp:   time.Now(),
		ExportedBy:  "S9s",
		ExportTime:  time.Now(),
		ContentSize: len(content),
	}

	result, err := e.Export(&data, FormatText, "")
	if err != nil {
		return "", err
	}
	return result.FilePath, nil
}

// Export exports job output to a file in the specified format
func (e *JobOutputExporter) Export(data *JobOutputData, format ExportFormat, customPath string) (*ExportResult, error) {
	result := &Result{
		Format:    format,
		Timestamp: time.Now(),
	}

	// Generate filename and determine output path
	filename := e.generateFilename(data.JobID, data.JobName, data.OutputType, format)
	outputPath := e.determinePath(customPath, filename)

	// Validate path
	validPath, err := e.validateAndSetPath(result, outputPath)
	if err != nil {
		return result, err
	}

	// Create directory
	if err := e.createExportDirectory(result, validPath); err != nil {
		return result, err
	}

	// Export by format
	if err := e.exportByFormat(result, data, format, validPath); err != nil {
		return result, err
	}

	// Get file size
	e.updateFileSize(result, validPath)

	result.Success = true
	return result, nil
}

// determinePath returns the path to export to
func (e *JobOutputExporter) determinePath(customPath, filename string) string {
	if customPath != "" {
		return customPath
	}
	return filepath.Join(e.defaultPath, filename)
}

// validateAndSetPath validates the output path is within safe directories
func (e *JobOutputExporter) validateAndSetPath(result *Result, outputPath string) (string, error) {
	homeDir, _ := os.UserHomeDir()
	validPath, validationErr := security.ValidatePathWithinBase(outputPath, e.defaultPath)
	if validationErr != nil && homeDir != "" {
		// Try validating against home directory as fallback
		validPath, validationErr = security.ValidatePathWithinBase(outputPath, homeDir)
	}
	if validationErr != nil {
		result.Error = fmt.Errorf("invalid export path %q: %w", outputPath, validationErr)
		return "", result.Error
	}
	result.FilePath = validPath
	return validPath, nil
}

// createExportDirectory ensures the export directory exists
func (e *JobOutputExporter) createExportDirectory(result *Result, outputPath string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, fileperms.DirUserOnly); err != nil {
		result.Error = fmt.Errorf("failed to create directory %s: %w", dir, err)
		return result.Error
	}
	return nil
}

// exportByFormat dispatches to the appropriate export format handler
func (e *JobOutputExporter) exportByFormat(result *Result, data *JobOutputData, format ExportFormat, outputPath string) error {
	var err error
	switch format {
	case FormatText:
		err = e.exportText(data, outputPath)
	case FormatJSON:
		err = e.exportJSON(data, outputPath)
	case FormatCSV:
		err = e.exportCSV(data, outputPath)
	case FormatMarkdown:
		err = e.exportMarkdown(data, outputPath)
	case FormatHTML:
		err = e.exportHTML(data, outputPath)
	default:
		err = fmt.Errorf("unsupported export format: %s", format)
	}
	if err != nil {
		result.Error = err
	}
	return err
}

// updateFileSize retrieves and updates the file size in result
func (e *JobOutputExporter) updateFileSize(result *Result, outputPath string) {
	if stat, err := os.Stat(outputPath); err == nil {
		result.Size = stat.Size()
	}
}

// exportText exports job output as plain text
func (e *JobOutputExporter) exportText(data *JobOutputData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write header
	header := fmt.Sprintf("Job Output Export\n"+
		"=================\n"+
		"Job ID: %s\n"+
		"Job Name: %s\n"+
		"Output Type: %s\n"+
		"Export Time: %s\n"+
		"Content Size: %d bytes\n"+
		"\n"+
		"Output Content:\n"+
		"---------------\n\n",
		data.JobID, data.JobName, data.OutputType,
		data.ExportTime.Format("2006-01-02 15:04:05"),
		data.ContentSize)

	if _, err := file.WriteString(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write content
	if _, err := file.WriteString(data.Content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// exportJSON exports job output as JSON
func (e *JobOutputExporter) exportJSON(data *JobOutputData, outputPath string) error {
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

// exportCSV exports job output as CSV (useful for tabular data or logs)
func (e *JobOutputExporter) exportCSV(data *JobOutputData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Job ID", "Job Name", "Output Type", "Export Time", "Content Size", "Line Number", "Content"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Pre-format common values to avoid repeated allocations
	exportTime := data.ExportTime.Format("2006-01-02 15:04:05")
	contentSizeStr := strconv.Itoa(data.ContentSize)

	// Reuse record slice to avoid allocations
	record := make([]string, 7)
	record[0] = data.JobID
	record[1] = data.JobName
	record[2] = data.OutputType
	record[3] = exportTime
	record[4] = contentSizeStr

	// Process content line by line without creating large intermediate slices
	content := data.Content
	lineNum := 1
	start := 0

	for i := 0; i <= len(content); i++ {
		// Found newline or end of content
		if i == len(content) || content[i] == '\n' {
			// Extract line without allocating new string (using slice of original)
			line := content[start:i]

			// Update record for this line - use efficient integer to string conversion
			record[5] = strconv.Itoa(lineNum)
			record[6] = line

			if err := writer.Write(record); err != nil {
				return fmt.Errorf("failed to write CSV record: %w", err)
			}

			lineNum++
			start = i + 1
		}
	}

	return nil
}

// exportMarkdown exports job output as Markdown
func (e *JobOutputExporter) exportMarkdown(data *JobOutputData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write Markdown content
	markdown := fmt.Sprintf("# Job Output Export\n\n"+
		"## Job Information\n\n"+
		"- **Job ID:** %s\n"+
		"- **Job Name:** %s\n"+
		"- **Output Type:** %s\n"+
		"- **Export Time:** %s\n"+
		"- **Content Size:** %d bytes\n\n"+
		"## Output Content\n\n"+
		"```\n%s\n```\n",
		data.JobID, data.JobName, data.OutputType,
		data.ExportTime.Format("2006-01-02 15:04:05"),
		data.ContentSize, data.Content)

	if _, err := file.WriteString(markdown); err != nil {
		return fmt.Errorf("failed to write markdown: %w", err)
	}

	return nil
}

// generateFilename creates a standardized filename for export
func (e *JobOutputExporter) generateFilename(jobID, jobName, outputType string, format ExportFormat) string {
	// Clean job name for filename
	cleanJobName := strings.ReplaceAll(jobName, " ", "_")
	cleanJobName = strings.ReplaceAll(cleanJobName, "/", "_")
	cleanJobName = strings.ReplaceAll(cleanJobName, "\\", "_")

	timestamp := time.Now().Format("20060102_150405")

	return fmt.Sprintf("job_%s_%s_%s_%s.%s",
		jobID, cleanJobName, outputType, timestamp, string(format))
}

// GetSupportedFormats returns all supported export formats
func (e *JobOutputExporter) GetSupportedFormats() []ExportFormat {
	return []ExportFormat{FormatText, FormatJSON, FormatCSV, FormatMarkdown, FormatHTML}
}

// GetDefaultPath returns the default export path
func (e *JobOutputExporter) GetDefaultPath() string {
	return e.defaultPath
}

// SetDefaultPath sets the default export path
func (e *JobOutputExporter) SetDefaultPath(path string) {
	e.defaultPath = path
	_ = os.MkdirAll(path, fileperms.DirUserOnly)
}

// BatchExport exports multiple job outputs in the specified format
func (e *JobOutputExporter) BatchExport(jobs []*JobOutputData, format ExportFormat, basePath string) ([]*ExportResult, error) {
	return e.BatchExportWithCallback(jobs, format, basePath, nil)
}

// BatchExportWithCallback exports multiple job outputs with optional progress callback
func (e *JobOutputExporter) BatchExportWithCallback(jobs []*JobOutputData, format ExportFormat, _ string, progressCallback func(current, total int, jobID string)) ([]*ExportResult, error) {
	results := make([]*ExportResult, 0, len(jobs))

	for i, job := range jobs {
		// Call progress callback if provided
		if progressCallback != nil {
			progressCallback(i+1, len(jobs), job.JobID)
		}

		result, err := e.Export(job, format, "")
		if err != nil {
			result.Error = err
		}
		results = append(results, result)

		// Force garbage collection of job content after processing to reduce memory usage
		job.Content = "" // Release the large content string
	}

	return results, nil
}

// StreamingBatchExport exports jobs one at a time to minimize memory usage
func (e *JobOutputExporter) StreamingBatchExport(jobProvider func() (*JobOutputData, bool), format ExportFormat, _ string, progressCallback func(current int, jobID string)) ([]*ExportResult, error) {
	var results []*ExportResult
	jobCount := 0

	for {
		job, hasMore := jobProvider()
		if !hasMore {
			break
		}

		jobCount++
		if progressCallback != nil {
			progressCallback(jobCount, job.JobID)
		}

		result, err := e.Export(job, format, "")
		if err != nil {
			result.Error = err
		}
		results = append(results, result)

		// Job content is automatically freed when job goes out of scope
	}

	return results, nil
}

// ExportSummary creates a summary of exported files
func (e *JobOutputExporter) ExportSummary(results []*ExportResult) string {
	successful := 0
	failed := 0
	var totalSize int64

	for _, result := range results {
		if result.Success {
			successful++
			totalSize += result.Size
		} else {
			failed++
		}
	}

	return fmt.Sprintf("Export Summary:\n"+
		"- Total Files: %d\n"+
		"- Successful: %d\n"+
		"- Failed: %d\n"+
		"- Total Size: %s\n"+
		"- Export Path: %s",
		len(results), successful, failed,
		formatBytes(totalSize), e.defaultPath)
}

// exportHTML exports job output as HTML
func (e *JobOutputExporter) exportHTML(data *JobOutputData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// HTML template
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>Job Output - {{.JobID}}</title>
    <style>
        body { font-family: 'Courier New', monospace; margin: 20px; background-color: #1e1e1e; color: #d4d4d4; }
        .container { background-color: #252526; padding: 20px; border-radius: 8px; }
        .header { background-color: #2d2d30; padding: 15px; border-radius: 4px; margin-bottom: 20px; }
        h1 { color: #569cd6; margin: 0; font-size: 24px; }
        .meta { color: #808080; margin-top: 10px; }
        .meta-item { margin-right: 20px; }
        .output-type { color: #ce9178; font-weight: bold; }
        .output-content { background-color: #1e1e1e; padding: 15px; border-radius: 4px; overflow-x: auto; white-space: pre-wrap; word-wrap: break-word; font-size: 14px; line-height: 1.4; }
        .timestamp { color: #608b4e; }
        .error { color: #f48771; }
        .warning { color: #dcdcaa; }
        .info { color: #9cdcfe; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Job Output Export</h1>
            <div class="meta">
                <span class="meta-item"><strong>Job ID:</strong> {{.JobID}}</span>
                <span class="meta-item"><strong>Job Name:</strong> {{.JobName}}</span>
                <span class="meta-item"><strong>Output Type:</strong> <span class="output-type">{{.OutputType}}</span></span>
            </div>
            <div class="meta">
                <span class="meta-item"><strong>Export Time:</strong> <span class="timestamp">{{.ExportTime.Format "2006-01-02 15:04:05"}}</span></span>
                <span class="meta-item"><strong>Content Size:</strong> {{.ContentSize}} bytes</span>
            </div>
        </div>
        <div class="output-content">{{.Content}}</div>
    </div>
</body>
</html>`

	// Parse and execute template
	tmpl, err := template.New("joboutput").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return nil
}

// formatBytes formats byte size in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
