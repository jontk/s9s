# SSH Integration Guide

Seamlessly access cluster nodes directly from S9S with powerful SSH integration features for debugging, monitoring, and interactive work.

## Overview

S9S SSH integration provides:
- One-click SSH access to any cluster node
- Automatic SSH key management and authentication
- Multi-node SSH sessions and command execution
- Integration with job debugging workflows
- SSH tunneling for secure connections
- Custom SSH configurations per cluster

## Quick SSH Access

### Basic SSH Operations

| Key | Action | Description |
|-----|--------|--------------|
| `s` | SSH to selected node | Direct SSH connection |
| `Shift+S` | SSH with options | Choose user, key, options |
| `Ctrl+S` | SSH in background | Open SSH in new terminal |
| `Alt+S` | SSH command mode | Execute single command via SSH |

### SSH from Different Views

**From Nodes View**:
```bash
# Navigate to nodes view
:view nodes

# Select a node and press 's'
node001  IDLE    16/32 cores  64GB/128GB  ← [s] SSH here
```

**From Jobs View**:
```bash
# Navigate to jobs view
:view jobs

# Press 's' on a running job to SSH to its allocated nodes
12345  alice  RUNNING  node[001-004]  ← [s] SSH to job nodes
```

**Direct SSH Command**:
```bash
# SSH to specific node
:ssh node001

# SSH with specific user
:ssh alice@node002

# SSH with custom command
:ssh node003 "htop"
```

## SSH Configuration

### Basic Configuration

Configure SSH settings in `~/.s9s/config.yaml`:

```yaml
ssh:
  # Default SSH user
  defaultUser: ${USER}

  # SSH key file
  keyFile: ~/.ssh/id_rsa

  # Known hosts file
  knownHostsFile: ~/.ssh/known_hosts

  # Connection options
  compression: true
  forwardAgent: true
  connectTimeout: 10s

  # Additional SSH arguments
  extraArgs: "-o StrictHostKeyChecking=ask -o ServerAliveInterval=60"
```

### Advanced SSH Configuration

```yaml
ssh:
  # Per-cluster SSH settings
  clusters:
    production:
      defaultUser: prod-user
      keyFile: ~/.ssh/prod_rsa
      proxyJump: gateway.prod.example.com

    development:
      defaultUser: dev-user
      keyFile: ~/.ssh/dev_rsa
      port: 2222

  # SSH client preferences
  client:
    terminal: "xterm-256color"
    shell: "/bin/bash"
    enableX11: true
    compression: true
    keepAlive: 60

  # Security settings
  security:
    strictHostKeyChecking: "ask"
    hostKeyAlgorithms: "ssh-ed25519,rsa-sha2-256"
    kexAlgorithms: "curve25519-sha256,diffie-hellman-group16-sha512"
```

### SSH Key Management

```yaml
ssh:
  keys:
    # Default key
    default: ~/.ssh/id_rsa

    # Per-user keys
    users:
      alice: ~/.ssh/alice_rsa
      bob: ~/.ssh/bob_ed25519

    # Per-partition keys
    partitions:
      gpu: ~/.ssh/gpu_access_rsa
      secure: ~/.ssh/secure_partition_ed25519

  # Key agent settings
  agent:
    useAgent: true
    addKeysOnConnect: true
    keyLifetime: 8h
```

## Interactive SSH Sessions

### Single Node SSH

Connect to individual nodes:

```bash
# Basic SSH connection
s  # Press 's' on selected node

# SSH session opens in new terminal:
user@node001:~$
```

### Multi-Node SSH

Connect to multiple nodes simultaneously:

```bash
# Select multiple nodes (Space to select)
node001  ✓ IDLE
node002  ✓ MIXED
node003  ✓ ALLOCATED

# Press 's' to open SSH to all selected nodes
# Opens multiple terminal windows/tabs
```

### SSH with Job Context

SSH directly to nodes running specific jobs:

```bash
# From jobs view, select a running job
12345  alice  RUNNING  node[001-004]  4h  ← Select this

# Press 's' to SSH to all nodes running this job
# Opens 4 SSH sessions (one per node)

# SSH with job environment loaded
:ssh --job 12345 --load-env
```

## SSH Command Execution

### Single Command Execution

Execute commands without interactive session:

```bash
# Execute command on selected node
:ssh node001 "uptime"

# Output displayed in S9S:
node001: 14:23:45 up 5 days, 3:21, 12 users, load average: 2.1, 1.8, 1.9

# Execute on multiple nodes
:ssh node[001-004] "df -h /scratch"

node001: /dev/sdb1  2.0T  1.8T  200G  90% /scratch
node002: /dev/sdb1  2.0T  1.2T  800G  60% /scratch
node003: /dev/sdb1  2.0T  1.9T  100G  95% /scratch
node004: /dev/sdb1  2.0T  0.5T  1.5T  25% /scratch
```

