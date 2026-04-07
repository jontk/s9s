# Troubleshooting Guide

This guide helps you resolve common issues with S9S. If you can't find a solution here, please check our [GitHub Issues](https://github.com/jontk/s9s/issues) or join our [Discord community](https://discord.gg/s9s).

## Common Issues

### Installation Problems

#### "Command not found" after installation

**Problem**: S9S is installed but not in PATH

**Solutions**:
```bash
# Check if S9S is installed
which s9s
ls -la ~/.local/bin/s9s

# Add to PATH (bash)
echo 'export PATH=$PATH:~/.local/bin' >> ~/.bashrc
source ~/.bashrc

# Add to PATH (zsh)
echo 'export PATH=$PATH:~/.local/bin' >> ~/.zshrc
source ~/.zshrc

# Or use full path
~/.local/bin/s9s
```

#### Permission denied during installation

**Problem**: Cannot write to directories

**Solutions**:
```bash
# Install to user directory (recommended)
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/
chmod +x ~/.local/bin/s9s
export PATH=$PATH:~/.local/bin

# Add to shell rc file for persistence
echo 'export PATH=$PATH:~/.local/bin' >> ~/.bashrc
```

### Connection Issues

#### Cannot connect to SLURM cluster

**Problem**: S9S cannot reach SLURM REST API

> **Important**: s9s requires `slurmrestd` (the SLURM REST API daemon, default port 6820). Having `slurmctld` (port 6817) and `slurmdbd` (port 6819) running is **not sufficient**. Check if slurmrestd is running: `ss -tlnp | grep 6820`

**Diagnostics**:
```bash
# Check if slurmrestd is running
ss -tlnp | grep 6820

# Test connection
s9s --debug
s9s config validate

# Check API endpoint
curl -k https://your-slurm-api.com/slurm/v0.0.43/ping

# Verify credentials
echo $SLURM_JWT
```

**Solutions**:

1. **Start slurmrestd** if it's not running:
   ```bash
   # Start slurmrestd (as root or SlurmUser)
   slurmrestd 0.0.0.0:6820

   # Or with systemd (if configured)
   sudo systemctl start slurmrestd

   # Verify it's listening
   ss -tlnp | grep 6820
   curl http://localhost:6820/slurm/v0.0.43/ping
   ```

2. **Check endpoint format**:
   ```yaml
   # Correct
   endpoint: "https://slurm.example.com:6820"

   # Incorrect
   endpoint: "slurm.example.com"  # Missing protocol
   endpoint: "https://slurm.example.com:6820/"  # Trailing slash
   ```

3. **Verify network access**:
   ```bash
   # Test connectivity
   ping slurm.example.com
   telnet slurm.example.com 6820

   # Check firewall
   sudo iptables -L | grep 6820
   ```

4. **Handle SSL/TLS issues**:
   ```yaml
   # For self-signed certificates
   clusters:
     - name: "default"
       cluster:
         endpoint: "https://slurm.example.com:6820"
         insecure: true
   ```

#### Authentication failures

**Problem**: Invalid credentials or token

**Solutions**:

1. **Token authentication**:
   ```bash
   # Verify token
   echo $SLURM_JWT

   # Test token directly
   curl -H "X-Auth-Token: $SLURM_JWT" \
        https://slurm.example.com/slurm/v0.0.43/jobs

   # Refresh token
   scontrol token
   ```

2. **Generate a new token**:
   ```bash
   # Generate a new SLURM JWT token
   scontrol token

   # Set it in your environment
   export SLURM_JWT="<new-token>"
   ```

### Display Issues

#### Corrupted or garbled display

**Problem**: Terminal compatibility issues

**Solutions**:

1. **Check terminal capabilities**:
   ```bash
   # Verify 256 color support
   tput colors

   # Test UTF-8 support
   echo $LANG
   locale

   # Set proper locale
   export LANG=en_US.UTF-8
   export LC_ALL=en_US.UTF-8
   ```

2. **Try different terminal**:
   - Recommended: iTerm2, Alacritty, kitty
   - Avoid: Windows Command Prompt
   - Use Windows Terminal or WSL2 on Windows

3. **Adjust S9S settings**:
   ```yaml
   ui:
     skin: "default"
     noIcons: true  # Disable icons if they render incorrectly
   ```

#### Screen flickering or slow updates

**Problem**: Performance issues

**Solutions**:

1. **Adjust refresh rate**:
   Change the `refreshRate` setting in your configuration file (`~/.s9s/config.yaml`):
   ```yaml
   refreshRate: "30s"  # Slower refresh (default is 10s)
   ```
   Or press `F6` at runtime to pause auto-refresh entirely; `F5` and `R`
   still work for manual refresh.

2. **Customize visible columns**:
   Configure columns in your configuration file:
   ```yaml
   views:
     jobs:
       columns: [id, name, state, time]
   ```

3. **Check system resources**:
   ```bash
   # Monitor S9S resource usage
   top -p $(pgrep s9s)

   # Check network latency
   ping -c 10 slurm.example.com
   ```

### Data Issues

#### Jobs not showing up

**Problem**: Missing or filtered jobs

**Diagnostics**:
- Press `/` to check if a filter is active, then press `Esc` to clear it
- Press `F5` to force a manual refresh

**Solutions**:

1. **Check permissions**:
   ```bash
   # Verify user can see jobs
   sacctmgr show user $USER

   # Check account associations
   sacctmgr show associations user=$USER
   ```

