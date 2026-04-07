# Configuration Reference

Complete reference for all S9S configuration options, settings, and customization.

## Configuration Files

S9S uses a hierarchical configuration system with the following precedence (highest to lowest):

1. Command-line flags
2. Environment variables
3. Current directory: `./config.yaml`
4. User config: `~/.s9s/config.yaml`
5. System config: `/etc/s9s/config.yaml`
6. Default values

## Configuration Modal (F10)

The in-app Configuration modal is opened with `F10` or the `:config` command. It contains two setting groups:

### General
- **Refresh Rate** (`refreshRate`) -- global auto-refresh interval applied to every view (default: `10s`). Edits to this field are re-applied live: click Save and the running ticker immediately re-arms at the new cadence. Set to an empty string to disable auto-refresh entirely; `F5` and `R` still work for manual refresh, and `F6` toggles the pause at runtime without touching the config file.
- **Default Cluster** (`defaultCluster`) -- which cluster to connect to on startup

### View Settings
The following settings take effect immediately when changed:
- **Max Jobs** (`maxJobs`) -- maximum number of jobs to display
- **Show Only Active** (`showOnlyActive`) -- show only active jobs
- **Group Nodes By** (`groupBy`) -- group nodes by partition, state, features, or none

The following settings are saved to the config file but only take effect on the next startup:
- **Job Columns** (`columns`) -- visible columns in the jobs view
- **Default Sort** (`defaultSort`) -- default sort column for jobs

All other configuration options (UI settings, feature flags, keyboard shortcuts, command aliases, plugins, and cluster contexts) are only configurable by editing the config file directly.

## Basic Structure

```yaml
# ~/.s9s/config.yaml

# General settings
refreshRate: "10s"
maxRetries: 3
defaultCluster: "default"

# Cluster connections
clusters: []

# UI settings (config file only, not editable from UI)
ui: {}

# View settings
views: {}

# Feature flags (config file only, not editable from UI)
features: {}

# Custom keyboard shortcuts (config file only, not editable from UI)
shortcuts: []

# Command aliases (config file only, not editable from UI)
aliases: {}

# Plugin configuration (config file only, not editable from UI)
plugins: []

# Plugin global settings (config file only, not editable from UI)
pluginSettings: {}

# Auto-discovery settings
discovery: {}
```

## Cluster Configuration

### Basic Cluster Setup
```yaml
clusters:
  - name: "default"
    cluster:
      # Required: SLURM REST API endpoint
      endpoint: "https://slurm.example.com:6820"

      # JWT authentication token
      token: "${SLURM_JWT}"

      # API version (auto-detected if omitted)
      apiVersion: "v0.0.43"

      # Skip TLS certificate verification
      insecure: false

      # Request timeout
      timeout: "30s"

    # Optional: namespace for multi-tenant setups
    namespace: ""

    # Optional: prevent write operations
    readOnly: false
```

### Multiple Clusters
```yaml
clusters:
  - name: "production"
    cluster:
      endpoint: "https://slurm-prod.example.com:6820"
      token: "${SLURM_JWT}"
      timeout: "30s"

  - name: "development"
    cluster:
      endpoint: "https://slurm-dev.example.com:6820"
      token: "${DEV_SLURM_JWT}"
      insecure: true

defaultCluster: "production"
```

## UI Configuration

> **Note:** UI settings are only configurable via the config file. They are not available in the Configuration modal (F10).

```yaml
ui:
  # Theme/skin selection
  skin: "default"

  # Hide logo
  logoless: false

  # Hide breadcrumbs
  crumbsless: false

  # Hide status bar
  statusless: false

  # Hide header
  headless: false

  # Disable icons
  noIcons: false

  # Enable mouse support
  enableMouse: true
```

## View Configuration

The Configuration modal (F10) provides a **View Settings** group where some of these settings can be changed at runtime. The following settings take effect immediately when changed in the modal:

- **Max Jobs** (`maxJobs`) -- limits the number of jobs displayed
- **Show Only Active** (`showOnlyActive`) -- filters to active jobs only
- **Group Nodes By** (`groupBy`) -- changes node grouping in the nodes view

The following settings can be edited in the modal and are saved to the config file, but are **not applied at runtime** (they take effect on the next startup):

