package benchschemas

import "github.com/ferranbt/fastssz/tracer"

import (
	"encoding/binary"
	"fmt"

	ssz "github.com/ferranbt/fastssz"
)

type DebugUnion struct {
	Selector byte
	Value    uint64
}

func (u *DebugUnion) MarshalSSZ() ([]byte, error) {
	sel := u.Selector & 1
	tracer.Record(12332355038645934268, tracer.ToScalar(sel))
	u.Selector = sel
	switch sel {
	case 0:
		return []byte{0}, nil
	case 1:
		buf := make([]byte, 1+8)
		tracer.Record(4550058912360985996, tracer.ToScalar(buf))
		buf[0] = 1
		binary.LittleEndian.PutUint64(buf[1:], u.Value)
		return buf, nil
	default:
		return nil, fmt.Errorf("invalid selector %d", u.Selector)
	}
}

func (u *DebugUnion) MarshalSSZTo(dst []byte) ([]byte, error) {
	serialized, err := u.MarshalSSZ()
	tracer.Record(9345820295806117970, tracer.ToScalar(err))
	tracer.Record(16959349194972922941, tracer.ToScalar(serialized))
	if err != nil {
		return dst, err
	}
	dst = append(dst, serialized...)
	tracer.Record(1938884375001766421, tracer.ToScalar(dst))
	return dst, nil
}

func (u *DebugUnion) SizeSSZ() int {
	if (u.Selector & 1) == 1 {
		return 9
	}
	return 1
}

func (u *DebugUnion) UnmarshalSSZ(buf []byte) error {
	if len(buf) < 1 {
		return ssz.ErrSize
	}
	u.Selector = buf[0]
	switch u.Selector {
	case 0:
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
	return []byte{}, nil
}

func (u *DebugUnion) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(u)
}

func (u *DebugUnion) HashTreeRootWith(hh ssz.HashWalker) error {
	indx := hh.Index()
	tracer.Record(2562346997231854237, tracer.ToScalar(indx))
	hh.PutUint8(u.Selector)
	hh.PutUint64(u.Value)
	hh.Merkleize(indx)
	return nil
}

func (u *DebugUnion) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(u)
}

type UnionBench struct {
	Slot          Slot
	ProposerIndex ValidatorIndex
	StateRoot     Root
	Padding       [512]byte
	Payload       DebugUnion
}

func (u *UnionBench) MarshalSSZ() ([]byte, error) {
	return marshalUnionBench(u.Slot, u.ProposerIndex, u.StateRoot[:], u.Padding[:], &u.Payload)
}

func (u *UnionBench) MarshalSSZTo(dst []byte) ([]byte, error) {
	payloadBytes, err := u.Payload.MarshalSSZ()
	tracer.Record(9345820295806117970, tracer.ToScalar(err))
	tracer.Record(10111477981541014458, tracer.ToScalar(payloadBytes))
	if err != nil {
		return dst, err
	}
	dst = ssz.MarshalValue(dst, uint64(u.Slot))
	tracer.Record(1938884375001766421, tracer.ToScalar(dst))
	dst = ssz.MarshalValue(dst, uint64(u.ProposerIndex))
	tracer.Record(1938884375001766421, tracer.ToScalar(dst))
	dst = append(dst, u.StateRoot[:]...)
	tracer.Record(1938884375001766421, tracer.ToScalar(dst))
	dst = append(dst, u.Padding[:]...)
	tracer.Record(1938884375001766421, tracer.ToScalar(dst))
	dst = append(dst, payloadBytes...)
	tracer.Record(1938884375001766421, tracer.ToScalar(dst))
	return dst, nil
}

func (u *UnionBench) SizeSSZ() int {
	return 8 + 8 + 32 + len(u.Padding) + u.Payload.SizeSSZ()
}

func (u *UnionBench) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(u)
}

func (u *UnionBench) UnmarshalSSZ(buf []byte) error {
	return unmarshalUnionBench(buf, &u.Slot, &u.ProposerIndex, u.StateRoot[:], u.Padding[:], &u.Payload)
}

func (u *UnionBench) UnmarshalSSZTail(buf []byte) ([]byte, error) {
	if err := u.UnmarshalSSZ(buf); err != nil {
		return nil, err
	}
	return []byte{}, nil
}

func (u *UnionBench) HashTreeRootWith(hh ssz.HashWalker) error {
	indx := hh.Index()
	tracer.Record(2562346997231854237, tracer.ToScalar(indx))
	hh.PutUint64(uint64(u.Slot))
	hh.PutUint64(uint64(u.ProposerIndex))
	hh.PutBytes(u.StateRoot[:])
	hh.PutBytes(u.Padding[:])
	if err := u.Payload.HashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}

func (u *UnionBench) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(u)
}

func marshalUnionBench(slot Slot, proposer ValidatorIndex, stateRoot []byte, padding []byte, payload *DebugUnion) ([]byte, error) {
	payloadBytes, err := payload.MarshalSSZ()
	tracer.Record(12157575802740089018, tracer.ToScalar(err))
	tracer.Record(8702588118236965554, tracer.ToScalar(payloadBytes))
	if err != nil {
		return nil, err
	}
	dst := make([]byte, 0, 8+8+len(stateRoot)+len(padding)+len(payloadBytes))
	tracer.Record(4781320654403282013, tracer.ToScalar(dst))
	dst = ssz.MarshalValue(dst, uint64(slot))
	tracer.Record(4781320654403282013, tracer.ToScalar(dst))
	dst = ssz.MarshalValue(dst, uint64(proposer))
	tracer.Record(4781320654403282013, tracer.ToScalar(dst))
	dst = append(dst, stateRoot...)
	tracer.Record(4781320654403282013, tracer.ToScalar(dst))
	dst = append(dst, padding...)
	tracer.Record(4781320654403282013, tracer.ToScalar(dst))
	dst = append(dst, payloadBytes...)
	tracer.Record(4781320654403282013, tracer.ToScalar(dst))
	return dst, nil
}

func unmarshalUnionBench(buf []byte, slot *Slot, proposer *ValidatorIndex, stateRoot []byte, padding []byte, payload *DebugUnion) error {
	fixedSize := 8 + 8 + len(stateRoot) + len(padding)
	tracer.Record(7343856561129173375, tracer.ToScalar(fixedSize))
	if len(buf) < fixedSize+1 {
		return ssz.ErrSize
	}
	*slot = Slot(binary.LittleEndian.Uint64(buf[:8]))
	*proposer = ValidatorIndex(binary.LittleEndian.Uint64(buf[8:16]))
	copy(stateRoot, buf[16:16+len(stateRoot)])
	paddingStart := 16 + len(stateRoot)
	tracer.Record(9930224050478120878, tracer.ToScalar(paddingStart))
	copy(padding, buf[paddingStart:paddingStart+len(padding)])
	return payload.UnmarshalSSZ(buf[fixedSize:])
}