### Batch SSH Operations

Execute commands across filtered nodes:

```bash
# Run command on all idle GPU nodes
/state:idle features:gpu
:ssh --selected "nvidia-smi"

# Update all nodes in maintenance
/state:maint
:ssh --selected "sudo apt update && sudo apt upgrade -y"

# Check disk space on all nodes
:ssh --all-nodes "df -h" --output disk-usage.txt
```

### Parallel SSH Execution

```bash
# Execute commands in parallel (default)
:ssh node[001-100] "hostname" --parallel --max-concurrent 20

# Execute sequentially
:ssh node[001-010] "reboot" --sequential --wait-between 30s

# Execute with timeout
:ssh node[001-050] "long-running-task.sh" --timeout 300s
```

## SSH for Debugging

### Job Debugging Workflow

Debug running and failed jobs:

```bash
# Debug a running job
:job 12345
:ssh --debug  # SSH with debugging context

# On the node:
user@node001:~$ ps aux | grep job_12345
user@node001:~$ gdb -p <pid>  # Attach debugger
user@node001:~$ strace -p <pid>  # Trace system calls
```

### Failed Job Analysis

```bash
# SSH to nodes where job failed
:job 12345  # Failed job
:ssh --post-mortem

# Automatically navigates to job directory and shows:
# - Job output files
# - Core dumps
# - System logs at time of failure
# - Resource usage at failure time
```

### Interactive Job Monitoring

```bash
# SSH and monitor running job
:ssh node001 --monitor-job 12345

# Opens SSH session with real-time monitoring:
user@node001:~$ # Job monitoring active
CPU: 95.2%  Memory: 18.5GB/32GB  GPU: 87%

# Press Ctrl+M to toggle monitoring display
# Press Ctrl+K to kill the job
# Press Ctrl+S to suspend the job
```

## SSH Tunneling

### Port Forwarding

Set up SSH tunnels for secure access:

```bash
# Forward local port to node service
:ssh node001 --tunnel 8080:localhost:8080
# Access http://localhost:8080 to reach node001:8080

# Forward multiple ports
:ssh node001 --tunnel 8080:localhost:80,9000:localhost:9000

# Dynamic SOCKS proxy
:ssh node001 --socks 1080
# Configure browser to use localhost:1080 as SOCKS proxy
```

### Jupyter/Web Interface Access

```bash
# SSH tunnel for Jupyter notebook
:ssh gpu-node --tunnel 8888:localhost:8888
# Job running Jupyter can be accessed at http://localhost:8888

# SSH tunnel for TensorBoard
:ssh ml-node --tunnel 6006:localhost:6006
# TensorBoard accessible at http://localhost:6006

# SSH tunnel for web-based monitoring
:ssh node001 --tunnel 3000:localhost:3000
```

## File Transfer Integration

### SCP Integration

Transfer files to/from nodes:

```bash
# Copy file to node
:scp local-file.txt node001:/tmp/

# Copy file from node
:scp node001:/scratch/results.dat ~/Downloads/

# Copy between nodes
:scp node001:/data/input.txt node002:/scratch/

# Recursive copy
:scp -r ~/experiment/ node001:/scratch/experiment/
```

### RSYNC Integration

```bash
# Sync directories with rsync
:rsync ~/project/ node001:/scratch/project/ --delete

# Sync with progress and compression
:rsync ~/large-dataset/ node[001-004]:/scratch/data/ \
  --progress --compress --parallel

# Sync job results back
:rsync node[001-004]:/scratch/results/ ~/results/ --merge
```

## SSH Security

### Authentication Methods

**SSH Key Authentication** (Recommended):
```yaml
ssh:
  auth:
    method: key
    keyFile: ~/.ssh/id_rsa
    keyType: ed25519  # or rsa, ecdsa
```

**Certificate Authentication**:
```yaml
ssh:
  auth:
    method: certificate
    certFile: ~/.ssh/id_rsa-cert.pub
    keyFile: ~/.ssh/id_rsa
    ca: ~/.ssh/ca.pub
```

**Multi-Factor Authentication**:
```yaml
ssh:
  auth:
    method: key+otp
    keyFile: ~/.ssh/id_rsa
    otpMethod: totp  # or hotp, yubikey
```

### Security Best Practices

