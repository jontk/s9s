# SSH Integration Guide

Interactive SSH access to cluster nodes directly from S9S for debugging, monitoring, and troubleshooting.

## Overview

S9S provides direct SSH access to cluster nodes, allowing you to quickly open interactive terminal sessions for debugging jobs, inspecting node status, and performing administrative tasks.

**Features:**
- One-click interactive SSH to cluster nodes
- SSH connection testing and validation
- Node information retrieval via SSH
- Integration with job debugging workflows
- SSH terminal session management

## Quick SSH Access

### Basic SSH Operations

From the Nodes View, press `s` on a selected node to open an interactive SSH session.

| Key | Action | Description |
|-----|--------|--------------|
| `s` | SSH to selected node | Direct interactive SSH connection |
| `S` | SSH to selected node | Same as lowercase `s` |

### SSH from Different Views

**From Nodes View**:
```bash
# Navigate to nodes view
:nodes

# Select a node and press 's' to open SSH session
node001  IDLE    16/32 cores  64GB/128GB  ← [s] SSH here
```

**From Jobs View**:
```bash
# Navigate to jobs view
:jobs

# Press 's' on a running job to SSH to its allocated nodes
12345  alice  RUNNING  node[001-004]  ← [s] SSH to job nodes
```

## SSH Configuration

S9S uses your system's default SSH configuration (`~/.ssh/config`, SSH agent, etc.). There is no `ssh:` section in the S9S configuration file. SSH connection parameters (username, port, key file) are handled programmatically per-connection based on the node being accessed.

To customize SSH behavior, configure your system SSH settings in `~/.ssh/config`:

```
Host node*
  User your-username
  IdentityFile ~/.ssh/cluster_key
  StrictHostKeyChecking no
  ServerAliveInterval 60
```

## Interactive SSH Sessions

### Single Node SSH

Connect to individual nodes for interactive work:

```bash
# Basic SSH connection
s  # Press 's' on selected node in Nodes View

# SSH session opens in your terminal, suspending s9s temporarily
user@node001:~$
```

When you exit the SSH session, S9S resumes automatically.

### SSH Terminal Manager

S9S provides an advanced SSH terminal manager for managing multiple SSH sessions:

```bash
# From nodes view, select a node
# Choose "SSH Terminal Manager" option

# Features:
# - View active SSH sessions
# - Switch between multiple node connections
# - Monitor session status
# - Quick access to node information
```

### SSH with Job Context

SSH directly to nodes running specific jobs:

```bash
# From jobs view, select a running job
12345  alice  RUNNING  node[001-004]  ← Select this

# Press 's' to SSH to the first node running this job
# Useful for debugging running jobs interactively
```

## SSH Features

### Node Information Retrieval

Gather basic node information via SSH from within the SSH Terminal Manager:

```bash
# In the SSH Terminal Manager, press 'i' to get node info
# Retrieves:
# - Hostname
# - Uptime
# - Memory usage
# - CPU count
# - Disk usage
```

### SSH Terminal Manager Access

