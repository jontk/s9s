# Configuration Reference

Complete reference for all S9S configuration options, settings, and customization possibilities.

## Configuration Files

S9S uses a hierarchical configuration system with the following precedence (highest to lowest):

1. Command-line flags
2. Environment variables
3. User config: `~/.s9s/config.yaml`
4. System config: `/etc/s9s/config.yaml`
5. Default values

## Basic Structure

```yaml
# ~/.s9s/config.yaml
version: 1

# Cluster connections
clusters: {}

# UI preferences
preferences: {}

# Authentication settings
auth: {}

# Plugin configuration
plugins: {}

# Performance tuning
performance: {}

# Security settings
security: {}

# Integration settings
integrations: {}
```

## Cluster Configuration

### Basic Cluster Setup
```yaml
clusters:
  default:
    # Required: SLURM REST API endpoint
    url: "https://slurm.example.com:6820"

    # Authentication configuration
    auth:
      method: "token"
      token: "${SLURM_TOKEN}"

    # Optional settings
    name: "Production Cluster"
    description: "Main production HPC cluster"
    default: true
    timeout: 30s
    retries: 3
```

### Advanced Cluster Options
```yaml
clusters:
  production:
    url: "https://prod-slurm.example.com:6820"

    # Authentication methods
    auth:
      method: "oauth2"  # token, basic, oauth2, cert
      client_id: "${OAUTH_CLIENT_ID}"
      client_secret: "${OAUTH_CLIENT_SECRET}"
      token_url: "https://auth.example.com/oauth/token"
      scopes: ["slurm.read", "slurm.write"]

    # TLS configuration
    tls:
      verify: true
      ca_file: "/etc/ssl/certs/ca-bundle.crt"
      cert_file: "/etc/s9s/client.crt"
      key_file: "/etc/s9s/client.key"
      insecure_skip_verify: false

    # Connection settings
    connection:
      timeout: 30s
      keep_alive: 60s
      max_idle_conns: 10
      max_conns_per_host: 5

    # HTTP settings
    http:
      headers:
        X-Custom-Header: "value"
        User-Agent: "S9S/1.0"
      proxy: "http://proxy.example.com:8080"

    # Rate limiting
    rate_limit:
      requests_per_second: 10
      burst: 20

    # Caching
    cache:
      enabled: true
      ttl: 60s
      max_size: 100MB
```

## UI Preferences

### Basic UI Settings
```yaml
preferences:
  # Theme selection
  theme: "dark"  # dark, light, terminal, high-contrast, custom
  custom_theme: "/path/to/custom-theme.yaml"

  # Default view
  default_view: "jobs"  # jobs, nodes, dashboard, users, partitions

  # Refresh settings
  refresh_interval: 5s  # 0 to disable auto-refresh
  auto_refresh: true

  # Display preferences
  show_line_numbers: false
  show_relative_time: true
  use_24_hour_time: false
  show_seconds: true
  timezone: "Local"  # Local, UTC, or specific timezone
  date_format: "2006-01-02 15:04:05"

  # Behavior
  confirm_actions: true
  save_window_state: true
  enable_animations: true
  wrap_long_lines: false

  # Pagination
  page_size: 50
  max_results: 1000

  # Performance
  lazy_loading: true
  virtual_scrolling: true
  debounce_delay: 300ms
```

### Advanced UI Configuration
```yaml
preferences:
  # Color customization
  colors:
    accent: "#00ff00"
    warning: "#ffaa00"
    error: "#ff0000"
    info: "#00aaff"

  # Font settings
  fonts:
    family: "JetBrains Mono"
    size: 14
    line_height: 1.2

  # Layout preferences
  layout:
    sidebar_width: 300
    panel_heights:
      main: 0.7
      details: 0.3
    compact_mode: false

  # Notifications
  notifications:
    enabled: true
    position: "top-right"  # top-left, top-right, bottom-left, bottom-right
    duration: 5s
    sound: true
    desktop: true
```

### Column Configuration
```yaml
# Per-view column settings
columns:
  jobs:
    visible:
      - "JobID"
      - "Name"
      - "User"
      - "State"
      - "Time"
      - "Nodes"
      - "Partition"
    widths:
      JobID: 100
      Name: 200
      User: 120
    sort:
      column: "SubmitTime"
      direction: "desc"

  nodes:
    visible:
      - "NodeName"
      - "State"
      - "CPULoad"
      - "Memory"
      - "Jobs"
    auto_width: true

  # Custom columns
  custom_columns:
    efficiency:
      title: "CPU Eff%"
      formula: "(cpu_time / (runtime * cpu_count)) * 100"
      format: "percentage"
      width: 80
```

