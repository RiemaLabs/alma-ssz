package schemas

import (
	ssz "github.com/ferranbt/fastssz"
)

// Canonicalize returns a canonicalized PendingAttestation for semantic bug detection.
//
// Ethereum consensus defines AggregationBits as Bitlist[2048], whose encoding must
// contain a sentinel (termination) bit. In particular, an empty bitlist must
// serialize as 0x01, and the last byte must never be 0x00.
//
// This canonicalizer normalizes missing-sentinel encodings to the canonical empty
// encoding, enabling the hash oracle to detect "null-bitlist" acceptance bugs.
func (p *PendingAttestation) Canonicalize() (ssz.Marshaler, error) {
	canonical := &PendingAttestation{}
	*canonical = *p

	if len(canonical.AggregationBits) == 0 {
		canonical.AggregationBits = []byte{0x01}
		return canonical, nil
	}

	if canonical.AggregationBits[len(canonical.AggregationBits)-1] == 0 {
		canonical.AggregationBits = []byte{0x01}
	}
	return canonical, nil
}

