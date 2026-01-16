#!/bin/bash
set -e

# Deploy Prometheus dependencies for s9s observability plugin
# This script sets up Prometheus, node-exporter, and cgroup-exporter for single-node testing

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="/opt/prometheus"
PROMETHEUS_VERSION="2.45.0"
NODE_EXPORTER_VERSION="1.6.1"
CGROUP_EXPORTER_VERSION="0.47.2"
PROMETHEUS_PORT=9090
NODE_EXPORTER_PORT=9100
CGROUP_EXPORTER_PORT=9306

echo "🚀 Setting up Prometheus stack for s9s observability plugin..."

# Create directories
sudo mkdir -p ${BASE_DIR}/{prometheus,node-exporter,cgroup-exporter}
sudo mkdir -p ${BASE_DIR}/prometheus/{config,data}

# Download and install Prometheus
echo "📊 Installing Prometheus ${PROMETHEUS_VERSION}..."
cd /tmp
wget -q https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz
tar xzf prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz
sudo cp prometheus-${PROMETHEUS_VERSION}.linux-amd64/prometheus ${BASE_DIR}/prometheus/
sudo cp prometheus-${PROMETHEUS_VERSION}.linux-amd64/promtool ${BASE_DIR}/prometheus/
sudo cp -r prometheus-${PROMETHEUS_VERSION}.linux-amd64/consoles ${BASE_DIR}/prometheus/
sudo cp -r prometheus-${PROMETHEUS_VERSION}.linux-amd64/console_libraries ${BASE_DIR}/prometheus/
rm -rf prometheus-${PROMETHEUS_VERSION}.linux-amd64*

# Download and install Node Exporter
echo "📈 Installing Node Exporter ${NODE_EXPORTER_VERSION}..."
wget -q https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz
tar xzf node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz
sudo cp node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64/node_exporter ${BASE_DIR}/node-exporter/
rm -rf node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64*

# Download and install cgroup-exporter
echo "🐳 Installing cgroup-exporter ${CGROUP_EXPORTER_VERSION}..."
wget -q https://github.com/google/cadvisor/releases/download/v${CGROUP_EXPORTER_VERSION}/cadvisor-v${CGROUP_EXPORTER_VERSION}-linux-amd64
sudo cp cadvisor-v${CGROUP_EXPORTER_VERSION}-linux-amd64 ${BASE_DIR}/cgroup-exporter/cadvisor
sudo chmod +x ${BASE_DIR}/cgroup-exporter/cadvisor
rm cadvisor-v${CGROUP_EXPORTER_VERSION}-linux-amd64

# Create Prometheus configuration
echo "⚙️  Creating Prometheus configuration..."
sudo tee ${BASE_DIR}/prometheus/config/prometheus.yml > /dev/null << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['localhost:9100']
    scrape_interval: 5s
    metrics_path: /metrics

  - job_name: 'cgroup-exporter'
    static_configs:
      - targets: ['localhost:9306']
    scrape_interval: 5s
    metrics_path: /metrics

rule_files:
  # Add alerting rules here if needed

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          # Add AlertManager targets here if needed
EOF

# Create systemd service for Prometheus
echo "🔧 Creating Prometheus systemd service..."
sudo tee /etc/systemd/system/prometheus.service > /dev/null << EOF
[Unit]
Description=Prometheus Server
Documentation=https://prometheus.io/docs/
After=network-online.target

[Service]
User=root
Restart=on-failure
ExecStart=${BASE_DIR}/prometheus/prometheus \\
  --config.file=${BASE_DIR}/prometheus/config/prometheus.yml \\
  --storage.tsdb.path=${BASE_DIR}/prometheus/data \\
  --web.console.templates=${BASE_DIR}/prometheus/consoles \\
  --web.console.libraries=${BASE_DIR}/prometheus/console_libraries \\
  --web.listen-address=0.0.0.0:${PROMETHEUS_PORT} \\
  --web.external-url=http://localhost:${PROMETHEUS_PORT}/ \\
  --storage.tsdb.retention.time=30d

[Install]
WantedBy=multi-user.target
EOF

# Create systemd service for Node Exporter
echo "📊 Creating Node Exporter systemd service..."
sudo tee /etc/systemd/system/node-exporter.service > /dev/null << EOF
[Unit]
Description=Prometheus Node Exporter
After=network.target

