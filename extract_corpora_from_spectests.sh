#!/usr/bin/env sh
set -e

# Fetch SSZ corpora from the new consensus-spec-tests repository
# Repo: https://github.com/ethereum/consensus-spec-tests
# Env:
#   SPEC_REF  - git ref (branch/tag/commit), default: master
#   SPEC_CFG  - config (mainnet|minimal), default: mainnet
#   LIMIT_PER - optional per-type cap (integer)

SPEC_REF=${SPEC_REF:-master}
SPEC_CFG=${SPEC_CFG:-mainnet}
LIMIT_PER=${LIMIT_PER:-}
ALL_TYPES=${ALL_TYPES:-}

ROOT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
CORPORA_DIR="$ROOT_DIR/workspace/corpora"

echo "[corpora] consensus-spec-tests @$SPEC_REF (config=$SPEC_CFG)"

# Require python-snappy for decompressing ssz_snappy; do not fallback
if ! python3 -c "import snappy" >/dev/null 2>&1; then
  echo "[corpora] ERROR: python-snappy not available. Please install it first:" >&2
  echo "         macOS:   brew install snappy && python3 -m pip install python-snappy" >&2
  echo "         Linux:   apt-get install -y libsnappy-dev python3-pip && python3 -m pip install python-snappy" >&2
  exit 1
fi

TMPDIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'corpora')
ARCHIVE="$TMPDIR/spec_tests.tar.gz"
EXTRACT="$TMPDIR/spec_tests"
mkdir -p "$EXTRACT"

curl -fsSL "https://codeload.github.com/ethereum/consensus-spec-tests/tar.gz/$SPEC_REF" -o "$ARCHIVE"
tar -xzf "$ARCHIVE" -C "$EXTRACT"

SPEC_ROOT=$(find "$EXTRACT" -maxdepth 1 -type d -name 'consensus-spec-tests-*' | head -n1)
if [ ! -d "$SPEC_ROOT" ]; then
  SPEC_ROOT=$(find "$EXTRACT" -maxdepth 1 -type d -name 'ethereum-consensus-spec-tests-*' | head -n1)
fi
if [ ! -d "$SPEC_ROOT" ]; then
  echo "[corpora] ERROR: cannot find extracted root" >&2
  exit 1
fi
TESTS_DIR="$SPEC_ROOT/tests/$SPEC_CFG"
if [ ! -d "$TESTS_DIR" ]; then
  echo "[corpora] ERROR: tests dir not found: $TESTS_DIR" >&2
  exit 1
fi

mkdir -p "$CORPORA_DIR/attestation" \
         "$CORPORA_DIR/attester_slashing" \
         "$CORPORA_DIR/proposer_slashing" \
         "$CORPORA_DIR/block" \
         "$CORPORA_DIR/block_header" \
         "$CORPORA_DIR/deposit" \
         "$CORPORA_DIR/voluntary_exit" \
         "$CORPORA_DIR/beaconstate" \
         "$CORPORA_DIR/enr" \
         "$CORPORA_DIR/discv5_packet" \
         "$CORPORA_DIR/bls"

# Decompress + write using content-hash filenames to avoid collisions
# Always emits: <dest>/<sha256>.ssz
extract_to_hashed() {
  src="$1"
  dest_dir="$2"
  python3 - "$src" "$dest_dir" <<'PY'
import sys, os, hashlib
try:
    import snappy
except Exception as e:
    snappy = None

src, dest = sys.argv[1], sys.argv[2]
with open(src, 'rb') as f:
    data = f.read()

if src.endswith('.ssz'):
    raw = data
else:
    if snappy is None:
        raise SystemExit('python-snappy required to decompress {}'.format(src))
    raw = snappy.decompress(data)

h = hashlib.sha256(raw).hexdigest()
out = os.path.join(dest, f"{h}.ssz")
with open(out, 'wb') as f:
    f.write(raw)
print(out)
PY
}

