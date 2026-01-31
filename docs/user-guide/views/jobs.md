# Jobs View

The Jobs view provides comprehensive job management capabilities including monitoring, submission, cancellation, and batch operations.

![Jobs Demo](/assets/demos/jobs.gif)

*Jobs view showing job list, filtering, and batch operations*

## Overview

The Jobs view displays a table of all SLURM jobs with detailed information and provides actions for job management. It supports both single-job operations and powerful batch operations for managing multiple jobs simultaneously.

## Table Columns

The jobs table displays 11 columns:

| Column | Width | Description | Alignment |
|--------|-------|-------------|-----------|
| **ID** | 10 | Job identifier | Left |
| **Name** | 20 | Job name | Left |
| **User** | 10 | Job owner username | Left |
| **Account** | 12 | Billing account | Left |
| **State** | 12 | Job status (color-coded) | Left |
| **Partition** | 10 | Compute partition | Left |
| **Nodes** | 8 | Number of nodes | Right |
| **Time** | 10 | Time used | Right |
| **Time Limit** | 10 | Maximum runtime | Right |
| **Priority** | 8 | Job priority | Right |
| **Submit Time** | 19 | Submission timestamp | Left |

### Color Coding
- **State column**: Color varies by job state
  - Green: RUNNING, COMPLETED
  - Yellow: PENDING, CONFIGURING
  - Red: FAILED, CANCELLED, TIMEOUT
  - Cyan: SUSPENDED
- **Table header**: Teal
- **Selected rows**: Yellow highlight

## Single Job Actions

### View Job Details
**Shortcut**: `Enter`

Shows detailed information about the selected job:
- Job ID, name, and user
- State, exit code, and reason
- Partition and QoS
- Node allocation
- Time information (submit, start, end)
- Resource usage (CPU, memory)
- Working directory and command
- Standard output/error paths

### Submit New Job
**Shortcuts**: `s/S` (wizard), `F2` (templates)

Two submission methods:
1. **Job Submission Wizard** (`s/S`): Step-by-step guided submission
2. **Job Templates** (`F2`): Pre-configured job templates

See [Job Management](../job-management.md) for detailed submission guide.

### Cancel Job
**Shortcut**: `c/C`

Cancels the selected job (available for RUNNING or PENDING states).

Confirmation dialog shows:
- Job ID and name
- Current state
- User and partition

### Hold Job
**Shortcut**: `H`

Places a PENDING job on hold, preventing it from starting.

### Release Job
**Shortcut**: `r`

Releases a held or suspended job, allowing it to run.

Available for:
- SUSPENDED jobs
- PENDING jobs (that were held)

### Requeue Job
**Shortcut**: `q/Q`

Requeues a completed, failed, or cancelled job for re-execution.

### View Job Output
**Shortcut**: `o/O`

Opens the job output viewer with:
- Standard output (stdout)
- Standard error (stderr)
- Real-time log streaming for running jobs

See [Job Streaming Guide](../../guides/job-streaming.md) for details.

### View Job Dependencies
**Shortcut**: `d/D`

Shows job dependency graph:
- Jobs this job depends on
- Jobs that depend on this job
- Dependency types (afterok, afterany, etc.)

## Batch Operations

### Enter Batch Mode
**Shortcut**: `b/B`

Opens the batch operations menu for managing multiple jobs.

### Multi-Select Mode
**Shortcut**: `v/V`

Toggles multi-select mode for selecting specific jobs.

**In multi-select mode:**
- `Space` - Toggle selection for current row
- `Ctrl+A` - Select all jobs
- `ESC` - Exit multi-select mode

Selection status shows: "Selected X jobs"

### Batch Actions

When multiple jobs are selected:
- **Cancel** - Cancel all selected jobs
- **Hold** - Hold all selected pending jobs
- **Release** - Release all selected held jobs

### State-Based Batch Operations

Operate on all jobs in a specific state:
- Cancel all PENDING jobs
- Hold all PENDING jobs
- Release all SUSPENDED jobs

## Filtering & Search

### Simple Text Filter
**Shortcut**: `/`

Activates the filter input box. Filters jobs by:
- Job ID
- Job name
- Username
- Account
- Partition
- Any displayed column value

**Special syntax:**
- `p:name` or `partition:name` - Filter by partition

