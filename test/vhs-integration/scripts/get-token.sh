#!/usr/bin/env bash
# Generate a JWT token from a Slurm cluster via kubectl
# Usage: get-token.sh <namespace>
# Output: JWT token string on stdout

set -euo pipefail

NAMESPACE="${1:?Usage: get-token.sh <namespace>}"

# Generate token via scontrol on the slurmctld pod
TOKEN=$(kubectl exec -n "$NAMESPACE" slurmctld-0 -c slurmctld -- scontrol token lifespan=3600 2>/dev/null \
    | grep -oP 'SLURM_JWT=\K.*')

if [[ -z "$TOKEN" ]]; then
    echo "ERROR: Failed to generate JWT token for namespace $NAMESPACE" >&2
    exit 1
fi

echo "$TOKEN"