```yaml
ssh:
  security:
    # Disable password auth
    passwordAuth: false

    # Require specific key types
    allowedKeyTypes: ["ed25519", "rsa-sha2-256"]

    # Connection limits
    maxConnections: 10
    connectionTimeout: 30s

    # Host verification
    strictHostKeyChecking: true
    verifyHostKeyDNS: true

    # Audit logging
    logConnections: true
    logCommands: true
    logFile: ~/.s9s/ssh.log
```

## SSH Automation

### Automated SSH Scripts

Create reusable SSH automation:

```yaml
# ~/.s9s/ssh-scripts/maintenance.yaml
name: "Node Maintenance"
description: "Standard node maintenance tasks"
script:
  - ssh: "sudo apt update"
  - ssh: "sudo apt upgrade -y"
  - ssh: "sudo reboot"
  - wait: 60s
  - ssh: "uptime"  # Verify reboot
```

Execute SSH scripts:
```bash
:ssh-script maintenance --nodes node[001-010]
```

### Scheduled SSH Tasks

```bash
# Schedule regular health checks
:schedule daily "health-check" \
  ":ssh --all-nodes 'df -h && free -m && uptime' --log"

# Schedule log rotation
:schedule weekly "log-rotation" \
  ":ssh --all-nodes 'sudo logrotate /etc/logrotate.conf'"
```

## SSH Troubleshooting

### Connection Issues

```bash
# Test SSH connectivity
:ssh-test node001

SSH Connection Test Results:
✅ DNS resolution: node001.cluster.edu (10.1.2.101)
✅ Port 22 connectivity: Open
✅ SSH handshake: Success
✅ Authentication: Key accepted
✅ Shell access: /bin/bash

Connection successful in 1.2s
```

### Debug SSH Problems

```bash
# Enable SSH debugging
:config set ssh.debug true

# SSH with verbose output
:ssh node001 --verbose

# Check SSH agent
:ssh-agent status

SSH Agent Status:
✅ Agent running (PID 12345)
✅ Keys loaded: 3
   - ~/.ssh/id_rsa (RSA 4096)
   - ~/.ssh/id_ed25519 (ED25519)
   - ~/.ssh/gpu_key (RSA 2048)
```

### Common SSH Fixes

```bash
# Fix known_hosts issues
:ssh-keygen --remove-host node001

# Reset SSH agent
:ssh-agent restart

# Update SSH config
:ssh-config validate
:ssh-config repair
```

## SSH Customization

### Custom SSH Commands

Define frequently used SSH commands:

```yaml
ssh:
  aliases:
    logs: "journalctl -f -n 100"
    top: "htop -u $USER"
    gpu: "nvidia-smi -l 5"
    temp: "sensors | grep temp"
    disk: "df -h && du -sh /scratch/$USER"
```

Use custom commands:
```bash
:ssh node001 logs    # Equivalent to: ssh node001 "journalctl -f -n 100"
:ssh gpu-node gpu    # Equivalent to: ssh gpu-node "nvidia-smi -l 5"
```

### SSH Profiles

Create SSH profiles for different scenarios:

```yaml
ssh:
  profiles:
    debug:
      terminal: tmux
      commands:
        - "cd /scratch/$USER"
        - "module load gdb"
        - "export DEBUG=1"

    monitoring:
      commands:
        - "watch -n 1 'ps aux | head -20'"
        - "tail -f /var/log/slurm/slurmd.log"
```

Use profiles:
```bash
:ssh node001 --profile debug
:ssh node002 --profile monitoring
```

## Best Practices

### SSH Usage

1. **Use SSH keys** - Never use password authentication
2. **Keep keys secure** - Protect private keys, rotate regularly
3. **Use SSH agent** - Avoid typing passphrases repeatedly
4. **Limit connections** - Don't open unnecessary SSH sessions
5. **Close idle sessions** - Set appropriate timeouts

### Security

1. **Verify host keys** - Always verify on first connection
2. **Use strong ciphers** - Prefer modern encryption algorithms
3. **Enable logging** - Audit SSH access and commands
4. **Restrict access** - Use SSH certificates for access control
5. **Monitor connections** - Watch for suspicious SSH activity

### Performance

1. **Use compression** - Enable SSH compression for slow networks
2. **Multiplex connections** - Reuse SSH connections when possible
3. **Optimize ciphers** - Choose appropriate cipher for your network
4. **Use connection pooling** - Maintain persistent connections
5. **Limit concurrent sessions** - Avoid overwhelming nodes

## Next Steps

- Configure [Node Operations](../node-operations.md) with SSH integration
- Learn [Job Management](../job-management.md) with SSH debugging
- Set up [Batch Operations](../batch-operations.md) with SSH automation
- Explore [Performance Monitoring](../performance.md) via SSH
