# Batch Operations Guide

Efficiently manage multiple jobs simultaneously with S9S's batch operation capabilities.

## Overview

Batch operations allow you to:
- Manage multiple jobs with a single command
- Apply operations to filtered jobs
- Perform bulk job maintenance tasks
- Export output from multiple jobs

## Selection Methods

### Visual Selection

Use visual selection mode to choose multiple items:

| Key | Action | Description |
|-----|--------|-------------|
| `Space` | Toggle selection | Select/deselect current item |
| `v`/`V` | Multi-select mode | Toggle multi-select mode |
| `Ctrl+A` | Select all | Select all visible items |

### Filter-Based Selection

Select items using filters:

```bash
# Quick filter for text search
/FAILED                # Jobs containing "FAILED"
/alice                 # Jobs containing "alice"
```

## Available Batch Operations

S9S supports the following batch operations on selected jobs:

### Cancel Jobs

Cancel all selected jobs:

```bash
# Select jobs and press 'c' or choose "Cancel Jobs" from the menu
# Confirmation dialog will appear before execution
```

**Operation**: Calls `scancel` for each selected job
**Use Case**: Stop running or pending jobs that are no longer needed

### Hold Jobs

Put selected jobs on hold:

```bash
# Select jobs and press 'H' or choose "Hold Jobs" from the menu
# Jobs will be prevented from starting
```

**Operation**: Calls `scontrol hold` for each selected job
**Use Case**: Temporarily prevent pending jobs from starting without canceling them
**Note**: Only works on pending jobs; cannot hold running jobs

### Release Jobs

Release held jobs:

```bash
# Select held jobs and press 'r' or choose "Release Jobs" from the menu
# Jobs will be eligible to run again
```

**Operation**: Calls `scontrol release` for each selected job
**Use Case**: Allow previously held jobs to start

### Requeue Jobs

Requeue selected jobs:

```bash
# Choose "Requeue Jobs" from the batch operations menu
# Jobs will be requeued for execution
```

**Operation**: Calls `scontrol requeue` for each selected job
**Use Case**: Restart failed jobs or re-run completed jobs
**Note**: Use the batch operations menu (`b`) to access requeue, since the `q` key is reserved for the global quit shortcut.

### Delete Jobs

Delete selected jobs:

```bash
# Select jobs and press 'd' or choose "Delete Jobs" from the menu
# Confirmation dialog will appear before execution
```

**Operation**: Cancels each selected job
**Use Case**: Remove unwanted jobs from the queue

### Set Priority

Set priority for selected jobs:

```bash
# Select jobs and press 'p' or choose "Set Priority" from the menu
# Enter the desired priority value
```

**Operation**: Sets priority for each selected job
**Use Case**: Adjust scheduling priority for a group of jobs

### Export Job Output

Export job output for all selected jobs:

```bash
# Select jobs and press 'e' or choose "Export Output" from the menu
# Choose output format: Text, JSON, CSV, or Markdown
# Files saved to ~/slurm_exports/ by default
```

**Formats Available**:
- **Text**: Plain text with header information
- **JSON**: Structured JSON with metadata
- **CSV**: CSV format (line-by-line for analysis)
- **Markdown**: Markdown format with code blocks
- **HTML**: HTML format for browser viewing

**Operation**: Retrieves job output and saves to local files
**Use Case**: Archive job results, analyze output across multiple jobs

## Interactive Batch Operations

### Using the Batch Operations Menu

1. **Select Jobs**: Use visual selection (`Space` key) or filters to select multiple jobs
2. **Open Batch Menu**: Press `b` (or configured batch key) to open the batch operations menu
3. **Choose Operation**: Navigate the menu and select an operation:
   - Cancel Jobs
   - Hold Jobs
   - Release Jobs
   - Requeue Jobs
   - Delete Jobs
   - Set Priority
   - Export Output
4. **Confirm**: Review the confirmation dialog showing affected jobs
5. **Execute**: Confirm to execute the batch operation
6. **Monitor Progress**: Watch the progress bar as operations are applied to each job

### Progress Tracking

The batch operations interface shows:
- Number of jobs being processed
- Current job being processed
- Success/failure count
- Overall completion status

## Common Workflows

### Cleanup Failed Jobs

```bash
# Step 1: Filter failed jobs using quick filter
/FAILED

# Step 2: Review the filtered list
# Step 3: Select jobs with Space or V for multi-select, then b for batch menu
# Step 4: Press 'c' to cancel, confirm the operation
```

### Hold User Jobs for Maintenance

```bash
# Step 1: Filter user's pending jobs
/alice

# Step 2: Select jobs, open batch menu with 'b', press 'H' to hold all jobs
# Step 3: Perform maintenance
# Step 4: Filter held jobs and use batch menu 'r' to release them
```

### Requeue Failed Jobs

```bash
# Step 1: Filter failed jobs
/FAILED

# Step 2: Select jobs, open batch menu with 'b', choose "Requeue Jobs"
# Step 3: Jobs will be requeued and eligible to run again
```

### Export Results from Completed Jobs

```bash
# Step 1: Filter completed jobs
/COMPLETED

# Step 2: Select jobs, open batch menu with 'b', press 'e' to export
# Step 3: Choose export format (Text, JSON, CSV, Markdown, or HTML)
# Step 4: Files saved to ~/slurm_exports/
```

## Safety and Validation

### Confirmation Dialogs

All destructive batch operations (cancel, hold, requeue) require confirmation:
- Shows the number of jobs affected
- Lists job IDs that will be modified
- Requires explicit "Yes" to proceed
- Can be canceled with "No" or `Esc`

### Progress Feedback

Batch operations provide real-time feedback:
- Progress bar showing completion percentage
- Current job being processed
- Success and failure counts
- Final summary of results

### Error Handling

If a batch operation fails on individual jobs:
- Operation continues with remaining jobs
- Errors are counted and reported
- Final summary shows successful vs. failed operations

## Export Defaults

Export files are saved to `~/slurm_exports/` by default. The directory is created automatically on first export. Export path and format are selected interactively in the export dialog.

## Troubleshooting

### Common Issues

**"No jobs selected for batch operations"**
- Solution: Select at least one job using `Space` or visual selection mode

**"Permission denied" errors during batch operations**
- Solution: You can only perform operations on your own jobs (unless admin)

**"Operation failed" for some jobs**
- Cause: Job state may have changed, or job may not support the operation
- Solution: Review the final summary to identify which jobs failed

**Export files not found**
- Solution: Check `~/slurm_exports/` directory or configured export path
- Note: Directory is created automatically on first export

## Keyboard Reference

| Key | Operation | Description |
|-----|-----------|-------------|
| `b` | Open batch menu | Open batch operations on selected jobs |
| `Space` | Toggle selection | Select/deselect individual jobs |
| `c` | Cancel jobs | Cancel all selected jobs |
| `H` | Hold jobs | Put selected jobs on hold |
| `r` | Release jobs | Release held jobs |
| (menu) | Requeue jobs | Requeue selected jobs (via batch menu) |
| `d` | Delete jobs | Delete all selected jobs |
| `p` | Set priority | Set priority for selected jobs |
| `e` | Export output | Export job output |
| `Esc` | Close menu | Close batch operations menu |

## Next Steps

- Learn more about [Advanced Filtering](../filtering.md) for precise job selection
- Explore [Export](./export.md) capabilities for detailed output analysis
- Review [Job Management](./job-management.md) for single-job operations
