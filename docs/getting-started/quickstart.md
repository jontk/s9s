# Quick Start Guide

Get up and running with s9s in just a few minutes! This guide covers the basics to help you start managing your SLURM cluster more efficiently.

## 🚀 First Launch

### With SLURM Cluster

If you have access to a SLURM cluster:

```bash
# Launch s9s
s9s

# Or connect to a specific cluster
s9s --cluster production
```

### Without SLURM (Mock Mode)

Try s9s without a SLURM cluster:

```bash
# Launch in mock mode with simulated data
s9s --mock
```

![Overview Demo](/assets/demos/overview.gif)

Mock mode is perfect for:
- Learning s9s navigation
- Testing workflows
- Demo purposes
- Development and testing

## 🎯 Basic Navigation

### View Switching

s9s organizes information into focused views. Switch between them using:

- **Tab** - Cycle forward through views
- **Shift+Tab** - Cycle backward through views
- **J** - Jump to Jobs view
- **N** - Jump to Nodes view
- **P** - Jump to Partitions view

Common views:
1. **Dashboard** - Cluster overview with metrics
2. **Jobs** - Monitor and manage jobs
3. **Nodes** - View and manage compute nodes
4. **Partitions** - Monitor partitions and queues
5. **Users** - View user accounts
6. **Accounts** - Account hierarchy
7. **QoS** - Quality of Service policies
8. **Reservations** - Resource reservations
9. **Health** - Cluster health monitoring

### Essential Keyboard Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `?` | Help | Show context-sensitive keyboard shortcuts |
| `q` | Quit | Exit s9s |
| `/` | Filter | Filter current view |
| `F3` | Advanced filter | Expression-based filtering |
| `Ctrl+F` | Global search | Search across all resources |
| `Tab` | Next view | Cycle through views |
| `Enter` | Details | View detailed information |
| `ESC` | Cancel | Exit dialog/filter/modal |
| `R` | Refresh | Manual data refresh |

See [Keyboard Shortcuts](../user-guide/keyboard-shortcuts.md) for complete reference.

## 📊 Dashboard View

The Dashboard is your cluster command center:

![Dashboard Demo](/assets/demos/dashboard.gif)

### What You See

- **Cluster Overview** - Health status, CPU/Memory usage
- **Jobs Summary** - Job states and queue depth
- **Nodes Summary** - Node availability and utilization
- **Partition Status** - Top partitions
- **Alerts & Issues** - System health warnings
- **Performance Trends** - Historical metrics

### Quick Actions

| Key | Action |
|-----|--------|
| `J` | Jump to Jobs view |
| `N` | Jump to Nodes view |
| `P` | Jump to Partitions view |
| `A` | Advanced analytics |
| `H` | Health check details |

## 📊 Jobs View

The Jobs view is where you'll manage your workload:

![Jobs Demo](/assets/demos/jobs.gif)

### View Jobs

```bash
# Launch s9s directly to jobs view
s9s --view jobs
```

### Common Operations

| Key | Action | Use Case |
|-----|--------|----------|
| `Enter` | View details | See full job information |
| `s/S` | Submit job | Launch job submission wizard |
| `c/C` | Cancel job | Cancel a running/pending job |
| `H` | Hold job | Prevent job from starting |
| `r` | Release job | Release a held job |
| `o/O` | View output | View job logs in real-time |
| `d/D` | Dependencies | Show job dependency graph |
| `b/B` | Batch operations | Manage multiple jobs |

### Filtering Jobs

Use `/` to filter jobs:

```
# Examples:
/RUNNING        # Show only running jobs
/gpu            # Find jobs with "gpu" in any field
/p:gpu          # Filter by partition "gpu"
/user:alice     # Jobs by user alice
```

Press `ESC` to clear filters.

For complex filtering, press `F3`:
```
state:RUNNING partition:gpu
user:alice priority:>500
nodes:>=8
```

## 💻 Nodes View

Monitor and manage compute nodes:

![Nodes Demo](/assets/demos/nodes.gif)

### Common Operations

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Node information and metrics |
| `d/D` | Drain | Mark node for maintenance |
| `r` | Resume | Return drained node to service |
| `s/S` | SSH | Connect to node via SSH |
| `g/G` | Group by | Group nodes (partition/state/features) |

### Node States

- **IDLE** (Green) - Available for jobs
- **ALLOCATED** (Cyan) - Running jobs
- **MIXED** (Yellow) - Partially allocated
- **DRAIN** (Orange) - Scheduled for maintenance
- **DOWN** (Red) - Offline/unavailable

### Resource Usage

Nodes display dual-bar visualization:
```
CPU: ████████░░░░░░░░ 8/16 (Load: 7.5)
     ^^^^^^^^          Actual usage (solid)
             ░░░░░░    Allocated but unused (hollow)
```

