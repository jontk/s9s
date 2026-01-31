# Partitions View

The Partitions view provides comprehensive partition monitoring with queue analytics, efficiency metrics, and wait time analysis.

![Partitions Demo](/assets/demos/partitions.gif)

*Partitions view showing queue depth, efficiency metrics, and analytics*

## Overview

The Partitions view displays all SLURM partitions with real-time queue information, resource allocation, and performance metrics. Use this view to understand partition utilization and identify bottlenecks.

## Table Columns

| Column | Description |
|--------|-------------|
| **Name** | Partition name |
| **State** | Partition status (UP/DOWN/DRAIN/INACTIVE) |
| **Total Nodes** | Number of nodes in partition |
| **CPUs** | Total CPU count |
| **Queue Depth** | Visual representation of pending vs. running jobs |
| **Jobs (Run/Pend)** | Running and pending job counts |
| **Avg Wait / Max Wait** | Average and maximum wait times |
| **Efficiency** | Cluster efficiency percentage |
| **QoS** | Associated Quality of Service policies |

## Queue Depth Visualization

The Queue Depth column shows a visual bar representing job distribution:

```
Queue: ████████░░░░
       ^^^^^^^^       Running jobs (solid)
               ░░░░   Pending jobs (hollow)
```

**Visual indicators:**
- **█** (solid) = Running jobs
- **░** (hollow) = Pending jobs
- **Length** = Relative proportion

**Color coding:**
- Green = Healthy (low queue depth)
- Yellow = Moderate queue
- Red = High queue depth (bottleneck)

## Partition States

| State | Color | Description |
|-------|-------|-------------|
| **UP** | Green | Active and accepting jobs |
| **DOWN** | Red | Unavailable |
| **DRAIN** | Orange | No new jobs, existing jobs continue |
| **INACTIVE** | Gray | Disabled partition |

## Efficiency Metrics

The Efficiency column shows partition resource utilization:

```
85% [████████████████░░]
```

**Rating levels:**
- **90-100%** (Green) - Excellent utilization
- **75-89%** (Yellow) - Good utilization
- **60-74%** (Orange) - Fair utilization
- **<60%** (Red) - Poor utilization

**Efficiency calculation:**
```
Efficiency = (Allocated Resources / Total Resources) × Job Success Rate
```

Factors considered:
- CPU allocation ratio
- Memory allocation ratio
- Job completion rate
- Resource waste from failed jobs

## Partition Actions

### View Partition Details
**Shortcut**: `Enter`

Shows comprehensive partition information:

**Basic Information:**
- Partition name and state
- Default/maximum time limits
- Node count and CPU count
- Priority level

**Resource Limits:**
- Maximum nodes per job
- Maximum CPUs per job
- Maximum memory per job
- Default resources

**Access Control:**
- Allowed accounts
- Allowed QoS levels
- User access restrictions

**Current Status:**
- Running jobs count
- Pending jobs count
- Allocated vs. total resources
- Current efficiency rating

### View Partition Jobs
**Shortcut**: `J`

Switches to Jobs view filtered by this partition. Shows all jobs (running and pending) assigned to the partition.

### View Partition Nodes
**Shortcut**: `N`

Switches to Nodes view filtered by this partition. Shows all nodes belonging to the partition with their current states.

### Partition Analytics
**Shortcut**: `A`

Opens detailed analytics dashboard for the partition:

**Queue Analytics:**
- Job submission rate trends
- Queue depth over time
- Wait time distribution
- Peak usage hours

**Resource Analytics:**
- CPU utilization trends
- Memory utilization trends
- Node availability over time
- Resource fragmentation analysis

**Performance Metrics:**
- Job throughput (jobs/hour)
- Average job duration
- Success rate percentage
- Failed job analysis

**Capacity Planning:**
- Projected growth
- Resource recommendations
- Bottleneck identification
- Optimization suggestions

### Wait Time Analytics
**Shortcut**: `W`

Shows cluster-wide wait time analysis:

**By Partition:**
- Average wait time per partition
- Maximum wait times
- Wait time trends
- Comparison across partitions

**By Job Size:**
- Wait times for small jobs (<10 nodes)
- Wait times for medium jobs (10-100 nodes)
- Wait times for large jobs (>100 nodes)

**Status Indicators:**
- 🟢 Good (<1 hour average wait)
- 🟡 Moderate (1-4 hours average wait)
- 🔴 Poor (>4 hours average wait)

**Historical Trends:**
- Wait time changes over 24 hours
- Week-over-week comparison
- Seasonal patterns

## Filtering & Search

### Simple Text Filter
**Shortcut**: `/`

Filters partitions by:
- Partition name
- State
- QoS policies
- Any displayed column

