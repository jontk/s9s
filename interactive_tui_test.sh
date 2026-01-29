#!/bin/bash

echo "=== S9S Interactive TUI Testing Setup ==="
echo ""

# Step 1: Get token from cluster
echo "[1/5] Getting fresh JWT token from cluster..."
TOKEN=$(ssh root@rocky9.ar.jontk.com 'scontrol token 2>/dev/null' | grep -oP '(?<=SLURM_JWT=).*')

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token"
    exit 1
fi

echo "✓ Token obtained"
echo ""

# Step 2: Create s9s configuration
echo "[2/5] Creating s9s configuration..."
mkdir -p ~/.s9s

cat > ~/.s9s/config.yaml << CFGEOF
clusters:
  rocky9:
    url: http://localhost:6820
    token: "${TOKEN}"
    api_version: v0.0.44
    default: true

preferences:
  theme: dark
  refreshInterval: 5s
  defaultView: jobs
CFGEOF

echo "✓ Configuration created at ~/.s9s/config.yaml"
echo ""

# Step 3: Validate configuration
echo "[3/5] Validating configuration..."
./s9s config validate && echo "✓ Configuration valid" || echo "⚠️  Configuration may need adjustment"
echo ""

# Step 4: Show cluster details
echo "[4/5] Cluster Details:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
ssh root@rocky9.ar.jontk.com << 'SSHEOF'
echo "Host: $(hostname)"
echo "SLURM: $(sinfo --version | head -1)"
echo "Partitions:"
sinfo -h -o "  %P (%l time limit)"
echo "Active Jobs: $(squeue --all -h | wc -l)"
echo "Nodes: $(sinfo -h -N | wc -l)"
SSHEOF
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Step 5: Instructions
echo "[5/5] Ready for Interactive Testing!"
echo ""
echo "To start s9s TUI on the cluster:"
echo "  ssh -t root@rocky9.ar.jontk.com"
echo "  Then run one of:"
echo "    1. Mock mode:  export S9S_ENABLE_MOCK=development && /path/to/s9s --mock"
echo "    2. Real cluster: /path/to/s9s --no-discovery"
echo ""
echo "TUI Keyboard Shortcuts to Test:"
echo "  Tab      - Switch views"
echo "  j        - Jobs view"
echo "  n        - Nodes view"
echo "  p        - Partitions view"
echo "  /        - Search"
echo "  ?        - Help menu"
echo "  q        - Quit"
echo ""
echo "Job Operations to Test:"
echo "  c        - Cancel job"
echo "  h        - Hold job"
echo "  r        - Release job"
echo "  d        - View details"
echo ""
echo "Configuration file saved at: ~/.s9s/config.yaml"
echo ""
