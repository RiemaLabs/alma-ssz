package schemas

import ssz "github.com/ferranbt/fastssz"

// AggregationBitsContainer models a consensus Bitlist field (e.g., Attestation/PendingAttestation aggregation bits).
// In Ethereum consensus, this is a Bitlist[2048], where an empty list must serialize as 0x01.
//
// We declare the field as an SSZ bitlist (via ssz:"bitlist") so that the target
// implementation performs bitlist validation. This lets the fuzzer surface the null-bitlist
// trap when the implementation is patched to skip sentinel-bit checks.
type AggregationBitsContainer struct {
	AggregationBits []byte `ssz:"bitlist" ssz-max:"2048"`
}

const aggregationBitsMax = 2048

func (a *AggregationBitsContainer) SizeSSZ() int {
	bitsLen := len(a.AggregationBits)
	// Canonical empty bitlist still occupies one byte (sentinel).
	if bitsLen == 0 {
		bitsLen = 1
	}
	return 4 + bitsLen
}

// MarshalSSZ serializes the container with a single variable Bitlist field.
func (a *AggregationBitsContainer) MarshalSSZ() ([]byte, error) {
	bits := a.AggregationBits
	if len(bits) == 0 {
		bits = []byte{0x01}
	}
	dst := make([]byte, 0, 4+len(bits))
	// Offset (0) AggregationBits
	dst = ssz.WriteOffset(dst, 4)
	// Tail (0) AggregationBits
	dst = append(dst, bits...)
	return dst, nil
}

func (a *AggregationBitsContainer) MarshalSSZTo(dst []byte) ([]byte, error) {
	serialized, err := a.MarshalSSZ()
	if err != nil {
		return dst, err
	}
	dst = append(dst, serialized...)
	return dst, nil
}

func (a *AggregationBitsContainer) UnmarshalSSZ(buf []byte) error {
	if len(buf) < 4 {
		return ssz.ErrSize
	}
	o0, _ := ssz.ReadOffset(buf)
	if o0 != 4 || int(o0) > len(buf) {
		return ssz.ErrOffset
	}
	tail := buf[o0:]
	b, err := ssz.UnmarshalBitList(a.AggregationBits[:0], tail, aggregationBitsMax)
	if err != nil {
		return err
	}
	a.AggregationBits = b
	return nil
}

func (a *AggregationBitsContainer) UnmarshalSSZTail(buf []byte) ([]byte, error) {
	if err := a.UnmarshalSSZ(buf); err != nil {
		return nil, err
	}
	return []byte{}, nil
}

func (a *AggregationBitsContainer) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(a)
}

func (a *AggregationBitsContainer) HashTreeRootWith(hh ssz.HashWalker) error {
	indx := hh.Index()
	if len(a.AggregationBits) == 0 {
		return ssz.ErrEmptyBitlist
	}
	hh.PutBitlist(a.AggregationBits, aggregationBitsMax)
	hh.Merkleize(indx)
	return nil
}

func (a *AggregationBitsContainer) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(a)
}

// Canonicalize returns a canonical form with a valid sentinel bit.
// If the sentinel is missing (last byte zero), we treat the value as an empty bitlist.
func (a *AggregationBitsContainer) Canonicalize() (ssz.Marshaler, error) {
	canonical := &AggregationBitsContainer{}
	if len(a.AggregationBits) == 0 {
		canonical.AggregationBits = []byte{0x01}
		return canonical, nil
	}
	canonical.AggregationBits = make([]byte, len(a.AggregationBits))
	copy(canonical.AggregationBits, a.AggregationBits)
	if canonical.AggregationBits[len(canonical.AggregationBits)-1] == 0 {
		canonical.AggregationBits = []byte{0x01}
	}
	return canonical, nil
}
