#!/bin/bash
#
# Usage: ./bug_toggle.sh [activate|deactivate] [ex1|ex2]
#
# This script checks out specific git commits in the fastssz workspace to
# introduce (activate) or remove (deactivate) historical bugs.

set -e

ACTION=$1
BUG_NAME=$2

if [ -z "$ACTION" ] || [ -z "$BUG_NAME" ]; then
    echo "Usage: $0 [activate|deactivate] [ex1|ex2]"
    exit 1
fi

# --- Commit Hashes ---
# The commit that FIXED the bug.
# To activate the bug, we check out the commit *before* the fix (parent).
EX1_FIX_COMMIT="7df50c8568f8"
EX2_FIX_COMMIT="571a8a27b64b9e64ac9943ddb933040b695b2ba1"
DEFAULT_BRANCH="main"
# ---------------------

SCRIPT_DIR=$(dirname "$0")
BASE_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
FASTSSZ_DIR="$BASE_DIR/workspace/fastssz"

cd "$FASTSSZ_DIR"

# Ensure the repository is clean before switching states
echo "Stashing any local changes in fastssz..."
git stash push -m "gemini-cli-temp-stash" || true
# In case there's nothing to stash
STASH_COUNT=$(git stash list | wc -l)

BUG_COMMIT=""
if [ "$BUG_NAME" == "ex1" ]; then
    BUG_COMMIT="${EX1_FIX_COMMIT}~1"
elif [ "$BUG_NAME" == "ex2" ]; then
    BUG_COMMIT="${EX2_FIX_COMMIT}~1"
else
    echo "Error: Invalid bug name '$BUG_NAME'. Use 'ex1' or 'ex2'."
    exit 1
fi

case "$ACTION" in
    activate)
        echo "Activating bug '$BUG_NAME' by checking out commit $BUG_COMMIT..."
        git checkout "$BUG_COMMIT"
        echo "Bug '$BUG_NAME' activated."
        ;;
    deactivate)
        echo "Deactivating bug '$BUG_NAME' by checking out branch '$DEFAULT_BRANCH'..."
        git checkout "$DEFAULT_BRANCH"
        # If we created a stash, pop it to restore local changes
        if [ "$STASH_COUNT" -lt "$(git stash list | wc -l)" ]; then
          echo "Restoring local changes..."
          git stash pop || true # Avoid error if stash is empty after checkout
        fi
        echo "Bug '$BUG_NAME' deactivated. Workspace is on the latest code."
        ;;
    *)
        echo "Error: Invalid action '$ACTION'. Use 'activate' or 'deactivate'."
        exit 1
        ;;
esac