## 🎯 Practical Examples

### Example 1: Submit a New Job

1. Press `s/S` in Jobs view
2. Fill in job submission wizard:
   - Job name
   - Partition
   - Number of nodes/CPUs
   - Time limit
   - Script or command
3. Review and submit

### Example 2: Cancel Failed Jobs

1. In Jobs view, press `/` and type `FAILED`
2. Navigate to a job with arrow keys
3. Press `c` to cancel
4. Confirm with `y` (or use batch operations for multiple)

### Example 3: SSH to a Node

1. Switch to Nodes view (press `N`)
2. Find your node (use `/nodename` to filter)
3. Press `s` to open SSH menu
4. Select "Quick Connect"
5. Interactive SSH session opens

### Example 4: Monitor GPU Jobs

1. In Jobs view, press `/`
2. Type `p:gpu` to filter GPU partition
3. Press `Enter` to apply
4. Press `Enter` on any job for GPU details

### Example 5: Batch Cancel Pending Jobs

1. In Jobs view, press `v/V` for multi-select mode
2. Navigate with arrow keys
3. Press `Space` to select specific jobs
4. Or press `b/B` and choose "Cancel all PENDING"

## 🔍 Advanced Features

### Global Search

Press `Ctrl+F` for cluster-wide search:

```
# Search finds jobs, nodes, partitions, etc.
Ctrl+F → type "gpu001"
# Shows: gpu001 node, jobs on gpu001, etc.
```

### Advanced Filtering

Press `F3` for expression-based filtering:

```bash
# Jobs view
state:RUNNING partition:gpu nodes:>4

# Nodes view
state:IDLE partition:compute cpus:>64

# Partitions view
efficiency:>80 qos:high
```

### Batch Operations

Select multiple items:

1. Press `v/V` to enter multi-select mode
2. Press `Space` on items to select
3. Press `Ctrl+A` to select all
4. Press `b/B` for batch menu
5. Choose operation (cancel, hold, release, etc.)

## ⚡ Command Mode

Press `:` to enter command mode (vim-style):

```bash
:help              # Show help
:quit              # Exit s9s
:view nodes        # Switch to nodes view
:filter user:john  # Apply filter
:refresh           # Force refresh
```

## 🎨 Quick Customization

### Auto-Refresh

In Jobs view:
```
Press m/M to toggle auto-refresh on/off
```

### Grouping (Nodes)

```
Press g/G in Nodes view
Choose: partition, state, features, or none
Press Space to expand/collapse groups
```

## 📝 Next Steps

Now that you know the basics, explore:

1. **[Views Overview](../user-guide/views/index.md)** - Detailed view documentation
2. **[Keyboard Shortcuts](../user-guide/keyboard-shortcuts.md)** - Complete shortcuts
3. **[Job Management](../user-guide/job-management.md)** - Advanced job operations
4. **[Configuration](configuration.md)** - Customize s9s
5. **[Filtering Guide](../user-guide/filtering.md)** - Master advanced filters

## 💡 Pro Tips

1. **Practice with mock mode** - `s9s --mock` is risk-free
2. **Use keyboard shortcuts** - Faster than mouse for everything
3. **Multi-select is powerful** - Batch operations save time
4. **Pin important filters** - Save frequently used filters
5. **Context help** - Press `?` in any view for specific shortcuts
6. **Group nodes** - Makes large clusters manageable
7. **SSH from UI** - No need to remember node names
8. **Watch job output** - Press `o/O` for real-time log streaming

## 🆘 Getting Help

- **In-app help**: Press `?` anywhere for context-sensitive help
- **View documentation**: Press `F1` for action menu
- **Full docs**: [https://s9s.dev/docs](https://s9s.dev/docs)
- **Troubleshooting**: [Troubleshooting Guide](../guides/troubleshooting.md)
- **GitHub Issues**: [Report bugs](https://github.com/jontk/s9s/issues)

## Common Questions

**Q: How do I exit s9s?**
A: Press `q` to quit.

**Q: Can I use s9s without SLURM?**
A: Yes! Use `s9s --mock` for a fully functional demo mode.

**Q: How do I cancel multiple jobs?**
A: Use multi-select (`v/V`), select jobs with `Space`, then press `b/B` for batch operations.

**Q: Can I SSH to nodes from s9s?**
A: Yes! Press `s/S` in Nodes view for SSH options.

**Q: How do I filter by partition?**
A: Use `/` and type `p:partitionname` or `partition:name`.

**Q: Is there a way to save my filters?**
A: Yes, see [Configuration Guide](configuration.md) for saved filters.

**Q: Can I customize colors/theme?**
A: Yes, see [Configuration Guide](configuration.md) for theme options.
