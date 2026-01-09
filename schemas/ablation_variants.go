package schemas

// Ablation-only schema variants to expand fuzzable boundary benchmarks.

// BitvectorPairStruct exposes two Bitvector[4] fields at the top level.
type BitvectorPairStruct struct {
	Magic   uint64
	BitsA   Bitvector4 `ssz-size:"1"`
	BitsB   Bitvector4 `ssz-size:"1"`
	Padding [128]byte
}

// BitvectorWideStruct increases search space with large padding.
type BitvectorWideStruct struct {
	Magic   uint64
	Padding [4096]byte
	BitsA   Bitvector4 `ssz-size:"1"`
	BitsB   Bitvector4 `ssz-size:"1"`
}

// BitvectorOffsetStruct mixes bitvectors with variable-length payloads.
type BitvectorOffsetStruct struct {
	Magic    uint64
	BitsA    Bitvector4 `ssz-size:"1"`
	PayloadA []byte     `ssz-max:"512"`
	BitsB    Bitvector4 `ssz-size:"1"`
	PayloadB []byte     `ssz-max:"512"`
}

// BitvectorScatterStruct spaces multiple bitvectors across large padding.
type BitvectorScatterStruct struct {
	Magic    uint64
	BitsA    Bitvector4 `ssz-size:"1"`
	PaddingA [256]byte
	BitsB    Bitvector4 `ssz-size:"1"`
	PaddingB [512]byte
	BitsC    Bitvector4 `ssz-size:"1"`
}

// BooleanPairStruct exposes two boolean fields at the top level.
type BooleanPairStruct struct {
	Magic   uint64
	FlagA   bool
	FlagB   bool
	Padding [128]byte
}

// BooleanWideStruct increases search space with large padding.
type BooleanWideStruct struct {
	Magic   uint64
	Padding [8192]byte
	FlagA   bool
	FlagB   bool
}

// BooleanOffsetStruct mixes boolean flags with variable-length payloads.
type BooleanOffsetStruct struct {
	Magic    uint64
	FlagA    bool
	PayloadA []byte `ssz-max:"256"`
	FlagB    bool
	PayloadB []byte `ssz-max:"256"`
	FlagC    bool
}

// BooleanScatterStruct spaces boolean flags across padding.
type BooleanScatterStruct struct {
	Magic    uint64
	FlagA    bool
	PaddingA [128]byte
	FlagB    bool
	PaddingB [256]byte
	FlagC    bool
}

// BitlistPairStruct exposes two Bitlist[2048] fields at the top level.
type BitlistPairStruct struct {
	BitsA []byte `ssz:"bitlist" ssz-max:"2048"`
	BitsB []byte `ssz:"bitlist" ssz-max:"2048"`
}

// BitlistWideStruct adds padding around two Bitlist[2048] fields.
type BitlistWideStruct struct {
	Magic   uint64
	Padding [2048]byte
	BitsA   []byte `ssz:"bitlist" ssz-max:"2048"`
	BitsB   []byte `ssz:"bitlist" ssz-max:"2048"`
}

// BitlistTriStruct adds a third bitlist to increase boundary interactions.
type BitlistTriStruct struct {
	BitsA []byte `ssz:"bitlist" ssz-max:"2048"`
	BitsB []byte `ssz:"bitlist" ssz-max:"2048"`
	BitsC []byte `ssz:"bitlist" ssz-max:"2048"`
}

// BitlistOffsetStruct interleaves bitlists with another variable-length field.
type BitlistOffsetStruct struct {
	Magic   uint64
	BitsA   []byte `ssz:"bitlist" ssz-max:"1024"`
	Payload []byte `ssz-max:"512"`
	BitsB   []byte `ssz:"bitlist" ssz-max:"1024"`
}

// GapPairStruct exposes two variable-length slices for offset-gap fuzzing.
type GapPairStruct struct {
	Magic    uint64
	PayloadA []byte `ssz-max:"1024"`
	PayloadB []byte `ssz-max:"1024"`
}

// GapWideStruct increases search space with padding and multiple slices.
type GapWideStruct struct {
	Magic    uint64
	Padding  [2048]byte
	PayloadA []byte `ssz-max:"512"`
	PayloadB []byte `ssz-max:"512"`
	PayloadC []byte `ssz-max:"512"`
}

// GapTriStruct adds a third payload without extra padding.
type GapTriStruct struct {
	Magic    uint64
	PayloadA []byte `ssz-max:"512"`
	PayloadB []byte `ssz-max:"512"`
	PayloadC []byte `ssz-max:"512"`
}

// GapScatterStruct interleaves fixed fields with multiple payload slices.
type GapScatterStruct struct {
	Magic    uint64
	PayloadA []byte `ssz-max:"128"`
	Stamp    uint32
	PayloadB []byte `ssz-max:"256"`
	Version  uint16
	PayloadC []byte `ssz-max:"512"`
	PayloadD []byte `ssz-max:"128"`
}
