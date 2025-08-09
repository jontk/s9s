# s9s Configuration Reference

This document provides a comprehensive reference for configuring s9s.

## Table of Contents

- [Configuration Files](#configuration-files)
- [Environment Variables](#environment-variables)
- [Command-line Flags](#command-line-flags)
- [Configuration Schema](#configuration-schema)
- [Authentication Methods](#authentication-methods)
- [Advanced Configuration](#advanced-configuration)
- [Examples](#examples)

## Configuration Files

s9s looks for configuration files in the following order (later files override earlier ones):

1. `/etc/s9s/config.yaml` - System-wide configuration
2. `~/.config/s9s/config.yaml` - User configuration (XDG standard)
3. `~/.s9s/config.yaml` - User configuration (legacy location)
4. `./.s9s.yaml` - Project-specific configuration
5. File specified by `--config` flag

### File Format

Configuration files use YAML format:

```yaml
# s9s configuration file
version: "1.0"

# Default cluster to connect to
default_cluster: production

# Cluster configurations
clusters:
  production:
    url: https://slurm.example.com
    auth:
      method: token
      token: ${SLURM_TOKEN}
    
  development:
    url: https://slurm-dev.example.com
    auth:
      method: basic
      username: ${SLURM_USER}
      password: ${SLURM_PASS}

# User preferences
preferences:
  theme: dark
  refresh_interval: 30s
  default_view: jobs
```

## Environment Variables

All environment variables are prefixed with `S9S_`.

### Core Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `S9S_CONFIG` | Configuration file path | `~/.s9s/config.yaml` | `/etc/s9s/config.yaml` |
| `S9S_DEBUG` | Enable debug mode | `false` | `true` |
| `S9S_LOG_FILE` | Debug log file path | `~/.s9s/debug.log` | `/var/log/s9s.log` |
| `S9S_MOCK` | Enable mock mode | `false` | `true` |
| `S9S_CLUSTER` | Default cluster name | First in config | `production` |

### SLURM Connection Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SLURM_URL` | SLURM REST API URL | - | `https://slurm.example.com` |
| `SLURM_API_VERSION` | API version | `v0.0.43` | `v0.0.43` |
| `SLURM_TOKEN` | Authentication token | - | `eyJhbGci...` |
| `SLURM_USER` | Username for basic auth | - | `admin` |
| `SLURM_PASS` | Password for basic auth | - | `secretpass` |
| `SLURM_CERT` | Client certificate path | - | `/path/to/cert.pem` |
| `SLURM_KEY` | Client key path | - | `/path/to/key.pem` |
| `SLURM_CA` | CA certificate path | - | `/path/to/ca.pem` |
| `SLURM_INSECURE` | Skip TLS verification | `false` | `true` |

### UI Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `S9S_THEME` | Color theme | `dark` | `light` |
| `S9S_REFRESH_INTERVAL` | Auto-refresh interval | `30s` | `1m` |
| `S9S_DEFAULT_VIEW` | Starting view | `jobs` | `nodes` |
| `S9S_EDITOR` | External editor | `$EDITOR` or `vi` | `vim` |
| `S9S_PAGER` | External pager | `$PAGER` or `less` | `more` |

## Command-line Flags

Command-line flags override all other configuration sources.

### Connection Flags

```bash
s9s [flags]

# Connection
--cluster NAME          Select cluster from config
--url URL              SLURM REST API URL
--token TOKEN          Authentication token
--username USER        Username for basic auth
--password PASS        Password for basic auth
--api-version VERSION  SLURM API version (default: v0.0.43)
--insecure            Skip TLS certificate verification

# Mock mode
--mock                 Run in mock mode (no SLURM required)
--mock-users N         Number of mock users (default: 50)
--mock-jobs N          Number of mock jobs (default: 200)
--mock-nodes N         Number of mock nodes (default: 100)
```

### UI Flags

```bash
# UI preferences
--theme THEME          Color theme (dark|light)
--refresh INTERVAL     Auto-refresh interval (e.g., 30s, 1m)
--no-refresh          Disable auto-refresh
--view VIEW           Start with specific view

# Output
--export FORMAT       Export data and exit (csv|json|yaml)
--output FILE         Output file for export
--quiet              Suppress non-error output
```

### Debug Flags

```bash
# Debugging
--debug               Enable debug mode
--log-file FILE       Debug log file path
--trace              Enable trace logging
--profile            Enable profiling
```

## Configuration Schema

### Complete Schema

```yaml
# Configuration file version
version: string  # Required: "1.0"

# Default cluster
default_cluster: string  # Optional

# Cluster configurations
clusters:
  <cluster_name>:
    # Connection settings
    url: string              # Required: SLURM REST API URL
    api_version: string      # Optional: API version (default: v0.0.43)
    timeout: duration        # Optional: Request timeout (default: 30s)
    retry_attempts: integer  # Optional: Retry attempts (default: 3)
    retry_delay: duration    # Optional: Retry delay (default: 1s)
    
    # Authentication
    auth:
      method: string         # Required: token|basic|cert
      
      # Token auth
      token: string          # Required for token auth
      token_file: string     # Alternative: read token from file
      
      # Basic auth
      username: string       # Required for basic auth
      password: string       # Required for basic auth
      
      # Certificate auth
      cert: string           # Required for cert auth
      key: string            # Required for cert auth
      ca: string             # Optional: CA certificate
    
    # TLS settings
    tls:
      insecure: boolean      # Skip verification (default: false)
      server_name: string    # Override server name
      min_version: string    # Minimum TLS version

# User preferences
preferences:
  # UI settings
  theme: string              # dark|light (default: dark)
  refresh_interval: duration # Auto-refresh (default: 30s)
  default_view: string       # Starting view (default: jobs)
  show_hints: boolean        # Show keyboard hints (default: true)
  
  # Display settings
  time_format: string        # Time format (default: "15:04:05")
  date_format: string        # Date format (default: "2006-01-02")
  
  # Table settings
  table:
    border: boolean          # Show borders (default: true)
    header: boolean          # Show headers (default: true)
    row_lines: boolean       # Show row lines (default: false)
    column_spacing: integer  # Column spacing (default: 1)
  
  # Job view settings
  jobs:
    default_filter: string   # Default filter
    show_completed: boolean  # Show completed jobs (default: false)
    columns: [string]        # Visible columns
    sort_by: string          # Default sort column
    sort_order: string       # asc|desc
  
  # Node view settings
  nodes:
    show_offline: boolean    # Show offline nodes (default: true)
    group_by: string         # Group by partition|state|none
    columns: [string]        # Visible columns
  
  # SSH settings
  ssh:
    command: string          # SSH command (default: ssh)
    options: [string]        # Additional SSH options
    user: string             # Default SSH user
  
  # Export settings
  export:
    default_format: string   # csv|json|yaml|markdown
    csv_separator: string    # CSV separator (default: ,)
    json_indent: integer     # JSON indent (default: 2)

# Logging settings
logging:
  level: string              # debug|info|warn|error (default: info)
  file: string               # Log file path
  max_size: integer          # Max file size in MB
  max_backups: integer       # Number of backups
  max_age: integer           # Max age in days
  
# Performance settings
performance:
  max_concurrent: integer    # Max concurrent operations
  cache_ttl: duration        # Cache TTL
  batch_size: integer        # Batch operation size

# Security settings
security:
  keyring: boolean           # Use system keyring (default: true)
  encrypt_config: boolean    # Encrypt sensitive config
```

## Authentication Methods

### Token Authentication

Most secure method using JWT tokens:

```yaml
clusters:
  production:
    url: https://slurm.example.com
    auth:
      method: token
      token: ${SLURM_TOKEN}  # From environment
      # OR
      token_file: ~/.slurm/token  # From file
```

Generate token:
```bash
scontrol token
export SLURM_TOKEN=<token>
```

### Basic Authentication

Username/password authentication:

```yaml
clusters:
  production:
    url: https://slurm.example.com
    auth:
      method: basic
      username: admin
      password: ${SLURM_PASS}  # From environment
```

### Certificate Authentication

Client certificate authentication:

```yaml
clusters:
  production:
    url: https://slurm.example.com
    auth:
      method: cert
      cert: /path/to/client.crt
      key: /path/to/client.key
      ca: /path/to/ca.crt  # Optional
```

## Advanced Configuration

### Multiple Clusters

```yaml
default_cluster: production

clusters:
  production:
    url: https://prod-slurm.example.com
    auth:
      method: token
      token: ${PROD_SLURM_TOKEN}
  
  development:
    url: https://dev-slurm.example.com
    auth:
      method: token
      token: ${DEV_SLURM_TOKEN}
  
  testing:
    url: https://test-slurm.example.com
    auth:
      method: basic
      username: testuser
      password: ${TEST_PASS}
```

Switch clusters:
```bash
# Use flag
s9s --cluster development

# Or environment
export S9S_CLUSTER=development
s9s
```

### Per-Project Configuration

Create `.s9s.yaml` in project directory:

```yaml
# Project-specific settings
default_cluster: project-cluster

clusters:
  project-cluster:
    url: https://project.slurm.local
    auth:
      method: token
      token_file: ./.slurm-token

preferences:
  default_view: jobs
  jobs:
    default_filter: "user:${USER}"
```

### Secure Configuration

For sensitive data:

1. **Use environment variables**:
   ```yaml
   auth:
     token: ${SLURM_TOKEN}
     password: ${SLURM_PASS}
   ```

2. **Use file references**:
   ```yaml
   auth:
     token_file: ~/.slurm/token
     cert: ~/.slurm/cert.pem
     key: ~/.slurm/key.pem
   ```

3. **Set file permissions**:
   ```bash
   chmod 600 ~/.s9s/config.yaml
   chmod 700 ~/.s9s
   ```

### Custom Themes

```yaml
preferences:
  theme: custom
  custom_theme:
    background: "#1e1e1e"
    foreground: "#d4d4d4"
    selection: "#264f78"
    cursor: "#ffffff"
    colors:
      black: "#000000"
      red: "#cd3131"
      green: "#0dbc79"
      yellow: "#e5e510"
      blue: "#2472c8"
      magenta: "#bc3fbc"
      cyan: "#11a8cd"
      white: "#e5e5e5"
```

## Examples

### Minimal Configuration

```yaml
clusters:
  default:
    url: https://slurm.example.com
    auth:
      method: token
      token: ${SLURM_TOKEN}
```

### Full-Featured Configuration

```yaml
version: "1.0"
default_cluster: main

clusters:
  main:
    url: https://slurm.example.com
    api_version: v0.0.43
    timeout: 60s
    retry_attempts: 5
    auth:
      method: token
      token: ${SLURM_TOKEN}
    tls:
      insecure: false

preferences:
  theme: dark
  refresh_interval: 15s
  default_view: dashboard
  show_hints: true
  
  time_format: "15:04:05"
  date_format: "Jan 02"
  
  jobs:
    show_completed: false
    columns: ["ID", "Name", "User", "State", "Time"]
    sort_by: "Submit Time"
    sort_order: desc
  
  nodes:
    show_offline: true
    group_by: partition
  
  ssh:
    user: ${USER}
    options: ["-o", "StrictHostKeyChecking=no"]
  
  export:
    default_format: csv
    csv_separator: ","

logging:
  level: info
  file: ~/.s9s/app.log
  max_size: 100
  max_backups: 3
  max_age: 30

performance:
  max_concurrent: 10
  cache_ttl: 5m
  batch_size: 100

security:
  keyring: true
```

### Development Configuration

```yaml
# Development setup with mock data
version: "1.0"

preferences:
  theme: dark
  refresh_interval: 5s
  default_view: jobs

logging:
  level: debug
  file: ./s9s-debug.log

# No clusters defined - will use mock mode
```

Run with:
```bash
s9s --mock --config dev-config.yaml
```

## Configuration Validation

s9s validates configuration on startup:

1. **Syntax validation**: YAML parsing
2. **Schema validation**: Required fields, types
3. **Connection validation**: Can connect to cluster
4. **Permission validation**: Can perform basic operations

Validation errors are reported clearly:
```
Error: Invalid configuration
  - clusters.production.auth.method: must be one of: token, basic, cert
  - preferences.refresh_interval: invalid duration format
```

## Migration Guide

### From Environment Variables

If migrating from environment-only setup:

```bash
# Old way
export SLURM_URL=https://slurm.example.com
export SLURM_TOKEN=abc123
s9s

# New way - create config
cat > ~/.s9s/config.yaml <<EOF
clusters:
  default:
    url: ${SLURM_URL}
    auth:
      method: token
      token: ${SLURM_TOKEN}
EOF
s9s
```

### From Other Tools

Import settings from other SLURM tools:

```bash
# Import from squeue defaults
s9s import squeue > ~/.s9s/config.yaml

# Import from existing config
s9s import --from /etc/slurm/config.yaml
```

---

For more information, see the [main documentation](https://s9s.dev/docs/configuration).