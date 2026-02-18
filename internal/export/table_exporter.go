package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/fileperms"
	"github.com/jontk/s9s/internal/security"
)

// TableData holds tabular data for export (headers + rows + metadata).
type TableData struct {
	Title     string     // e.g. "Jobs", "Nodes", "Partitions"
	Headers   []string   // column names
	Rows      [][]string // raw (uncolored) row values
	ExportedAt time.Time
}

// TableExporter exports tabular data to various file formats.
type TableExporter struct {
	defaultPath string
}

// NewTableExporter creates a new TableExporter.
// If defaultPath is empty it defaults to ~/slurm_exports.
func NewTableExporter(defaultPath string) *TableExporter {
	if defaultPath == "" {
		homeDir, _ := os.UserHomeDir()
		defaultPath = filepath.Join(homeDir, "slurm_exports")
	}
	_ = os.MkdirAll(defaultPath, fileperms.DirUserOnly)
	return &TableExporter{defaultPath: defaultPath}
}

// SetDefaultPath updates the default export directory.
func (e *TableExporter) SetDefaultPath(path string) {
	e.defaultPath = path
	_ = os.MkdirAll(path, fileperms.DirUserOnly)
}

// Export writes td to a file and returns a Result.
// If customPath is non-empty it is used as the full output path;
// otherwise a timestamped filename is generated in defaultPath.
func (e *TableExporter) Export(td *TableData, format ExportFormat, customPath string) (*ExportResult, error) {
	if td.ExportedAt.IsZero() {
		td.ExportedAt = time.Now()
	}

	result := &ExportResult{
		Format:    format,
		Timestamp: td.ExportedAt,
	}

	filename := e.generateFilename(td.Title, format)
	outputPath := e.determinePath(customPath, filename)

	validPath, err := e.validatePath(result, outputPath)
	if err != nil {
		return result, err
	}

	if err := e.ensureDir(result, validPath); err != nil {
		return result, err
	}

	if err := e.writeByFormat(result, td, format, validPath); err != nil {
		return result, err
	}

	if stat, err := os.Stat(validPath); err == nil {
		result.Size = stat.Size()
	}
	result.Success = true
	return result, nil
}

func (e *TableExporter) generateFilename(title string, format ExportFormat) string {
	clean := strings.ToLower(strings.ReplaceAll(title, " ", "_"))
	ts := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s.%s", clean, ts, string(format))
}

func (e *TableExporter) determinePath(customPath, filename string) string {
	if customPath != "" {
		return customPath
	}
	return filepath.Join(e.defaultPath, filename)
}

func (e *TableExporter) validatePath(result *ExportResult, outputPath string) (string, error) {
	homeDir, _ := os.UserHomeDir()
	validPath, err := security.ValidatePathWithinBase(outputPath, e.defaultPath)
	if err != nil && homeDir != "" {
		validPath, err = security.ValidatePathWithinBase(outputPath, homeDir)
	}
	if err != nil {
		result.Error = fmt.Errorf("invalid export path %q: %w", outputPath, err)
		return "", result.Error
	}
	result.FilePath = validPath
	return validPath, nil
}

func (e *TableExporter) ensureDir(result *ExportResult, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, fileperms.DirUserOnly); err != nil {
		result.Error = fmt.Errorf("failed to create directory %s: %w", dir, err)
		return result.Error
	}
	return nil
}

func (e *TableExporter) writeByFormat(result *ExportResult, td *TableData, format ExportFormat, path string) error {
	var err error
	switch format {
	case FormatText:
		err = e.writeText(td, path)
	case FormatJSON:
		err = e.writeJSON(td, path)
	case FormatCSV:
		err = e.writeCSV(td, path)
	case FormatMarkdown:
		err = e.writeMarkdown(td, path)
	case FormatHTML:
		err = e.writeHTML(td, path)
	default:
		err = fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		result.Error = err
	}
	return err
}

