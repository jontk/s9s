# Export & Reporting Guide

Export S9S data in multiple formats for analysis, reporting, and integration with other tools and workflows.

## Quick Export

### Basic Export Commands

```bash
# Export current view to CSV
:export csv

# Export selected jobs
/user:alice state:COMPLETED
:export --selected json

# Export with custom filename
:export csv --output my-jobs.csv
```

### One-Click Exports

| Key | Format | Description |
|-----|--------|--------------|
| `Ctrl+E` | CSV | Export to CSV format |
| `Ctrl+Shift+E` | JSON | Export to JSON format |
| `Alt+E` | Excel | Export to Excel format |
| `Ctrl+P` | PDF | Generate PDF report |

## Supported Formats

### Structured Data Formats

**CSV (Comma-Separated Values)**
- Best for: Spreadsheet analysis, data processing
- Features: Headers, custom delimiters, UTF-8 encoding
- Extensions: `.csv`, `.tsv`

**JSON (JavaScript Object Notation)**
- Best for: API integration, web applications
- Features: Nested data, metadata, schema validation
- Extensions: `.json`, `.jsonl` (JSON Lines)

**Excel (Microsoft Excel)**
- Best for: Business reports, formatted presentations
- Features: Multiple sheets, formatting, charts
- Extensions: `.xlsx`, `.xls`

**Parquet (Apache Parquet)**
- Best for: Big data analysis, columnar storage
- Features: Compression, schema evolution, fast queries
- Extensions: `.parquet`

### Report Formats

**PDF (Portable Document Format)**
- Best for: Reports, documentation, archival
- Features: Formatting, charts, custom templates
- Extensions: `.pdf`

**HTML (HyperText Markup Language)**
- Best for: Web reports, interactive content
- Features: Styling, links, embedded media
- Extensions: `.html`, `.htm`

**Markdown**
- Best for: Documentation, version control
- Features: Tables, formatting, compatibility
- Extensions: `.md`, `.markdown`

## Export Options

### Field Selection

Choose specific fields to export:

```bash
# Export specific job fields
:export csv --fields=JobID,User,State,Runtime

# Export all available fields
:export json --fields=all

# Export minimal fields
:export csv --fields=minimal

# Custom field list
:export excel --fields="Job ID,User,Partition,Start Time,End Time"
```

### Time Range Filters

```bash
# Export jobs from specific time period
:export csv --time-range="2023-12-01..2023-12-31"

# Export recent data
:export json --time-range="last-7d"

# Export with relative time
:export csv --submitted=">1h" --completed="<24h"
```

### Data Filtering

```bash
# Export with filters
:export csv --filter="user:alice state:COMPLETED"

# Complex filtering
:export json --filter="partition:gpu nodes:>4 runtime:>2h"

# Multiple conditions
:export excel --user=alice,bob --state=RUNNING,COMPLETED
```

## Advanced Reporting

### Job Reports

**Job Summary Report**:
```bash
:report job-summary --period=month --format=pdf
```

Includes:
- Total jobs by state
- Resource utilization
- User activity
- Queue wait times
- Success/failure rates

**User Activity Report**:
```bash
:report user-activity --users=alice,bob --period=week
```

Includes:
- Jobs submitted/completed per user
- Resource consumption
- Efficiency metrics
- Cost allocation

**Resource Utilization Report**:
```bash
:report utilization --partitions=gpu,cpu --format=excel
```

Includes:
- CPU/GPU utilization over time
- Memory usage patterns
- Node efficiency
- Queue backlog analysis

### Node Reports

**Node Health Report**:
```bash
:report node-health --nodes=node[001-100] --format=html
```

**Maintenance Report**:
```bash
:report maintenance --period=quarter --include-scheduled
```

### Custom Reports

Create custom report templates:

```yaml
# ~/.s9s/reports/custom-template.yaml
name: "Weekly HPC Report"
format: pdf
sections:
  - type: summary
    title: "Cluster Overview"
    metrics: [total_jobs, utilization, availability]
  - type: chart
    title: "Job Trends"
    chart_type: line
    data: job_counts_by_day
  - type: table
    title: "Top Users"
    data: user_statistics
    sort: jobs_submitted
    limit: 10
```

Generate custom reports:
```bash
:report custom --template=custom-template --output=weekly-report.pdf
```

## Automated Exports

### Scheduled Exports

Set up automatic data exports:

```bash
# Daily job export
:schedule daily "job-export" \
  ":export csv --filter='submitted:today' --output='/reports/jobs-{date}.csv'"

# Weekly utilization report
:schedule weekly "utilization-report" \
  ":report utilization --format=pdf --email=admin@example.com"

# Monthly user reports
:schedule monthly "user-reports" \
  ":report user-activity --all-users --format=excel --upload=s3://reports/"
```

### Export Automation

Automate exports with triggers:

```yaml
# ~/.s9s/automation/exports.yaml
triggers:
  job_completed:
    condition: "state == 'COMPLETED' and runtime > '24h'"
    action: "export"
    format: "json"
    destination: "webhook://analytics.example.com/job-data"

  maintenance_complete:
    condition: "node_state_change to 'IDLE' after 'MAINT'"
    action: "report"
    template: "maintenance-summary"
    email: "ops-team@example.com"
```

