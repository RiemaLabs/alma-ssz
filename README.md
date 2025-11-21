SSZ Internals playground for structural fuzzing.

## Layout
- `workspace/`: heavyweight inputs cloned locally (consensus-specs, consensus-spec-tests mainnet slice, Prysm, fastssz, corpora, ...). Everything under this directory is ignored by git so you can freely sync upstream changes.
- `internal/`: Go packages for corpus loading, mutation helpers, and oracles.
- `fuzz/`: Native Go fuzz entrypoints (`go test -run=Fuzz`).
- `cmd/`: helper CLIs (corpus exporter, roundtrip generator, etc.).
- `config/`: JSON descriptors that feed generators (currently `roundtrip_targets.json`).

## Quickstart
1. Clone upstream dependencies (Phase 1 of `agent.md`):
   ```bash
   mkdir -p workspace
   git clone https://github.com/ethereum/consensus-specs workspace/consensus-specs
   git clone https://github.com/ethereum/consensus-spec-tests workspace/tests   # mainnet slice only is fine
   git clone https://github.com/prysmaticlabs/prysm -b v5.0.4 workspace/prysm
   git clone https://github.com/ferranbt/fastssz workspace/fastssz
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