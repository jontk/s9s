#!/usr/bin/env bash
# VHS Integration Test Runner
# Runs VHS tapes against real Slurm clusters and compares output to golden files.
#
# Usage:
#   ./test/vhs-integration/scripts/run-integration.sh                    # Test all versions
#   ./test/vhs-integration/scripts/run-integration.sh --version slurm-2405
#   ./test/vhs-integration/scripts/run-integration.sh --update-golden
#   ./test/vhs-integration/scripts/run-integration.sh --update-golden --version slurm-2411

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ROOT="$(cd "$BASE_DIR/../.." && pwd)"

# Cluster definitions: namespace -> highest supported API version
declare -A CLUSTER_API_VERSION=(
    [slurm-2405]="v0.0.41"
    [slurm-2411]="v0.0.42"
    [slurm-2505]="v0.0.43"
    [slurm-2511]="v0.0.44"
)

ALL_VERSIONS=("slurm-2405" "slurm-2411" "slurm-2505" "slurm-2511")
TAPES_DIR="$BASE_DIR/tapes"
GOLDEN_DIR="$BASE_DIR/golden"
OUTPUT_DIR="$BASE_DIR/output"
NORMALIZE="$SCRIPT_DIR/normalize-output.sh"
GET_TOKEN="$SCRIPT_DIR/get-token.sh"
SEED_CLUSTER="$SCRIPT_DIR/seed-cluster.sh"

