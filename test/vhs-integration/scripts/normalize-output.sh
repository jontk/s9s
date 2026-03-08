#!/usr/bin/env bash
# Extract the last TUI frame from VHS ASCII output and normalize dynamic data
# Usage: normalize-output.sh <input.ascii> <output.ascii>
#
# VHS ASCII files contain multiple rendered frames. We extract the LAST rendered
# TUI screen (identified by the "S9S - SLURM Terminal UI" header) and normalize
# dynamic values so diffs only catch structural/behavioral changes.

set -euo pipefail

INPUT="${1:?Usage: normalize-output.sh <input.ascii> <output.ascii>}"
OUTPUT="${2:?Usage: normalize-output.sh <input.ascii> <output.ascii>}"

if [[ ! -f "$INPUT" ]]; then
    echo "ERROR: Input file not found: $INPUT" >&2
    exit 1
fi

# Step 1: Strip ANSI escapes and extract the last TUI frame
python3 -c "
import re, sys

with open(sys.argv[1], 'r', errors='replace') as f:
    content = f.read()

# Strip ANSI escape sequences
clean = re.sub(r'\x1b\[[0-9;]*[a-zA-Z]', '', content)
clean = re.sub(r'\x1b\][^\x07]*\x07', '', clean)
clean = re.sub(r'\x1b[^[\x1b]', '', clean)

# Split into lines and find the LAST occurrence of the TUI header
lines = clean.split('\n')
last_header_idx = -1
for i, line in enumerate(lines):
    if 'S9S - SLURM Terminal UI' in line:
        last_header_idx = i

if last_header_idx >= 0:
    # Extract from header to end of TUI content (before log/exit lines)
    frame_lines = lines[last_header_idx:]
    end_idx = len(frame_lines)
    for i, line in enumerate(frame_lines):
        stripped = line.strip()
        if i > 2 and (re.match(r'^\d{4}-\d{2}-\d{2}T', stripped) or
                       stripped == '>' or
                       'shutdown complete' in stripped):
            end_idx = i
            break
    print('\n'.join(frame_lines[:end_idx]))
else:
    # No TUI header found - use last 30 non-empty lines as fallback
    non_empty = [l for l in lines if l.strip()]
    print('\n'.join(non_empty[-30:]))
" "$INPUT" > "$OUTPUT.tmp"

# Step 2: Normalize dynamic data
sed -E \
    -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}/YYYY-MM-DDTHH:MM:SS/g' \
    -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}/YYYY-MM-DD HH:MM:SS/g' \
    -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}/YYYY-MM-DD HH:MM/g' \
    -e 's/ *-?[0-9]+:-?[0-9]+:-?[0-9]+/HH:MM:SS/g' \
    -e 's/ *-?[0-9]+:-?[0-9]+\.\.\./HH:MM.../g' \
    -e 's/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/X.X.X.X/g' \
    -e 's/[0-9]+d [0-9]+h/Xd Xh/g' \
    -e 's/[0-9]+h [0-9]+m/Xh Xm/g' \
    -e 's/[0-9]+m [0-9]+s/Xm Xs/g' \
    -e 's/ [0-9]+(\.[0-9]+)?d([│ ])/ Xd\2/g' \
    -e 's/ [0-9]+(\.[0-9]+)?h([│ ])/ Xh\2/g' \
    -e 's/ [0-9]+(\.[0-9]+)?m([│ ])/ Xm\2/g' \
    -e 's/ [0-9]+(\.[0-9]+)?s([│ ])/ Xs\2/g' \
    -e 's/\|HH:MM:SS$/| HH:MM/g' \
    -e 's/token lifespan=[0-9]+/token lifespan=XXXX/g' \
    -e 's/─{2,}/──/g' \
    -e 's/ +│/│/g' \
    -e 's/ {2,}/ /g' \
    "$OUTPUT.tmp" > "$OUTPUT.tmp2"

# Step 3: Remove trailing artifacts
sed -e '/^>$/d' -e '/^[[:space:]]*$/d' "$OUTPUT.tmp2" > "$OUTPUT"

rm -f "$OUTPUT.tmp" "$OUTPUT.tmp2"
