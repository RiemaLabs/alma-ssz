# Bucketed SSZ Input Model (BeaconState)

This note explains how we partition the SSZ input space into balanced buckets and how those buckets are applied to the `BeaconState` schema. The goal is to make each step pick a single bucket per field and then sample uniformly inside that bucket, so coverage stays broad without relying on magic constants.

## SSZ background
- Fixed vs. variable sections: A container writes all fixed-size fields first. Every variable-size field (list or byte list) is represented in that fixed section by a little-endian offset pointing to where its data starts in the variable section that follows.
- Offset discipline: With multiple variable fields, offsets must be monotonically increasing and non-overlapping, and each offset must land on the start of its field’s payload. Misplaced offsets shift or collide variable data.
- Bitvectors/bitlists: Bits are packed into whole bytes. Unused padding bits in the final byte must be zero; non-zero padding makes the encoding invalid and produces a semantic mismatch when re-encoded.

## Bucket templates

The same template is applied to every field of a given type. All buckets are closed intervals and are chosen to be as even as possible.

- Unsigned integers (width `MAX`): 10 buckets. `B0={0}`, `B1={1}`, `B2..B9` are eight equal-width slices of `[2, MAX]` with width `w = ceil((MAX-1)/8)`. Bucket `Bk` covers `[2 + (k-2)·w , min(1 + k·w, MAX)]` for `k ∈ {2..9}`. Example for `uint8`: `B0=0`, `B1=1`, `B2=2–31`, `B3=32–63`, `B4=64–95`, `B5=96–127`, `B6=128–159`, `B7=160–191`, `B8=192–223`, `B9=224–255`.
- Booleans: three buckets: `false (0x00)`, `true (0x01)`, `dirty (0x02–0xFF)` to expose illegal encodings.
- Byte values (used for roots, byte lists, and bitvector/bitlist elements): 12 equal slices over `0x00–0xFF`. Example ranges: `B0 0x00–0x15`, `B1 0x16–0x2B`, `B2 0x2C–0x41`, `B3 0x42–0x57`, `B4 0x58–0x6D`, `B5 0x6E–0x83`, `B6 0x84–0x99`, `B7 0x9A–0xAF`, `B8 0xB0–0xC5`, `B9 0xC6–0xDB`, `B10 0xDC–0xF1`, `B11 0xF2–0xFF`. High buckets naturally inject dirty padding.
- Lengths for lists/bitlists (max `L`): six buckets. `B0={0}`, `B1={1}`, `B2..B5` split `[2, L]` into four equal slices with width `w = ceil((L-1)/4)`; each bucket `Bi` covers `[2 + (i-2)·w , min(1 + i·w, L)]` for `i ∈ {2..5}`.
- Offsets (for variable parts): four buckets. `B0={0}`, `B1..B3` split `[1, Omax]` into three equal slices using `w = ceil(Omax/3)`; bucket `Bi` covers `[1 + (i-1)·w , min(i·w, Omax)]` for `i ∈ {1..3}`. Here `Omax` is the largest legal offset in bytes for that container: fixed-section size plus the maximum total size of all its variable payloads. These buckets apply to the offset words that sit in the fixed section and point to the start of each variable payload.
- Containers: assign one bucket per scalar field. For lists/byte lists, also pick a length bucket and an offset bucket because they live in the variable section and are referenced from the fixed section via offsets. Fixed-size vectors are fully inlined in the fixed section, so they skip length and offset buckets.
- Byte strings and roots: apply the byte-value buckets independently to each byte; no aggregate bucket is used.

## BeaconState schema

