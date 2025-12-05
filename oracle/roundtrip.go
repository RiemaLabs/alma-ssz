package oracle

import (
	"bytes"
	"errors"
	"fmt"

	ssz "github.com/ferranbt/fastssz"
)

// ErrInvalidInput signals that the SSZ payload failed to decode.
var ErrInvalidInput = errors.New("oracle: invalid input")

// RoundTripTarget constrains SSZ structs usable by the oracle.
type RoundTripTarget[T any] interface {
	*T
	ssz.Marshaler
	UnmarshalSSZ([]byte) error
}

// RoundTrip enforces Encode(Decode(x)) == x for SSZ types that implement fastssz.
func RoundTrip[T any, PT RoundTripTarget[T]](data []byte) error {
	var obj PT = PT(new(T))

	if err := obj.UnmarshalSSZ(data); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	out, err := obj.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("oracle: marshal failed: %w", err)
	}

	if !bytes.Equal(out, data) {
		return fmt.Errorf("oracle: bug triggered! non-canonical roundtrip (dirty padding?) (input=%d output=%d)", len(data), len(out))
	}
	return nil
}