## Export Destinations

### Local Files

```bash
# Export to specific directory
:export csv --output="/data/exports/jobs.csv"

# Export with timestamp
:export json --output="jobs-{timestamp}.json"

# Export to user directory
:export excel --output="~/reports/cluster-report.xlsx"
```

### Cloud Storage

```bash
# AWS S3
:export csv --upload="s3://my-bucket/cluster-data/"

# Google Cloud Storage
:export json --upload="gs://analytics-bucket/s9s-data/"

# Azure Blob Storage
:export parquet --upload="https://account.blob.core.windows.net/container/"
```

### Database Integration

```bash
# Export to PostgreSQL
:export --database="postgresql://user:pass@host/db" --table="job_history"

# Export to InfluxDB
:export --influx="http://influx:8086/mydb" --measurement="slurm_jobs"

# Export to Elasticsearch
:export --elastic="http://elastic:9200/slurm-index"
```

### API Endpoints

```bash
# POST to webhook
:export json --webhook="https://api.example.com/slurm-data"

# Stream to Apache Kafka
:export jsonl --kafka="kafka:9092" --topic="slurm-events"

# Send to monitoring system
:export --prometheus="http://prometheus:9090/api/v1/receive"
```

## Export Configuration

### Default Settings

```yaml
# ~/.s9s/config.yaml
export:
  # Default format
  defaultFormat: csv

  # Default output directory
  outputDir: ~/s9s-exports

  # Include headers in CSV
  includeHeaders: true

  # Date format in filenames
  dateFormat: "2006-01-02"

  # Compression
  compress: true
  compressionFormat: gzip

  # Field formatting
  timeFormat: RFC3339
  durationFormat: seconds

  # Limits
  maxRecords: 1000000
  maxFileSize: 100MB
```

### Format-Specific Settings

```yaml
export:
  formats:
    csv:
      delimiter: ","
      quote: '"'
      encoding: utf-8
      lineEnding: unix

    json:
      indent: 2
      sortKeys: true
      includeSchema: true

    excel:
      worksheet: "S9S Data"
      autoWidth: true
      freezeHeader: true

    pdf:
      template: "default"
      margins: [20, 20, 20, 20]
      orientation: portrait
```

## Data Visualization

### Built-in Charts

Generate charts during export:

```bash
# Job state distribution pie chart
:export html --chart=pie --group-by=state

# Resource utilization over time
:export pdf --chart=line --x-axis=time --y-axis=utilization

# User activity bar chart
:export html --chart=bar --group-by=user --metric=job_count
```

### Integration with BI Tools

**Tableau**:
```bash
# Export Tableau data extract
:export tde --output=cluster-data.tde
```

**Power BI**:
```bash
# Export for Power BI
:export csv --powerbi-format --output=powerbi-data.csv
```

**Grafana**:
```bash
# Export to InfluxDB for Grafana
:export --influx=http://influx:8086/grafana --measurement=slurm
```

## Security and Privacy

### Data Sanitization

```bash
# Remove sensitive information
:export csv --sanitize --fields=JobID,State,Runtime

# Anonymize user data
:export json --anonymize-users --hash-method=sha256

# Filter sensitive partitions
:export csv --exclude-partitions=confidential,private
```

### Access Control

```yaml
export:
  security:
    requirePermission: true
    allowedFormats: [csv, json]
    maxRecordsPerUser: 10000
    auditExports: true
    restrictFields: [script_path, environment]
```

### Encryption

```bash
# Encrypt exports
:export csv --encrypt --key-file=~/.s9s/export.key

# Sign exports
:export json --sign --cert-file=~/.s9s/export.crt
```

## Best Practices

### Performance Optimization

1. **Use filters** to limit data volume
2. **Choose appropriate formats** (Parquet for large datasets)
3. **Export incrementally** for large historical data
4. **Compress exports** to save storage and bandwidth
5. **Use streaming** for real-time data

### Data Management

1. **Version your exports** with timestamps
2. **Archive old exports** regularly
3. **Document export schemas** for consistency
4. **Validate exported data** before use
5. **Monitor export jobs** for failures

### Integration

1. **Standardize formats** across tools
2. **Use APIs** instead of files when possible
3. **Implement error handling** in downstream systems
4. **Set up monitoring** for data pipelines
5. **Test exports regularly** to ensure quality

## Troubleshooting

### Common Issues

**Export fails with "Permission denied"**:
- Check file/directory permissions
- Verify export destination accessibility
- Ensure sufficient disk space

**Large exports timeout**:
- Use smaller time ranges
- Export in batches
- Increase timeout settings
- Use streaming export

**Invalid date formats**:
- Check date format configuration
- Verify timezone settings
- Use ISO 8601 format for compatibility

### Debug Mode

```bash
# Enable export debugging
:config set export.debug true

# Verbose export logging
:export csv --debug --verbose

# Dry run export
:export json --dry-run --output=test.json
```

## Next Steps

- Learn [Batch Operations](./batch-operations.md) for bulk exports
- Explore [Node Operations](./node-operations.md) for node data analysis
- Explore [Advanced Filtering](../filtering.md) to refine export data
