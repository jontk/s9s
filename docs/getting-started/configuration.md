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

S9s looks for `config.yaml` in the following paths (first match wins):

1. `./config.yaml` - Current directory
2. `$HOME/.s9s/config.yaml` - User configuration
3. `/etc/s9s/config.yaml` - System-wide configuration
4. File specified by `--config` flag (overrides search)

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
      # user: admin  # Override SLURM username (default: OS user)

  - name: development
    cluster:
      endpoint: https://slurm-dev.example.com:6820
      token: ${DEV_SLURM_JWT}
      user: root  # Use root for dev clusters where tokens are generated for root

# UI preferences
ui:
  skin: default
```

## Environment Variables

S9s uses the `S9S_` prefix for its own environment variables and also supports unprefixed SLURM variables. Viper's `AutomaticEnv()` maps environment variables to config keys using the `S9S_` prefix with `.` replaced by `_` (e.g., `S9S_UI_SKIN` maps to `ui.skin`).

### SLURM Connection Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SLURM_REST_URL` or `S9S_SLURM_REST_URL` | SLURM REST API URL | - | `https://slurm.example.com:6820` |
| `SLURM_JWT` or `S9S_SLURM_JWT` | Authentication token (auto-discovered via `scontrol token` if not set) | - | `eyJhbGci...` |
| `SLURM_API_VERSION` | API version | `v0.0.43` | `v0.0.40` |
| `SLURM_USER_NAME` | Override SLURM username for `X-SLURM-USER-NAME` header (takes precedence over config `cluster.user` and OS user) | OS user | `root` |

### Mock Mode Variable

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `S9S_ENABLE_MOCK` | Enable mock mode (required for `--mock` flag); any non-empty value enables it | unset | `1`, `true`, `dev` |

### Environment Variable Substitution

Use `${VAR_NAME}` syntax in configuration files to reference environment variables:

```yaml
clusters:
  - name: default
    cluster:
      endpoint: ${SLURM_REST_URL}
      token: ${SLURM_JWT}
```

> **Note:** The `${VAR:-default}` syntax with fallback values is not supported. Use plain `${VAR}` references only.

## Command-line Flags

Command-line flags override all other configuration sources.

### Connection Flags

```bash
s9s [flags]

# Configuration
--config PATH          Custom config file location
--cluster NAME         Select cluster from config
--debug                Enable debug logging

# Discovery
--no-discovery         Disable auto-discovery
--discovery-timeout    Timeout for auto-discovery (e.g., 10s, 30s)

# Version
--version, -v          Show version information

# Subcommands
s9s update             Check for and install updates
s9s update --check     Only check, don't install
s9s update --target    Update to a specific version
```

## Configuration Schema

### Complete Schema

The following schema reflects the actual `Config` struct in the codebase. All field names use camelCase as defined by the `mapstructure` tags.