- **Job Columns** (`columns`) -- visible columns in the jobs view
- **Default Sort** (`defaultSort`) -- default sort column for jobs

All other view settings below are configurable only via the config file.

### Jobs View
```yaml
views:
  jobs:
    # Visible columns (saved via UI but applied on next startup)
    columns: ["id", "name", "user", "state", "time", "nodes", "priority"]

    # Show only active jobs (editable and applied at runtime via F10)
    showOnlyActive: true

    # Default sort column (saved via UI but applied on next startup)
    defaultSort: "time"

    # Maximum number of jobs to display (editable and applied at runtime via F10)
    maxJobs: 1000

    # Job submission configuration
    submission:
      # Default values for job submission form
      formDefaults:
        partition: "compute"
        nodes: 1
      # Fields to hide from submission form
      hiddenFields: ["comment"]
      # Custom dropdown options for fields
      fieldOptions:
        partition: ["compute", "gpu", "highmem"]
      # Show built-in templates (default: true)
      showBuiltinTemplates: true
      # Template sources to load: builtin, config, saved
      templateSources: ["builtin", "config", "saved"]
      # Custom job templates
      templates:
        - name: "GPU Training Job"
          description: "PyTorch training on GPU partition"
          defaults:
            partition: "gpu"
            gres: "gpu:1"
          hiddenFields: []
```

### Nodes View
```yaml
views:
  nodes:
    # Group nodes by field (editable and applied at runtime via F10)
    groupBy: "partition"

    # Show utilization metrics
    showUtilization: true

    # Maximum number of nodes to display
    maxNodes: 500
```

### Partitions View
```yaml
views:
  partitions:
    # Show queue depth information
    showQueueDepth: true

    # Show estimated wait times
    showWaitTime: true
```

## Feature Flags

> **Note:** Feature flags are only configurable via the config file. They are not available in the Configuration modal (F10).

```yaml
features:
  # Enable job streaming
  streaming: true

  # Enable Pulseye integration
  pulseye: true

  # Enable X-ray diagnostics
  xray: false

  # Enable app diagnostics
  appDiagnostics: false
```

## Keyboard Shortcuts

> **Note:** Custom keyboard shortcuts are only configurable via the config file. They are not available in the Configuration modal (F10).

Custom keyboard shortcuts use the `shortcuts` array with `key`, `action`, and `description` fields:

```yaml
shortcuts:
  - key: "ctrl+e"
    action: "export csv"
    description: "Export current view as CSV"

  - key: "ctrl+s"
    action: "submit job"
    description: "Open job submission form"
```

## Command Aliases

> **Note:** Command aliases are only configurable via the config file. They are not available in the Configuration modal (F10).

```yaml
aliases:
  ctx: "context"
  kj: "kill job"
  dj: "describe job"
  dn: "describe node"
  sub: "submit job"
```

## General Settings

```yaml
# Auto-refresh interval applied to every view (default: 10s).
# Set to "" to disable auto-refresh; F5/R still work for manual refresh,
# and F6 toggles the pause at runtime without touching the config file.
refreshRate: "10s"

# Maximum API retries (default: 3)
maxRetries: 3

# Default cluster context
defaultCluster: "default"

# Use mock SLURM client (requires S9S_ENABLE_MOCK env var)
useMockClient: false
```

## Auto-Update Configuration

```yaml
update:
  # Enable background update checks on TUI startup (default: true)
  # Set to false for air-gapped clusters or managed deployments
  enabled: true

  # Automatically download and install updates (default: false)
  # When true, s9s replaces its own binary on startup and shows
  # "Updated to X — restart s9s to use the new version"
  # When false, only a notification is shown
  autoInstall: false

  # How often to check for new versions (default: "24h")
  checkInterval: "24h"

  # Include pre-release versions in update checks (default: false)
  preRelease: false
```

The background check runs in a goroutine with a 3-second timeout and caches results to `~/.s9s/update-state.json`. Behavior depends on `autoInstall`:

- **`autoInstall: false`** (default) — shows a status bar notification: `Update available: 0.8.0 -> 0.9.0 (run 's9s update')`
- **`autoInstall: true`** — downloads and replaces the binary, then shows: `Updated to 0.9.0 — restart s9s to use the new version`. If the auto-install fails (e.g., permissions), falls back to the notification.

