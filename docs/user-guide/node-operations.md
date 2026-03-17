# Node Operations Guide

Master node management in S9S with powerful operations for monitoring, maintenance, and direct access to cluster nodes.

## Node View Overview

Press `2` or `:nodes` to access the nodes view, where you can:

- Monitor all cluster nodes in real-time
- View detailed node specifications and utilization
- Perform maintenance operations
- Access nodes directly via SSH
- Drain and resume nodes for maintenance

## Node Information

### Node States

S9S displays nodes in various states:

| State | Description | Color |
|-------|-------------|-------|
| **IDLE** | Available for new jobs | Green |
| **MIXED** | Some CPUs allocated, some free | Yellow |
| **ALLOCATED** | Fully utilized by jobs | Blue |
| **DOWN** | Node is offline | Red |
| **DRAIN** | Being drained for maintenance | Orange |
| **DRAINING** | Actively draining jobs | Orange |
| **FAIL** | Node has failed | Red |
| **MAINT** | In maintenance mode | Gray |

### Node Details

View detailed information with `Enter`:

```bash
Node: node001.cluster.edu
State: MIXED (4/16 CPUs allocated)
Features: gpu,nvme,infiniband
OS: Linux 5.4.0-74-generic
Architecture: x86_64
Real Memory: 128 GB
Allocated Memory: 32 GB (25%)
Free Memory: 96 GB
GPUs: 2x NVIDIA A100-SXM4-40GB
Jobs: 2 running, 0 pending
Boot Time: 2023-12-01 08:30:15
Last Seen: 2023-12-15 14:23:42 (5s ago)
```

## Node Operations

### Basic Operations

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Show details | View comprehensive node information |
| `d`/`D` | Drain | Prepare node for maintenance |
| `r` | Resume | Return node to service |
| `R` | Refresh | Update node information |
| `s` | SSH to node | Direct SSH access |

### State Filter Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `a`/`A` | All states | Clear state filter |
| `i`/`I` | Idle filter | Toggle idle state filter |
| `m`/`M` | Mixed filter | Toggle mixed state filter |
| `p`/`P` | Partition filter | Prompt for partition filter |
| `g`/`G` | Group by | Group nodes by partition, state, or features |
| `e`/`E` | Export | Open export dialog |

## Maintenance Workflows

### Planned Maintenance

1. **Drain the node**:
   ```bash
   # In nodes view, select node and press D
   # Or use command mode
   :drain node001 Planned maintenance
   ```

2. **Wait for jobs to complete**:
   - Monitor draining progress
   - Jobs will finish naturally
   - New jobs won't be scheduled

3. **Perform maintenance**:
   ```bash
   # SSH to node for maintenance
   s  # Press 's' on selected node
   ```

4. **Resume the node**:
   ```bash
   # After maintenance, resume the node
   :resume node001
   ```

### Emergency Maintenance

1. **Drain the node with a reason**:
   ```bash
   :drain node001 Emergency maintenance
   ```

2. **Monitor draining progress** -- jobs will finish naturally
3. **Perform maintenance once drained**

### Batch Maintenance

Drain individual nodes using the command mode:

```bash
# Drain nodes one at a time
:drain node001 OS update
:drain node002 OS update

# Resume nodes after maintenance
:resume node001
:resume node002

# Filter to see specific nodes
/node001
```

> **Note:** The `:drain` command accepts a single node name and an optional reason string. Node ranges and `--reason`/`--timeout` flags are planned. See [#119](https://github.com/jontk/s9s/issues/119).

## SSH Integration

### Direct SSH Access

Press `s` on any node to SSH directly:

```bash
# Automatically connects with your configured SSH settings
ssh user@node001.cluster.edu
```

### SSH Operations

| Key | Action | Description |
|-----|--------|-------------|
| `s` | SSH to node | Interactive SSH session to selected node |

Press `s` on a selected node in the Nodes view to open an SSH session. See [SSH Integration Guide](../guides/ssh-integration.md) for configuration.

## Node Monitoring

### Resource Utilization

Monitor real-time resource usage:

```bash
# CPU utilization
CPU: ████████░░░░░░░░ 8/16 cores (50%)

# Memory usage
Memory: ██████░░░░░░░░░░ 32/128 GB (25%)

# GPU utilization
GPU 0: ████████████████ 100% (job_12345)
GPU 1: ░░░░░░░░░░░░░░░░ 0% (idle)
```

### Resource Display

S9S displays the following node resource information:

- **CPU Usage**: Allocated vs total CPUs, CPU load
- **Memory Usage**: Allocated vs total memory
- **State**: Current node state with color coding
- **Partitions**: Which partitions the node belongs to
- **Features**: Node feature tags
- **Reason**: Drain/down reason if applicable

## Node Filtering and Search

### Find Specific Nodes

The `/` quick filter performs plain text search across all visible columns:

```bash
# Find nodes by name
/compute              # Nodes containing "compute"
/gpu                  # Nodes with "gpu" in any column
/node001              # Specific node
```

For field-specific filtering, use the advanced filter (`Ctrl+F`):

```bash
# Advanced filter examples (Ctrl+F)
state=idle                              # Idle nodes
state=mixed                             # Mixed-state nodes
name~compute                            # Nodes containing "compute"
memory>64000                            # Nodes with >64GB RAM (in MB)
```

## Troubleshooting Node Issues

### Common Node Problems

**Node shows as DOWN**:
1. Check network connectivity
2. Verify SLURM daemon is running
3. Check system logs
4. Restart slurmd if needed

**Node not accepting jobs**:
1. Check if node is drained
2. Verify available resources
3. Check job constraints vs node features
4. Review partition configuration

**High load but no jobs**:
1. Check for system processes
2. Look for hung or zombie processes
3. Check for I/O wait issues
4. Review system logs

### Node Diagnostics

View node details by selecting a node and pressing `Enter` in the Nodes view. For deeper diagnostics, SSH to the node with `s`.

> **Note:** Command-mode diagnostic commands (`:diag`, `:ping`, `:logs`, `:slurm-status`) are planned. See [#119](https://github.com/jontk/s9s/issues/119).

## Best Practices

### Node Management

1. **Plan maintenance windows** - Use drain with timeout
2. **Monitor during drainage** - Ensure jobs complete cleanly
3. **Verify after maintenance** - Test functionality before resuming
4. **Document changes** - Use descriptive drain reasons
5. **Batch operations** - Maintain multiple nodes efficiently

### Resource Monitoring

1. **Set up alerts** - Proactive monitoring prevents issues
2. **Regular health checks** - Monitor trends over time
3. **Capacity planning** - Track utilization patterns
4. **Performance baselines** - Know normal vs abnormal behavior

### SSH Security

1. **Use SSH keys** - Avoid password authentication
2. **Limit access** - Restrict SSH to necessary users
3. **Audit connections** - Log and monitor SSH usage
4. **Keep keys secure** - Rotate and protect SSH keys

## Next Steps

- Learn about [Batch Operations](./batch-operations.md) to manage multiple nodes efficiently
- Explore [Advanced Filtering](../filtering.md) for powerful node queries
- Set up [Export capabilities](./export.md) to analyze node data
