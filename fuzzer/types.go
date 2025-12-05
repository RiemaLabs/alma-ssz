package fuzzer

import ssz "github.com/ferranbt/fastssz" // Explicitly alias

// Canonicalizer is an optional interface that schemas can implement
// to provide a canonical representation of themselves.
// This is used by the fuzzer to detect semantic bugs where a non-canonical
// but accepted encoding leads to a different hash tree root.
type Canonicalizer interface {
	Canonicalize() (ssz.Marshaler, error)
}