// writeText writes a plain-text table.
func (e *TableExporter) writeText(td *TableData, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Compute column widths.
	widths := make([]int, len(td.Headers))
	for i, h := range td.Headers {
		widths[i] = len(h)
	}
	for _, row := range td.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	separator := buildSeparator(widths)

	header := fmt.Sprintf("%s Export\n", td.Title)
	header += fmt.Sprintf("Exported at: %s\n", td.ExportedAt.Format("2006-01-02 15:04:05"))
	header += fmt.Sprintf("Total records: %d\n\n", len(td.Rows))
	if _, err := fmt.Fprint(f, header); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(f, separator); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, buildRow(td.Headers, widths)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, separator); err != nil {
		return err
	}
	for _, row := range td.Rows {
		if _, err := fmt.Fprintln(f, buildRow(row, widths)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(f, separator); err != nil {
		return err
	}
	return nil
}

func buildSeparator(widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("-", w+2)
	}
	return "+" + strings.Join(parts, "+") + "+"
}

func buildRow(cells []string, widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts[i] = fmt.Sprintf(" %-*s ", w, cell)
	}
	return "|" + strings.Join(parts, "|") + "|"
}

// writeCSV writes a CSV file.
func (e *TableExporter) writeCSV(td *TableData, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(td.Headers); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, row := range td.Rows {
		if err := w.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}
	return nil
}

// writeJSON writes a JSON array of objects keyed by header name.
func (e *TableExporter) writeJSON(td *TableData, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	type envelope struct {
		Title      string              `json:"title"`
		ExportedAt string              `json:"exported_at"`
		Total      int                 `json:"total"`
		Records    []map[string]string `json:"records"`
	}

	records := make([]map[string]string, len(td.Rows))
	for i, row := range td.Rows {
		m := make(map[string]string, len(td.Headers))
		for j, h := range td.Headers {
			if j < len(row) {
				m[h] = row[j]
			}
		}
		records[i] = m
	}

	env := envelope{
		Title:      td.Title,
		ExportedAt: td.ExportedAt.Format(time.RFC3339),
		Total:      len(td.Rows),
		Records:    records,
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(env); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}

// writeMarkdown writes a Markdown table.
func (e *TableExporter) writeMarkdown(td *TableData, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	fmt.Fprintf(f, "# %s Export\n\n", td.Title)
	fmt.Fprintf(f, "_Exported at: %s — %d records_\n\n", td.ExportedAt.Format("2006-01-02 15:04:05"), len(td.Rows))

	// Header row
	fmt.Fprintf(f, "| %s |\n", strings.Join(td.Headers, " | "))
	// Separator
	seps := make([]string, len(td.Headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Fprintf(f, "| %s |\n", strings.Join(seps, " | "))
	// Data rows
	for _, row := range td.Rows {
		// Pad row to match header count
		cells := make([]string, len(td.Headers))
		for i := range cells {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		fmt.Fprintf(f, "| %s |\n", strings.Join(cells, " | "))
	}
	return nil
}

// writeHTML writes an HTML table.
func (e *TableExporter) writeHTML(td *TableData, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	const tmplStr = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}} Export</title>
  <style>
    body{font-family:monospace;background:#1e1e1e;color:#d4d4d4;margin:20px}
    h1{color:#569cd6}
    .meta{color:#808080;margin-bottom:16px}
    table{border-collapse:collapse;width:100%}
    th{background:#2d2d30;color:#9cdcfe;padding:8px 12px;text-align:left;border:1px solid #3c3c3c}
    td{padding:6px 12px;border:1px solid #3c3c3c}
    tr:nth-child(even){background:#252526}
  </style>
</head>
<body>
  <h1>{{.Title}} Export</h1>
  <p class="meta">Exported at: {{.ExportedAt.Format "2006-01-02 15:04:05"}} &mdash; {{len .Rows}} records</p>
  <table>
    <thead><tr>{{range .Headers}}<th>{{.}}</th>{{end}}</tr></thead>
    <tbody>
      {{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>`

	tmpl, err := template.New("table").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	if err := tmpl.Execute(f, td); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	return nil
}
