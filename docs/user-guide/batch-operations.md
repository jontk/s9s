# Batch Operations Guide

Efficiently manage multiple jobs, nodes, and resources simultaneously with S9S's powerful batch operation capabilities.

## Overview

Batch operations allow you to:
- Manage multiple jobs with a single command
- Apply operations to filtered resources
- Perform bulk maintenance tasks
- Execute commands across node ranges
- Automate repetitive workflows

## Selection Methods

### Visual Selection

Use visual selection mode to choose multiple items:

| Key | Action | Description |
|-----|--------|-------------|
| `Space` | Toggle selection | Select/deselect current item |
| `v` | Visual mode | Enter visual selection mode |
| `V` | Visual line mode | Select entire rows |
| `Ctrl+A` | Select all | Select all visible items |
| `Ctrl+D` | Deselect all | Clear all selections |
| `Ctrl+I` | Invert selection | Invert current selection |

### Filter-Based Selection

Select items using filters:

```bash
# Select all failed jobs
/state:FAILED

# Select all idle GPU nodes
/state:idle features:gpu

# Select jobs by user
/user:alice

# Select nodes in specific rack
/node:rack[01-04]*
```

### Range Selection

Select ranges of items:

```bash
# Job ID ranges
:select jobs 1000-2000

# Node ranges
:select nodes node[001-100]

# Time-based ranges
:select jobs --submitted=today

# Priority ranges
:select jobs --priority=1000-9999
```

## Job Batch Operations

### Job Management

Operate on multiple jobs simultaneously:

```bash
# Cancel multiple jobs
:cancel job[1000-1010]

# Hold all user's jobs
/user:alice
:hold --selected --reason="Maintenance window"

# Release held jobs
/state:HELD user:alice
:release --selected

# Change priority of job range
:priority job[2000-2100] --priority=500

# Requeue failed jobs
/state:FAILED user:bob
:requeue --selected
```

### Advanced Job Operations

```bash
# Modify job arrays
:modify job_array_123 --array=1-100:2  # Every other task

# Update time limits for running jobs
/state:RUNNING partition:gpu
:time --selected --time=+2:00:00  # Add 2 hours

# Change partition for pending jobs
/state:PENDING user:alice
:partition --selected --partition=highmem

# Notify on job completion
/user:alice state:RUNNING
:notify --selected --email --webhook
```

### Conditional Operations

Apply operations based on conditions:

```bash
# Cancel long-running jobs over 48 hours
/state:RUNNING time:>48h
:cancel --selected --reason="Runtime limit exceeded"

# Hold jobs exceeding memory limits
/state:RUNNING memory:>90%
:hold --selected --reason="High memory usage"

# Lower priority of idle jobs
/state:PENDING time:>24h
:priority --selected --priority=100
```

## Node Batch Operations

### Node Maintenance

Manage multiple nodes efficiently:

```bash
# Drain node range for maintenance
:drain node[001-020] --reason="OS update" --timeout=2h

# Resume multiple nodes after maintenance
:resume node[001-020]

# Put nodes in maintenance mode
/features:gpu state:idle
:maintenance --selected --reason="GPU firmware update"

# Update node features
:features node[100-200] --features=+nvme,-infiniband
```

### Bulk Node Configuration

```bash
# Set node weights for load balancing
:weight node[001-050] --weight=100
:weight node[051-100] --weight=200

# Configure power management
/features:power_save
:power --selected --policy=ondemand

# Update node availability
:available node[200-250] --available=no --reason="Hardware upgrade"
```

## Resource Monitoring

### Batch Status Checks

Monitor multiple resources:

```bash
# Check job status for user
:status --user alice --format table

# Node health summary
:health --nodes node[001-100] --summary

# Resource utilization report
:utilization --partition gpu --time-range 24h

# Queue analysis
:queue-analysis --all-partitions --export csv
```

### Performance Analytics

```bash
# Job efficiency analysis
/state:COMPLETED user:alice
:efficiency --selected --threshold 0.8

# Resource usage trends
:usage-trend --nodes node[001-050] --period week

# Utilization heatmap
:heatmap --partition gpu --time-range month
```

## Workflow Automation

### Scheduled Operations

Create recurring batch operations:

```bash
# Daily cleanup of old failed jobs
:schedule daily "cleanup-failed-jobs" \
  "/state:FAILED time:>7d" \
  ":cancel --selected --purge"

# Weekly maintenance check
:schedule weekly "maintenance-check" \
  "/load:>16 jobs:0" \
  ":maintenance --selected --auto-resume 4h"

# Hourly priority adjustment
:schedule hourly "priority-boost" \
  "/state:PENDING time:>6h priority:<500" \
  ":priority --selected --priority=+100"
```

### Template Operations

Save and reuse batch operation templates:

