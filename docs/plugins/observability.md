# Observability Plugin

A comprehensive observability plugin for the s9s SLURM management interface that integrates with Prometheus to provide real-time monitoring, historical analysis, and intelligent resource optimization recommendations.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [API Reference](#api-reference)
- [Architecture](#architecture)
- [Metrics](#metrics)
- [Efficiency Scoring](#efficiency-scoring)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

## Features

### Core Monitoring

- **Real-time Metrics**: Live CPU, memory, storage, and network utilization
- **Prometheus Integration**: Native connection to existing Prometheus infrastructure
- **Cached Queries**: Intelligent caching system to reduce Prometheus load
- **Visual Overlays**: Seamless metric overlays on existing s9s views

### Historical Analysis

- **Time Series Collection**: Automated collection and storage of historical metrics
- **30-Day Retention**: Configurable data retention with automatic cleanup
- **Statistical Analysis**: Comprehensive trend analysis with linear regression
- **Anomaly Detection**: Z-score based anomaly detection with configurable sensitivity
- **Seasonal Patterns**: Daily, weekly, and custom seasonal pattern analysis

### Resource Efficiency

- **Comprehensive Scoring**: Multi-factor efficiency scoring (0-100 scale)
- **Resource Analysis**: Individual analysis for CPU, memory, storage, network, and GPU
- **Optimization Recommendations**: AI-driven recommendations with cost impact analysis
- **Cluster-wide Insights**: Aggregate efficiency analysis across the entire cluster
- **ROI Calculations**: Return on investment analysis for optimization suggestions

### Data Subscriptions

- **Real-time Updates**: Subscribe to metric updates with customizable intervals
- **Persistent Subscriptions**: Subscriptions survive plugin restarts
- **Change Detection**: Intelligent notification system for significant metric changes
- **Callback System**: Flexible callback system for custom integrations

### External API

- **HTTP REST API**: Complete RESTful API for external integrations
- **Authentication**: Optional bearer token authentication
- **JSON Responses**: Structured JSON responses for all endpoints
- **Rate Limiting**: Built-in protection against excessive requests

## Installation

1. Place the observability plugin directory in your s9s plugins folder:
   ```bash
   cp -r plugins/observability /path/to/s9s/plugins/
   ```

2. Configure your s9s instance to load the plugin:
   ```yaml
   plugins:
     - name: observability
       enabled: true
       config:
         prometheus.endpoint: "http://your-prometheus:9090"
         prometheus.timeout: "10s"
         display.refreshInterval: "30s"
         display.showOverlays: true
         alerts.enabled: true
   ```

## Configuration

### Basic Configuration

```yaml
observability:
  # Prometheus connection settings
  prometheus:
    endpoint: "http://localhost:9090"
    timeout: "10s"

    # Authentication (optional)
    auth:
      type: "basic"  # or "bearer"
      username: "admin"
      password: "secret"
      # token: "bearer-token"  # for bearer auth

    # TLS settings (optional)
    tls:
      enabled: true
      insecureSkipVerify: false
      caFile: "/path/to/ca.pem"
      certFile: "/path/to/cert.pem"
      keyFile: "/path/to/key.pem"

  # Display configuration
  display:
    refreshInterval: "30s"
    showOverlays: true
    showSparklines: true
    sparklinePoints: 20
    colorScheme: "default"
    decimalPrecision: 2

  # Alert settings
  alerts:
    enabled: true
    checkInterval: "60s"
    loadPredefinedRules: true
    showNotifications: true

  # Caching configuration
  cache:
    enabled: true
    defaultTTL: "1m"
    maxSize: 1000
    cleanupInterval: "5m"

  # API configuration
  api:
    enabled: false
    port: 8080
    auth_token: "your-secret-token"
```

### Advanced Configuration

```yaml
observability:
  # Historical data collection
  historical:
    dataDir: "./data/historical"
    retention: "720h"  # 30 days
    collectInterval: "5m"
    maxDataPoints: 10000

    # Custom queries for data collection
    queries:
      node_cpu: '100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)'
      node_memory: '(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100'
      node_load: 'node_load1'
      job_count: 'slurm_job_total'
      queue_length: 'slurm_queue_pending_jobs'

  # Metric collection settings
  metrics:
    node:
      nodeLabel: "instance"
      rateRange: "5m"
      enabledMetrics: ["cpu", "memory", "disk", "network"]

    job:
      enabled: true
      cgroupPattern: "/slurm/uid_%d/job_%d"
      enabledMetrics: ["cpu", "memory", "io"]
```

## Usage

### Web Interface

1. **Observability View**: Access the main observability dashboard by pressing 'o' in the s9s interface
2. **Metric Overlays**: View real-time metrics overlaid on jobs and nodes views
3. **Historical Charts**: Access time-series charts and trend analysis
4. **Efficiency Dashboard**: Review resource efficiency scores and recommendations

### External API

The plugin exposes a comprehensive REST API when enabled.

#### Authentication

All API requests require a Bearer token when authentication is enabled:

```bash
curl -H "Authorization: Bearer your-token" http://localhost:8080/api/v1/status
```

#### Metrics Endpoints

**Query Metrics**

Instant query:
```bash
curl "http://localhost:8080/api/v1/metrics/query?query=up"
```

Range query:
```bash
curl "http://localhost:8080/api/v1/metrics/query_range?query=node_cpu&start=2023-01-01T00:00:00Z&end=2023-01-01T23:59:59Z&step=15m"
```

**Historical Data**

Get historical data:
```bash
curl "http://localhost:8080/api/v1/historical/data?metric=node_cpu&start=2023-01-01T00:00:00Z&end=2023-01-02T00:00:00Z"
```

Get statistics:
```bash
curl "http://localhost:8080/api/v1/historical/statistics?metric=node_cpu&duration=24h"
```

#### Analysis Endpoints

**Trend Analysis**

```bash
curl "http://localhost:8080/api/v1/analysis/trend?metric=node_cpu&duration=7d"
```

**Anomaly Detection**

```bash
curl "http://localhost:8080/api/v1/analysis/anomaly?metric=node_cpu&duration=24h&sensitivity=2.0"
```

**Seasonal Analysis**

```bash
curl "http://localhost:8080/api/v1/analysis/seasonal?metric=node_cpu&duration=168h"
```

#### Efficiency Analysis

**Resource Efficiency**

```bash
curl "http://localhost:8080/api/v1/efficiency/resource?type=cpu&duration=168h"
curl "http://localhost:8080/api/v1/efficiency/resource?type=memory&duration=168h"
```

**Cluster Efficiency**

```bash
curl "http://localhost:8080/api/v1/efficiency/cluster?duration=168h"
```

#### Subscription Management

**List Subscriptions**

```bash
curl "http://localhost:8080/api/v1/subscriptions"
```

**Create Subscription**

```bash
curl -X POST "http://localhost:8080/api/v1/subscriptions/create" \
  -H "Content-Type: application/json" \
  -d '{"provider_id": "prometheus-metrics", "params": {"query": "up", "update_interval": "30s"}}'
```

**Delete Subscription**

```bash
curl -X DELETE "http://localhost:8080/api/v1/subscriptions/delete?id=subscription-id"
```

## Architecture

### Component Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  s9s Interface  │    │  External Apps  │    │   Prometheus    │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │                      │                      │
    ┌─────▼──────────────────────▼──────────────────────▼─────┐
    │                                                        │
    │               Observability Plugin                     │
    │                                                        │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
    │  │    Views    │  │ External API│  │ Prometheus  │     │
    │  │             │  │             │  │   Client    │     │
    │  └─────────────┘  └─────────────┘  └─────────────┘     │
    │                                                        │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
    │  │  Overlays   │  │Subscription │  │ Historical  │     │
    │  │             │  │  Manager    │  │  Collector  │     │
    │  └─────────────┘  └─────────────┘  └─────────────┘     │
    │                                                        │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
    │  │ Efficiency  │  │   Config    │  │    Cache    │     │
    │  │  Analyzer   │  │   Manager   │  │   Manager   │     │
    │  └─────────────┘  └─────────────┘  └─────────────┘     │
    │                                                        │
    └────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Metric Collection**: Prometheus client queries metrics based on configured intervals
2. **Caching**: Frequently accessed metrics are cached to reduce Prometheus load
3. **Historical Storage**: Time-series data is collected and stored locally for analysis
4. **Analysis Pipeline**: Historical data feeds into trend, anomaly, and efficiency analyzers
5. **Subscription System**: Real-time updates are distributed to subscribers
6. **API Exposure**: External API provides programmatic access to all functionality

### Storage Structure

```
data/
├── observability/           # Subscription persistence
│   ├── subscriptions.json
│   └── notifications.json
└── historical/              # Historical data storage
    ├── node_cpu.json
    ├── node_memory.json
    ├── node_load.json
    └── ...
```

## Metrics

### Default Collected Metrics

- **node_cpu**: CPU utilization percentage per node
- **node_memory**: Memory utilization percentage per node
- **node_load**: System load average per node
- **job_count**: Total number of SLURM jobs
- **queue_length**: Number of pending jobs in queue

### Custom Metrics

Add custom metrics by extending the historical collector configuration:

```yaml
historical:
  queries:
    custom_metric: 'your_prometheus_query_here'
    gpu_usage: 'nvidia_gpu_utilization_percent'
    network_io: 'rate(node_network_receive_bytes_total[5m]) + rate(node_network_transmit_bytes_total[5m])'
```

## Efficiency Scoring

The efficiency analyzer uses a multi-factor scoring system:

### Scoring Components

- **Utilization Score** (50%): Optimal range 70-85%
- **Stability Score** (30%): Lower standard deviation is better
- **Waste Score** (20%): Penalty for unused allocated resources

### Resource-Specific Multipliers

- **CPU**: 1.1x (performance critical)
- **Memory**: 1.05x (stability critical)
- **Storage**: 1.0x (baseline)
- **Network**: 0.95x (less critical for most workloads)

### Efficiency Levels

- **Excellent** (90-100): Optimal resource utilization
- **Good** (75-89): Minor optimization opportunities
- **Fair** (60-74): Moderate inefficiencies detected
- **Poor** (40-59): Significant waste or instability
- **Critical** (0-39): Severe inefficiencies requiring attention

## Troubleshooting

### Common Issues

**Plugin fails to start**
- Verify Prometheus endpoint is accessible
- Check authentication credentials
- Ensure required directories are writable

**No data in historical views**
- Confirm data collection is enabled
- Check historical collector is running
- Verify Prometheus queries return data

**API authentication failures**
- Ensure correct bearer token format
- Check token matches configuration
- Verify API is enabled in configuration

**Performance issues**
- Increase cache TTL to reduce Prometheus load
- Reduce collection frequency for large clusters
- Consider increasing maxDataPoints for longer retention

### Debug Mode

Enable debug logging by setting log level to debug:

```bash
export LOG_LEVEL=debug
```

### Health Checks

Monitor plugin health through the API:

```bash
curl http://localhost:8080/health
```

Or use the plugin's internal health check:
- Plugin status shows "healthy" when Prometheus is accessible
- Cache statistics indicate query performance
- Subscription statistics show active data flows

## Development

### Building

```bash
cd plugins/observability
go build -o observability.so -buildmode=plugin .
```

### Testing

Unit tests:
```bash
go test ./...
```

Integration tests with mock Prometheus:
```bash
go test -v ./integration_test.go
```

Benchmark tests:
```bash
go test -bench=. -benchmem
```

### Contributing

1. Follow Go coding standards
2. Add comprehensive tests for new features
3. Update documentation for configuration changes
4. Ensure backward compatibility

## License

This plugin is licensed under the MIT License. See LICENSE file for details.
