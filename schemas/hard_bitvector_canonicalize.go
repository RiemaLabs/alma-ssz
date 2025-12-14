package schemas

import ssz "github.com/ferranbt/fastssz"

// Canonicalize returns a canonical version of HardBitvectorStruct.
// It clears unused padding bits in the Bitvector[4] target field.
// This enables semantic bug detection via the round-trip/hash oracle.
func (h *HardBitvectorStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := &HardBitvectorStruct{}
	*canonical = *h
	canonical.Target[0] &= 0x0F
	return canonical, nil
}

