# Health View

The Health view provides comprehensive cluster health monitoring with alerts, health checks, and system diagnostics.

![Health Demo](/assets/demos/health.gif)

*Health view showing cluster health status, alerts, and monitoring*

## Overview

The Health view monitors cluster status and identifies issues requiring attention. It provides:
- Overall cluster health score
- Active alerts with severity levels
- Health check summaries
- Alert acknowledgment and resolution
- System diagnostics

## Health Status

### Overall Status Indicator

The top of the Health view shows cluster-wide health:

| Status | Color | Score Range | Description |
|--------|-------|-------------|-------------|
| **Healthy** | Green | 90-100 | No issues, optimal performance |
| **Warning** | Yellow | 60-89 | Minor issues, attention recommended |
| **Critical** | Red | 0-59 | Serious issues, immediate action needed |

### Health Score Calculation

Score based on multiple factors:
- Node availability (down nodes reduce score)
- Job queue health (failed jobs reduce score)
- Resource utilization (extreme usage reduces score)
- Partition status (down partitions reduce score)

**Example:**
```
Health Score: 85/100 (Warning)

Components:
  Nodes:     95/100 (1 down node)
  Jobs:      80/100 (5 failed jobs)
  Resources: 90/100 (High CPU usage)
  Partitions: 100/100 (All operational)
```

## Alerts Section

### Alert Display

Alerts are shown with:
- **Severity** indicator (color + icon)
- **Alert message**
- **Timestamp**
- **Status** (Active/Acknowledged/Resolved)
- **Actions** available

### Alert Severity Levels

| Severity | Icon | Color | Priority | Example |
|----------|------|-------|----------|---------|
| **Critical** | ✗ | Red | Highest | Cluster controller down, multiple nodes down |
| **Warning** | ⚠ | Yellow | Medium | High memory usage, long wait times |
| **Info** | ℹ | Blue | Low | Maintenance scheduled, quota approaching |

### Alert Types

**Node Alerts:**
- Node down/unreachable
- Node drain status
- High node resource usage
- Hardware failures

**Job Alerts:**
- High failed job count (>10)
- Jobs stuck in pending (>24 hours)
- Job queue backlog
- Submission failures

**Resource Alerts:**
- High CPU utilization (>90%)
- High memory utilization (>90%)
- Resource exhaustion
- Disk space warnings

**System Alerts:**
- Partition down
- Controller issues
- Database connectivity
- Configuration errors

## Health Checks

The Health view runs periodic health checks:

### Health Check Categories

**Node Health:**
- Down node count and percentage
- Draining node count
- Node responsiveness
- Hardware status

**Job Queue Health:**
- Failed job count
- Long-pending job count (>24h)
- Queue depth
- Submission rate

**Resource Health:**
- CPU utilization vs. critical threshold (95%)
- Memory utilization vs. critical threshold (95%)
- Resource allocation efficiency
- Fragmentation

**Partition Health:**
- Operational partition count
- Down partition count
- Partition capacity
- Queue status per partition

### Health Check Status

Each check shows:
- ✓ (Green) - Passing, healthy
- ⚠ (Yellow) - Warning threshold exceeded
- ✗ (Red) - Critical threshold exceeded

## Health Actions

### View Alert Details
**Shortcut**: `Enter`

Shows comprehensive alert information:

**Alert Information:**
- Full alert message
- Severity level
- Source component
- First occurrence time
- Last update time
- Occurrence count (if repeated)

**Impact Assessment:**
- Affected resources
- User impact estimate
- Service degradation details

**Recommended Actions:**
- Suggested remediation steps
- Related documentation links
- Contact information

**History:**
- Alert timeline
- State changes
- Acknowledgments
- Resolution notes

### Acknowledge Alert
**Shortcut**: `a/A`

Acknowledges the first unacknowledged alert:
- Marks alert as "seen"
- Doesn't resolve the alert
- Useful for tracking who's aware of issue
- Shows acknowledgment timestamp

