# Configuration Guide

S9s offers extensive configuration options to customize your experience. This guide covers all configuration methods and options.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration Files](#configuration-files)
- [Environment Variables](#environment-variables)
- [Command-line Flags](#command-line-flags)
- [Configuration Schema](#configuration-schema)
- [Authentication](#authentication)
- [UI and Display Configuration](#ui-and-display-configuration)
- [Advanced Configuration](#advanced-configuration)
- [Security Configuration](#security-configuration)
- [Configuration Management](#configuration-management)
- [Best Practices](#best-practices)

## Quick Start

Create a minimal configuration file at `~/.s9s/config.yaml`:

```yaml
clusters:
  - name: default
    cluster:
      endpoint: https://slurm.example.com:6820
      token: ${SLURM_JWT}  # or discovered via scontrol token
```

Then run s9s:

```bash
export SLURM_JWT=$(scontrol token)
s9s
```

For testing without a real SLURM cluster, enable mock mode (requires `S9S_ENABLE_MOCK=true` environment variable):

```bash
S9S_ENABLE_MOCK=true s9s --mock
```

See the [Mock Mode Guide](../guides/mock-mode.md) for details.

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

# Default cluster to connect to
defaultCluster: production

# Cluster configurations
clusters:
  - name: production
    cluster:
      endpoint: https://slurm.example.com:6820
      token: ${SLURM_JWT}  # or discovered via scontrol token / SLURM_JWT env var

  - name: development
    cluster:
      endpoint: https://slurm-dev.example.com:6820
      token: ${DEV_SLURM_JWT}

# UI preferences
ui:
  skin: default
```

### Project-Specific Configuration

Create `.s9s.yaml` in your project directory for project-specific settings:

```yaml
# Project-specific settings
defaultCluster: project-cluster

clusters:
  - name: project-cluster
    cluster:
      endpoint: https://project.slurm.local:6820
      token: ${SLURM_JWT}
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
| `SLURM_URL` | SLURM REST API URL | - | `https://slurm.example.com:6820` |
| `SLURM_API_VERSION` | API version | auto-detected | `v0.0.43` |
| `SLURM_JWT` | Authentication token (auto-discovered via `scontrol token` if not set) | - | `eyJhbGci...` |
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
--api-version VERSION  SLURM API version (default: auto-detected)
--insecure            Skip TLS certificate verification

# Mock mode (requires S9S_ENABLE_MOCK=true)
--mock                 Run in mock mode (no SLURM required)
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

> **Note:** Some configuration options shown below are planned features. See [#117](https://github.com/jontk/s9s/issues/117) for status. The currently implemented config struct supports: `RefreshRate`, `MaxRetries`, `DefaultCluster`, `Clusters[]` (with `Endpoint`, `Token`, `APIVersion`, `Insecure`, `Timeout`), `UI` (skin, logoless, crumbsless, statusless, headless, noIcons, enableMouse), `Views` (jobs, nodes, partitions), `Features` (streaming, pulseye, xray), `Shortcuts[]`, `Aliases`, `Plugins[]`, `UseMockClient`, `PluginSettings`, and `Discovery`.

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
    api_version: string      # Optional: API version (default: auto-detected)
    timeout: duration        # Optional: Request timeout (default: 30s)
    retry_attempts: integer  # Optional: Retry attempts (default: 3)
    retry_delay: duration    # Optional: Retry delay (default: 1s)

    # Authentication (token-based via SLURM JWT)
    token: string            # JWT token or env var reference (e.g., ${SLURM_JWT})
                             # Auto-discovered via scontrol token if not set
    insecure: boolean        # Skip TLS verification (default: false)

# User preferences (many of these are planned -- see note above)
preferences:
  # UI settings
  theme: string              # dark|light (default: dark)
  refresh_interval: duration # Auto-refresh (default: 30s)
  default_view: string       # Starting view (default: jobs)

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

# Notification settings (planned)
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

# Performance settings (planned)
performance:
  max_concurrent: integer    # Max concurrent operations
  cache_ttl: duration        # Cache TTL
  batch_size: integer        # Batch operation size

# Security settings (planned)
security:
  keyring: boolean           # Use system keyring (default: true)
  encrypt_config: boolean    # Encrypt sensitive config
```

## Authentication

S9s uses token-based authentication with SLURM's JWT tokens. The token is resolved in the following order:

1. **Explicit config** -- the `token` field in your cluster configuration
2. **Environment variable** -- `SLURM_JWT`
3. **Auto-discovery** -- runs `scontrol token` to obtain a token automatically

```yaml
clusters:
  - name: production
    cluster:
      endpoint: https://slurm.example.com:6820
      token: ${SLURM_JWT}  # or discovered via scontrol token
```

Generate a token manually:

```bash
export SLURM_JWT=$(scontrol token)
s9s
```

If you are running s9s on a node where `scontrol` is available and your user has permission to generate tokens, no explicit token configuration is needed -- s9s will auto-discover it.

## UI and Display Configuration

### Basic UI Settings

```yaml
preferences:
  # Theme settings
  theme: dark|light

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

### Job Submission Configuration

Customize the job submission form with default values, hidden fields, dropdown options, and reusable templates. These settings live under the `views.jobs.submission` key:

```yaml
views:
  jobs:
    submission:
      # Global defaults applied to all new jobs
      formDefaults:
        partition: "compute"
        timeLimit: "04:00:00"
        nodes: 1
        cpus: 4
        memory: "8G"

      # Fields to hide from the form
      hiddenFields:
        - arraySpec
        - exclusive
        - requeue

      # Restrict dropdown options
      fieldOptions:
        partition: ["compute", "gpu", "highmem"]
        qos: ["normal", "high"]

      # Control template sources: builtin, config, saved
      templateSources: ["builtin", "config", "saved"]

      # Custom templates
      templates:
        - name: "GPU Training Job"
          description: "PyTorch training on GPU partition"
          defaults:
            partition: "gpu"
            timeLimit: "24:00:00"
            cpus: 8
            memory: "32G"
            gpus: 2
            cpusPerTRES: "gres/gpu:4"
          hiddenFields: ["arraySpec"]
```

**`formDefaults`** -- Default values pre-filled in the submission form for all new jobs (including "Custom Job"). Template-specific defaults override these values. Any field from the [field reference](../user-guide/job-management.md#submission-wizard-fields) can be used as a key (camelCase).

**`hiddenFields`** -- A list of form fields to hide globally. These fields are hidden for all templates and Custom Job. Per-template `hiddenFields` are additive — a template cannot un-hide a globally hidden field. Field names use the camelCase config key format.

**`fieldOptions`** -- Restricts dropdown menus to a specific set of values. Supported fields: `partition`, `qos`, `account`.

**`templateSources`** -- Controls which template sources appear in the template picker. Possible values:

| Value | Description |
|-------|-------------|
| `"builtin"` | 8 built-in templates shipped with s9s |
| `"config"` | Templates defined in this config file under `templates` |
| `"saved"` | User-saved templates from `~/.s9s/templates/*.json` |

Default: `["builtin", "config", "saved"]` (all three). Invalid values are silently filtered; if all values are invalid, falls back to showing all three sources.

Examples:
```yaml
# Only show templates defined in config (org-managed)
templateSources: ["config"]

# Only show user-saved templates
templateSources: ["saved"]

# Hide built-ins, show config + saved
templateSources: ["config", "saved"]
```

**`templates`** -- Define reusable job templates. Each template has a `name`, optional `description`, a `defaults` map (same keys as `formDefaults`), and an optional `hiddenFields` list that applies when the template is selected.

- **Name-based override**: If a config template has the same name as a built-in template, it replaces the built-in entirely. Similarly, a saved template with the same name replaces both config and built-in versions.
- **Per-template `hiddenFields`**: These are additive with the global `hiddenFields`. When the template is selected, fields in both lists are hidden.
- **Advanced field visibility**: Fields not in the default-visible set are automatically hidden unless the template's `defaults` sets a non-zero value for them. This means a template only shows the fields it actually uses.

See the [Template Management Commands](../reference/commands.md#template-management-commands) for CLI operations and the [field reference](../user-guide/job-management.md#submission-wizard-fields) for all available field keys.

#### Default Value Precedence

Values are applied in this order (later overrides earlier):

1. **Hardcoded defaults** — `timeLimit: "01:00:00"`, `nodes: 1`, `cpus: 1`
2. **`formDefaults`** from config
3. **Current SLURM user** — `account` defaults to user's `DefaultAccount` (or username if no default), `qos` defaults to user's `DefaultQoS`
4. **Template defaults** — if a template is selected
5. **Working directory** — defaults to the directory where s9s was launched

#### Field Options Behavior

> **`partition` and `account`**: The configured values are *intersected* with values fetched from the cluster API. Only values that exist in both the config list and the cluster are shown. If the intersection is empty (e.g., the cluster doesn't have any of the configured partitions), all cluster values are shown as a fallback. If no cluster data is available, the config values are shown as-is.
>
> **`qos`**: QoS values are fetched from the cluster API and filtered by `fieldOptions.qos` (same intersection behavior as partition/account). If neither the cluster nor config provides QoS values, a plain text input is shown instead of a dropdown.
>
> **`account` and `qos` dropdowns** include an empty option at the top, allowing users to submit without specifying these fields. `partition` does not have an empty option since it is required.

#### Boolean Overlay Limitation

> Boolean fields (`exclusive`, `requeue`, `hold`, `contiguous`, `overcommit`, etc.) use zero-value overlay semantics. Once set to `true` by `formDefaults`, a template cannot set them back to `false`. To work around this, avoid setting boolean fields in `formDefaults` and instead set them per-template.

#### Migration from `showBuiltinTemplates`

> `showBuiltinTemplates` (boolean) is the legacy equivalent of `templateSources`. Setting `showBuiltinTemplates: false` is equivalent to `templateSources: ["config", "saved"]`. If both are set, `templateSources` takes precedence. New configurations should use `templateSources`.

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

### UI Skin

The UI skin can be set via the `ui.skin` configuration key:

```yaml
ui:
  skin: "default"
```

## Advanced Configuration

### Multiple Clusters

```yaml
defaultCluster: production

clusters:
  - name: production
    cluster:
      endpoint: https://prod-slurm.example.com:6820
      token: ${PROD_SLURM_JWT}

  - name: development
    cluster:
      endpoint: https://dev-slurm.example.com:6820
      token: ${DEV_SLURM_JWT}

  - name: testing
    cluster:
      endpoint: https://test-slurm.example.com:6820
      token: ${TEST_SLURM_JWT}
```

Switch clusters:

```bash
# Use flag
s9s --cluster development

# Or environment
export S9S_CLUSTER=development
s9s

# Or interactively with Ctrl+K while s9s is running
```

When multiple clusters are configured, the active cluster name is shown in the header bar. Press `Ctrl+K` to open the cluster switcher and select a different cluster without restarting.

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

### Best Practices for Sensitive Data

1. **Use environment variables**:
   ```yaml
   clusters:
     - name: production
       cluster:
         endpoint: https://slurm.example.com:6820
         token: ${SLURM_JWT}
   ```

2. **Use `scontrol token` auto-discovery** -- if running on a node with `scontrol` available, omit the `token` field entirely and s9s will discover it automatically.

3. **Set file permissions**:
   ```bash
   chmod 600 ~/.s9s/config.yaml
   chmod 700 ~/.s9s
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

If auto-discovery doesn't find your cluster, run the setup wizard to configure your endpoint and authentication:

```bash
s9s setup
```

## Configuration Examples

### Minimal Configuration

```yaml
clusters:
  - name: default
    cluster:
      endpoint: https://slurm.example.com:6820
      token: ${SLURM_JWT}
```

### Full-Featured Configuration

```yaml
defaultCluster: main

clusters:
  - name: main
    cluster:
      endpoint: https://slurm.example.com:6820
      apiVersion: v0.0.43  # Optional: omit to auto-detect from slurmrestd
      token: ${SLURM_JWT}
      timeout: 60s
      insecure: false

refreshRate: 15s
maxRetries: 5

ui:
  skin: default
  enableMouse: true

features:
  streaming: true
  pulseye: true
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
export SLURM_JWT=abc123
s9s

# New way - create config
cat > ~/.s9s/config.yaml <<EOF
clusters:
  - name: default
    cluster:
      endpoint: ${SLURM_URL}
      token: ${SLURM_JWT}
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
  - clusters[0].cluster.endpoint: required field missing
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
