#!/bin/bash
#
# Usage: ./scripts/measure_fastssz_regressions.sh
#
# Measures exposure time for each fastssz regression by applying its patch
# and running regression checks. Outputs a CSV summary.

set -euo pipefail

SCRIPT_DIR=$(dirname "$0")
BASE_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
FASTSSZ_DIR="$BASE_DIR/workspace/fastssz"
RESULTS_FILE="${RESULTS_FILE:-$BASE_DIR/fastssz_regression_tte.csv}"

BUGS=(
  FSSZ-INT-01
  FSSZ-222
  FSSZ-INT-02
  FSSZ-INT-03
  FSSZ-181
  FSSZ-162
  FSSZ-152
  FSSZ-127
  FSSZ-54
  FSSZ-76
  FSSZ-153
  FSSZ-158
  FSSZ-166
  FSSZ-136
  FSSZ-156
  FSSZ-159
  FSSZ-164
  FSSZ-188
  FSSZ-86
  FSSZ-100
  FSSZ-149
  FSSZ-151
  FSSZ-1
  FSSZ-49
  FSSZ-52
  FSSZ-173
  FSSZ-147
  FSSZ-119
  FSSZ-111
  FSSZ-110
  FSSZ-98
  FSSZ-96
  FSSZ-9
  FSSZ-23
)

SKIP_TOGGLE=(
  FSSZ-173
)

now_ms() {
  python3 - <<'PY'
import time
print(int(time.time() * 1000))
PY
}

tmp_out="$(mktemp)"
cleanup() {
  rm -f "$tmp_out"
}
trap cleanup EXIT

echo "bug,found,tte_ms,stage" > "$RESULTS_FILE"

for bug in "${BUGS[@]}"; do
  echo "==> Measuring $bug"
  already_active=0
  skip_toggle=0
  for skip in "${SKIP_TOGGLE[@]}"; do
    if [ "$bug" = "$skip" ]; then
      skip_toggle=1
      already_active=1
      break
    fi
  done

  if [ $skip_toggle -eq 0 ]; then
    activate_output="$("$BASE_DIR/scripts/bug_toggle.sh" activate "$bug")"
    echo "$activate_output"
    if echo "$activate_output" | grep -q "already active"; then
      already_active=1
    fi
  else
    echo "Skipping toggle for $bug (treated as already active)."
  fi

  start_ms="$(now_ms)"
  found=0
  stage=""
  regenerated=0

  set +e
  (
    cd "$FASTSSZ_DIR"
    go test ./...
  ) >"$tmp_out" 2>&1
  status=$?
  set -e

  if [ $status -ne 0 ]; then
    found=1
    stage="fastssz-go-test"
  else
    set +e
    (
      cd "$FASTSSZ_DIR/sszgen/testcases"
      go generate ./...
    ) >"$tmp_out" 2>&1
    status=$?
    set -e
    regenerated=1

    if [ $status -ne 0 ]; then
      found=1
      stage="sszgen-go-generate"
    else
      set +e
      (
        cd "$FASTSSZ_DIR/sszgen/testcases"
        go test ./...
      ) >"$tmp_out" 2>&1
      status=$?
      set -e

      if [ $status -ne 0 ]; then
        found=1
        stage="sszgen-go-test"
      fi
    fi
  fi

  end_ms="$(now_ms)"
  tte_ms=$((end_ms - start_ms))

  if [ $found -eq 1 ]; then
    echo "$bug,1,$tte_ms,$stage" >> "$RESULTS_FILE"
  else
    echo "$bug,0,,$stage" >> "$RESULTS_FILE"
  fi

  if [ $skip_toggle -eq 0 ] && [ $already_active -eq 0 ]; then
    "$BASE_DIR/scripts/bug_toggle.sh" deactivate "$bug" > /dev/null 2>&1 || true
  fi

  if [ $regenerated -eq 1 ]; then
    (
      cd "$FASTSSZ_DIR/sszgen/testcases"
      go generate ./...
    ) >"$tmp_out" 2>&1 || true
  fi
done

RESULTS_FILE="$RESULTS_FILE" python3 - <<'PY'
import csv
import statistics
import os

path = os.environ["RESULTS_FILE"]
durations = []
found = 0
total = 0
with open(path, newline="") as f:
    reader = csv.DictReader(f)
    for row in reader:
        total += 1
        if row["found"] == "1":
            found += 1
            durations.append(int(row["tte_ms"]))

median = statistics.median(durations) if durations else None
print(f"Reproduced: {found}/{total}")
if median is not None:
    print(f"Median TTE (ms): {median}")
else:
    print("Median TTE (ms): --")
print(f"Results written to {path}")
PY
