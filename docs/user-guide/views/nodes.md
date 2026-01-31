# Nodes View

The Nodes view provides comprehensive node monitoring and management with visual resource utilization, state tracking, and SSH integration.

![Nodes Demo](/assets/demos/nodes.gif)

*Nodes view showing resource usage, node states, and grouping capabilities*

## Overview

The Nodes view displays all compute nodes in your cluster with real-time resource usage visualization, state information, and administrative actions like drain/resume and SSH access.

## Table Columns

| Column | Description |
|--------|-------------|
| **Node Name** | Node identifier |
| **State** | Color-coded node status |
| **Partitions** | Associated partition names |
| **CPU Usage** | Dual visual bar with allocated vs. actual usage |
| **Memory Usage** | Dual visual bar with allocated vs. actual usage |
| **CPU Total** | Total available CPUs |
| **Memory Total** | Total available memory |
| **Features** | Node feature tags |
| **Reason** | Drain reason or status message |

## Node States

Node states are color-coded for quick identification:

| State | Color | Description |
|-------|-------|-------------|
| **IDLE** | Green | Available for work |
| **ALLOCATED** | Cyan | Fully assigned to jobs |
| **MIXED** | Yellow | Partially allocated |
| **DOWN** | Red | Unavailable/offline |
| **DRAIN** | Orange | Being drained |
| **IDLE+DRAIN** | Orange | Idle but draining |
| **COMPLETING** | Blue | Jobs completing |
| **RESERVED** | Magenta | Reserved |

## Resource Usage Visualization

CPU and Memory columns show dual-bar visualizations:

```
CPU: ████████░░░░░░░░ 8/16 (Load: 7.5)
     ^^^^^^^^          Actual usage (solid █)
             ░░░░░░    Allocated but unused (hollow □)
                   ···· Free (dots ·)
```

### Visual Indicators
- **█** (solid block) = Actual resource usage
- **░** (light shade) = Allocated but not actively used
- **·** (dot) = Free/unallocated

### CPU Usage Details
Shows:
- Allocated CPUs / Total CPUs
- System load average
- Visual bar showing allocation and usage

**Example**: `8/16 (Load: 7.5)` means 8 of 16 CPUs allocated with load average of 7.5

### Memory Usage Details
Shows:
- Allocated memory / Total memory (formatted as GB/TB)
- Visual bar showing allocation and usage

**Example**: `32GB/64GB` means 32 GB allocated out of 64 GB total

## Node Actions

### View Node Details
**Shortcut**: `Enter`

Shows comprehensive node information:

**Node Information:**
- Node name and state
- Associated partitions
- Node features/capabilities
- Reason for current state (if draining/down)

**Resource Details:**
- Total CPUs and allocated CPUs
- CPU load efficiency (load/allocated ratio)
- Total memory and allocated memory
- Memory efficiency (usage/allocated ratio)

**Running Jobs:**
- List of active jobs on the node
- Job IDs and users

**Performance Metrics:**
- CPU efficiency percentage
- Memory efficiency percentage
- Overall resource utilization

### Drain Node
**Shortcut**: `d/D`

Drains the selected node with a custom reason.

**When to drain:**
- Node needs maintenance
- Hardware issues detected
- Preparing for shutdown

**Process:**
1. Enter drain reason
2. Confirm action
3. Node state changes to DRAIN
4. Existing jobs continue running
5. No new jobs will be allocated

### Resume Node
**Shortcut**: `r`

Resumes a drained node, making it available for new jobs.

**Confirmation dialog shows:**
- Node name
- Current state
- Drain reason (if set)

**Effect:**
- Node becomes available for allocation
- State changes from DRAIN to IDLE (or appropriate state)

### SSH Access
**Shortcut**: `s/S`

Opens SSH options menu with four choices:

#### 1. SSH Terminal Manager
Advanced session management with:
- Multiple concurrent SSH sessions
- Session naming and organization
- Quick session switching
- Terminal multiplexing

#### 2. Quick Connect
Direct SSH connection to the node:
- Opens in-app SSH terminal
- Automatic connection using cluster config
- Interactive shell access

#### 3. Test Connection
Verifies SSH connectivity:
- Checks SSH daemon status
- Tests authentication
- Reports connection success/failure
- Useful for troubleshooting

#### 4. Get Node Info
Retrieves system information via SSH:
- OS version and kernel
- CPU model and count
- Memory size
- Disk usage
- Network interfaces
- Running SLURM daemons

See [SSH Integration Guide](../../guides/ssh-integration.md) for detailed SSH usage.

