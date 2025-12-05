package schemas

// HardBitvector4 corresponds to Bitvector[4] in the hard case.
type HardBitvectorStruct struct {
	Magic uint64 // Must match specific value to trigger bug check
	// Massive padding to force large input generation and offset arithmetic
	Padding [4096]byte
	F0      uint64
	F1      []byte `ssz-max:"2048"` // Requires valid Offset 1
	// Nested container
	F2     HardNestedContainer
	F3     []byte     `ssz-max:"2048"` // Requires valid Offset 3
	Target Bitvector4 `ssz-size:"1"`
}

type HardNestedContainer struct {
	// Inner padding
	InnerPadding [1024]byte
	A            []uint64 `ssz-max:"128"`
	B            []byte   `ssz-max:"1024"`
	C            uint64
}

// HardBooleanStruct wraps the vulnerable Boolean.
type HardBooleanStruct struct {
	Magic uint64
	// Large fixed buffer to consume entropy
	LargeBuffer [8192]byte
	Name        []byte `ssz-max:"256"` // Variable
	Age         uint64
	// Nested fixed size array to add "distance"
	Meta   []uint64 `ssz-size:"4"`
	Target bool
}

// HardGapStruct wraps the Gap vulnerability.
type HardGapStruct struct {
	Magic   uint64
	Padding [4096]byte
	Header  uint64
	// Many Variable fields to create many offset boundaries
	Payload1 []byte `ssz-max:"1024"`
	Payload2 []byte `ssz-max:"1024"`
	Payload3 []byte `ssz-max:"1024"`
	Payload4 []byte `ssz-max:"1024"`
	Payload5 []byte `ssz-max:"1024"`
}