**Example**: `/` then type "gpu" to find all jobs with "gpu" in any field.

### Advanced Filter
**Shortcut**: `F3`

Opens the advanced filter bar with expression-based filtering.

**Filter expressions:**
```
state:RUNNING
user:alice
partition:gpu
nodes:>4
priority:>=1000
```

**Supported fields:**
- `id` - Job ID
- `name` - Job name
- `user` - Username
- `account` - Account name
- `state` - Job state
- `partition` - Partition name
- `nodes` - Node count (supports >, <, >=, <=, =)
- `timelimit` - Time limit
- `timeused` - Time used
- `priority` - Priority (supports comparison operators)
- `qos` - Quality of Service
- `submittime` - Submission time
- `starttime` - Start time
- `endtime` - End time
- `workdir` - Working directory
- `command` - Job command

**Operators:**
- `=` - Exact match
- `:` - Contains
- `>`, `<`, `>=`, `<=` - Numeric comparison

**Example filters:**
```
state:RUNNING partition:gpu
user:alice priority:>500
nodes:>=8 state:PENDING
```

Press `ESC` to exit advanced filter mode.

### State Filtering
| Shortcut | Filter |
|----------|--------|
| `a/A` | Show all states |
| `p/P` | Show pending jobs only |

### User Filtering
**Shortcut**: `u/U`

Opens dialog to filter jobs by specific username.

### Global Search
**Shortcut**: `Ctrl+F`

Opens global search across all views (jobs, nodes, partitions, users, accounts, QoS, reservations).

## Sorting

Sort jobs by clicking column headers or using keyboard shortcuts.

**Sortable columns:**
- State
- Priority
- Submit Time

Press column number `1-9` to sort by that column. Press again to reverse sort direction.

## Auto-Refresh

Jobs view auto-refreshes every **30 seconds** by default.

**Toggle auto-refresh**: `m/M`

When disabled, use `R` for manual refresh.

## Action Menu

**Shortcut**: `F1`

Shows context-sensitive action menu with all available actions for the selected job.

## Keyboard Shortcuts Reference

### Job Operations
| Key | Action |
|-----|--------|
| `Enter` | View job details |
| `s/S` | Submit job (wizard) |
| `F2` | Job templates |
| `c/C` | Cancel job |
| `H` | Hold job |
| `r` | Release job |
| `q/Q` | Requeue job |
| `o/O` | View output |
| `d/D` | View dependencies |

### Selection & Batch
| Key | Action |
|-----|--------|
| `b/B` | Batch operations menu |
| `v/V` | Toggle multi-select mode |
| `Space` | Toggle row selection (in multi-select) |
| `Ctrl+A` | Select all (in multi-select) |

### Filtering & Search
| Key | Action |
|-----|--------|
| `/` | Simple filter |
| `F3` | Advanced filter |
| `Ctrl+F` | Global search |
| `a/A` | Filter all states |
| `p/P` | Filter pending |
| `u/U` | Filter by user |
| `ESC` | Exit filter mode |

### Data Management
| Key | Action |
|-----|--------|
| `R` | Manual refresh |
| `m/M` | Toggle auto-refresh |
| `F1` | Action menu |
| `1-9` | Sort by column |

## Job Submission

![Job Submission Demo](/assets/demos/job-submission.gif)

*Job submission wizard with step-by-step configuration*

The job submission wizard (`s/S`) guides you through:

1. **Basic Information**
   - Job name
   - Script or command
   - Working directory

2. **Resource Requirements**
   - Partition
   - Number of nodes
   - CPUs per task
   - Memory requirements

3. **Time Limits**
   - Wall time
   - Time limit format (days-hours:minutes:seconds)

4. **Advanced Options**
   - Account
   - QoS
   - Job dependencies
   - Email notifications
   - Output/error file paths

See [Job Management](../job-management.md) for detailed submission guide.

## Tips

- Use `v/V` for multi-select when you need to operate on specific jobs
- Use batch operations by state when you want to affect all jobs in a state
- Press `F3` for powerful filtering with expressions
- Use `partition:name` syntax in simple filter for quick partition filtering
- Check job output with `o/O` to debug issues
- Use `d/D` to understand job dependencies before canceling
- Enable auto-refresh (`m/M`) for monitoring active jobs
- Press `F1` when unsure what actions are available for a job
