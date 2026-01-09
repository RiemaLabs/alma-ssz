# Alma: specification-guided structural fuzzing for Ethereum SSZ implementations

## Layout
- `workspace/`: heavyweight inputs cloned locally (consensus-specs, consensus-spec-tests mainnet slice, Prysm, fastssz, corpora, ...). Everything under this directory is ignored by git so you can freely sync upstream changes.
- `benchschemas/`: benchmark harness schemas (fastssz regressions + py-ssz oracles).
- `internal/`: Go packages for corpus loading, mutation helpers, and oracles.
- `fuzz/`: Native Go fuzz entrypoints (`go test -run=Fuzz`).
- `cmd/`: helper CLIs (corpus exporter, roundtrip generator, etc.).
- `config/`: JSON descriptors that feed generators (currently `roundtrip_targets.json`).
- `oracle/`: external oracles (py-ssz bridge).
- `schemas/`: SSZ schema variants used by fuzzers and ablations.
- `scripts/`: benchmark/ablation automation helpers.

## Quickstart
1. Clone upstream dependencies (Phase 1 of `agent.md`):
   ```bash
   mkdir -p workspace
   git clone https://github.com/ethereum/consensus-specs workspace/consensus-specs
   git clone https://github.com/ethereum/consensus-spec-tests workspace/tests   # mainnet slice only is fine
   git clone https://github.com/prysmaticlabs/prysm -b v5.0.4 workspace/prysm
   git clone https://github.com/ferranbt/fastssz workspace/fastssz
   # Optional: separate fastssz workspace used for patching benchmarks (can be a symlink).
   git clone https://github.com/ferranbt/fastssz workspace/fastssz_bench
   # Optional: py-ssz for cross-client oracle benchmarks.
   git clone https://github.com/ethereum/py-ssz workspace/py-ssz
   ```
2. Install `sszgen` from the cloned fastssz repo:
   ```bash
   cd workspace/fastssz
   go install ./sszgen
   ```
3. Run the round-trip oracle fuzzers (native Go fuzzing). The harness automatically loads up to 256 seeds per target from `workspace/tests`, so you start with hundreds of real SSZ samples without touching `testdata/`:
   ```bash
   go test ./fuzz -run=Fuzz
   ```
   Under the hood we parse `.ssz` / `.ssz_snappy` vectors straight from `workspace/tests/mainnet/**/{BeaconBlockBody,Attestation,SignedBeaconBlock,IndexedAttestation}` and feed them through the round-trip oracle before mutation begins.
4. Add/remove roundtrip fuzz targets declaratively by editing `config/roundtrip_targets.json`, then regenerate the fuzz harness:
   ```bash
   go generate ./fuzz
   ```
   This rewrites `fuzz/roundtrip_targets_generated_test.go` so every SSZ struct follows the same template without copy/pasting Go code.

To materialize the raw SSZ seeds (for archiving or debugging), run:
```bash
go run ./cmd/corpusseed -out corpus/export -limit 256 -format zip
```

## Optional: py-ssz Oracle Setup
The py-ssz oracle powers cross-client benchmarks and schema-validation checks.
```bash
python3 -m venv .venv
. .venv/bin/activate
python -m pip install -U pip
python -m pip install -e workspace/py-ssz
```
If you are not using the default `.venv`, set:
```bash
export ALMA_PYSSZ_PYTHON=/path/to/python3
```

## Benchmark + Ablation Measurements
- Run the full ablation suite (writes `ablation_results.csv`):
  ```bash
  python3 scripts/measure_ablation.py --budget 30s --max-steps 200000 --batch-size 50 --trials 5
  ```
- Measure fastssz regression exposure times (writes `fastssz_regression_tte.csv`):
  ```bash
  ./scripts/measure_fastssz_regressions.sh
  ```
- Build the consolidated benchmark CSV used by the paper:
  ```bash
  python3 scripts/build_benchmark_results.py
  ```

## Regression Testing Workflow

This project includes a workflow to test historical bugs from `fastssz`. The process uses patches to inject a bug and a generic fuzzing script to test for it. Patches are applied in `workspace/fastssz_bench` (use a second clone or symlink to `workspace/fastssz`).

### Scripts

- `scripts/bug_toggle.sh`: Activates or deactivates a bug by applying or reverting a patch in the `workspace/fastssz_bench` directory.
- `scripts/run_fastssz_regression.sh`: Applies a patch and runs `go test` for a single fastssz regression.
- `scripts/measure_fastssz_regressions.sh`: Batch measurement for all fastssz regressions.

### Example Workflow: Testing a fastssz Regression

Here is a step-by-step guide to test `FSSZ-147`.

**1. Activate the Bug**

Introduce the bug by applying the regression patch.

```bash
./scripts/bug_toggle.sh activate FSSZ-147
```

**2. Run the Fuzz Tester**

Run the fastssz regression tests.

```bash
./scripts/run_fastssz_regression.sh FSSZ-147
```

**3. Deactivate the Bug**

Revert the fastssz workspace to its clean state.

```bash
./scripts/bug_toggle.sh deactivate FSSZ-147
```
