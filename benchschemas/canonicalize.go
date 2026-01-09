package benchschemas

import ssz "github.com/ferranbt/fastssz"

func (b *BeaconStateBench) Canonicalize() (ssz.Marshaler, error) {
	canonical := *b
	canonical.JustificationBits[0] &= 0x0F
	return &canonical, nil
}

func (a *AttestationEnvelope) Canonicalize() (ssz.Marshaler, error) {
	canonical := *a
	canonical.AggregationBits = canonicalizeBitlist(a.AggregationBits)
	canonical.CustodyBits = canonicalizeBitlist(a.CustodyBits)
	return &canonical, nil
}

func canonicalizeBitlist(bits []byte) []byte {
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
