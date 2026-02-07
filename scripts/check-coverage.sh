#!/bin/bash
# Check per-package coverage thresholds
# Usage: ./scripts/check-coverage.sh

set -e

# Package-specific minimum coverage thresholds (percentage)
# These are initial baseline thresholds - gradually increase as tests are added
declare -A THRESHOLDS=(
    ["./internal/config"]=20
    ["./internal/dao"]=5
    ["./internal/monitoring"]=45  # Current: 47.6%, gradually increase to 80%
    ["./internal/app"]=30
)

echo "Checking package coverage thresholds..."
echo

# Track failures
FAILED=0
FAILURES=""

# Check each package
for package in "${!THRESHOLDS[@]}"; do
    threshold=${THRESHOLDS[$package]}

    # Run go test with coverage for this specific package
    output=$(go test -cover "$package" 2>&1 || true)

    # Extract coverage percentage
    coverage=$(echo "$output" | grep -oP 'coverage: \K[0-9.]+' | head -1 || echo "0.0")

    if [ -z "$coverage" ]; then
        coverage="0.0"
    fi

    # Compare coverage to threshold
    result=$(awk -v cov="$coverage" -v thresh="$threshold" 'BEGIN { if (cov >= thresh) print "PASS"; else print "FAIL" }')

    if [ "$result" = "PASS" ]; then
        echo "✓ ${package}: ${coverage}% (threshold: ${threshold}%)"
    else
        echo "✗ ${package}: ${coverage}% (threshold: ${threshold}%) - FAILED"
        FAILED=1
        FAILURES="${FAILURES}\n  - ${package}: ${coverage}% < ${threshold}%"
    fi
done

echo
if [ $FAILED -eq 0 ]; then
    echo "✓ All package coverage thresholds met!"
    exit 0
else
    echo "✗ Coverage threshold failures:"
    echo -e "$FAILURES"
    echo
    echo "Please add tests to meet minimum coverage requirements."
    echo "See docs/development/testing.md for testing guidelines."
    exit 1
fi
