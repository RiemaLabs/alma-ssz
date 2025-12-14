package schemas

import ssz "github.com/ferranbt/fastssz"

// Canonicalize returns a canonicalized BeaconState for semantic-bug detection.
// In Phase0, JustificationBits is Bitvector[4] stored in one byte, where the
// high 4 bits are padding and must be zero in canonical encodings.
func (b *BeaconState) Canonicalize() (ssz.Marshaler, error) {
	canonical := &BeaconState{}
	*canonical = *b
	canonical.JustificationBits[0] &= 0x0F
	return canonical, nil
}

