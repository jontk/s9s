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
ls -la /usr/local/bin/s9s

# Add to PATH (bash)
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
source ~/.bashrc

# Add to PATH (zsh)
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.zshrc
source ~/.zshrc

# Or use full path
/usr/local/bin/s9s
```

#### Permission denied during installation

**Problem**: Cannot write to system directories

**Solutions**:
```bash
# Option 1: Use sudo
sudo mv s9s /usr/local/bin/

# Option 2: Install to user directory
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/
export PATH=$PATH:~/.local/bin

# Option 3: Fix permissions
sudo chown $USER:$USER /usr/local/bin/s9s
chmod +x /usr/local/bin/s9s
```

### Connection Issues

#### Cannot connect to SLURM cluster

**Problem**: S9S cannot reach SLURM REST API

**Diagnostics**:
```bash
# Test connection
s9s --debug
s9s config test

# Check API endpoint
curl -k https://your-slurm-api.com/slurm/v0.0.40/ping

# Verify credentials
echo $SLURM_TOKEN
```

**Solutions**:

1. **Check URL format**:
   ```yaml
   # Correct
   url: https://slurm.example.com:6820

   # Incorrect
   url: slurm.example.com  # Missing protocol
   url: https://slurm.example.com:6820/  # Trailing slash
   ```

2. **Verify network access**:
   ```bash
   # Test connectivity
   ping slurm.example.com
   telnet slurm.example.com 6820

   # Check firewall
   sudo iptables -L | grep 6820
   ```

3. **Handle SSL/TLS issues**:
   ```yaml
   # For self-signed certificates
   clusters:
     default:
       insecureTLS: true

   # Or specify CA certificate
   clusters:
     default:
       tls:
         caFile: /path/to/ca.crt
   ```

#### Authentication failures

**Problem**: Invalid credentials or token

**Solutions**:

1. **Token authentication**:
   ```bash
   # Verify token
   echo $SLURM_TOKEN

   # Test token directly
   curl -H "X-Auth-Token: $SLURM_TOKEN" \
        https://slurm.example.com/slurm/v0.0.40/jobs

   # Refresh token
   scontrol token
   ```

2. **Basic authentication**:
   ```yaml
   auth:
     method: basic
     username: ${SLURM_USER}
     password: ${SLURM_PASS}
   ```

3. **OAuth2 issues**:
   ```bash
   # Test OAuth2 flow
   s9s auth login --cluster production

   # Clear cached tokens
   rm -rf ~/.s9s/tokens/
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
   preferences:
     unicodeSupport: false
     colorMode: 16  # Fallback to 16 colors
     theme: simple  # Basic theme
   ```

#### Screen flickering or slow updates

**Problem**: Performance issues

**Solutions**:

1. **Adjust refresh rate**:
   ```bash
   # Slower refresh
   :set refresh 10s

   # Disable auto-refresh
   :set refresh 0
   ```

2. **Reduce data displayed**:
   ```bash
   # Limit results
   :set pageSize 25

   # Hide unnecessary columns
   :columns JobID,Name,State,Time
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
```bash
# Check active filters
:filters show

# Clear all filters
:clear

# Verify with squeue
:!squeue -u $USER
```

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
     default:
       apiVersion: v0.0.40  # or latest
   ```

3. **Partition visibility**:
   ```bash
   # List visible partitions
   :partitions list

   # Check partition access
   sinfo -s
   ```

#### Incorrect job states

**Problem**: Stale or wrong job information

**Solutions**:

1. **Force refresh**:
   ```bash
   # Manual refresh
   Ctrl+R

   # Clear cache
   :cache clear
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

1. **Optimize queries**:
   ```yaml
   performance:
     maxResults: 100  # Limit results
     cacheEnabled: true
     cacheTTL: 60s
   ```

2. **Debug mode analysis**:
   ```bash
   # Enable profiling
   s9s --profile

   # Check debug log
   tail -f ~/.s9s/debug.log
   ```

3. **Network optimization**:
   ```yaml
   clusters:
     default:
       timeout: 60s  # Increase timeout
       compression: true  # Enable compression
   ```

### SSH Issues

#### Cannot SSH to nodes

**Problem**: SSH connection fails from S9S

**Solutions**:

1. **Configure SSH settings**:
   ```yaml
   ssh:
     defaultUser: ${USER}
     keyFile: ~/.ssh/id_rsa
     extraArgs: "-o StrictHostKeyChecking=no"
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

Enable comprehensive debugging:

```bash
# Start with debug logging
s9s --debug --log-level=trace

# Debug specific component
s9s --debug-component=api
s9s --debug-component=ui
s9s --debug-component=ssh

# Save debug session
s9s --debug 2>&1 | tee debug.log
```

### Configuration Validation

Verify configuration:

```bash
# Validate syntax
s9s config validate

# Test specific cluster
s9s config test --cluster production

# Show effective configuration
s9s config show --resolved
```

### API Testing

Test SLURM API directly:

```bash
# Test API endpoint
curl -k -H "X-Auth-Token: $SLURM_TOKEN" \
     https://slurm.example.com/slurm/v0.0.40/ping

# List jobs via API
curl -k -H "X-Auth-Token: $SLURM_TOKEN" \
     https://slurm.example.com/slurm/v0.0.40/jobs

# Test with S9S
s9s api GET /jobs
s9s api GET /nodes
```

### Log Analysis

Check S9S logs:

```bash
# View recent logs
tail -n 100 ~/.s9s/s9s.log

# Search for errors
grep ERROR ~/.s9s/s9s.log

# Monitor logs
tail -f ~/.s9s/s9s.log

# Rotate logs
s9s logs rotate
```

## Diagnostic Commands

### Built-in Diagnostics

```bash
# System information
:diag system

# Connection test
:diag connection

# Performance metrics
:diag performance

# Configuration check
:diag config

# Full diagnostic report
:diag full > diagnostic-report.txt
```

### Health Checks

```bash
# API health
:health api

# Cache status
:health cache

# Plugin status
:health plugins

# Overall health
:health all
```

## Getting Help

### Collect Debug Information

When reporting issues, include:

```bash
# Generate support bundle
s9s support-bundle

# Manual collection
s9s --version > support.txt
s9s config show --sanitized >> support.txt
s9s diag full >> support.txt
tar czf s9s-debug.tar.gz ~/.s9s/logs/
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

# Reset to defaults
s9s reset

# Or manual reset
rm -rf ~/.s9s
s9s setup
```

### Clear Cache

```bash
# Clear all caches
s9s cache clear

# Clear specific cache
s9s cache clear --type=api
s9s cache clear --type=ui

# Manual cache clear
rm -rf ~/.s9s/cache/
```

### Reinstall S9S

```bash
# Backup config
cp ~/.s9s/config.yaml ~/s9s-config-backup.yaml

# Remove S9S
sudo rm /usr/local/bin/s9s
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

- Review [Configuration Guide](../configuration.md) for optimization
- Learn [Performance Tuning](../performance.md)
- Set up [Monitoring](../monitoring.md)
- Join our [Community](https://discord.gg/s9s) for help
