#!/bin/bash
#
# Usage: ./run_fuzz.sh <StructName> [FuzzTime]
#
# Example: ./run_fuzz.sh SignedBeaconBlock 30s
#
# This script runs the round-trip fuzz test for a given SSZ struct.

set -e

STRUCT_NAME=$1
FUZZ_TIME=${2:-30s} # Default to 30s if not provided

if [ -z "$STRUCT_NAME" ]; then
    echo "Usage: $0 <StructName> [FuzzTime]"
    echo "Example: $0 SignedBeaconBlock 30s"
    exit 1
fi

FUZZ_TARGET_NAME="Fuzz${STRUCT_NAME}RoundTrip"

echo "Running fuzz test for '$STRUCT_NAME'..."
echo "Fuzz target: $FUZZ_TARGET_NAME"
echo "Time limit: $FUZZ_TIME"

# The fuzz tests are located in the ./fuzz directory.
go test -a -p 1 ./fuzz -run="^${FUZZ_TARGET_NAME}$" -fuzz="^${FUZZ_TARGET_NAME}$" -fuzztime=$FUZZ_TIME