```yaml
# ~/.s9s/templates/maintenance.yaml
name: "Weekly Maintenance"
description: "Standard weekly maintenance workflow"
steps:
  - filter: "/features:gpu state:idle jobs:0"
  - operation: "drain"
    options:
      reason: "Weekly maintenance"
      timeout: "2h"
  - wait: "jobs-complete"
  - operation: "maintenance"
    options:
      script: "/opt/maintenance/gpu-check.sh"
  - operation: "resume"
```

Execute templates:

```bash
# Run saved template
:template run maintenance

# Run with overrides
:template run maintenance --timeout=4h --nodes=node[100-200]
```

## Batch Reporting

### Generate Reports

Create comprehensive reports across resources:

```bash
# User activity report
:report user-activity --users alice,bob,charlie \
  --period month --format pdf --output ~/reports/

# Cluster utilization report
:report utilization --all-partitions \
  --time-range "2023-12-01..2023-12-31" \
  --format excel

# Job efficiency report
:report efficiency --min-runtime 1h \
  --threshold 0.7 --format json
```

### Export Batch Data

```bash
# Export job history for analysis
/user:alice state:COMPLETED
:export --selected --format csv --fields=all \
  --output ~/data/alice_jobs.csv

# Export node configuration
:export nodes --format yaml --config-only \
  --output cluster-config.yaml

# Export performance metrics
/partition:gpu
:export --selected --metrics --time-range week \
  --format parquet --output gpu-metrics/
```

## Safety and Validation

### Dry Run Mode

Test batch operations safely:

```bash
# Preview operation without executing
:cancel job[1000-2000] --dry-run

# Validate template
:template validate maintenance --dry-run

# Check impact of drain operation
:drain node[001-020] --dry-run --report-impact
```

### Confirmation and Safeguards

```bash
# Require explicit confirmation for destructive operations
:cancel /state:RUNNING --confirm-each

# Set maximum operation limits
:config set batch.max_operations 100
:config set batch.require_confirmation true

# Enable operation logging
:config set logging.batch_operations true
```

### Rollback Capabilities

```bash
# Undo last batch operation
:undo

# Rollback specific operation
:rollback operation_id_12345

# View operation history
:history --operations --limit 10
```

## Interactive Batch Mode

### Batch Command Interface

Enter interactive batch mode:

```bash
# Start batch mode
:batch

# In batch mode
batch> select /user:alice state:PENDING
batch> priority --selected 1000
batch> notify --selected --email
batch> execute
```

### Batch Scripting

Create reusable batch scripts:

```bash
#!/usr/bin/env s9s-batch

# Batch script: daily-cleanup.s9s
select /state:FAILED time:>7d
cancel --selected --purge

select /state:COMPLETED time:>30d
archive --selected --location s3://archive-bucket/

select /partition:debug state:PENDING time:>1h
cancel --selected --reason="Debug queue timeout"

notify --summary --slack
```

Execute batch scripts:

```bash
s9s batch-run daily-cleanup.s9s
```

## Batch Operation Examples

### Common Workflows

**1. Weekly Maintenance**:
```bash
# Identify maintenance candidates
/state:idle jobs:0 features:gpu

# Drain selected nodes
:drain --selected --reason="Weekly maintenance" --timeout=2h

# Wait and perform maintenance
:wait jobs-complete
:ssh --selected "sudo /opt/maintenance/weekly-check.sh"

# Resume nodes
:resume --selected
```

**2. Job Cleanup**:
```bash
# Old completed jobs
/state:COMPLETED time:>30d
:archive --selected --location /archive/jobs/

# Failed jobs with low priority
/state:FAILED priority:<100 time:>7d
:cancel --selected --purge

# Stuck pending jobs
/state:PENDING time:>72h
:cancel --selected --reason="Timeout in queue"
```

**3. Resource Rebalancing**:
```bash
# High-priority jobs stuck in queue
/state:PENDING priority:>5000 time:>6h
:partition --selected --partition=express

# Long-running jobs hogging resources
/state:RUNNING time:>48h nodes:>8
:time --selected --time=+4h --notify
```

## Configuration

### Batch Operation Settings

```yaml
# ~/.s9s/config.yaml
batch:
  # Maximum operations per batch
  maxOperations: 1000

  # Require confirmation for destructive operations
  requireConfirmation: true

  # Enable dry-run mode by default
  defaultDryRun: false

  # Parallel execution settings
  maxConcurrent: 10
  timeout: 300s

  # Safety limits
  maxNodesPerOperation: 100
  maxJobsPerOperation: 1000

  # Logging
  logOperations: true
  logFile: ~/.s9s/batch.log
```

## Next Steps

- Master [Advanced Filtering](../filtering.md) for precise resource selection
- Learn [Node Operations](./node-operations.md) for single-node management
- Explore [Export](./export.md) capabilities for data analysis