## Authentication Configuration

### Token Authentication
```yaml
auth:
  method: "token"
  token: "${SLURM_TOKEN}"
  # OR
  token_file: "/path/to/token"
  # OR
  token_command: "vault kv get -field=token secret/slurm"
```

### Basic Authentication
```yaml
auth:
  method: "basic"
  username: "${SLURM_USER}"
  password: "${SLURM_PASS}"
  # OR
  credentials_file: "/path/to/credentials"
```

### OAuth 2.0
```yaml
auth:
  method: "oauth2"
  client_id: "${OAUTH_CLIENT_ID}"
  client_secret: "${OAUTH_CLIENT_SECRET}"
  token_url: "https://auth.example.com/oauth/token"
  scopes: ["slurm.read", "slurm.write"]

  # Optional OAuth settings
  auth_url: "https://auth.example.com/oauth/authorize"
  redirect_url: "http://localhost:8080/callback"
  state: "random-state-string"
  pkce: true
```

### Certificate Authentication
```yaml
auth:
  method: "cert"
  cert_file: "/path/to/client.crt"
  key_file: "/path/to/client.key"
  ca_file: "/path/to/ca.crt"
```

## Keyboard Shortcuts

### Global Shortcuts
```yaml
keybindings:
  global:
    "q": "quit"
    "?": "help"
    "r": "refresh"
    "ctrl+c": "interrupt"
    "esc": "cancel"
    "/": "filter"
    ":": "command_mode"
    "tab": "next_view"
    "shift+tab": "prev_view"

    # View switching
    "1": "view jobs"
    "2": "view nodes"
    "3": "view users"
    "4": "view partitions"
    "0": "view dashboard"
```

### View-Specific Shortcuts
```yaml
keybindings:
  jobs:
    "c": "cancel"
    "h": "hold"
    "r": "release"
    "k": "kill"
    "d": "details"
    "s": "ssh"
    "o": "output"
    "l": "logs"
    "p": "priority"

  nodes:
    "d": "drain"
    "r": "resume"
    "s": "ssh"
    "m": "maintenance"
    "i": "info"
    "j": "jobs"
    "ctrl+r": "reboot"

  # Custom shortcuts
  custom:
    "ctrl+e": "export csv"
    "ctrl+shift+e": "export json"
    "alt+s": "save_filter"
```

## Filtering Configuration

### Default Filters
```yaml
filters:
  # Predefined filters
  presets:
    my_jobs:
      name: "My Jobs"
      filter: "user:${USER}"
      view: "jobs"

    running_gpu:
      name: "Running GPU Jobs"
      filter: "state:RUNNING partition:gpu"
      view: "jobs"

    idle_nodes:
      name: "Idle Nodes"
      filter: "state:IDLE"
      view: "nodes"

  # Filter behavior
  behavior:
    case_sensitive: false
    regex_enabled: true
    auto_complete: true
    save_history: true
    max_history: 50
```

### Search Configuration
```yaml
search:
  # Search providers
  providers:
    - name: "jobs"
      fields: ["JobID", "JobName", "User", "Account"]
      weight: 1.0

    - name: "nodes"
      fields: ["NodeName", "Features", "State"]
      weight: 0.8

  # Search behavior
  fuzzy_matching: true
  max_results: 100
  highlight_matches: true

  # Indexing
  indexing:
    enabled: true
    update_interval: 60s
    max_index_size: 10MB
```

## Export Configuration

### Default Export Settings
```yaml
export:
  # Default format and location
  default_format: "csv"
  output_directory: "~/s9s-exports"

  # File naming
  filename_template: "{view}_{timestamp}.{ext}"
  timestamp_format: "20060102_150405"

  # Data options
  include_headers: true
  max_records: 1000000

  # Format-specific settings
  formats:
    csv:
      delimiter: ","
      quote_char: '"'
      encoding: "utf-8"
      line_ending: "unix"

    json:
      indent: 2
      sort_keys: true
      ensure_ascii: false

    excel:
      sheet_name: "S9S Data"
      auto_width: true
      freeze_panes: true
      include_charts: false
```

### Cloud Export Settings
```yaml
export:
  # Cloud destinations
  destinations:
    s3:
      enabled: true
      bucket: "my-s9s-exports"
      prefix: "exports/"
      region: "us-west-2"
      credentials:
        access_key: "${AWS_ACCESS_KEY}"
        secret_key: "${AWS_SECRET_KEY}"

    gcs:
      enabled: false
      bucket: "my-gcs-bucket"
      prefix: "s9s-exports/"
      credentials_file: "/path/to/service-account.json"

    azure:
      enabled: false
      account: "mystorageaccount"
      container: "s9s-exports"
      sas_token: "${AZURE_SAS_TOKEN}"
```

