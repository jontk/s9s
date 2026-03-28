# Dashboard View

The Dashboard provides a real-time overview of your SLURM cluster with comprehensive metrics, health indicators, and quick navigation.

![Dashboard Demo](/assets/demos/dashboard.gif)

*Dashboard showing cluster metrics, job distribution, and node status*

## Overview

The Dashboard provides a cluster overview accessible via key `8`, displaying six information panels organized in a responsive layout:

- **Cluster Overview** - System information and health status
- **Jobs Summary** - Job queue and state distribution
- **Nodes Summary** - Node availability and resource usage
- **Partition Status** - Top partitions by size
- **Alerts & Issues** - System health warnings
- **Performance Trends** - Historical throughput and efficiency

## Display Panels

### Cluster Overview (Top Left)

Displays essential cluster information:
- Cluster name, version, and endpoint
- Overall health status (EXCELLENT/GOOD/FAIR/POOR/CRITICAL)
- CPU and Memory usage percentages
- Visual utilization bars

Health scoring is calculated from:
- Down nodes (2 points per %)
- Failed jobs (1 point per %)
- High CPU/Memory utilization (up to 10 points each for >95%)

### Jobs Summary (Top Center)

Shows current job queue status:
- Total job count
- Job state breakdown:
  - **Running** - Currently executing jobs
  - **Pending** - Queued jobs waiting for resources
  - **Completed** - Successfully finished jobs
  - **Failed** - Jobs that ended with errors
  - **Canceled** - User-canceled jobs
- Job state distribution visualization
- Average wait time for pending jobs

### Nodes Summary (Top Right)

Provides node availability metrics:
- Total node count
- Node state counts:
  - **Idle** - Available for work
  - **Allocated** - Fully assigned
  - **Mixed** - Partially allocated
  - **Down** - Unavailable
  - **Drain** - Being drained
- CPU and Memory utilization percentages
- Node availability percentage

### Partition Status (Middle Left)

Lists the top 8 partitions by size:
- Partition name
- Node count
- CPU count
- Current state
- Sorted by node count (largest first)

### Alerts & Issues (Middle Right)

Displays system alerts when issues are detected:
- Down nodes warning
- High memory utilization (>90%)
- High CPU utilization (>90%)
- Long waiting jobs (>24 hours)
- Failed jobs (>10)

Shows "No issues detected" with green checkmark when system is healthy.

### Performance Trends (Bottom)

Displays cluster performance visualizations:
- Resource efficiency bar (average of CPU and Memory utilization)
- System health score with visual bar

These values are computed from real cluster metrics collected during the session.

## Actions & Shortcuts

### Layout
| Key | Action |
|-----|--------|
| `L` | Switch layout (Default / Monitoring) |

### Analytics
| Key | Action |
|-----|--------|
| `A` | Show Advanced Analytics modal |
| `H` | Show Health Check modal |

### Data Management
| Key | Action |
|-----|--------|
| `R` | Manual refresh all dashboard data |

## Advanced Analytics Modal

Press `A` to open the Advanced Analytics modal with comprehensive cluster analysis:

### Resource Efficiency
- CPU efficiency percentage and assessment
- Memory efficiency percentage and assessment
- Overall resource efficiency rating

### Job Analysis
- Job state distribution
- Average wait times by state
- Queue depth analysis

### Node Analysis
- Node state distribution
- Node utilization metrics
- Resource availability trends

### Recommendations
The system applies rule-based heuristics to current metrics and provides:
- Performance optimization suggestions
- Resource allocation recommendations
- Potential issue warnings

**Modal Shortcuts:**
- `R` - Refresh analytics
- `ESC` - Close modal

## Health Check Modal

Press `H` to open detailed health check report:

### Overall Health Score
- Numeric score (0-100)
- Color-coded health rating
- Status summary

### Component Health Checks

**Nodes**
- Down node count and percentage
- Draining node count
- Impact assessment

**Job Queue**
- Failed job count
- Long-running stuck jobs
- Queue health status

**Resource Utilization**
- CPU usage vs. critical threshold
- Memory usage vs. critical threshold
- Utilization warnings

**Partitions**
- Operational partition count
- Down partition count
- Partition availability

Each component shows:
- Status indicator (✓/⚠/✗)
- Detailed metrics
- Threshold information

**Modal Shortcuts:**
- `R` - Refresh health check
- `ESC` - Close modal

## Auto-Refresh

The Dashboard loads data on initial display but does not automatically refresh afterward. Press `R` for a manual refresh to update the dashboard data.

## Visual Design

The Dashboard uses a 6-panel color system for status indicators:
- **Green** - Healthy, good status
- **Red** - Critical issues, failures
- **Yellow** - Warnings, attention needed
- **Orange** - Moderate issues
- **Cyan** - Informational
- **Teal** - Headers and accents

## Technical Details

### Data Fetching
The Dashboard fetches data concurrently from multiple sources:
- Cluster information and version
- Current metrics (CPU/Memory)
- Job list with all states
- Node list with states
- Partition information

This parallel fetching ensures quick dashboard updates even on large clusters.

### Layout Adaptation
The Dashboard uses flexible layout proportions to adapt to different terminal sizes:
- Minimum recommended width: 120 columns
- Minimum recommended height: 30 rows
- Panels resize proportionally
- Text truncates gracefully

## Tips

- Use the Dashboard as your cluster monitoring hub
- Press `A` regularly to review recommendations
- Check `H` for detailed health diagnostics
- Press `L` to switch between the default dashboard and the monitoring layout
- Press `R` to manually refresh the dashboard data when you want the latest metrics
