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

## Basic Structure

```yaml
# ~/.s9s/config.yaml

# General settings
refreshRate: "2s"
maxRetries: 3
defaultCluster: "default"

# Cluster connections
clusters: []

# UI settings
ui: {}

# View settings
views: {}

# Feature flags
features: {}

# Custom keyboard shortcuts
shortcuts: []

# Command aliases
aliases: {}

# Plugin configuration
plugins: []

# Plugin global settings
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

### Jobs View
```yaml
views:
  jobs:
    # Visible columns
    columns: ["id", "name", "user", "state", "time", "nodes", "priority"]

    # Show only active jobs
    showOnlyActive: true

    # Default sort column
    defaultSort: "time"

    # Maximum number of jobs to display
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
    # Group nodes by field
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
# Auto-refresh interval (default: 2s)
refreshRate: "2s"

# Maximum API retries (default: 3)
maxRetries: 3

# Default cluster context
defaultCluster: "default"

# Use mock SLURM client (requires S9S_ENABLE_MOCK env var)
useMockClient: false
```

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

Environment variables override configuration file settings. The `S9S_` prefixed versions take precedence over unprefixed versions.

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