When you press `s` on a node in the Nodes View, S9S opens the SSH Terminal Manager directly. From there, you use keyboard shortcuts to perform various actions such as connecting to sessions, retrieving node information, and managing terminals. See the [Keyboard Reference](#keyboard-reference) below for available keybindings.

## SSH Security

### Authentication

S9S relies on your system SSH configuration for authentication. SSH key authentication is recommended.

### Security Best Practices

Configure security settings in your system SSH config (`~/.ssh/config`):

```
Host node*
  StrictHostKeyChecking yes
  UserKnownHostsFile ~/.ssh/known_hosts
  ConnectTimeout 30
```

**Important Security Note**: For production environments, enable strict host key checking in your SSH configuration. In cluster environments where nodes are frequently rebuilt, you may choose to disable it, but be aware of the security implications.

## SSH Troubleshooting

### Connection Issues

If SSH connection fails:

1. **Verify SSH connectivity manually**:
   ```bash
   ssh <nodename>
   ```

2. **Check SSH agent** (if using SSH agent):
   ```bash
   ssh-add -l
   ```

3. **Verify SSH key permissions**:
   ```bash
   chmod 600 ~/.ssh/id_rsa
   ```

4. **Check system SSH configuration**:
   ```bash
   # Verify your SSH config for cluster nodes
   cat ~/.ssh/config
   ```

### Common SSH Issues

**Problem**: "Permission denied (publickey)"
- **Solution**: Ensure your SSH public key is authorized on the target node
- Verify `~/.ssh/authorized_keys` on the node contains your public key

**Problem**: "Connection timeout"
- **Solution**: Check network connectivity to the node
- Verify the node is reachable: `ping <nodename>`

**Problem**: "Host key verification failed"
- **Solution**: Update known_hosts file
- Remove old key: `ssh-keygen -R <nodename>`
- Or disable strict host key checking (less secure)

## Best Practices

### SSH Usage

1. **Use SSH keys** - Never use password authentication
2. **Keep keys secure** - Protect private keys with file permissions (600)
3. **Use SSH agent** - Avoid entering passphrases repeatedly
4. **Close sessions** - Exit SSH sessions when done to free resources
5. **Verify node state** - Check node status before SSH (avoid DOWN or DRAIN nodes)

### Security

1. **Verify host keys** - Use strict host key checking in production
2. **Monitor connections** - Enable SSH connection logging
3. **Restrict access** - Ensure only authorized users have SSH access to nodes
4. **Audit regularly** - Review SSH logs for suspicious activity

## Workflow Examples

### Debug a Running Job

```bash
# 1. Navigate to jobs view
:jobs

# 2. Find your running job
12345  alice  RUNNING  node[001-004]

# 3. Press 's' to SSH to a job node
# S9S suspends, SSH session opens

user@node001:~$ ps aux | grep <your_program>
user@node001:~$ htop -u alice
user@node001:~$ tail -f /path/to/job/output

# 4. Exit SSH session (Ctrl+D or 'exit')
# S9S resumes automatically
```

### Check Node Health

```bash
# 1. Navigate to nodes view
:nodes

# 2. Select a problematic node
# 3. Press 's' to open the SSH Terminal Manager
# 4. Press 'i' to retrieve node info

# S9S retrieves and displays:
# - Uptime
# - Memory usage
# - Disk space
# - CPU count

# Or press Enter/t to open a full SSH terminal session
```

### Investigate Failed Job

```bash
# 1. Find failed job in jobs view
12345  alice  FAILED  node003

# 2. Press 's' to SSH to the node where it failed
# 3. Investigate logs, check for errors

user@node003:~$ cd /scratch/alice/job_12345
user@node003:~$ less slurm-12345.out
user@node003:~$ dmesg | tail
```

## Integration with S9S Workflows

SSH access integrates seamlessly with S9S cluster management:

- **Job Debugging**: SSH to nodes running specific jobs
- **Node Inspection**: Quick access from node status screens
- **Troubleshooting**: Direct access to nodes showing problems
- **Performance Analysis**: Interactive exploration of node resources

## Keyboard Reference

**From Nodes View:**
- `s` or `S` - Open SSH to selected node

**From Jobs View:**
- `s` or `S` - Open SSH to first node running selected job

**From SSH Terminal Manager:**
- `Enter` - Connect to selected node/session
- `c` - Create new SSH connection
- `i` - Show node information
- `t` - Open terminal session
- `s` - Show system information
- `m` or `M` - Monitor session status
- `x` or `X` - Close selected session
- `r` or `R` - Refresh sessions
- `Esc` - Close SSH interface

## Next Steps

- Explore [Job Management](job-management.md) with SSH debugging
- Review [Troubleshooting Guide](troubleshooting.md) for common issues