To update manually, use `s9s update`. See the [Installation Guide](../getting-started/installation.md#upgrading) for details.

## Auto-Discovery Configuration

```yaml
discovery:
  # Enable auto-discovery of slurmrestd endpoint and token
  enabled: true

  # Auto-discover endpoint
  enableEndpoint: true

  # Auto-discover/generate JWT token
  enableToken: true

  # Discovery timeout
  timeout: "10s"

  # Default slurmrestd port
  defaultPort: 6820

  # Path to scontrol binary
  scontrolPath: "scontrol"
```

## Plugin Configuration

> **Note:** Plugin settings are only configurable via the config file. They are not available in the Configuration modal (F10).

```yaml
# Individual plugin entries
plugins:
  - name: "my-plugin"
    enabled: true
    path: "/path/to/plugin"
    config:
      customSetting: "value"

# Global plugin settings
pluginSettings:
  enableAll: false
  pluginDir: "$HOME/.s9s/plugins"
  autoDiscover: true
  safeMode: false          # Disable external plugins
  maxMemoryMB: 100         # Memory limit per plugin
  maxCPUPercent: 25.0      # CPU limit per plugin
```

## Environment Variables

S9S recognizes the following environment variables:

```bash
# SLURM REST API endpoint (S9S-prefixed takes precedence)
export SLURM_REST_URL="https://slurm.example.com:6820"
export S9S_SLURM_REST_URL="https://slurm.example.com:6820"

# SLURM JWT authentication token (S9S-prefixed takes precedence)
export SLURM_JWT="your-token-here"
export S9S_SLURM_JWT="your-token-here"

# SLURM API version (default: v0.0.43)
export SLURM_API_VERSION="v0.0.43"

# SLURM request timeout
export SLURM_TIMEOUT="30s"

# Enable mock mode (any non-empty value)
export S9S_ENABLE_MOCK=1
```

Environment variables override configuration file settings. Only `SLURM_REST_URL` and `SLURM_JWT` have `S9S_` prefixed variants (`S9S_SLURM_REST_URL` and `S9S_SLURM_JWT`), and the prefixed versions take precedence over the unprefixed versions. The `SLURM_API_VERSION` and `SLURM_TIMEOUT` environment variables do not have `S9S_` prefixed variants.

## Configuration Examples

### Minimal Configuration
```yaml
clusters:
  - name: "default"
    cluster:
      endpoint: "https://slurm.example.com:6820"
      token: "${SLURM_JWT}"
```

### Complete Example
```yaml
refreshRate: "5s"
maxRetries: 3
defaultCluster: "production"

clusters:
  - name: "production"
    cluster:
      endpoint: "https://slurm-prod.example.com:6820"
      token: "${SLURM_JWT}"
      apiVersion: "v0.0.43"
      timeout: "30s"

  - name: "development"
    cluster:
      endpoint: "https://slurm-dev.example.com:6820"
      token: "${DEV_SLURM_JWT}"
      insecure: true

ui:
  skin: "default"
  enableMouse: true
  noIcons: false

views:
  jobs:
    columns: ["id", "name", "user", "state", "time", "nodes", "priority"]
    showOnlyActive: true
    defaultSort: "time"
    maxJobs: 1000
  nodes:
    groupBy: "partition"
    showUtilization: true
    maxNodes: 500
  partitions:
    showQueueDepth: true
    showWaitTime: true

features:
  streaming: true
  pulseye: true
  xray: false

shortcuts:
  - key: "ctrl+e"
    action: "export csv"
    description: "Export as CSV"

aliases:
  ctx: "context"
  kj: "kill job"
  dj: "describe job"
  dn: "describe node"
  sub: "submit job"

plugins: []

discovery:
  enabled: true
  enableEndpoint: true
  enableToken: true
  timeout: "10s"
  defaultPort: 6820
  scontrolPath: "scontrol"
```

## Configuration Validation

```bash
# Validate configuration syntax and values
s9s config validate

# Show effective configuration
s9s config show

# Edit configuration file in your default editor
s9s config edit
```

## Next Steps

- Review [command reference](./commands.md)
- Explore [API integration](./api.md)
- Start with a minimal configuration in [getting started guide](../getting-started/quickstart.md)
- Configure [plugins](../plugins/overview.md)
