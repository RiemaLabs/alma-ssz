#!/bin/bash
#
# Usage: ./bug_toggle.sh [activate|deactivate] [bitvector|boolean|gap]
#
# This script applies/reverses patches in the fastssz workspace to
# introduce (activate) or remove (deactivate) specific bugs.

set -e

ACTION=$1
BUG_NAME=$2

if [ -z "$ACTION" ] || [ -z "$BUG_NAME" ]; then
    echo "Usage: $0 [activate|deactivate] [bitvector|boolean|gap]"
    exit 1
fi

SCRIPT_DIR=$(dirname "$0")
BASE_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
FASTSSZ_DIR="$BASE_DIR/workspace/fastssz"
PATCHES_DIR="$BASE_DIR/patches"

PATCH_FILE=""
case "$BUG_NAME" in
    "bitvector")
        PATCH_FILE="$PATCHES_DIR/Bitvector_Dirty_Padding.patch"
        ;;
    "boolean")
        PATCH_FILE="$PATCHES_DIR/Dirty_Boolean.patch"
        ;;
    "gap")
        PATCH_FILE="$PATCHES_DIR/Container_Gap.patch"
        ;;
    *)
        echo "Error: Invalid bug name '$BUG_NAME'. Valid options: bitvector, boolean, gap."
        exit 1
        ;;
esac

if [ ! -f "$PATCH_FILE" ]; then
    echo "Error: Patch file not found at $PATCH_FILE"
    exit 1
fi

cd "$FASTSSZ_DIR"

case "$ACTION" in
    activate)
        echo "Activating bug '$BUG_NAME' by applying patch..."
        # Check if already applied? Hard to check generically.
        # git apply checks if it can be applied.
        if git apply --check "$PATCH_FILE"; then
            git apply "$PATCH_FILE"
            echo "Bug '$BUG_NAME' activated."
        else
            echo "Error: Cannot apply patch. It might already be applied or conflict with other changes."
            exit 1
        fi
        ;;
    deactivate)
        echo "Deactivating bug '$BUG_NAME' by reversing patch..."
        if git apply --check -R "$PATCH_FILE"; then
            git apply -R "$PATCH_FILE"
            echo "Bug '$BUG_NAME' deactivated."
        else
            echo "Error: Cannot reverse patch. It might not be applied."
            exit 1
        fi
        ;;
    *)
        echo "Error: Invalid action '$ACTION'. Use 'activate' or 'deactivate'."
        exit 1
        ;;
esac