# Parse arguments
UPDATE_GOLDEN=false
SKIP_SEED=false
VERSIONS=()
while [[ $# -gt 0 ]]; do
    case "$1" in
        --update-golden)
            UPDATE_GOLDEN=true
            shift
            ;;
        --version)
            VERSIONS+=("$2")
            shift 2
            ;;
        --no-seed)
            SKIP_SEED=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--update-golden] [--version <slurm-XXXX>] [--no-seed]"
            echo ""
            echo "Options:"
            echo "  --update-golden    Update golden files instead of comparing"
            echo "  --version <name>   Test only this version (can be repeated)"
            echo "  --no-seed          Skip seeding clusters with test data"
            echo ""
            echo "Versions: ${ALL_VERSIONS[*]}"
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Default to all versions if none specified
if [[ ${#VERSIONS[@]} -eq 0 ]]; then
    VERSIONS=("${ALL_VERSIONS[@]}")
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info()  { echo -e "${BLUE}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[PASS]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail()  { echo -e "${RED}[FAIL]${NC} $*"; }

# Ensure required tools are available
for tool in vhs kubectl curl python3; do
    if ! command -v "$tool" &>/dev/null; then
        fail "Required tool not found: $tool"
        exit 1
    fi
done

# Build s9s if needed
BINARY="$PROJECT_ROOT/build/s9s"
if [[ ! -f "$BINARY" ]]; then
    info "Building s9s..."
    make -C "$PROJECT_ROOT" build
fi

# Copy binary to project root for VHS (tapes use ./s9s)
cp "$BINARY" "$PROJECT_ROOT/s9s"
trap 'rm -f "$PROJECT_ROOT/s9s"' EXIT

# Collect test tapes
TAPES=()
for tape in "$TAPES_DIR"/*.tape; do
    TAPES+=("$(basename "$tape" .tape)")
done

if [[ ${#TAPES[@]} -eq 0 ]]; then
    fail "No test tapes found in $TAPES_DIR"
    exit 1
fi

info "Found ${#TAPES[@]} test tapes: ${TAPES[*]}"

# Track results
TOTAL=0
PASSED=0
FAILED=0
SKIPPED=0
FAILURES=()

# Run tests for each version
for VERSION in "${VERSIONS[@]}"; do
    echo ""
    info "=== Testing against $VERSION ==="

    # Get ClusterIP for slurmrestd service
    CLUSTER_IP=$(kubectl get svc slurmrestd -n "$VERSION" -o jsonpath='{.spec.clusterIP}' 2>/dev/null) || true
    if [[ -z "$CLUSTER_IP" ]]; then
        warn "Could not resolve slurmrestd ClusterIP for $VERSION - skipping"
        SKIPPED=$((SKIPPED + ${#TAPES[@]}))
        continue
    fi
    SLURM_URL="http://${CLUSTER_IP}:6820"

    # Generate JWT token
    info "Generating JWT token for $VERSION..."
    TOKEN=$("$GET_TOKEN" "$VERSION" 2>/dev/null) || true
    if [[ -z "$TOKEN" ]]; then
        warn "Could not generate token for $VERSION - skipping"
        SKIPPED=$((SKIPPED + ${#TAPES[@]}))
        continue
    fi

    # Verify connectivity
    API_VER="${CLUSTER_API_VERSION[$VERSION]}"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "X-SLURM-USER-NAME: root" \
        -H "X-SLURM-USER-TOKEN: $TOKEN" \
        "$SLURM_URL/slurm/$API_VER/ping" 2>/dev/null) || true
    if [[ "$HTTP_CODE" != "200" ]]; then
        warn "Connectivity check failed for $VERSION (HTTP $HTTP_CODE) - skipping"
        SKIPPED=$((SKIPPED + ${#TAPES[@]}))
        continue
    fi
    ok "Connected to $VERSION at $SLURM_URL (API $API_VER)"

    # Seed cluster with test data (accounts, users, jobs, etc.)
    if ! $SKIP_SEED; then
        info "Seeding $VERSION with test data..."
        if ! "$SEED_CLUSTER" "$VERSION"; then
            warn "Seeding failed for $VERSION - tests may see empty views"
        fi
        # Give Slurm a moment to schedule the submitted jobs
        sleep 2
    fi

    # Create output directory for this version
    VERSION_OUTPUT="$OUTPUT_DIR/$VERSION"
    mkdir -p "$VERSION_OUTPUT"

    # Run each tape
    for TAPE_NAME in "${TAPES[@]}"; do
        TOTAL=$((TOTAL + 1))
        TAPE_FILE="$TAPES_DIR/$TAPE_NAME.tape"
        RAW_OUTPUT="$VERSION_OUTPUT/$TAPE_NAME.ascii"
        NORMALIZED="$VERSION_OUTPUT/$TAPE_NAME.normalized.ascii"
        GOLDEN_FILE="$GOLDEN_DIR/$VERSION/$TAPE_NAME.ascii"

        info "Running tape: $TAPE_NAME against $VERSION..."

        # Create wrapper tape that prepends Output directive
        WRAPPER="$PROJECT_ROOT/.vhs-integration-run.tape"
        echo "Output \"$RAW_OUTPUT\"" > "$WRAPPER"
        cat "$TAPE_FILE" >> "$WRAPPER"

        # Run VHS with environment variables for Slurm connection
        if ! SLURM_REST_URL="$SLURM_URL" SLURM_JWT="$TOKEN" SLURM_API_VERSION="$API_VER" \
            vhs "$WRAPPER" 2>/dev/null; then
            fail "$VERSION/$TAPE_NAME - VHS execution failed"
            FAILED=$((FAILED + 1))
            FAILURES+=("$VERSION/$TAPE_NAME (VHS error)")
            rm -f "$WRAPPER"
            continue
        fi
        rm -f "$WRAPPER"

        # Check that output was produced
        if [[ ! -f "$RAW_OUTPUT" ]]; then
            fail "$VERSION/$TAPE_NAME - No output produced"
            FAILED=$((FAILED + 1))
            FAILURES+=("$VERSION/$TAPE_NAME (no output)")
            continue
        fi

        # Normalize output
        "$NORMALIZE" "$RAW_OUTPUT" "$NORMALIZED"

        if $UPDATE_GOLDEN; then
            # Update golden file
            mkdir -p "$(dirname "$GOLDEN_FILE")"
            cp "$NORMALIZED" "$GOLDEN_FILE"
            ok "$VERSION/$TAPE_NAME - Golden file updated"
            PASSED=$((PASSED + 1))
        else
            # Compare against golden
            if [[ ! -f "$GOLDEN_FILE" ]]; then
                fail "$VERSION/$TAPE_NAME - No golden file (run with --update-golden first)"
                FAILED=$((FAILED + 1))
                FAILURES+=("$VERSION/$TAPE_NAME (no golden)")
                continue
            fi

            if diff -u "$GOLDEN_FILE" "$NORMALIZED" > "$VERSION_OUTPUT/$TAPE_NAME.diff" 2>&1; then
                ok "$VERSION/$TAPE_NAME"
                PASSED=$((PASSED + 1))
                rm -f "$VERSION_OUTPUT/$TAPE_NAME.diff"
            else
                fail "$VERSION/$TAPE_NAME - Output differs from golden"
                echo "  Diff: $VERSION_OUTPUT/$TAPE_NAME.diff"
                head -20 "$VERSION_OUTPUT/$TAPE_NAME.diff" | sed 's/^/  /'
                FAILED=$((FAILED + 1))
                FAILURES+=("$VERSION/$TAPE_NAME")
            fi
        fi
    done
done

# Summary
echo ""
echo "=============================="
if $UPDATE_GOLDEN; then
    info "Golden files updated"
else
    info "Test Results"
fi
echo "  Total:   $TOTAL"
echo -e "  ${GREEN}Passed:  $PASSED${NC}"
if [[ $FAILED -gt 0 ]]; then
    echo -e "  ${RED}Failed:  $FAILED${NC}"
fi
if [[ $SKIPPED -gt 0 ]]; then
    echo -e "  ${YELLOW}Skipped: $SKIPPED${NC}"
fi
echo "=============================="

if [[ ${#FAILURES[@]} -gt 0 ]]; then
    echo ""
    fail "Failed tests:"
    for f in "${FAILURES[@]}"; do
        echo "  - $f"
    done
    exit 1
fi

if [[ $TOTAL -eq 0 ]]; then
    warn "No tests were run (all versions skipped)"
    exit 1
fi

exit 0
