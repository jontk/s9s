#!/run/current-system/sw/bin/sh

# Run s9s with gotty
# Usage: ./run-with-gotty.sh [gotty options]

# Default gotty options
GOTTY_OPTIONS=${GOTTY_OPTIONS:-"-p 8080 --permit-write --reconnect --title 's9s - SLURM TUI'"}

# Path to s9s binary
S9S_BINARY="./s9s"

# Check if s9s binary exists
if [ ! -f "$S9S_BINARY" ]; then
    echo "Error: s9s binary not found at $S9S_BINARY"
    echo "Please build s9s first with: go build -o s9s cmd/s9s/main.go"
    exit 1
fi

# Check if gotty is installed
if ! command -v gotty &> /dev/null; then
    echo "Error: gotty is not installed"
    echo "Install gotty with one of these methods:"
    echo "  - Download from: https://github.com/yudai/gotty/releases"
    echo "  - macOS: brew install yudai/gotty/gotty"
    echo "  - Or use go: go install github.com/yudai/gotty@latest"
    exit 1
fi

echo "Starting s9s with gotty..."
echo "Access s9s at: http://localhost:8080"
echo "Press Ctrl+C to stop"

# Run gotty with s9s
gotty $GOTTY_OPTIONS $S9S_BINARY