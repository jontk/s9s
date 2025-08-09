#!/bin/bash

# Run s9s with ttyd (formerly gotty)
# Usage: ./run-with-gotty.sh [ttyd options]
# 
# Note: This script now uses ttyd instead of gotty for better compatibility

# Default ttyd options (compatible with gotty usage)
TTYD_PORT=${TTYD_PORT:-8080}
TTYD_OPTIONS=${TTYD_OPTIONS:-"-p $TTYD_PORT -W"}

# Path to s9s binary
S9S_BINARY="./s9s"

# Check if s9s binary exists
if [ ! -f "$S9S_BINARY" ]; then
    echo "Error: s9s binary not found at $S9S_BINARY"
    echo "Please build s9s first with: go build -o s9s cmd/s9s/main.go"
    exit 1
fi

# Check if ttyd is installed
if ! command -v ttyd &> /dev/null; then
    echo "Error: ttyd is not installed (replacing gotty)"
    echo "Install ttyd with one of these methods:"
    echo "  - macOS: brew install ttyd"
    echo "  - Linux: apt-get install ttyd (or equivalent)"
    echo "  - Download from: https://github.com/tsl0922/ttyd/releases"
    echo ""
    echo "Note: ttyd is a more modern alternative to gotty with better compatibility"
    exit 1
fi

echo "Starting s9s with ttyd (gotty replacement)..."
echo "Access s9s at: http://localhost:$TTYD_PORT"
echo "Press Ctrl+C to stop"

# Run ttyd with s9s
ttyd $TTYD_OPTIONS $S9S_BINARY
