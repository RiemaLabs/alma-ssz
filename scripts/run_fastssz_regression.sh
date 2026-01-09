#!/bin/bash
#
# Usage: ./scripts/run_fastssz_regression.sh <FSSZ-ID>
#
# Example: ./scripts/run_fastssz_regression.sh FSSZ-147
#
# This script applies a fastssz regression patch, runs fastssz tests,
# and then reverts the patch.

set -euo pipefail

BUG_NAME=${1:-}
if [ -z "$BUG_NAME" ]; then
    echo "Usage: $0 <FSSZ-ID>"
    exit 1
fi

SCRIPT_DIR=$(dirname "$0")
BASE_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
FASTSSZ_DIR="$BASE_DIR/workspace/fastssz"

cleanup() {
    "$BASE_DIR/scripts/bug_toggle.sh" deactivate "$BUG_NAME" > /dev/null 2>&1 || true
}

trap cleanup EXIT

"$BASE_DIR/scripts/bug_toggle.sh" activate "$BUG_NAME"

echo ">> Running fastssz tests for $BUG_NAME..."
(
    cd "$FASTSSZ_DIR"
    go test ./...
)