**Acknowledgment indicates:**
- Someone is aware of the issue
- Someone is investigating
- Does not mean issue is fixed

### Resolve Alert
**Shortcut**: `r`

Resolves the first active alert:
- Marks alert as resolved
- Adds resolution timestamp
- Alert moves to resolved list
- Optional resolution notes

**Only resolve when:**
- Issue is actually fixed
- Root cause is addressed
- Monitoring confirms resolution

### Clear Resolved Alerts
**Shortcut**: `c/C`

Removes resolved alerts older than 24 hours:
- Cleans up historical alerts
- Keeps recent resolved for reference
- Archived for long-term logging

### Show Health Check Details
**Shortcut**: `H`

Opens detailed health check report:

**Overall Health:**
- Composite score (0-100)
- Color-coded rating
- Trend indicator (improving/degrading/stable)

**Component Details:**
Each component shows:
- Current value vs. threshold
- Status (✓/⚠/✗)
- Impact on overall score
- Recommended threshold
- Trend data

**Health Checks Included:**
1. **Node Availability**
   - Down nodes: 0 critical, <5% warning
   - Drain nodes: <10% warning

2. **Job Queue Health**
   - Failed jobs: <5 good, <10 warning, >10 critical
   - Long-pending: <5% warning, <10% critical

3. **CPU Utilization**
   - <80% good, 80-95% warning, >95% critical
   - Averaged across cluster

4. **Memory Utilization**
   - <80% good, 80-95% warning, >95% critical
   - Averaged across cluster

5. **Partition Status**
   - All UP = good
   - Any DOWN = warning/critical

### Show Health Statistics
**Shortcut**: `s/S`

Opens health statistics dashboard:

**Alert Statistics:**
- Total alerts (all time)
- Active alerts by severity
- Average resolution time
- Alert frequency trends

**System Uptime:**
- Cluster uptime
- Node availability percentage
- Service disruptions

**Performance Metrics:**
- Job success rate
- Average queue wait time
- Resource utilization trends
- System throughput

**Historical Trends:**
- Health score over time (24h, 7d, 30d)
- Alert frequency patterns
- Recurring issues
- Improvement/degradation trends

### Refresh Health Data
**Shortcut**: `R`

Manually refreshes all health checks and alerts.

Auto-refresh occurs every 10-30 seconds.

## Filtering

### Simple Filter
**Shortcut**: Not available (alerts auto-sorted by severity)

### View Filters

**Active Alerts:**
- Unresolved issues requiring action
- Default view

**Acknowledged Alerts:**
- Issues being investigated
- Shows who acknowledged and when

**Resolved Alerts:**
- Fixed issues (last 24 hours)
- Useful for post-incident review

## Keyboard Shortcuts Reference

### Alert Operations
| Key | Action |
|-----|--------|
| `Enter` | View alert details |
| `a/A` | Acknowledge first unacknowledged alert |
| `r` | Resolve first active alert |
| `c/C` | Clear resolved alerts (>24h old) |

### Health Information
| Key | Action |
|-----|--------|
| `H` | Show health check details |
| `s/S` | Show health statistics |

### Data Management
| Key | Action |
|-----|--------|
| `R` | Manual refresh |
| `ESC` | Close modals |

## Auto-Refresh

Health view auto-refreshes every **10-30 seconds** to provide real-time monitoring.

More frequent during active incidents.

## Alert Example

```
┌─ Active Alerts ─────────────────────────────────┐
│                                                  │
│ ✗ CRITICAL - Multiple nodes down                │
│   3 nodes (gpu001, gpu002, gpu003) unresponsive │
│   First seen: 2024-01-15 14:23:45               │
│   Impact: 144 cores unavailable                 │
│   Status: Active                                │
│                                                  │
│ ⚠ WARNING - High memory utilization             │
│   Cluster memory usage: 92% (>90% threshold)    │
│   First seen: 2024-01-15 14:30:12               │
│   Impact: Reduced capacity for new jobs         │
│   Status: Acknowledged by alice                 │
│                                                  │
│ ℹ INFO - Scheduled maintenance                  │
│   Partition gpu will be drained Sunday 02:00    │
│   First seen: 2024-01-15 10:00:00               │
│   Impact: gpu partition unavailable 4 hours     │
│   Status: Acknowledged by admin                 │
│                                                  │
└──────────────────────────────────────────────────┘
```