copy_type_dir() {
  type_dir_glob="$1"   # e.g., Attestation*
  dest="$2"
  copied=0
  # Search both ssz_static and ssz_generic across forks
  for fork in $(find "$TESTS_DIR" -maxdepth 1 -mindepth 1 -type d); do
    for base in ssz_static ssz_generic; do
      [ -d "$fork/$base" ] || continue
      # Type directories directly under base
      for typ in $(find "$fork/$base" -maxdepth 1 -type d -name "$type_dir_glob" 2>/dev/null); do
        # Cases under type
        for case_dir in $(find "$typ" -mindepth 1 -maxdepth 3 -type d 2>/dev/null); do
          # Try common filenames
          for fname in ssz_snappy serialized.ssz_snappy serialized.ssz; do
            f="$case_dir/$fname"
            if [ -f "$f" ]; then
              extract_to_hashed "$f" "$dest"
              copied=$((copied+1))
              break
            fi
          done
          if [ -n "$LIMIT_PER" ] && [ "$copied" -ge "$LIMIT_PER" ]; then
            echo "$copied"; return
          fi
        done
      done
    done
  done
  echo "$copied"
}

att=$(copy_type_dir 'Attestation*' "$CORPORA_DIR/attestation")
asl=$(copy_type_dir 'AttesterSlashing*' "$CORPORA_DIR/attester_slashing")
psl=$(copy_type_dir 'ProposerSlashing*' "$CORPORA_DIR/proposer_slashing")
dep=$(copy_type_dir 'Deposit*' "$CORPORA_DIR/deposit")
sblk=$(copy_type_dir 'SignedBeaconBlock*' "$CORPORA_DIR/block")
bblk=$(copy_type_dir 'BeaconBlock*' "$CORPORA_DIR/block_header")
sve=$(copy_type_dir 'SignedVoluntaryExit*' "$CORPORA_DIR/voluntary_exit")
bst=$(copy_type_dir 'BeaconState*' "$CORPORA_DIR/beaconstate")

echo "[corpora] Copied counts:"
printf "  Attestation:          %s\n" "$att"
printf "  AttesterSlashing:     %s\n" "$asl"
printf "  ProposerSlashing:     %s\n" "$psl"
printf "  Deposit:              %s\n" "$dep"
printf "  SignedBeaconBlock->block:    %s\n" "$sblk"
printf "  BeaconBlock->block_header:   %s\n" "$bblk"
printf "  SignedVoluntaryExit:  %s\n" "$sve"
printf "  BeaconState:          %s\n" "$bst"

# Optionally copy ALL types present in spec tests into extras/<Type>
if [ "$ALL_TYPES" = "1" ] || [ "$ALL_TYPES" = "true" ]; then
  echo "[corpora] ALL_TYPES=1 enabled: collecting extra types into $CORPORA_DIR/extras/<Type>"
  CORE_TYPES='^(Attestation|AttesterSlashing|ProposerSlashing|Deposit|SignedBeaconBlock|BeaconBlock|SignedVoluntaryExit|BeaconState)$'
  EXTRAS_DIR="$CORPORA_DIR/extras"
  mkdir -p "$EXTRAS_DIR"

  # Enumerate forks and bases
  for fork in $(find "$TESTS_DIR" -maxdepth 1 -mindepth 1 -type d); do
    for base in ssz_static ssz_generic; do
      [ -d "$fork/$base" ] || continue
      # Types under base
      for typ_dir in $(find "$fork/$base" -maxdepth 1 -mindepth 1 -type d 2>/dev/null); do
        type_name=$(basename "$typ_dir")
        # skip core types, only extras
        echo "$type_name" | grep -Eq "$CORE_TYPES" && continue
        dest="$EXTRAS_DIR/$type_name"
        mkdir -p "$dest"
        copied=0
        # Walk cases
        for case_dir in $(find "$typ_dir" -mindepth 1 -maxdepth 3 -type d 2>/dev/null); do
          for fname in ssz_snappy serialized.ssz_snappy serialized.ssz; do
            f="$case_dir/$fname"
            if [ -f "$f" ]; then
              extract_to_hashed "$f" "$dest"
              copied=$((copied+1))
              break
            fi
          done
          if [ -n "$LIMIT_PER" ] && [ "$copied" -ge "$LIMIT_PER" ]; then
            break
          fi
        done
        # Print per-type summary line if any copied
        printf "  [extras] %-24s %s\n" "$type_name:" "$copied"
      done
    done
  done
fi

# Placeholders for inputs not provided by spec tests
for d in enr discv5_packet bls; do
  if [ ! -e "$CORPORA_DIR/$d/seed" ]; then
    printf X > "$CORPORA_DIR/$d/seed"
  fi
done

echo "[corpora] Done: $CORPORA_DIR"