[Service]
User=root
Restart=on-failure
ExecStart=${BASE_DIR}/node-exporter/node_exporter \\
  --web.listen-address=0.0.0.0:${NODE_EXPORTER_PORT} \\
  --collector.systemd \\
  --collector.processes \\
  --collector.interrupts \\
  --collector.tcpstat \\
  --collector.mountstats

[Install]
WantedBy=multi-user.target
EOF

# Create systemd service for cgroup-exporter (cAdvisor)
echo "🐳 Creating cgroup-exporter systemd service..."
sudo tee /etc/systemd/system/cgroup-exporter.service > /dev/null << EOF
[Unit]
Description=cAdvisor (cgroup-exporter)
After=network.target

[Service]
User=root
Restart=on-failure
ExecStart=${BASE_DIR}/cgroup-exporter/cadvisor \\
  --port=${CGROUP_EXPORTER_PORT} \\
  --housekeeping_interval=10s \\
  --max_housekeeping_interval=15s \\
  --event_storage_event_limit=default=0 \\
  --event_storage_age_limit=default=0 \\
  --disable_metrics=accelerator,cpu_topology,disk,memory_numa,tcp,udp,percpu,sched,process,hugetlb,referenced_memory,resctrl,cpuset,advtcp \\
  --store_container_labels=false \\
  --whitelisted_container_labels=io.kubernetes.container.name,io.kubernetes.pod.name,io.kubernetes.pod.namespace \\
  --docker_only=false

[Install]
WantedBy=multi-user.target
EOF

# Set proper permissions
sudo chown -R root:root ${BASE_DIR}
sudo chmod +x ${BASE_DIR}/prometheus/prometheus
sudo chmod +x ${BASE_DIR}/prometheus/promtool
sudo chmod +x ${BASE_DIR}/node-exporter/node_exporter
sudo chmod +x ${BASE_DIR}/cgroup-exporter/cadvisor

# Reload systemd and enable services
echo "🔄 Enabling and starting services..."
sudo systemctl daemon-reload
sudo systemctl enable prometheus node-exporter cgroup-exporter

# Start services
sudo systemctl start prometheus
sudo systemctl start node-exporter
sudo systemctl start cgroup-exporter

# Wait for services to start
echo "⏳ Waiting for services to start..."
sleep 5

# Check service status
echo "✅ Checking service status..."
echo "Prometheus:"
sudo systemctl status prometheus --no-pager -l
echo -e "\nNode Exporter:"
sudo systemctl status node-exporter --no-pager -l
echo -e "\ncgroup-exporter:"
sudo systemctl status cgroup-exporter --no-pager -l

# Test connectivity
echo "🧪 Testing connectivity..."
echo "Testing Prometheus (port ${PROMETHEUS_PORT}):"
curl -s http://localhost:${PROMETHEUS_PORT}/api/v1/status/buildinfo | head -1 || echo "❌ Prometheus not responding"

echo "Testing Node Exporter (port ${NODE_EXPORTER_PORT}):"
curl -s http://localhost:${NODE_EXPORTER_PORT}/metrics | head -1 || echo "❌ Node Exporter not responding"

echo "Testing cgroup-exporter (port ${CGROUP_EXPORTER_PORT}):"
curl -s http://localhost:${CGROUP_EXPORTER_PORT}/metrics | head -1 || echo "❌ cgroup-exporter not responding"

# Show running processes
echo -e "\n🔍 Running processes:"
ps aux | grep -E "(prometheus|node_exporter|cadvisor)" | grep -v grep

# Show listening ports
echo -e "\n🔊 Listening ports:"
ss -tlnp | grep -E "(${PROMETHEUS_PORT}|${NODE_EXPORTER_PORT}|${CGROUP_EXPORTER_PORT})"

echo -e "\n🎉 Prometheus stack deployment complete!"
echo "Access URLs:"
echo "  Prometheus: http://localhost:${PROMETHEUS_PORT}"
echo "  Node Exporter: http://localhost:${NODE_EXPORTER_PORT}/metrics"
echo "  cgroup-exporter: http://localhost:${CGROUP_EXPORTER_PORT}/metrics"
echo ""
echo "To check logs:"
echo "  journalctl -u prometheus -f"
echo "  journalctl -u node-exporter -f"
echo "  journalctl -u cgroup-exporter -f"
echo ""
echo "To stop services:"
echo "  sudo systemctl stop prometheus node-exporter cgroup-exporter"
echo ""
echo "To restart services:"
echo "  sudo systemctl restart prometheus node-exporter cgroup-exporter"