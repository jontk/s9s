# Configuration Guide

S9s offers extensive configuration options to customize your experience. This guide covers all configuration methods and options.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration Files](#configuration-files)
- [Environment Variables](#environment-variables)
- [Command-line Flags](#command-line-flags)
- [Configuration Schema](#configuration-schema)
- [Authentication Methods](#authentication-methods)
- [UI and Display Configuration](#ui-and-display-configuration)
- [Advanced Configuration](#advanced-configuration)
- [Security Configuration](#security-configuration)
- [Configuration Management](#configuration-management)
- [Best Practices](#best-practices)

## Quick Start

Create a minimal configuration file at `~/.s9s/config.yaml`:

```yaml
clusters:
  default:
    url: https://slurm.example.com
    auth:
      method: token
      token: ${SLURM_TOKEN}
```

Then run s9s:

```bash
export SLURM_TOKEN=your-token-here
s9s
```

For testing without a real SLURM cluster:

```bash
s9s --mock --mock-jobs 200 --mock-nodes 100
```

## Configuration Files

S9s looks for configuration files in the following order (later files override earlier ones):

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

### Project-Specific Configuration

Create `.s9s.yaml` in your project directory for project-specific settings:

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
| `S9S_THEME` | Color theme | `dark` | `light` |
| `S9S_REFRESH_INTERVAL` | Auto-refresh interval | `30s` | `1m` |
| `S9S_DEFAULT_VIEW` | Starting view | `jobs` | `nodes` |

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
| `S9S_EDITOR` | External editor | `$EDITOR` or `vi` | `vim` |
| `S9S_PAGER` | External pager | `$PAGER` or `less` | `more` |

### Environment Variable Substitution

Use `${VAR_NAME}` syntax in configuration files:

```yaml
clusters:
  default:
    url: ${SLURM_API_URL}
    auth:
      token: ${SLURM_TOKEN}

# With defaults
preferences:
  theme: ${S9S_THEME:-dark}
  refreshInterval: ${S9S_REFRESH:-5s}
```

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
      method: string         # Required: token|basic|cert|oauth2

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

      # OAuth2 auth
      clientId: string       # OAuth2 client ID
      clientSecret: string   # OAuth2 client secret
      tokenURL: string       # OAuth2 token URL
      scopes: [string]       # OAuth2 scopes
      redirectURL: string    # OAuth2 redirect URL

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
  confirm_actions: boolean   # Confirm destructive actions (default: true)

  # Display settings
  time_format: string        # Time format (default: "15:04:05")
  date_format: string        # Date format (default: "2006-01-02")
  show_relative_time: boolean # Show relative times (default: true)
  timezone: string           # Timezone (default: "UTC")

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
    key_file: string         # SSH key file path
    known_hosts_file: string # Known hosts file

  # Export settings
  export:
    default_format: string   # csv|json|yaml|markdown
    csv_separator: string    # CSV separator (default: ,)
    json_indent: integer     # JSON indent (default: 2)

# Column configuration
columns:
  jobs:
    - JobID
    - Name
    - User
    - State
  nodes:
    - NodeName
    - State
    - CPULoad

# Keyboard bindings
keybindings:
  global:
    "ctrl+q": "quit"
    "ctrl+s": "save"
  jobs:
    "d": "delete"

# Filter presets
filters:
  my-jobs:
    view: jobs
    filter: "user:${USER}"

# Notification settings
notifications:
  enabled: boolean
  desktop: boolean
  sound: boolean
  webhook:
    url: string
    events: [string]

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

Generate a token:

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

### OAuth2 Authentication

OAuth2-based authentication:

```yaml
clusters:
  research:
    url: https://research.example.edu:6820
    auth:
      method: oauth2
      clientId: ${OAUTH_CLIENT_ID}
      clientSecret: ${OAUTH_CLIENT_SECRET}
      tokenURL: https://auth.example.edu/token
      scopes: ["slurm.read", "slurm.write"]
      redirectURL: http://localhost:8080/callback
```

## UI and Display Configuration

### Basic UI Settings

```yaml
preferences:
  # Theme settings
  theme: dark|light|terminal|custom

  # View settings
  default_view: jobs|nodes|dashboard
  refresh_interval: 5s|10s|30s|0 (disabled)

  # Display options
  show_relative_time: true|false
  use_24hour_time: true|false
  show_seconds: true|false
  date_format: "2006-01-02 15:04:05"
  timezone: "UTC"|"Local"|"America/New_York"

  # Behavior
  confirm_actions: true|false
  auto_refresh: true|false
  show_hints: true|false
```

### Column Configuration

Customize visible columns per view:

```yaml
columns:
  jobs:
    - JobID
    - Name
    - User
    - State
    - Time
    - Nodes
    - Partition
    - Priority
    - QoS
    - Account

  nodes:
    - NodeName
    - State
    - CPULoad
    - Memory
    - RealMemory
    - AllocMemory
    - FreeMem
    - GPUs
    - Jobs
    - Features
```

### Custom Keybindings

Override default shortcuts:

```yaml
keybindings:
  # Global shortcuts (work everywhere)
  global:
    "ctrl+q": "quit"
    "ctrl+s": "save"
    "ctrl+/": "search"
    "f1": "help"
    "ctrl+\\": "toggle-sidebar"

  # View-specific shortcuts
  jobs:
    "d": "delete"
    "D": "delete --force"
    "ctrl+c": "cancel"
    "ctrl+h": "hold"
    "ctrl+r": "release"

  nodes:
    "shift+d": "drain --reason='Maintenance'"
    "shift+r": "resume"
```

### Built-in Themes

- `dark` - Dark background, light text
- `light` - Light background, dark text
- `terminal` - Classic terminal green
- `high-contrast` - Accessibility optimized

### Custom Themes

Create `~/.s9s/themes/custom.yaml`:

```yaml
name: "My Theme"
colors:
  # Base colors
  background: "#1a1a1a"
  foreground: "#e0e0e0"
  selection: "#3a3a3a"
  cursor: "#ffffff"

  # UI elements
  border: "#404040"
  title: "#ffffff"
  subtitle: "#b0b0b0"

  # Status colors
  success: "#50fa7b"
  warning: "#f1fa8c"
  error: "#ff5555"
  info: "#8be9fd"

  # Job states
  running: "#50fa7b"
  pending: "#f1fa8c"
  failed: "#ff5555"
  completed: "#8be9fd"
  cancelled: "#ff79c6"
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

### Filter Presets

Create commonly used filters:

```yaml
filters:
  my-jobs:
    view: jobs
    filter: "user:${USER}"
  gpu-jobs:
    view: jobs
    filter: "partition:gpu state:RUNNING"
  failed-today:
    view: jobs
    filter: "state:FAILED time:>today"
```

### Notification Settings

```yaml
notifications:
  enabled: true
  desktop: true
  sound: true
  webhook:
    url: https://slack.example.com/webhook
    events:
      - job_complete
      - job_failed
      - node_down
```

### Logging Configuration

```yaml
logging:
  level: debug|info|warn|error
  file: ~/.s9s/s9s.log
  max_size: 100MB
  max_backups: 3
  max_age: 30d
  compress: true

  # Separate log levels
  levels:
    api: debug
    ui: info
    ssh: warn
```

### Performance Tuning

```yaml
performance:
  # Connection pooling
  max_idle_connections: 100
  max_connections_per_host: 10
  idle_connection_timeout: 90s

  # Caching
  cache:
    enabled: true
    size: 100MB
    ttl: 60s
    compression: true

  # Request handling
  request_timeout: 30s
  response_timeout: 60s
  keep_alive: 30s

  # UI performance
  virtual_scrolling: true
  lazy_loading: true
  debounce_delay: 300ms
```

### SSH Configuration

```yaml
preferences:
  ssh:
    command: ssh
    user: ${USER}
    key_file: ~/.ssh/id_rsa
    known_hosts_file: ~/.ssh/known_hosts
    compression: true
    forward_agent: true
    extra_args: "-o StrictHostKeyChecking=ask"
```

### Export Settings

```yaml
preferences:
  export:
    default_format: csv
    default_path: ~/Documents/s9s-exports
    include_headers: true
    date_format: RFC3339
    csv_separator: ","
    json_indent: 2
```

## Security Configuration

### Secure Token Storage

Use system keyring (recommended):

```yaml
clusters:
  secure:
    url: https://slurm.example.com
    auth:
      method: token
      token_provider: keyring
      token_key: s9s-prod-token
```

### Best Practices for Sensitive Data

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
   chmod 600 ~/.slurm/token
   ```

4. **Certificate validation**:
   ```yaml
   clusters:
     strict:
       url: https://slurm.example.com
       tls:
         insecure: false
         ca_file: /etc/ssl/certs/ca-bundle.crt
         verify_hostname: true
         min_version: "1.2"
   ```

## Configuration Management

### Validate Configuration

```bash
s9s config validate
s9s config test --cluster production
```

### Export/Import Configuration

```bash
# Export current config
s9s config export > config-backup.yaml

# Import configuration
s9s config import config-backup.yaml

# Merge configurations
s9s config merge additional-config.yaml
```

### Configuration Wizard

```bash
# Interactive setup
s9s setup

# Guided cluster addition
s9s config add-cluster

# Update specific settings
s9s config set preferences.theme dark
s9s config set clusters.default.url https://new-url.com
```

## Configuration Examples

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
version: "1.0"

preferences:
  theme: dark
  refresh_interval: 5s
  default_view: jobs

logging:
  level: debug
  file: ./s9s-debug.log
```

Run with:

```bash
s9s --mock --config dev-config.yaml
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

## Configuration Validation

S9s validates configuration on startup:

1. **Syntax validation**: YAML parsing
2. **Schema validation**: Required fields, types
3. **Connection validation**: Can connect to cluster
4. **Permission validation**: Can perform basic operations

Validation errors are reported clearly:

```
Error: Invalid configuration
  - clusters.production.auth.method: must be one of: token, basic, cert, oauth2
  - preferences.refresh_interval: invalid duration format
```

## Best Practices

1. **Use environment variables** for sensitive data
2. **Version control** your config (exclude secrets with `.gitignore`)
3. **Test changes** with `s9s config test`
4. **Backup** before major changes
5. **Use keyring** for token storage
6. **Separate** dev/prod configurations
7. **Document** custom settings
8. **Set proper file permissions** on configuration files
9. **Validate** configuration after changes
10. **Keep tokens** in files with restricted permissions

## Next Steps

- See [Installation Guide](./installation.md) for setup instructions
- Check [Quick Start](./quickstart.md) to get started
- Explore [User Guide](../user-guide/index.md) for feature documentation
- Learn about [SSH Integration](../guides/ssh-integration.md)
- Set up [Notifications](../guides/notifications.md)
- Create [Custom Themes](../guides/theming.md)
