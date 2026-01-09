#!/bin/bash
# Usage: ./scripts/measure_baseline.sh <bug_name> [hard]
# bug_name: bitvector, boolean, bitlist, gap
# hard: (optional) if "hard", run hard version

BUG=$1
MODE=$2

if [ -z "$BUG" ]; then
    echo "Usage: $0 <bug_name> [hard]"
    exit 1
fi

if [ "$MODE" == "hard" ]; then
    case "$BUG" in
        bitvector) TARGET="FuzzHardBitvectorBug" ;;
        boolean)   TARGET="FuzzHardBooleanBug" ;;
        bitlist)   TARGET="FuzzBitlistBug" ;;
        gap)       TARGET="FuzzHardGapBug" ;;
        *) echo "Unknown bug: $BUG"; exit 1 ;;
    esac
else
    case "$BUG" in
        bitvector) TARGET="FuzzBitvectorBug" ;;
        boolean)   TARGET="FuzzBooleanBug" ;;
        bitlist)   TARGET="FuzzBitlistBug" ;;
        gap)       TARGET="FuzzGapBug" ;;
        *) echo "Unknown bug: $BUG"; exit 1 ;;
    esac
fi

echo "================================================="
echo "Measuring baseline fuzzing time for bug: $BUG (Mode: ${MODE:-easy})"
echo "Target: $TARGET"
echo "================================================="

# Ensure bug is activated
echo ">> Activating bug..."
./scripts/bug_toggle.sh activate $BUG > /dev/null 2>&1

# Regenerate SSZ code to propagate generator changes
echo ">> Regenerating SSZ code..."
go run workspace/fastssz/sszgen/main.go --path fuzz/schemas --objs BitvectorStruct,BooleanStruct,BitlistStruct,GapStruct,HardBitvectorStruct,HardBooleanStruct,HardGapStruct > /dev/null 2>&1

# --- CRITICAL CHANGE: Clean corpus before fuzzing ---
FUZZ_CORPUS_DIR="fuzz/schemas/testdata/$TARGET"
echo ">> Cleaning fuzz corpus directory: $FUZZ_CORPUS_DIR"
rm -rf "$FUZZ_CORPUS_DIR"
echo ">> Cleaning fuzz cache..."
go clean -fuzzcache

# Run fuzzing
echo ">> Starting fuzzer..."
START_TIME=$(date +%s%N)
# Run fuzzing. Timeout 60s.
OUTFILE=$(mktemp)

set +e # Allow failure
# -fuzztime=60s
go test -fuzz="^$TARGET$" -fuzztime=1800s ./fuzz/schemas > "$OUTFILE" 2>&1
EXIT_CODE=$?
END_TIME=$(date +%s%N)
set -e

DURATION=$(( (END_TIME - START_TIME) / 1000000 )) # milliseconds

echo ">> Fuzzer finished with exit code $EXIT_CODE"

if [ $EXIT_CODE -ne 0 ]; then
    # Check if it was a bug trigger or build error
    if grep -q "Bug triggered" "$OUTFILE"; then
        echo "SUCCESS: Bug triggered!"
        echo "Time taken: ${DURATION} ms"
        # Extract exec count if available
        grep "fuzz: elapsed" "$OUTFILE" | tail -1
    else
        echo "FAILURE: Fuzzer failed but not due to bug trigger (or timeout/build error)."
        cat "$OUTFILE"
    fi
else
    echo "FAILURE: Fuzzer timed out (60s) without triggering the bug."
    grep "fuzz: elapsed" "$OUTFILE" | tail -1
fi

rm "$OUTFILE"

# Deactivate bug
echo ">> Deactivating bug..."
./scripts/bug_toggle.sh deactivate $BUG > /dev/null 2>&1
echo "================================================="
