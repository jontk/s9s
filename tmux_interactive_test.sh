#!/bin/bash

TMUX_SESSION="s9s-interactive-test"
TEST_LOG="tmux_test_results.log"

echo "=== S9S Interactive TUI Testing via Tmux ===" | tee $TEST_LOG
echo "" | tee -a $TEST_LOG

# Kill existing session
tmux kill-session -t $TMUX_SESSION 2>/dev/null

# Create new tmux session
echo "[1/8] Creating tmux session..." | tee -a $TEST_LOG
tmux new-session -d -s $TMUX_SESSION -x 200 -y 50
sleep 1

# SSH to cluster
echo "[2/8] Connecting to cluster..." | tee -a $TEST_LOG
tmux send-keys -t $TMUX_SESSION "ssh -t root@rocky9.ar.jontk.com" Enter
sleep 3

# Check connection
echo "[3/8] Verifying connection..." | tee -a $TEST_LOG
tmux capture-pane -t $TMUX_SESSION -p | tail -5 | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG

# Test 1: Check s9s availability (we'll need to copy it first)
echo "[4/8] Preparing s9s binary on remote..." | tee -a $TEST_LOG
tmux send-keys -t $TMUX_SESSION "ls -lh /usr/local/bin/s9s 2>/dev/null || echo 's9s not in /usr/local/bin'" Enter
sleep 1
tmux capture-pane -t $TMUX_SESSION -p | tail -3 | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG

# Test 2: Check SLURM commands
echo "[5/8] Testing SLURM commands..." | tee -a $TEST_LOG
tmux send-keys -t $TMUX_SESSION "echo '=== SLURM Status ===' && sinfo && echo '' && squeue" Enter
sleep 2
tmux capture-pane -t $TMUX_SESSION -p | tail -15 | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG

# Test 3: Test API endpoint
echo "[6/8] Testing RestD API endpoint..." | tee -a $TEST_LOG
tmux send-keys -t $TMUX_SESSION "curl -s http://localhost:6820/slurm/v0.0.44/ping 2>&1 | head -1" Enter
sleep 1
tmux capture-pane -t $TMUX_SESSION -p | tail -2 | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG

# Test 4: Submit test job for TUI testing
echo "[7/8] Submitting test job for TUI interaction..." | tee -a $TEST_LOG
tmux send-keys -t $TMUX_SESSION "sbatch --partition=debug --time=00:10:00 <<< '#!/bin/bash
echo \"Interactive TUI Test Job\"
sleep 300'" Enter
sleep 2
tmux capture-pane -t $TMUX_SESSION -p | tail -3 | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG

# Test 5: Get job list for verification
echo "[8/8] Current job queue (for TUI testing)..." | tee -a $TEST_LOG
tmux send-keys -t $TMUX_SESSION "squeue -o '%.8i %.9P %.30j %.8u %.2t %.10M %.6D %R' | head -10" Enter
sleep 1
tmux capture-pane -t $TMUX_SESSION -p | tail -12 | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" | tee -a $TEST_LOG
echo "Tmux Session Ready: $TMUX_SESSION" | tee -a $TEST_LOG
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG
echo "To view the session:" | tee -a $TEST_LOG
echo "  tmux attach -t $TMUX_SESSION" | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG
echo "To test s9s TUI manually:" | tee -a $TEST_LOG
echo "  1. Attach to tmux session" | tee -a $TEST_LOG
echo "  2. Copy s9s binary: scp ./s9s root@rocky9.ar.jontk.com:/tmp/" | tee -a $TEST_LOG
echo "  3. Run: /tmp/s9s --no-discovery (or mock mode)" | tee -a $TEST_LOG
echo "  4. Test all keyboard shortcuts" | tee -a $TEST_LOG
echo "" | tee -a $TEST_LOG
echo "Test log saved to: $TEST_LOG" | tee -a $TEST_LOG
