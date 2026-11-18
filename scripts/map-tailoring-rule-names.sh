#!/usr/bin/env bash
# Helper script to fill in rule_name placeholders in exported tailoring files
# by cross-referencing with existing tailoring files that have known rule_name values.
#
# Usage: ./scripts/map-tailoring-rule-names.sh EXPORTED_FILE REFERENCE_DIR
#
# Example:
#   ./scripts/map-tailoring-rule-names.sh \
#     /tmp/export/RHEL_9_STIG.tailoring.yaml \
#     benchmarks/

set -euo pipefail

if [ $# -ne 2 ]; then
    cat >&2 << 'EOF'
Usage: map-tailoring-rule-names.sh EXPORTED_FILE REFERENCE_DIR

Attempts to fill in PLACEHOLDER rule_name values in an exported tailoring
file by matching stig_id with existing tailoring files in REFERENCE_DIR.

Arguments:
  EXPORTED_FILE   - Tailoring file with PLACEHOLDER rule_name values
  REFERENCE_DIR   - Directory containing reference tailoring files with known rule_name values

Example:
  ./scripts/map-tailoring-rule-names.sh \
    /tmp/export/RHEL_9_STIG.tailoring.yaml \
    benchmarks/

The script prints a report showing which rule_names were found and which remain PLACEHOLDER.
It does NOT modify the exported file - review the output and apply changes manually.
EOF
    exit 2
fi

EXPORTED_FILE="$1"
REFERENCE_DIR="$2"

if [ ! -f "$EXPORTED_FILE" ]; then
    echo "Error: Exported file not found: $EXPORTED_FILE" >&2
    exit 1
fi

if [ ! -d "$REFERENCE_DIR" ]; then
    echo "Error: Reference directory not found: $REFERENCE_DIR" >&2
    exit 1
fi

# Extract all STIG IDs from the exported file that have PLACEHOLDER
PLACEHOLDERS=$(grep -A1 "rule_name: PLACEHOLDER" "$EXPORTED_FILE" | grep "stig_id:" | sed 's/.*stig_id: //' || true)

if [ -z "$PLACEHOLDERS" ]; then
    echo "No PLACEHOLDER rule_names found in $EXPORTED_FILE"
    exit 0
fi

echo "Searching for rule_name mappings in $REFERENCE_DIR..."
echo ""

FOUND=0
NOT_FOUND=0

while IFS= read -r STIG_ID; do
    if [ -z "$STIG_ID" ]; then
        continue
    fi
    
    # Search for this STIG ID in reference files
    MATCH=$(grep -A1 "$STIG_ID" "$REFERENCE_DIR"/*.tailoring.yaml 2>/dev/null | grep "rule_name:" | head -1 | sed 's/.*rule_name: //' || true)
    
    if [ -n "$MATCH" ]; then
        echo "✓ $STIG_ID → rule_name: $MATCH"
        FOUND=$((FOUND + 1))
    else
        echo "✗ $STIG_ID → rule_name: NOT FOUND in references"
        NOT_FOUND=$((NOT_FOUND + 1))
    fi
done <<< "$PLACEHOLDERS"

echo ""
echo "Summary:"
echo "  Found: $FOUND"
echo "  Not found: $NOT_FOUND"
echo ""

if [ $NOT_FOUND -gt 0 ]; then
    echo "For STIG IDs not found in references:"
    echo "  1. Consult SSG (SCAP Security Guide) content for your OS"
    echo "  2. Check docket converter documentation"
    echo "  3. Consider leaving rule_name as PLACEHOLDER if not used for automated remediation"
fi