## Filtering & Search

### Simple Text Filter
**Shortcut**: `/`

Filters nodes by any column value:
- Node name
- State
- Partition
- Features
- Reason

**Special syntax:**
- `p:partition` or `partition:name` - Filter by partition

### Advanced Filter
**Shortcut**: `F3`

Expression-based filtering with field matching:

```
state:IDLE
partition:gpu
features:nvlink
cpus:>64
memory:>256GB
```

Press `ESC` to exit advanced filter.

### State Filters
| Shortcut | Filter |
|----------|--------|
| `a/A` | Toggle "all states" filter |
| `i/I` | Toggle idle state filter |
| `m/M` | Toggle mixed state filter |

### Partition Filter
**Shortcut**: `p/P`

Opens dialog to filter nodes by specific partition.

### Global Search
**Shortcut**: `Ctrl+F`

Search across all cluster resources from the nodes view.

## Node Grouping

### Group By Dialog
**Shortcut**: `g/G`

Group nodes by:
- **None** - Flat list (default)
- **Partition** - Group by partition membership
- **State** - Group by node state
- **Features** - Group by feature tags

### Group Navigation
When grouped:
- **Space** - Toggle group expansion/collapse
- Groups show summary counts
- Expand groups to see individual nodes

**Example grouped view:**
```
▼ Partition: gpu [8 nodes]
  gpu001  IDLE      ...
  gpu002  ALLOCATED ...
  gpu003  MIXED     ...

▼ Partition: cpu [16 nodes]
  cpu001  IDLE      ...
  cpu002  IDLE      ...
```

## Sorting

Sort nodes by clicking column headers or using number keys.

**Sortable columns:**
- Node Name
- State
- CPUs (total or allocated)
- Memory (total or allocated)

Press `1-9` to sort by column number.

## Keyboard Shortcuts Reference

### Node Operations
| Key | Action |
|-----|--------|
| `Enter` | View node details |
| `d/D` | Drain node |
| `r` | Resume node |
| `s/S` | SSH to node |

### Filtering & Search
| Key | Action |
|-----|--------|
| `/` | Simple filter |
| `F3` | Advanced filter |
| `Ctrl+F` | Global search |
| `p/P` | Filter by partition |
| `a/A` | Toggle all states |
| `i/I` | Toggle idle filter |
| `m/M` | Toggle mixed filter |
| `ESC` | Exit filter mode |

### Grouping
| Key | Action |
|-----|--------|
| `g/G` | Group by dialog |
| `Space` | Toggle group expansion |

### Data Management
| Key | Action |
|-----|--------|
| `R` | Manual refresh |
| `1-9` | Sort by column |

## Node Details Example

When viewing node details (`Enter`), you see:

```
Node: gpu001
State: ALLOCATED
Partitions: gpu, bigmem

Resources:
  CPUs: 48 total, 40 allocated
  CPU Load Efficiency: 95.2% (38.1/40)
  Memory: 256GB total, 220GB allocated
  Memory Efficiency: 98.6% (217GB/220GB)

Features:
  - nvidia_a100
  - nvlink
  - infiniband

Running Jobs:
  12345 (alice) - 20 CPUs, 100GB
  12346 (bob)   - 20 CPUs, 120GB

Performance:
  CPU Efficiency: 95%
  Memory Efficiency: 99%
  Overall Utilization: 97%
```

## Tips

- **Monitor drain status**: Check the Reason column for why nodes are drained
- **Use grouping**: Group by state to quickly find idle nodes
- **Check efficiency**: Node details show CPU/memory efficiency to identify underutilized allocations
- **SSH troubleshooting**: Use "Test Connection" if direct SSH fails
- **Visual bars**: Dual bars help identify over-allocated resources (allocated > actual usage)
- **Filter by partition**: Use `p:gpu` syntax for quick partition filtering
- **Batch drain**: When draining multiple nodes for maintenance, use the filter to identify targets
- **Resume safely**: Check node details before resuming to understand why it was drained

## Resource Efficiency

Node details include efficiency metrics:

**CPU Load Efficiency** = (Actual Load / Allocated CPUs) × 100
- >100% = Oversubscribed
- 80-100% = Efficient utilization
- <80% = Under-utilized

**Memory Efficiency** = (Used Memory / Allocated Memory) × 100
- >100% = Over-committed (potential issue)
- 80-100% = Good utilization
- <80% = Under-utilized

Use these metrics to:
- Identify poorly sized job requests
- Optimize partition configurations
- Guide user education on resource requests