### Advanced Filter
**Shortcut**: `F3`

Expression-based filtering:

```
state:UP
nodes:>100
efficiency:>80
qos:high
```

**Supported fields:**
- `name` - Partition name
- `state` - Partition state
- `nodes` - Node count (supports >, <, >=, <=)
- `cpus` - CPU count (supports comparison)
- `efficiency` - Efficiency percentage (supports comparison)
- `jobs` - Running job count
- `pending` - Pending job count
- `qos` - QoS name

Press `ESC` to exit advanced filter.

### Global Search
**Shortcut**: `Ctrl+F`

Search across all cluster resources.

## Sorting

Sort partitions by clicking column headers or using number keys.

**Common sorting:**
- By efficiency (identify underutilized partitions)
- By queue depth (find bottlenecks)
- By wait time (prioritize improvements)
- By node count (understand capacity)

Press `1-9` to sort by column number.

## Keyboard Shortcuts Reference

### Partition Operations
| Key | Action |
|-----|--------|
| `Enter` | View partition details |
| `J` | View partition jobs |
| `N` | View partition nodes |
| `A` | Partition analytics dashboard |
| `W` | Wait time analytics |

### Filtering & Search
| Key | Action |
|-----|--------|
| `/` | Simple filter |
| `F3` | Advanced filter |
| `Ctrl+F` | Global search |
| `ESC` | Exit filter mode |

### Data Management
| Key | Action |
|-----|--------|
| `R` | Manual refresh |
| `1-9` | Sort by column |

## Partition Details Example

When viewing partition details (`Enter`):

```
Partition: gpu
State: UP
Priority: 100

Time Limits:
  Default: 2-00:00:00 (2 days)
  Maximum: 7-00:00:00 (7 days)

Resources:
  Nodes: 32
  CPUs: 1536 (48 per node)
  Memory: 8TB total

Limits:
  Max Nodes per Job: 16
  Max CPUs per Job: 768
  Max Memory per Job: 4TB

Access:
  Allowed Accounts: research, ml-team
  Allowed QoS: normal, high
  Default QoS: normal

Current Status:
  Running Jobs: 28
  Pending Jobs: 15
  Allocated CPUs: 1200/1536 (78%)
  Efficiency: 85% (Good)
```

## Understanding Queue Metrics

### Average Wait Time
The average time pending jobs wait before starting execution.

**Factors affecting wait time:**
- Partition capacity
- Job priority and QoS
- Resource request size
- Current queue depth
- Fair-share scheduling

**Healthy ranges:**
- <1 hour: Excellent responsiveness
- 1-4 hours: Good for production
- 4-12 hours: Acceptable for batch processing
- >12 hours: Investigate bottleneck

### Maximum Wait Time
The longest current wait for any pending job.

**When to investigate:**
- Max wait >24 hours: Check for stuck jobs
- Max wait >> Avg wait: Potential scheduling issue
- Max wait increasing: Growing bottleneck

## Efficiency Analysis

### High Efficiency (90-100%)
**Indicators:**
- Green efficiency bar
- High resource utilization
- Low waste

**Potential issues:**
- May be over-subscribed
- Users might experience long waits
- Consider expanding capacity

### Low Efficiency (<60%)
**Indicators:**
- Red efficiency bar
- Low resource utilization
- High waste

**Common causes:**
- Poor resource request sizing
- Job failures
- Partition misconfiguration
- Seasonal low usage

**Actions:**
- Review job submission patterns
- Educate users on right-sizing
- Consider consolidating partitions
- Check for hardware issues

## Tips

- **Monitor wait times**: Use `W` to identify partitions needing attention
- **Check efficiency regularly**: Low efficiency may indicate training opportunities
- **Use analytics**: Press `A` for in-depth partition analysis
- **Navigate to jobs**: Press `J` to see exactly what's running on a partition
- **Navigate to nodes**: Press `N` to check partition node health
- **Compare partitions**: Sort by efficiency to compare partition utilization
- **Queue depth visualization**: Use visual bars to quickly spot bottlenecks
- **Filter by state**: Use `F3` with `state:UP` to focus on active partitions
- **Capacity planning**: Review analytics regularly for growth trends

## Capacity Planning

Use partition analytics to plan capacity:

1. **Identify bottlenecks**: High queue depth + long wait times
2. **Find underutilized**: Low efficiency + low queue depth
3. **Analyze trends**: Job submission patterns over time
4. **Right-size partitions**: Match partition size to demand
5. **Balance resources**: Distribute workload across partitions

**Analytics dashboard** (`A`) provides:
- Historical utilization trends
- Projected resource needs
- Optimization recommendations
- Cost-benefit analysis for expansion