## Health Check Details Example

When viewing health check details (`H`):

```
Overall Health: 85/100 (Warning)
Trend: Stable

Component Health Checks:

✓ Node Availability: 95/100
  Down Nodes: 1/128 (0.8%)
  Threshold: <5% warning
  Status: Good

⚠ Job Queue: 75/100
  Failed Jobs: 8
  Threshold: <10 warning, <5 good
  Status: Warning
  Long-pending: 3 jobs >24h

✓ CPU Utilization: 88/100
  Average: 78%
  Threshold: <95% critical, <80% good
  Status: Good

⚠ Memory Utilization: 70/100
  Average: 92%
  Threshold: <95% critical, <80% good
  Status: Warning

✓ Partitions: 100/100
  Operational: 8/8
  Status: All UP

Recommendations:
  - Investigate failed jobs in job queue
  - Monitor memory usage on high-utilization nodes
  - Consider expanding capacity if trend continues
```

## Health Statistics Example

When viewing statistics (`s/S`):

```
Health Statistics

Alert Summary (Last 7 Days):
  Total Alerts: 47
  Critical: 2
  Warning: 28
  Info: 17
  Average Resolution Time: 2h 15m

Current Status:
  Active Alerts: 3
  Acknowledged: 1
  Resolved (24h): 8

System Uptime:
  Cluster: 99.8%
  Average Node Availability: 98.5%
  Last Outage: 3 days ago (15 minutes)

Performance Metrics:
  Job Success Rate: 94.2%
  Average Wait Time: 1h 45m
  Resource Utilization: 78%
  Throughput: 245 jobs/hour

Health Trend (7 days):
  Average Score: 87/100
  Trend: Improving (+3 points)
  Best Day: Monday (95/100)
  Worst Day: Wednesday (78/100)
```

## Tips

- **Monitor regularly**: Check Health view daily for early issue detection
- **Acknowledge promptly**: Let team know you're investigating
- **Resolve only when fixed**: Don't mark as resolved prematurely
- **Use statistics**: Identify patterns and recurring issues
- **Check health details**: Understand component-level health
- **Respond to critical**: Critical alerts need immediate attention
- **Clear old alerts**: Keep view focused on current issues
- **Review trends**: Use historical data for capacity planning
- **Set up notifications**: Configure external alerts for critical issues
- **Document resolutions**: Add notes when resolving for future reference

## Common Alert Patterns

### Recurring Alerts
Same alert appearing repeatedly:
- Indicates underlying issue not fixed
- Need permanent solution, not temporary fix
- Review root cause

### Escalating Severity
Warning → Critical:
- Issue worsening over time
- Requires intervention before critical
- Monitor trends

### Correlated Alerts
Multiple related alerts:
- Single root cause
- Fix root cause to resolve all
- Example: Controller down → many node alerts

## Best Practices

**For Administrators:**
- Monitor Health view in dashboard or mobile
- Set up external alerting (email/Slack)
- Document common issues and solutions
- Review health statistics weekly
- Tune alert thresholds for your cluster
- Archive resolved alerts for compliance

**For Users:**
- Check Health before submitting large jobs
- Report persistent issues to admins
- Understand impact of active alerts on your jobs
- Use health info to choose partitions wisely

**For Troubleshooting:**
- Start with Health view when investigating issues
- Correlate alerts with job failures
- Check health check details for specific metrics
- Review alert history for patterns
- Use statistics to identify trends
