package schemas

import ssz "github.com/ferranbt/fastssz"

func canonicalizeBitvector4(bits Bitvector4) Bitvector4 {
	bits[0] &= 0x0F
	return bits
}

func canonicalizeBitlistBits(bits []byte) []byte {
	if len(bits) == 0 {
		return []byte{0x01}
	}
	dst := make([]byte, len(bits))
	copy(dst, bits)
	if dst[len(dst)-1] == 0 {
		return []byte{0x01}
	}
	return dst
}

// Canonicalize normalizes bitvectors by clearing padding bits.
func (b *BitvectorPairStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.BitsA = canonicalizeBitvector4(b.BitsA)
	canonical.BitsB = canonicalizeBitvector4(b.BitsB)
	return &canonical, nil
}

// Canonicalize normalizes bitvectors by clearing padding bits.
func (b *BitvectorWideStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.BitsA = canonicalizeBitvector4(b.BitsA)
	canonical.BitsB = canonicalizeBitvector4(b.BitsB)
	return &canonical, nil
}

// Canonicalize normalizes bitvectors by clearing padding bits.
func (b *BitvectorOffsetStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.BitsA = canonicalizeBitvector4(b.BitsA)
	canonical.BitsB = canonicalizeBitvector4(b.BitsB)
	return &canonical, nil
}

// Canonicalize normalizes bitvectors by clearing padding bits.
func (b *BitvectorScatterStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.BitsA = canonicalizeBitvector4(b.BitsA)
	canonical.BitsB = canonicalizeBitvector4(b.BitsB)
	canonical.BitsC = canonicalizeBitvector4(b.BitsC)
	return &canonical, nil
}

// Canonicalize normalizes bitlists to a canonical sentinel representation.
func (b *BitlistPairStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.BitsA = canonicalizeBitlistBits(b.BitsA)
	canonical.BitsB = canonicalizeBitlistBits(b.BitsB)
	return &canonical, nil
}

// Canonicalize normalizes bitlists to a canonical sentinel representation.
func (b *BitlistWideStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.BitsA = canonicalizeBitlistBits(b.BitsA)
	canonical.BitsB = canonicalizeBitlistBits(b.BitsB)
	return &canonical, nil
}

// Canonicalize normalizes bitlists to a canonical sentinel representation.
func (b *BitlistTriStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.BitsA = canonicalizeBitlistBits(b.BitsA)
	canonical.BitsB = canonicalizeBitlistBits(b.BitsB)
	canonical.BitsC = canonicalizeBitlistBits(b.BitsC)
	return &canonical, nil
}

// Canonicalize normalizes bitlists to a canonical sentinel representation.
func (b *BitlistOffsetStruct) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.BitsA = canonicalizeBitlistBits(b.BitsA)
	canonical.BitsB = canonicalizeBitlistBits(b.BitsB)
	return &canonical, nil
}