## Plugin Configuration

### Plugin Management
```yaml
plugins:
  # Global plugin settings
  enabled: true
  directory: "~/.s9s/plugins"
  auto_load: true

  # Plugin repositories
  repositories:
    - url: "https://plugins.s9s.dev"
      type: "official"

    - url: "https://github.com/myorg/s9s-plugins"
      type: "git"
      auth:
        username: "${GITHUB_USER}"
        token: "${GITHUB_TOKEN}"

  # Auto-load plugins
  autoload:
    - "efficiency-analyzer"
    - "cost-tracker"
    - "notification-manager"

  # Plugin-specific configuration
  config:
    efficiency-analyzer:
      threshold: 0.8
      report_frequency: "daily"

    cost-tracker:
      currency: "USD"
      rates:
        cpu_hour: 0.05
        gpu_hour: 2.50

    notification-manager:
      email:
        smtp_server: "smtp.example.com"
        port: 587
        username: "${EMAIL_USER}"
        password: "${EMAIL_PASS}"
```

## Performance Configuration

### Caching Settings
```yaml
performance:
  # Data caching
  cache:
    enabled: true
    size: "500MB"
    ttl:
      jobs: 30s
      nodes: 60s
      users: 300s
      partitions: 600s

    # Cache storage
    storage:
      type: "memory"  # memory, file, redis
      # For file storage
      directory: "~/.s9s/cache"
      # For Redis
      redis_url: "redis://localhost:6379"

  # Connection pooling
  connections:
    max_idle: 10
    max_open: 50
    max_lifetime: 300s

  # Request settings
  requests:
    timeout: 30s
    retries: 3
    backoff: "exponential"
    max_backoff: 60s

  # UI performance
  ui:
    virtual_scrolling: true
    lazy_loading: true
    debounce: 300ms
    batch_size: 100
```

### Logging Configuration
```yaml
logging:
  # Log level
  level: "info"  # debug, info, warn, error

  # Log destinations
  outputs:
    - type: "console"
      format: "text"

    - type: "file"
      path: "~/.s9s/logs/s9s.log"
      format: "json"
      max_size: "100MB"
      max_backups: 5
      max_age: "30d"
      compress: true

    - type: "syslog"
      network: "tcp"
      address: "log.example.com:514"
      tag: "s9s"

  # Component-specific logging
  components:
    api: "debug"
    ui: "info"
    cache: "warn"
    plugins: "info"

  # Request logging
  requests:
    enabled: true
    headers: false
    body: false
```

## Security Configuration

### TLS Settings
```yaml
security:
  tls:
    # Certificate verification
    verify_certificates: true
    ca_file: "/etc/ssl/certs/ca-bundle.crt"

    # Client certificates
    client_cert: "/path/to/client.crt"
    client_key: "/path/to/client.key"

    # TLS versions and ciphers
    min_version: "1.2"
    max_version: "1.3"
    cipher_suites:
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_CHACHA20_POLY1305_SHA256"

  # Encryption
  encryption:
    # Data at rest
    encrypt_cache: true
    encryption_key_file: "/path/to/key"

    # Data in transit
    require_https: true
```

### Access Control
```yaml
security:
  access_control:
    # IP restrictions
    allowed_ips:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"

    # Rate limiting
    rate_limits:
      requests_per_minute: 1000
      requests_per_hour: 10000

    # Session management
    session:
      timeout: "8h"
      max_sessions: 5

  # Audit logging
  audit:
    enabled: true
    log_file: "/var/log/s9s/audit.log"
    events:
      - "authentication"
      - "job_operations"
      - "configuration_changes"
```

## Notification Configuration

### Basic Notifications
```yaml
notifications:
  enabled: true

  # Desktop notifications
  desktop:
    enabled: true
    duration: 5000  # milliseconds

  # Sound notifications
  sound:
    enabled: true
    file: "/path/to/notification.wav"

  # Job completion notifications
  job_completion:
    enabled: true
    only_long_jobs: true
    min_runtime: "1h"
```