```yaml
# General settings
refreshRate: duration        # Auto-refresh interval (default: 2s)
maxRetries: integer          # Max API retry attempts (default: 3)
defaultCluster: string       # Active cluster name (default: "default")
useMockClient: boolean       # Enable mock SLURM client (default: false)

# Cluster configurations (list format with name field)
clusters:
  - name: string             # Cluster identifier
    cluster:
      endpoint: string       # SLURM REST API URL
      token: string          # JWT token or ${SLURM_JWT} env var reference
      apiVersion: string     # API version (default: auto-detected, e.g., v0.0.43)
      insecure: boolean      # Skip TLS verification (default: false)
      timeout: duration      # Request timeout (default: 30s)
    namespace: string        # Optional namespace
    readOnly: boolean        # Prevent write operations (default: false)

# UI settings
ui:
  skin: string               # Theme name (default: "default")
  logoless: boolean          # Hide logo (default: false)
  crumbsless: boolean        # Hide breadcrumbs (default: false)
  statusless: boolean        # Hide status bar (default: false)
  headless: boolean          # Hide header (default: false)
  noIcons: boolean           # Disable icons (default: false)
  enableMouse: boolean       # Enable mouse support (default: true)

# View-specific settings
views:
  jobs:
    columns: [string]        # Visible columns (default: [id, name, user, state, time, nodes, priority])
    showOnlyActive: boolean  # Show only active jobs (default: true)
    defaultSort: string      # Default sort column (default: "time")
    maxJobs: integer         # Max jobs to display (default: 1000)
    submission:              # Job submission wizard settings
      formDefaults: map      # Default form values
      hiddenFields: [string] # Fields to hide
      fieldOptions: map      # Restrict dropdown options
      showBuiltinTemplates: boolean  # Legacy: show built-in templates
      templateSources: [string]      # Template sources: builtin, config, saved
      templates:             # Custom job templates
        - name: string
          description: string
          defaults: map
          hiddenFields: [string]

  nodes:
    groupBy: string          # Group by: partition|state|feature|none (default: "partition")
    showUtilization: boolean # Show utilization bars (default: true)
    maxNodes: integer        # Max nodes to display (default: 500)

  partitions:
    showQueueDepth: boolean  # Show queue depth (default: true)
    showWaitTime: boolean    # Show wait time (default: true)

# Feature flags
features:
  streaming: boolean         # Real-time updates via WebSocket (default: true)
  pulseye: boolean           # Health scanner (default: true)
  xray: boolean              # Deep inspection mode (default: false)
  appDiagnostics: boolean    # Application diagnostics (default: false)

# Custom keyboard shortcuts (list of objects)
shortcuts:
  - key: string              # Key combination (e.g., "ctrl+j")
    action: string           # Action to perform (e.g., "view:jobs")
    description: string      # Human-readable description

# Command aliases
aliases:
  string: string             # e.g., ctx: "context", kj: "kill job"

# Plugin configuration
plugins:
  - name: string             # Plugin name
    enabled: boolean         # Enable/disable
    path: string             # Plugin binary path
    config: map              # Plugin-specific settings

# Plugin global settings
pluginSettings:
  enableAll: boolean         # Enable all plugins (default: false)
  pluginDir: string          # Plugin directory (default: "$HOME/.s9s/plugins")
  autoDiscover: boolean      # Auto-discover plugins (default: true)
  safeMode: boolean          # Disable external plugins (default: false)
  maxMemoryMB: integer       # Memory limit per plugin (default: 100)
  maxCPUPercent: float       # CPU limit per plugin (default: 25.0)

# Auto-update settings
update:
  enabled: boolean           # Enable background update checks (default: true)
  autoInstall: boolean       # Auto-download and replace binary on startup (default: false)
  checkInterval: duration    # How often to check for updates (default: "24h")
  preRelease: boolean        # Include pre-release versions (default: false)

# Auto-discovery settings
discovery:
  enabled: boolean           # Enable auto-discovery (default: true)
  enableEndpoint: boolean    # Auto-discover slurmrestd endpoint (default: true)
  enableToken: boolean       # Auto-discover token via scontrol (default: true)
  timeout: duration          # Discovery timeout (default: 10s)
  defaultPort: integer       # Default slurmrestd port (default: 6820)
  scontrolPath: string       # Path to scontrol binary (default: "scontrol")
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
ui:
  skin: default              # Theme skin
  logoless: false            # Hide the logo
  crumbsless: false          # Hide breadcrumbs
  statusless: false          # Hide status bar
  headless: false            # Hide header
  noIcons: false             # Disable icons
  enableMouse: true          # Enable mouse support

refreshRate: 2s              # Auto-refresh interval
```

### Column Configuration

Customize visible columns in the jobs view:

```yaml
views:
  jobs:
    columns:
      - id
      - name
      - user
      - state
      - time
      - nodes
      - priority
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

### Custom Shortcuts

Define custom keyboard shortcuts as a list of objects:

```yaml
shortcuts:
  - key: ctrl+j
    action: "view:jobs"
    description: "Switch to jobs view"

  - key: ctrl+n
    action: "view:nodes"
    description: "Switch to nodes view"

  - key: ctrl+x
    action: "xray:toggle"
    description: "Toggle Xray mode"

  - key: ctrl+h
    action: "pulseye:scan"
    description: "Run health scan"
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

# Or interactively with Ctrl+K while s9s is running
```

When multiple clusters are configured, the active cluster name is shown in the header bar. Press `Ctrl+K` to open the cluster switcher and select a different cluster without restarting.

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

refreshRate: 2s
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
defaultCluster: dev

clusters:
  - name: dev
    cluster:
      endpoint: https://slurm-dev.example.com:6820
      token: ${SLURM_JWT}

refreshRate: 2s

ui:
  skin: default

features:
  streaming: true
  pulseye: true
```

Run with:

```bash
s9s --config dev-config.yaml
```

## Migration Guide

### From Environment Variables

If migrating from environment-only setup:

```bash
# Environment variables still work
export SLURM_REST_URL=https://slurm.example.com:6820
export SLURM_JWT=abc123
s9s

# Or create a config file
cat > ~/.s9s/config.yaml <<EOF
clusters:
  - name: default
    cluster:
      endpoint: https://slurm.example.com:6820
      token: \${SLURM_JWT}
EOF
s9s
```

## Configuration Validation

S9s validates configuration on startup:

1. **Syntax validation**: YAML parsing
2. **Schema validation**: Required fields and types
3. **Cluster resolution**: The `defaultCluster` must match a cluster name in the `clusters` list (unless auto-discovery is enabled)

## Best Practices

1. **Use environment variables** for sensitive data (tokens, endpoints)
2. **Version control** your config (exclude secrets with `.gitignore`)
3. **Backup** before major changes
4. **Separate** dev/prod configurations using multiple cluster entries
5. **Document** custom settings
6. **Set proper file permissions** on configuration files (`chmod 600 ~/.s9s/config.yaml`)
7. **Use auto-discovery** when running on SLURM nodes to avoid hardcoding tokens

## Next Steps

- See [Installation Guide](./installation.md) for setup instructions
- Check [Quick Start](./quickstart.md) to get started
- Explore [User Guide](../user-guide/index.md) for feature documentation
- Learn about [SSH Integration](../guides/ssh-integration.md)
