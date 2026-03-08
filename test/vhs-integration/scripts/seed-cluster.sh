#!/usr/bin/env bash
# Seed a Slurm cluster with test data for VHS integration tests
# Usage: seed-cluster.sh <namespace>
#
# Creates: accounts, users, QOS policies, a reservation, and submits sample jobs
# in various states across partitions. Idempotent - safe to run multiple times.

set -euo pipefail

NAMESPACE="${1:?Usage: seed-cluster.sh <namespace>}"

run() {
    kubectl exec -n "$NAMESPACE" slurmctld-0 -c slurmctld -- bash -c "$1" 2>&1
}

echo "[seed] Seeding cluster in namespace: $NAMESPACE"

# --- Accounts ---
echo "[seed] Creating accounts..."
run "sacctmgr -i add account research Description='Research Group' Organization='University' 2>/dev/null || true"
run "sacctmgr -i add account engineering Description='Engineering Team' Organization='Company' 2>/dev/null || true"
run "sacctmgr -i add account training Description='ML Training' Organization='University' parent=research 2>/dev/null || true"

# --- Users ---
echo "[seed] Creating users..."
run "sacctmgr -i add user alice Account=research 2>/dev/null || true"
run "sacctmgr -i add user bob Account=engineering 2>/dev/null || true"
run "sacctmgr -i add user carol Account=training 2>/dev/null || true"
run "sacctmgr -i add user dave Account=research 2>/dev/null || true"

# --- QOS ---
echo "[seed] Creating QOS policies..."
run "sacctmgr -i add qos high Priority=100 MaxWall=7-00:00:00 2>/dev/null || true"
run "sacctmgr -i add qos low Priority=10 MaxWall=1-00:00:00 2>/dev/null || true"
run "sacctmgr -i add qos gpu MaxTRES=gres/gpu=4 Priority=50 2>/dev/null || true"

# --- Submit jobs as root ---
# All jobs submitted as root (the only user with guaranteed scheduler access).
# This populates all views with real data across partitions and states.
echo "[seed] Cleaning up old test jobs..."
run "scancel --name='test-simulation' --name='test-analysis' --name='test-training' --name='test-debug-run' --name='test-gpu-train' --name='test-bigmem' --name='test-pending-large' --name='test-pending-gpu' --name='test-array' 2>/dev/null || true"

echo "[seed] Submitting test jobs..."

# Long-running jobs across partitions (will be RUNNING)
run "sbatch --job-name=test-simulation --partition=compute --time=06:00:00 --nodes=1 --wrap='sleep 21600'"
run "sbatch --job-name=test-analysis --partition=compute --time=04:00:00 --nodes=1 --wrap='sleep 14400'"
run "sbatch --job-name=test-training --partition=compute --time=08:00:00 --nodes=2 --wrap='sleep 28800'"
run "sbatch --job-name=test-debug-run --partition=debug --time=00:30:00 --nodes=1 --wrap='sleep 1800'"
run "sbatch --job-name=test-gpu-train --partition=gpu --time=12:00:00 --nodes=1 --wrap='sleep 43200'"
run "sbatch --job-name=test-bigmem --partition=highmem --time=02:00:00 --nodes=1 --wrap='sleep 7200'"

# Jobs that will be PENDING (request more nodes than available after above jobs allocate)
run "sbatch --job-name=test-pending-large --partition=compute --time=01:00:00 --nodes=4 --exclusive --wrap='sleep 3600'"
run "sbatch --job-name=test-pending-gpu --partition=gpu --time=06:00:00 --nodes=2 --exclusive --wrap='sleep 21600'"

# Job array
run "sbatch --job-name=test-array --partition=compute --time=01:00:00 --array=1-3 --wrap='sleep 3600'"

# --- Reservation (optional, may fail on minimal clusters) ---
echo "[seed] Creating reservation..."
run "scontrol create reservation ReservationName=maintenance StartTime=now+7days Duration=04:00:00 Users=root Nodes=slurmd-0 Flags=MAINT 2>/dev/null || true"

# --- Verify ---
echo "[seed] Verification:"
echo "[seed] Jobs:"
run "squeue"
echo "[seed] Nodes:"
run "sinfo"
echo "[seed] Accounts:"
run "sacctmgr -n show account"
echo "[seed] Users:"
run "sacctmgr -n show user"
echo "[seed] QOS:"
run "sacctmgr -n show qos format=Name,Priority,MaxWall,MaxTRES"
echo "[seed] Reservations:"
run "scontrol show reservation 2>/dev/null || echo '  (none)'"

echo "[seed] Done seeding $NAMESPACE"
