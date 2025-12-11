package schemas

import (
	"encoding/binary"
	"fmt"

	ssz "github.com/ferranbt/fastssz"
)

// DebugUnion models a minimal union with two variants:
// Selector 0 => none (no payload). Selector 1 => uint64 payload.
// Bug: selector 0 silently accepts trailing bytes instead of rejecting them.
type DebugUnion struct {
	Selector byte
	Value    uint64
}

// MarshalSSZ serializes the union in canonical form.
func (u *DebugUnion) MarshalSSZ() ([]byte, error) {
	sel := u.Selector & 1 // clamp to two supported variants
	u.Selector = sel
	switch sel {
	case 0:
		return []byte{0}, nil
	case 1:
		buf := make([]byte, 1+8)
		buf[0] = 1
		binary.LittleEndian.PutUint64(buf[1:], u.Value)
		return buf, nil
	default:
		return nil, fmt.Errorf("invalid selector %d", u.Selector)
	}
}

func (u *DebugUnion) MarshalSSZTo(dst []byte) ([]byte, error) {
	serialized, err := u.MarshalSSZ()
	if err != nil {
		return dst, err
	}
	dst = append(dst, serialized...)
	return dst, nil
}

// SizeSSZ reports the serialized size of the union.
func (u *DebugUnion) SizeSSZ() int {
	if (u.Selector & 1) == 1 {
		return 9
	}
	// Selector 0 or invalid -> single selector byte
	return 1
}

// UnmarshalSSZ parses the union. For selector 0, it BUGGILY ignores any trailing data.
func (u *DebugUnion) UnmarshalSSZ(buf []byte) error {
	if len(buf) < 1 {
		return ssz.ErrSize
	}
	u.Selector = buf[0]
	switch u.Selector {
	case 0:
		// BUG: accept and discard any trailing payload for the None variant.
		return nil
	case 1:
		if len(buf) < 1+8 {
			return ssz.ErrSize
		}
		u.Value = binary.LittleEndian.Uint64(buf[1:])
		return nil
	default:
		return fmt.Errorf("invalid selector %d", u.Selector)
	}
}

func (u *DebugUnion) UnmarshalSSZTail(buf []byte) ([]byte, error) {
	if err := u.UnmarshalSSZ(buf); err != nil {
		return nil, err
	}
	// Buggy behavior: consume (and discard) all remaining bytes for selector 0.
	return []byte{}, nil
}

// HashTreeRoot provides a hash for the union.
func (u *DebugUnion) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(u)
}

func (u *DebugUnion) HashTreeRootWith(hh ssz.HashWalker) error {
	indx := hh.Index()
	hh.PutUint8(u.Selector)
	hh.PutUint64(u.Value)
	hh.Merkleize(indx)
	return nil
}

func (u *DebugUnion) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(u)
}

// UnionStruct is a container embedding a union and a lightweight magic gate.
type UnionStruct struct {
	Magic   uint32
	Padding [128]byte
	Payload DebugUnion
}

// HardUnionStruct uses a larger padding region to dilute search space; gate logic is shared.
type HardUnionStruct struct {
	Magic   uint32
	Padding [1024]byte
	Payload DebugUnion
}

func (u *UnionStruct) MarshalSSZ() ([]byte, error) {
	return marshalUnionContainer(u.Magic, u.Padding[:], &u.Payload)
}

func (u *UnionStruct) MarshalSSZTo(dst []byte) ([]byte, error) {
	return marshalUnionContainerTo(dst, u.Magic, u.Padding[:], &u.Payload)
}

func (u *UnionStruct) SizeSSZ() int {
	return 4 + len(u.Padding) + u.Payload.SizeSSZ()
}

func (u *UnionStruct) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(u)
}

func (u *UnionStruct) UnmarshalSSZ(buf []byte) error {
	return unmarshalUnionContainer(buf, &u.Magic, u.Padding[:], &u.Payload)
}

func (u *UnionStruct) UnmarshalSSZTail(buf []byte) ([]byte, error) {
	if err := u.UnmarshalSSZ(buf); err != nil {
		return nil, err
	}
	return []byte{}, nil
}

func (u *UnionStruct) HashTreeRootWith(hh ssz.HashWalker) error {
	indx := hh.Index()
	hh.PutUint32(u.Magic)
	hh.PutBytes(u.Padding[:])
	if err := u.Payload.HashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}

func (u *UnionStruct) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(u)
}

func (u *HardUnionStruct) MarshalSSZ() ([]byte, error) {
	return marshalUnionContainer(u.Magic, u.Padding[:], &u.Payload)
}

func (u *HardUnionStruct) MarshalSSZTo(dst []byte) ([]byte, error) {
	return marshalUnionContainerTo(dst, u.Magic, u.Padding[:], &u.Payload)
}

func (u *HardUnionStruct) SizeSSZ() int {
	return 4 + len(u.Padding) + u.Payload.SizeSSZ()
}

func (u *HardUnionStruct) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(u)
}

func (u *HardUnionStruct) UnmarshalSSZ(buf []byte) error {
	return unmarshalUnionContainer(buf, &u.Magic, u.Padding[:], &u.Payload)
}

func (u *HardUnionStruct) UnmarshalSSZTail(buf []byte) ([]byte, error) {
	if err := u.UnmarshalSSZ(buf); err != nil {
		return nil, err
	}
	return []byte{}, nil
}

func (u *HardUnionStruct) HashTreeRootWith(hh ssz.HashWalker) error {
	indx := hh.Index()
	hh.PutUint32(u.Magic)
	hh.PutBytes(u.Padding[:])
	if err := u.Payload.HashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}

func (u *HardUnionStruct) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(u)
}

// Shared helpers for container marshal/unmarshal.

func marshalUnionContainer(magic uint32, padding []byte, payload *DebugUnion) ([]byte, error) {
	payloadBytes, err := payload.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	dst := make([]byte, 0, 4+len(padding)+len(payloadBytes))
	return marshalUnionContainerTo(dst, magic, padding, payload)
}

func marshalUnionContainerTo(dst []byte, magic uint32, padding []byte, payload *DebugUnion) ([]byte, error) {
	payloadBytes, err := payload.MarshalSSZ()
	if err != nil {
		return dst, err
	}
	dst = ssz.MarshalValue(dst, magic)
	dst = append(dst, padding...)
	dst = append(dst, payloadBytes...)
	return dst, nil
}

func unmarshalUnionContainer(buf []byte, magic *uint32, padding []byte, payload *DebugUnion) error {
	minSize := 4 + len(padding) + 1 // need at least selector byte
	if len(buf) < minSize {
		return ssz.ErrSize
	}

	var err error
	*magic, buf = ssz.UnmarshallValue[uint32](buf)
	buf = ssz.UnmarshalFixedBytes(padding, buf)

	if err = payload.UnmarshalSSZ(buf); err != nil {
		return err
	}

	// BUG is triggered when selector == 0 (None) AND trailing bytes were provided (len(buf) > 1)
	// UnmarshalSSZ already ignores the tail; the mismatch surfaces later in roundtrip.

	// Light gate disabled to avoid over-filtering inputs; keep magic as-is.
	return nil
}
