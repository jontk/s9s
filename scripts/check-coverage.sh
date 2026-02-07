#!/bin/bash
# Check per-package coverage thresholds
# Usage: ./scripts/check-coverage.sh

set -e

# Package-specific minimum coverage thresholds (percentage)
# These thresholds are set based on current coverage to prevent regression
# and gradually increased as more tests are added
#
# Aspirational goals (from docs/development/testing.md):
# - internal/monitoring: 80%
# - internal/dao: 80%
# - internal/app: 80%
# - internal/config: 80%
#
# Strengthened coverage thresholds (raised from previous 5%/20%/30%/45%)
#
# Previous thresholds were too low (5-45%) for safety-critical code.
# New thresholds enforce current coverage levels to prevent regression
# and provide foundation for gradual improvement toward 80% target.
#
# Changes from previous thresholds:
#   - internal/config: 20% -> 20% (no change, already at current coverage floor)
#   - internal/dao: 5% -> 6% (+1%, raised from dangerously low 5%)
#   - internal/monitoring: 45% -> 47% (+2%, raised to current coverage floor)
#   - internal/app: 30% -> 33% (+3%, raised to current coverage floor)
#   - Global minimum: 5% -> 6% (stronger baseline for all packages)
#
# Current actual coverage (2026-02-07):
#   internal/config: 20.2%, internal/dao: 6.4%
#   internal/monitoring: 47.6%, internal/app: 33.5%
#
# These thresholds now prevent ANY regression from current coverage levels.
# Previous thresholds were dangerously low and allowed significant quality degradation.
#
# Roadmap to 80% target (docs/development/testing.md:109):
#   Phase 1 (DONE): Raise thresholds to prevent regression
#   Phase 2 (NEXT): Add tests for critical paths (s9s-4zb, s9s-9xk, s9s-atd)
#   Phase 3 (FUTURE): Incrementally raise to 50% -> 60% -> 70% -> 80%
#
declare -A THRESHOLDS=(
    ["./internal/config"]=20
    ["./internal/dao"]=6
    ["./internal/monitoring"]=47
    ["./internal/app"]=33
)

# Global minimum threshold - raised from 5% to 6%
# Applies to all packages not explicitly listed above
GLOBAL_MIN=6

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

    # Use the higher of package-specific threshold or global minimum
    effective_threshold=$threshold
    if (( GLOBAL_MIN > threshold )); then
        effective_threshold=$GLOBAL_MIN
    fi

    # Compare coverage to effective threshold
    result=$(awk -v cov="$coverage" -v thresh="$effective_threshold" 'BEGIN { if (cov >= thresh) print "PASS"; else print "FAIL" }')

    if [ "$result" = "PASS" ]; then
        echo "✓ ${package}: ${coverage}% (threshold: ${effective_threshold}%)"
    else
        echo "✗ ${package}: ${coverage}% (threshold: ${effective_threshold}%) - FAILED"
        FAILED=1
        FAILURES="${FAILURES}\n  - ${package}: ${coverage}% < ${effective_threshold}%"
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
