# Node Operations Guide

Master node management in S9S with powerful operations for monitoring, maintenance, and direct access to cluster nodes.

## Node View Overview

Press `2` or `:view nodes` to access the nodes view, where you can:

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

View detailed information with `Enter` or `d`:

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
| `d` | Show details | View comprehensive node information |
| `l` | View logs | Show node logs and messages |
| `j` | View jobs | List all jobs on this node |
| `s` | SSH to node | Direct SSH access |
| `r` | Refresh | Update node information |

### Maintenance Operations

| Key | Action | Description |
|-----|--------|-------------|
| `D` | Drain node | Prepare node for maintenance |
| `R` | Resume node | Return node to service |
| `U` | Update state | Force state update |
| `M` | Maintenance mode | Put node in maintenance |

### Advanced Operations

| Key | Action | Description |
|-----|--------|-------------|
| `Ctrl+R` | Reboot node | Restart node (admin only) |
| `Ctrl+P` | Power cycle | Hard power cycle |
| `Ctrl+D` | Force drain | Immediate drain with job termination |

## Maintenance Workflows

### Planned Maintenance

1. **Drain the node**:
   ```bash
   # In nodes view, select node and press D
   # Or use command mode
   :drain node001 --reason="Planned maintenance" --timeout=1h
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

1. **Force drain immediately**:
   ```bash
   :drain node001 --force --reason="Emergency maintenance"
   ```

2. **Jobs are immediately terminated**
3. **Node is ready for maintenance**

### Batch Maintenance

Maintain multiple nodes efficiently:

```bash
# Drain multiple nodes
:drain node[001-010] --reason="OS update" --timeout=2h

# Resume multiple nodes
:resume node[001-010]

# Check status of node range
/node:node[001-010]
```

## SSH Integration

### Direct SSH Access

Press `s` on any node to SSH directly:

```bash
# Automatically connects with your configured SSH settings
ssh user@node001.cluster.edu
```

### SSH Configuration

Configure SSH in `~/.s9s/config.yaml`:

```yaml
ssh:
  defaultUser: ${USER}
  keyFile: ~/.ssh/id_rsa
  knownHostsFile: ~/.ssh/known_hosts
  compression: true
  forwardAgent: true
  extraArgs: "-o StrictHostKeyChecking=ask"
```

### SSH Operations

| Key | Action | Description |
|-----|--------|-------------|
| `s` | SSH to node | Interactive SSH session |
| `Ctrl+S` | SSH with options | Choose user, key, options |
| `Alt+S` | Background SSH | SSH in new terminal |

### Bulk SSH Operations

Execute commands across multiple nodes:

```bash
# Run command on all idle nodes
:ssh --filter="state:idle" "uptime"

# Update all nodes in maintenance
:ssh --nodes="node[001-010]" "sudo apt update"

# Check disk space on GPU nodes
:ssh --filter="features:gpu" "df -h"
```

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

### Health Monitoring

S9S monitors node health indicators:

- **Load Average**: System load over 1, 5, 15 minutes
- **Memory Pressure**: Available vs allocated memory
- **Disk Space**: Available disk space on filesystems
- **Network**: Network connectivity and bandwidth
- **Temperature**: Hardware temperature sensors
- **Jobs**: Running and pending job counts

### Alerts and Notifications

Configure alerts for node issues:

```yaml
# In config.yaml
notifications:
  nodeAlerts:
    - condition: "load > 32"
      severity: warning
      message: "High load on {node}"
    - condition: "memory < 10%"
      severity: critical
      message: "Low memory on {node}"
    - condition: "state == DOWN"
      severity: critical
      message: "Node {node} is down"
```

## Node Filtering and Search

### Find Specific Nodes

```bash
# Find nodes by name pattern
/node:compute*

# Find nodes by state
/state:idle

# Find GPU nodes
/features:gpu

# Find nodes with high memory
/memory:>64GB

# Find nodes with specific job count
/jobs:>4
```

### Complex Node Queries

```bash
# Idle GPU nodes with >100GB RAM
/state:idle features:gpu memory:>100GB

# Nodes that haven't been seen recently
/lastseen:>1h state:!DOWN

# Overutilized nodes
/load:>16 cpus:<=16

# Nodes ready for maintenance
/jobs:0 state:idle
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

```bash
# View detailed node diagnostics
:diag node001

# Check node connectivity
:ping node001

# View system logs
:logs node001 --system --lines=100

# Check SLURM daemon status
:slurm-status node001
```

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