### Advanced Notifications
```yaml
notifications:
  providers:
    email:
      enabled: true
      smtp:
        server: "smtp.gmail.com"
        port: 587
        username: "${EMAIL_USER}"
        password: "${EMAIL_PASS}"
        tls: true
      from: "s9s@example.com"

    slack:
      enabled: true
      webhook_url: "${SLACK_WEBHOOK}"
      channel: "#hpc-notifications"
      username: "S9S"
      icon: ":computer:"

    discord:
      enabled: false
      webhook_url: "${DISCORD_WEBHOOK}"

    webhook:
      enabled: false
      url: "https://api.example.com/webhook"
      headers:
        Authorization: "Bearer ${WEBHOOK_TOKEN}"

  # Event-based notifications
  events:
    job_completed:
      providers: ["email", "slack"]
      conditions:
        - "runtime > 1h"
        - "user == '${USER}'"

    job_failed:
      providers: ["email", "slack"]
      urgent: true

    node_down:
      providers: ["email", "slack", "webhook"]
      urgent: true
```

## Environment Variables

### Core Variables
```bash
# Configuration
export S9S_CONFIG="/path/to/config.yaml"
export S9S_CLUSTER="production"
export S9S_DEBUG=true

# Authentication
export SLURM_TOKEN="your-token-here"
export SLURM_USER="username"
export SLURM_PASS="password"

# UI Preferences
export S9S_THEME="dark"
export S9S_REFRESH_INTERVAL="5s"
export S9S_DEFAULT_VIEW="jobs"

# Performance
export S9S_CACHE_SIZE="500MB"
export S9S_MAX_RESULTS=1000
export S9S_REQUEST_TIMEOUT="30s"

# Logging
export S9S_LOG_LEVEL="info"
export S9S_LOG_FILE="/var/log/s9s.log"
```

### Integration Variables
```bash
# Cloud exports
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export AWS_DEFAULT_REGION="us-west-2"

# Notifications
export SLACK_WEBHOOK="https://hooks.slack.com/..."
export EMAIL_USER="user@example.com"
export EMAIL_PASS="password"

# OAuth
export OAUTH_CLIENT_ID="client-id"
export OAUTH_CLIENT_SECRET="client-secret"
```

## Configuration Examples

### Complete Example
```yaml
# ~/.s9s/config.yaml
version: 1

# Cluster configuration
clusters:
  production:
    url: "https://slurm-prod.example.com:6820"
    auth:
      method: "token"
      token: "${SLURM_TOKEN}"
    default: true
    timeout: 30s

  development:
    url: "https://slurm-dev.example.com:6820"
    auth:
      method: "basic"
      username: "${DEV_USER}"
      password: "${DEV_PASS}"

# UI preferences
preferences:
  theme: "dark"
  default_view: "jobs"
  refresh_interval: 5s
  show_relative_time: true
  confirm_actions: true

# Keyboard shortcuts
keybindings:
  global:
    "ctrl+q": "quit"
    "f5": "refresh"
  jobs:
    "space": "toggle_selection"
    "c": "cancel"
    "d": "details"

# Column configuration
columns:
  jobs:
    visible: ["JobID", "Name", "User", "State", "Time", "Nodes"]
    sort:
      column: "SubmitTime"
      direction: "desc"

# Export settings
export:
  default_format: "csv"
  output_directory: "~/exports"
  include_headers: true

# Plugin configuration
plugins:
  enabled: true
  autoload: ["efficiency-analyzer"]
  config:
    efficiency-analyzer:
      threshold: 0.8

# Performance settings
performance:
  cache:
    enabled: true
    size: "200MB"
    ttl:
      jobs: 30s
      nodes: 60s

# Logging
logging:
  level: "info"
  outputs:
    - type: "file"
      path: "~/.s9s/s9s.log"
      max_size: "50MB"

# Notifications
notifications:
  enabled: true
  desktop:
    enabled: true
  providers:
    slack:
      enabled: true
      webhook_url: "${SLACK_WEBHOOK}"
      channel: "#hpc"
```

### Minimal Configuration
```yaml
version: 1

clusters:
  default:
    url: "https://your-slurm-api.com"
    auth:
      method: "token"
      token: "${SLURM_TOKEN}"

preferences:
  theme: "dark"
  refresh_interval: 10s
```

## Configuration Validation

### Validate Configuration
```bash
# Check configuration syntax and values
s9s config validate

# Test cluster connections
s9s config test-connections

# Show effective configuration
s9s config show

# Check for deprecated settings
s9s config check-deprecated
```

### Configuration Migration
```bash
# Upgrade configuration format
s9s config migrate --from=0.9 --to=1.0

# Export configuration
s9s config export > backup.yaml

# Import configuration
s9s config import backup.yaml
```

## Next Steps

- Review [command reference](./commands.md)
- Explore [API integration](./api.md)
- Start with a minimal configuration in [getting started guide](../guides/quickstart.md)
- Configure [plugins](../guides/plugins.md)
