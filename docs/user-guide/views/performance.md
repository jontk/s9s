# Performance View

The Performance view provides real-time cluster-wide metrics and utilization statistics, giving you a high-level overview of your SLURM cluster's health and resource usage.

![Performance Demo](/assets/demos/performance.gif)

*Performance view showing cluster-wide job, node, and resource metrics with auto-refresh*

## Overview

The Performance view displays three main metric categories:

- **Jobs**: Total job counts and distribution
- **Nodes**: Node availability and status
- **Resources**: Cluster-wide CPU and Memory utilization

This view is designed for quick cluster health assessment and capacity planning.

## Access

Press **`9`** or navigate to "Performance" from the view switcher.

## Display Sections

### Jobs Metrics

Shows cluster-wide job statistics:

- **Total**: All jobs in the system
- **Running**: Currently executing jobs (green)
- **Pending**: Jobs waiting in queue (blue)

**Use Cases**:
- Monitor queue depth
- Identify bottlenecks (high pending count)
- Track overall cluster load

### Nodes Metrics

Shows node availability across the cluster:

- **Total**: All configured nodes
- **Active**: Nodes running jobs (green)
- **Idle**: Available nodes with no jobs (blue)
- **Down**: Offline or unavailable nodes (red)

**Use Cases**:
- Identify hardware issues (down nodes)
- Check capacity (idle nodes available)
- Monitor resource utilization

### Resources Metrics

Shows aggregate cluster utilization:

- **CPU**: Cluster-wide CPU usage percentage
- **Memory**: Cluster-wide memory usage percentage
- Visual bars with color-coded thresholds:
  - 🟢 Green: 0-75% (healthy)
  - 🟡 Yellow: 75-90% (high)
  - 🔴 Red: 90-100% (critical)

**Use Cases**:
- Capacity planning
- Identify resource saturation
- Performance trending

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `R` | Toggle auto-refresh on/off |
| `F5` | Manual refresh |
| `?` | Show help |
| `q` | Exit view |

## Auto-Refresh

The Performance view automatically refreshes every **5 seconds** by default when auto-refresh is enabled.

- **Enable/Disable**: Press `R` to toggle
- **Manual Refresh**: Press `F5` to update immediately
- **Status Indicator**: Control bar shows auto-refresh state

## Interpretation Guide

### Healthy Cluster Signs

- ✅ Low pending job count relative to running jobs
- ✅ Few or no down nodes
- ✅ CPU/Memory utilization in green/yellow range
- ✅ Some idle nodes available for burst capacity

### Warning Signs

- ⚠️ High pending to running job ratio (potential bottleneck)
- ⚠️ Multiple down nodes (hardware issues)
- ⚠️ Sustained red resource utilization (capacity limit reached)
- ⚠️ Zero idle nodes (no burst capacity)

### Critical Issues

- 🚨 More pending than running jobs (severe bottleneck)
- 🚨 Majority of nodes down (cluster failure)
- 🚨 100% resource utilization sustained (oversubscribed)

## Example Scenarios

### Scenario 1: Healthy Cluster
```
Jobs:           Nodes:          Resources:
Total: 45       Total: 20       CPU: 45%  ████░░░░
Running: 30     Active: 12      Mem: 52%  █████░░░
Pending: 15     Idle: 8
                Down: 0
```
**Analysis**: Good balance, capacity available, no issues.

### Scenario 2: Queue Bottleneck
```
Jobs:           Nodes:          Resources:
Total: 120      Total: 20       CPU: 85%  ████████
Running: 20     Active: 20      Mem: 89%  ████████
Pending: 100    Idle: 0
                Down: 0
```
**Analysis**: All nodes busy, large queue, near capacity. Consider:
- Adding more nodes
- Reviewing job priorities
- Checking for inefficient jobs

### Scenario 3: Hardware Issues
```
Jobs:           Nodes:          Resources:
Total: 25       Total: 20       CPU: 92%  █████████
Running: 22     Active: 14      Mem: 88%  ████████
Pending: 3      Idle: 0
                Down: 6
```
**Analysis**: 30% of nodes down, remaining nodes overloaded. Action required:
- Investigate down nodes immediately
- High utilization due to reduced capacity

## Integration with Other Views

The Performance view provides a high-level overview. Drill down for details:

- **Jobs view** (`J`): See specific job details and queue analysis
- **Nodes view** (`N`): Investigate individual node status and down nodes
- **Partitions view** (`P`): Check partition-specific utilization
- **Dashboard view** (`D`): See health checks and detailed metrics

## Tips

1. **Monitor During Peak Hours**: Check Performance view during typical peak usage times to understand baseline
2. **Trend Analysis**: Note patterns over time (daily/weekly cycles)
3. **Capacity Planning**: If consistently high utilization, plan for expansion
4. **Quick Health Check**: Performance view is perfect for quick "is everything okay?" checks

## Metrics Source

All metrics are pulled from the SLURM cluster via `sinfo`, `squeue`, and cluster statistics APIs. The view shows real-time data from your actual cluster, updated every 5 seconds.

## Related Documentation

- [Dashboard View](dashboard.md) - Detailed cluster health
- [Jobs View](jobs.md) - Job management and monitoring
- [Nodes View](nodes.md) - Node-level details
- [Health View](health.md) - Cluster health checks

---

## Developer Note: App Diagnostics

For s9s developers, there's an **App Diagnostics** view that monitors the s9s CLI application itself (memory, goroutines, internal operations). This is hidden by default and can be enabled with:

```yaml
# ~/.config/s9s/config.yaml
features:
  appDiagnostics: true
```

This is useful for debugging s9s performance issues, not for cluster monitoring.
