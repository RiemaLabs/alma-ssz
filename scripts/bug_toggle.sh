#!/bin/bash
#
# Usage: ./bug_toggle.sh [activate|deactivate] [bitvector|boolean|bitlist|gap|FSSZ-###]
#
# This script applies/reverses patches in the fastssz workspace to
# introduce (activate) or remove (deactivate) specific bugs.

set -e

ACTION=$1
BUG_NAME=$2

if [ -z "$ACTION" ] || [ -z "$BUG_NAME" ]; then
    echo "Usage: $0 [activate|deactivate] [bitvector|boolean|bitlist|gap|FSSZ-###]"
    exit 1
fi

SCRIPT_DIR=$(dirname "$0")
BASE_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
FASTSSZ_DIR="$BASE_DIR/workspace/fastssz_bench"
PATCHES_DIR="$BASE_DIR/patches"

PATCH_FILE=""
case "$BUG_NAME" in
    "bitvector")
        PATCH_FILE="$PATCHES_DIR/Bitvector_Dirty_Padding.patch"
        ;;
    "boolean")
        PATCH_FILE="$PATCHES_DIR/Dirty_Boolean.patch"
        ;;
    "bitlist")
        PATCH_FILE="$PATCHES_DIR/Null_Bitlist.patch"
        ;;
    "gap")
        PATCH_FILE="$PATCHES_DIR/Container_Gap.patch"
        ;;
    "FSSZ-INT-01"|"fssz-int-01")
        PATCH_FILE="$PATCHES_DIR/Bitvector_Dirty_Padding.patch"
        ;;
    "FSSZ-222"|"fssz-222")
        PATCH_FILE="$PATCHES_DIR/Dirty_Boolean.patch"
        ;;
    "FSSZ-INT-02"|"fssz-int-02")
        PATCH_FILE="$PATCHES_DIR/Null_Bitlist.patch"
        ;;
    "FSSZ-INT-03"|"fssz-int-03")
        PATCH_FILE="$PATCHES_DIR/Container_Gap.patch"
        ;;
    *)
        if [[ "$BUG_NAME" == FSSZ-* ]]; then
            PATCH_FILE="$PATCHES_DIR/fastssz/$BUG_NAME.patch"
        elif [[ "$BUG_NAME" == fssz-* ]]; then
            BUG_UPPER=$(printf '%s' "$BUG_NAME" | tr '[:lower:]' '[:upper:]')
            PATCH_FILE="$PATCHES_DIR/fastssz/$BUG_UPPER.patch"
        else
            echo "Error: Invalid bug name '$BUG_NAME'. Valid options: bitvector, boolean, bitlist, gap, FSSZ-###."
            exit 1
        fi
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
        if git apply --check "$PATCH_FILE"; then
            git apply "$PATCH_FILE"
            echo "Bug '$BUG_NAME' activated."
        elif git apply --check -R "$PATCH_FILE"; then
            echo "Bug '$BUG_NAME' already active."
        else
            echo "Error: Cannot apply patch. It might conflict with other changes."
            exit 1
        fi
        ;;
    deactivate)
        echo "Deactivating bug '$BUG_NAME' by reversing patch..."
        if git apply --check -R "$PATCH_FILE"; then
            git apply -R "$PATCH_FILE"
            echo "Bug '$BUG_NAME' deactivated."
        elif git apply --check "$PATCH_FILE"; then
            echo "Bug '$BUG_NAME' already inactive."
        else
            echo "Error: Cannot reverse patch. It might conflict with other changes."
            exit 1
        fi
        ;;
    *)
        echo "Error: Invalid action '$ACTION'. Use 'activate' or 'deactivate'."
        exit 1
        ;;
esac
