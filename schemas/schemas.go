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

// GapStruct for Variable-Length Container Gap vulnerability.
// It has a fixed part (offset) and a variable part.
type GapStruct struct {
	// List of bytes, max size 1024.
	Data []byte `ssz-max:"1024"`
}

// --- BeaconState and Dependencies ---

type Root = [32]byte
type Slot = uint64
type ValidatorIndex = uint64
type Gwei = uint64

type Fork struct {
	PreviousVersion [4]byte
	CurrentVersion  [4]byte
	Epoch           uint64
}

type BeaconBlockHeader struct {
	Slot          Slot
	ProposerIndex ValidatorIndex
	ParentRoot    Root
	StateRoot     Root
	BodyRoot      Root
}

type Eth1Data struct {
	DepositRoot  Root
	DepositCount uint64
	BlockHash    Root
}

type Checkpoint struct {
	Epoch uint64
	Root  Root
}

type Validator struct {
	Pubkey                     [48]byte
	WithdrawalCredentials      [32]byte
	EffectiveBalance           Gwei
	Slashed                    bool
	ActivationEligibilityEpoch uint64
	ActivationEpoch            uint64
	ExitEpoch                  uint64
	WithdrawableEpoch          uint64
}

type AttestationData struct {
	Slot            Slot
	Index           uint64
	BeaconBlockRoot Root
	Source          Checkpoint
	Target          Checkpoint
}

type PendingAttestation struct {
	AggregationBits []byte `ssz-max:"2048"`
	Data            AttestationData
	InclusionDelay  Slot
	ProposerIndex   ValidatorIndex
}

// BeaconState simplified for fuzzing but structurally equivalent
type BeaconState struct {
	GenesisTime                 uint64
	GenesisValidatorsRoot       Root
	Slot                        Slot
	Fork                        Fork
	LatestBlockHeader           BeaconBlockHeader
	BlockRoots                  [][32]byte `ssz-size:"4"` // Reduced size
	StateRoots                  [][32]byte `ssz-size:"4"` // Reduced size
	HistoricalRoots             [][32]byte `ssz-max:"4"`  // Reduced size
	Eth1Data                    Eth1Data
	Eth1DataVotes               []Eth1Data `ssz-max:"4"`
	Eth1DepositIndex            uint64
	Validators                  []Validator          `ssz-max:"4"`
	Balances                    []Gwei               `ssz-max:"4"`
	RandaoMixes                 [][32]byte           `ssz-size:"4"`
	Slashings                   []Gwei               `ssz-size:"4"`
	PreviousEpochAttestations   []PendingAttestation `ssz-max:"4"`
	CurrentEpochAttestations    []PendingAttestation `ssz-max:"4"`
	JustificationBits           Bitvector4           `ssz-size:"1"` // Bitvector[4]
	PreviousJustifiedCheckpoint Checkpoint
	CurrentJustifiedCheckpoint  Checkpoint
	FinalizedCheckpoint         Checkpoint
}
