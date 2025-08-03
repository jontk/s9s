package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ExportFormat represents different output export formats
type ExportFormat string

const (
	FormatText     ExportFormat = "txt"
	FormatJSON     ExportFormat = "json"
	FormatCSV      ExportFormat = "csv"
	FormatMarkdown ExportFormat = "md"
)

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
	os.MkdirAll(defaultPath, 0755)
	
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

// ExportResult contains information about the export operation
type ExportResult struct {
	FilePath    string
	Format      ExportFormat
	Size        int64
	Success     bool
	Error       error
	Timestamp   time.Time
}

// ExportJobOutput exports job output to a file in the specified format
func (e *JobOutputExporter) ExportJobOutput(data JobOutputData, format ExportFormat, customPath string) (*ExportResult, error) {
	result := &ExportResult{
		Format:    format,
		Timestamp: time.Now(),
	}

	// Generate filename
	filename := e.generateFilename(data.JobID, data.JobName, data.OutputType, format)
	
	// Determine output path
	var outputPath string
	if customPath != "" {
		outputPath = customPath
	} else {
		outputPath = filepath.Join(e.defaultPath, filename)
	}
	
	result.FilePath = outputPath

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create directory %s: %w", dir, err)
		return result, result.Error
	}

	// Export based on format
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

// exportText exports job output as plain text
func (e *JobOutputExporter) exportText(data JobOutputData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

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
func (e *JobOutputExporter) exportJSON(data JobOutputData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// exportCSV exports job output as CSV (useful for tabular data or logs)
func (e *JobOutputExporter) exportCSV(data JobOutputData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

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
func (e *JobOutputExporter) exportMarkdown(data JobOutputData, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

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
	return []ExportFormat{FormatText, FormatJSON, FormatCSV, FormatMarkdown}
}

// GetDefaultPath returns the default export path
func (e *JobOutputExporter) GetDefaultPath() string {
	return e.defaultPath
}

// SetDefaultPath sets the default export path
func (e *JobOutputExporter) SetDefaultPath(path string) {
	e.defaultPath = path
	os.MkdirAll(path, 0755)
}

// BatchExport exports multiple job outputs in the specified format
func (e *JobOutputExporter) BatchExport(jobs []JobOutputData, format ExportFormat, basePath string) ([]*ExportResult, error) {
	return e.BatchExportWithCallback(jobs, format, basePath, nil)
}

// BatchExportWithCallback exports multiple job outputs with optional progress callback
func (e *JobOutputExporter) BatchExportWithCallback(jobs []JobOutputData, format ExportFormat, basePath string, progressCallback func(current, total int, jobID string)) ([]*ExportResult, error) {
	results := make([]*ExportResult, 0, len(jobs))
	
	for i, job := range jobs {
		// Call progress callback if provided
		if progressCallback != nil {
			progressCallback(i+1, len(jobs), job.JobID)
		}
		
		result, err := e.ExportJobOutput(job, format, "")
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
func (e *JobOutputExporter) StreamingBatchExport(jobProvider func() (JobOutputData, bool), format ExportFormat, basePath string, progressCallback func(current int, jobID string)) ([]*ExportResult, error) {
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
		
		result, err := e.ExportJobOutput(job, format, "")
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