```go
type BeaconState struct {
    GenesisTime                 uint64
    GenesisValidatorsRoot       [32]byte
    Slot                        uint64
    Fork                        Fork
    LatestBlockHeader           BeaconBlockHeader
    BlockRoots                  [][32]byte `ssz-size:"4"`
    StateRoots                  [][32]byte `ssz-size:"4"`
    HistoricalRoots             [][32]byte `ssz-max:"4"`
    Eth1Data                    Eth1Data
    Eth1DataVotes               []Eth1Data `ssz-max:"4"`
    Eth1DepositIndex            uint64
    Validators                  []Validator          `ssz-max:"4"`
    Balances                    []uint64             `ssz-max:"4"`
    RandaoMixes                 [][32]byte           `ssz-size:"4"`
    Slashings                   []uint64             `ssz-size:"4"`
    PreviousEpochAttestations   []PendingAttestation `ssz-max:"4"`
    CurrentEpochAttestations    []PendingAttestation `ssz-max:"4"`
    JustificationBits           [1]byte              `ssz-size:"1"` // Bitvector[4]
    PreviousJustifiedCheckpoint Checkpoint
    CurrentJustifiedCheckpoint  Checkpoint
    FinalizedCheckpoint         Checkpoint
}
```

This variant keeps the structure of the consensus `BeaconState` but reduces list bounds for faster experimentation.

Type widths in this schema:
- `Slot`, `Epoch`, `Gwei`, `ValidatorIndex`, and all other counters are `uint64` (`MAX = 2^64-1`).
- `Root` is `[32]byte`.
- `JustificationBits` is `Bitvector[4]` stored as a single byte with 4 data bits and 4 padding bits.

## Bucket assignment for each field

- Time and index scalars: `GenesisTime`, `Slot`, `Eth1DepositIndex`, all `Fork`/`Checkpoint` epochs, `AttestationData` slots and indices, `InclusionDelay`, `ProposerIndex`, and validator epoch fields use the unsigned-integer template with `MAX = 2^64-1`.
- Roots and hashes: `GenesisValidatorsRoot`, `BeaconBlockHeader` roots, `BlockRoots`, `StateRoots`, `HistoricalRoots`, `Eth1Data.DepositRoot`, `Eth1Data.BlockHash`, `RandaoMixes`, and the roots inside checkpoints use the byte-value template per byte. `BlockRoots`, `StateRoots`, and `RandaoMixes` have fixed length 4 (no length bucket); `HistoricalRoots` also gets a length bucket.
- Checkpoints: `Epoch` uses the `uint64` bucket template; `Root` uses the byte-value template.
- Beacon block headers: `Slot` and `ProposerIndex` use the `uint64` template; all roots use byte buckets.
- Attestation data: `Slot` and `Index` use the `uint64` template; `BeaconBlockRoot` uses byte buckets; `Source` and `Target` are checkpoints and inherit their epoch/root buckets.
- Eth1 data and votes: `Eth1Data.DepositCount` uses the unsigned-integer template. `Eth1DataVotes` has a length bucket plus per-element `Eth1Data` buckets.
- Validators: the list length uses the length template. Each validator uses byte buckets for `Pubkey` and `WithdrawalCredentials`, an unsigned-integer bucket for `EffectiveBalance` and epoch fields, and the boolean bucket for `Slashed`.
- Balances and slashings: `Balances` length uses the length template and each balance uses the unsigned-integer bucket. `Slashings` is a fixed-size vector of 4 balances, so only the per-element unsigned-integer buckets apply.
- Attestations: both attestation lists use a length bucket. `AggregationBits` (byte list) uses a length bucket plus byte buckets for each byte. `AttestationData` fields follow the scalar/root rules above.
- Justification bits (Bitvector[4]): one byte governed by the byte-value template. Only the lower 4 bits are meaningful; upper 4 bits are padding that must be zero.

## Bug focus: dirty padding in bitvectors

- SSZ rule: for bitvectors the unused padding bits in the final byte must be zero. For `Bitvector[4]`, bits 4–7 of the single byte are padding.
- Bucket-driven trigger: choose the highest byte bucket (e.g., `B11: 0xF2–0xFF`) for `JustificationBits`. A sample value `0b11110001` sets a legitimate low nibble (`0001`) but leaves all padding bits high (`1111`).
- Failure manifestation: the decoder accepts the byte, but a re-encoding zeroes the padding to `0b00010001`, so the two encodings hash differently. The mismatch is reported as a semantic mismatch bug on `JustificationBits`.

This bucketed view turns padding bugs into targeted choices: every field always picks exactly one bucket, the buckets are balanced, and high-byte buckets naturally surface dirty padding without any special-case logic.