2. **API version mismatch**:
   ```yaml
   # Update API version
   clusters:
     - name: "default"
       cluster:
         apiVersion: v0.0.43  # or latest
   ```

3. **Partition visibility**:
   ```bash
   # Switch to partitions view in s9s
   :partitions

   # Or check partition access from the command line
   sinfo -s
   ```

#### Incorrect job states

**Problem**: Stale or wrong job information

**Solutions**:

1. **Force refresh**:
   ```bash
   # Manual refresh
   F5       # Press F5 in any view
   ```

2. **Check time sync**:
   ```bash
   # Verify time sync
   timedatectl status

   # Sync time
   sudo ntpdate -s time.nist.gov
   ```

### Performance Problems

#### S9S is slow or unresponsive

**Problem**: Performance degradation

**Solutions**:

1. **Limit displayed jobs**:
   ```yaml
   views:
     jobs:
       maxJobs: 100  # Limit results (default 1000)
   ```

2. **Debug mode analysis**:
   ```bash
   # Enable debug logging
   s9s --debug

   # Check debug log (written to ./s9s-debug.log in current directory)
   tail -f ./s9s-debug.log

   # Check app log (general application log)
   tail -f ~/.s9s/s9s.log
   ```

3. **Increase cluster timeout**:
   ```yaml
   clusters:
     - name: "default"
       cluster:
         endpoint: "https://slurm.example.com:6820"
         timeout: "60s"  # Increase timeout
   ```

### SSH Issues

#### Cannot SSH to nodes

**Problem**: SSH connection fails from S9S

**Solutions**:

1. **Configure SSH via system settings** (`~/.ssh/config`):
   ```
   Host node*
     User your-username
     IdentityFile ~/.ssh/id_rsa
     StrictHostKeyChecking no
   ```

2. **Test SSH manually**:
   ```bash
   # Test connection
   ssh node001

   # Check SSH agent
   ssh-add -l

   # Add key to agent
   ssh-add ~/.ssh/id_rsa
   ```

3. **Node name resolution**:
   ```bash
   # Check DNS
   nslookup node001

   # Add to hosts file
   echo "10.0.0.1 node001" | sudo tee -a /etc/hosts
   ```

## Advanced Troubleshooting

### Debug Mode

Enable debug logging:

```bash
# Start with debug logging
s9s --debug

# Save debug output to a file
s9s --debug 2>&1 | tee debug.log
```

### API Testing

Test the SLURM REST API directly to isolate connection issues:

```bash
# Test API endpoint
curl -k -H "X-Auth-Token: $SLURM_JWT" \
     https://slurm.example.com/slurm/v0.0.43/ping

# List jobs via API
curl -k -H "X-Auth-Token: $SLURM_JWT" \
     https://slurm.example.com/slurm/v0.0.43/jobs
```

### Log Analysis

The `--debug` flag writes a debug log to `./s9s-debug.log` in the current working directory. The general app log is at `~/.s9s/s9s.log`.

```bash
# View recent debug log (created by --debug flag)
tail -n 100 ./s9s-debug.log

# Search for errors in debug log
grep ERROR ./s9s-debug.log

# Monitor debug log in real time
tail -f ./s9s-debug.log

# View general app log
tail -n 100 ~/.s9s/s9s.log
```

## Diagnostic Information

To view cluster health information, use the built-in health view:

```bash
# Switch to health view
:health

# Or press 9 to switch to the health view
```

For configuration issues, use the config view:

```bash
# Open configuration
:config
```

> **Note**: Additional diagnostic commands are planned. See [#119](https://github.com/jontk/s9s/issues/119) for planned diagnostic commands.

## Getting Help

### Collect Debug Information

When reporting issues, include:

```bash
# Check s9s version
s9s --version

# Run with debug logging
s9s --debug 2>&1 | tee debug.log

# Collect debug log if available
tar czf s9s-debug.tar.gz ./s9s-debug.log ~/.s9s/s9s.log
```

### Community Support

- **Discord**: [Join our server](https://discord.gg/s9s)
- **GitHub Issues**: [Report bugs](https://github.com/jontk/s9s/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jontk/s9s/discussions)

### Enterprise Support

For enterprise support:
- Email: support@s9s.dev
- Priority support available
- SLA guarantees
- Custom development

## Recovery Procedures

### Reset S9S

Complete reset:

```bash
# Backup configuration
cp -r ~/.s9s ~/.s9s.backup

# Manual reset
rm -rf ~/.s9s

# S9S will recreate defaults on next launch
s9s
```

### Clear Cache

```bash
# Manual cache clear
rm -rf ~/.s9s/cache/
```

### Reinstall S9S

```bash
# Backup config
cp ~/.s9s/config.yaml ~/s9s-config-backup.yaml

# Remove S9S
rm ~/.local/bin/s9s
rm -rf ~/.s9s

# Reinstall
curl -sSL https://get.s9s.dev | bash

# Restore config
mkdir -p ~/.s9s
cp ~/s9s-config-backup.yaml ~/.s9s/config.yaml
```

## Prevention Tips

1. **Keep S9S updated**: Check for updates regularly
2. **Monitor logs**: Set up log rotation and monitoring
3. **Test changes**: Use mock mode for testing
4. **Backup config**: Version control your configuration
5. **Document issues**: Keep notes on resolved problems

## Next Steps

- Review [Configuration Reference](../reference/configuration.md) for optimization
- Join our [Community](https://discord.gg/s9s) for help
