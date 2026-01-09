package schemas

// Bitvector4 is a custom type to trigger fastssz Bitvector logic.
// It is an array to ensure proper memory layout for fixed-size unmarshaling.
type Bitvector4 [1]byte

// BitvectorStruct corresponds to Bitvector[4].
type BitvectorStruct struct {
	// Bitvector[4]
	ValidationBits Bitvector4 `ssz-size:"1"`
}

// BooleanStruct for Dirty Boolean vulnerability.
type BooleanStruct struct {
	Val bool
}

// BitlistStruct for Null-Bitlist vulnerability.
type BitlistStruct struct {
	Bits []byte `ssz:"bitlist" ssz-max:"2048"`
}

// GapStruct for Variable-Length Container Gap vulnerability.
// It has a fixed part (offset) and a variable part.
type GapStruct struct {
	// List of bytes, max size 1024.
	Data []byte `ssz-max:"1024"`
}
