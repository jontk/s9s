# Observability Plugin

The Observability plugin provides real-time metrics monitoring for s9s by integrating with Prometheus. It displays metrics from node-exporter and cgroup-exporter alongside SLURM job and node information.

## Features

- **Real-time Metrics Dashboard**: Comprehensive view showing cluster, node, and job metrics
- **Prometheus Integration**: Connects to your existing Prometheus server
- **Metric Overlays**: Adds CPU and memory usage to existing jobs and nodes views
- **Alert Monitoring**: Configurable alerts for resource usage thresholds
- **Caching**: Intelligent caching to reduce Prometheus query load
- **Visual Widgets**: Gauges, sparklines, and tables for metric visualization

## Installation

The observability plugin is included with s9s. To enable it, add the following to your s9s configuration:

```yaml
plugins:
  - name: observability
    enabled: true
    config:
      prometheus:
        endpoint: "http://your-prometheus:9090"
```

## Configuration

See the included `config.yaml.template` for a complete configuration example. Key settings include:

### Prometheus Connection
```yaml
prometheus:
  endpoint: "http://localhost:9090"  # Your Prometheus server
  timeout: "10s"                      # Query timeout
  auth:
    type: "none"                      # Options: none, basic, bearer
```

### Display Settings
```yaml
display:
  refreshInterval: "30s"              # How often to update metrics
  showOverlays: true                  # Show metrics in jobs/nodes views
  showSparklines: true                # Enable sparkline charts
  colorScheme: "default"              # Options: default, colorblind, monochrome
```

### Alerts
```yaml
alerts:
  enabled: true
  rules:
    - name: "High CPU Usage"
      metric: "cpu_usage"
      operator: ">"
      threshold: 90.0
      duration: "5m"
      severity: "warning"
```

## Required Prometheus Exporters

The plugin expects the following exporters to be running on your cluster nodes:

1. **node-exporter**: For node-level metrics (CPU, memory, disk, network)
   - Standard installation: https://github.com/prometheus/node_exporter
   
2. **cgroup-exporter**: For job-level metrics from SLURM cgroups
   - Provides per-job CPU and memory usage
   - Must be configured to expose SLURM job cgroups

## Usage

### Accessing the Observability View

Press `o` from any view to open the observability dashboard, or navigate through the view menu.

### Keyboard Shortcuts

- `Tab`/`Shift+Tab`: Navigate between panels
- `n`: Focus node metrics table
- `j`: Focus job metrics table
- `a`: Focus alerts panel
- `r` or `Ctrl+R`: Refresh metrics
- `h` or `?`: Show help
- `Esc`: Return to previous view

### Understanding the Display

#### Cluster Overview
Shows aggregate cluster statistics including:
- Total active/down nodes
- CPU core count and load averages
- Running and pending job counts

#### Node Metrics Table
Displays per-node metrics:
- CPU usage percentage with color coding
- Memory usage percentage
- Load average
- Active job count
- Network traffic (receive/transmit)
- Disk I/O rates

#### Job Metrics Table
Shows per-job resource consumption:
- Actual CPU usage vs allocated
- Memory usage vs limit
- Resource efficiency percentage
- Job status

#### Alerts Panel
Active alerts with severity indicators:
- 🔴 Critical alerts
- 🟡 Warning alerts
- 🔵 Informational alerts

## Metric Sources

### Node Metrics (from node-exporter)
- `node_cpu_seconds_total`: CPU usage
- `node_memory_MemTotal_bytes`: Total memory
- `node_memory_MemAvailable_bytes`: Available memory
- `node_load1`, `node_load5`, `node_load15`: Load averages
- `node_disk_read_bytes_total`: Disk read bytes
- `node_disk_write_bytes_total`: Disk write bytes
- `node_network_receive_bytes_total`: Network receive
- `node_network_transmit_bytes_total`: Network transmit

### Job Metrics (from cgroup-exporter)
- `container_cpu_usage_seconds_total`: Job CPU usage
- `container_memory_usage_bytes`: Job memory usage
- `container_spec_memory_limit_bytes`: Job memory limit
- `container_cpu_throttled_seconds_total`: CPU throttling

## Troubleshooting

### Connection Issues
1. Verify Prometheus is accessible:
   ```bash
   curl http://your-prometheus:9090/-/healthy
   ```

2. Check exporter endpoints:
   ```bash
   curl http://node:9100/metrics | grep node_cpu
   ```

3. Enable debug logging:
   ```yaml
   advanced:
     debug: true
   ```

### Missing Metrics
- Ensure node-exporter is running on all nodes
- Verify cgroup-exporter is configured for SLURM
- Check Prometheus scrape configuration
- Verify metric names match your exporter versions

### Performance Issues
- Increase cache TTL for slower refresh
- Reduce sparkline points for less history
- Disable overlays if not needed

## Custom Queries

Add custom PromQL queries in the configuration:

```yaml
metrics:
  customQueries:
    gpu_usage: 'avg(nvidia_gpu_utilization{instance=~"{{.NodePattern}}"})'
    ib_traffic: 'rate(node_infiniband_port_data_received_bytes_total[5m])'
```

## Development

### Adding New Metrics

1. Define the query in `prometheus/queries.go`
2. Add collection logic in the view's refresh method
3. Create or update widgets to display the data

### Creating Custom Widgets

Extend the widget base classes in `views/widgets/`:
- `GaugeWidget`: For percentage/threshold displays
- `SparklineWidget`: For time series data
- `AlertsWidget`: For alert notifications

## Future Enhancements

- GPU metrics support (nvidia-smi exporter)
- InfiniBand metrics
- Lustre filesystem metrics
- Predictive analytics
- Anomaly detection
- Capacity planning

## License

This plugin is part of s9s and is licensed under the same terms.