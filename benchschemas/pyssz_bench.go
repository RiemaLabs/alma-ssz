package benchschemas

// Minimal, spec-derived schemas for cross-language (py-ssz) benchmarks.

type PSSZBoolBench struct {
	Slashed bool
	Epoch   Slot
	Root    Root
}

type PSSZBitvectorBench struct {
	Slot              Slot
	Root              Root
	JustificationBits Bitvector4 `ssz-size:"1"`
}

type PSSZBitlistBench struct {
	AggregationBits []byte `ssz:"bitlist" ssz-max:"2048"`
	Slot            Slot
}

type PSSZByteListBench struct {
	Data []byte `ssz-max:"31"`
}

type PSSZGapBench struct {
	FieldA []Root `ssz-max:"4"`
	FieldB []Root `ssz-max:"4"`
}

type PSSZTailBench struct {
	Slot Slot
}

type PSSZHTRListBench struct {
	Balances []Gwei `ssz-max:"128"`
}

type PSSZHeaderListBench struct {
	Headers []BeaconBlockHeader `ssz-max:"4"`
}
