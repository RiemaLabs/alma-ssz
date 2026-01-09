package benchschemas

type Root = [32]byte
type Slot = uint64
type ValidatorIndex = uint64
type Gwei = uint64

type Bitvector4 [1]byte

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
	AggregationBits []byte `ssz:"bitlist" ssz-max:"2048"`
	Data            AttestationData
	InclusionDelay  Slot
	ProposerIndex   ValidatorIndex
}

type BeaconStateBench struct {
	GenesisTime                 uint64
	GenesisValidatorsRoot       Root
	Slot                        Slot
	Fork                        Fork
	LatestBlockHeader           BeaconBlockHeader
	BlockRoots                  [][32]byte `ssz-size:"64"`
	StateRoots                  [][32]byte `ssz-size:"64"`
	HistoricalRoots             [][32]byte `ssz-max:"64"`
	Eth1Data                    Eth1Data
	Eth1DataVotes               []Eth1Data `ssz-max:"128"`
	Eth1DepositIndex            uint64
	Validators                  []Validator          `ssz-max:"128"`
	Balances                    []Gwei               `ssz-max:"128"`
	RandaoMixes                 [][32]byte           `ssz-size:"64"`
	Slashings                   []Gwei               `ssz-size:"64"`
	PreviousEpochAttestations   []PendingAttestation `ssz-max:"64"`
	CurrentEpochAttestations    []PendingAttestation `ssz-max:"64"`
	JustificationBits           Bitvector4           `ssz-size:"1"`
	PreviousJustifiedCheckpoint Checkpoint
	CurrentJustifiedCheckpoint  Checkpoint
	FinalizedCheckpoint         Checkpoint
}

type ValidatorEnvelope struct {
	Slashed                    bool
	Pubkey                     [48]byte
	WithdrawalCredentials      [32]byte
	EffectiveBalance           Gwei
	ActivationEligibilityEpoch uint64
	ActivationEpoch            uint64
	ExitEpoch                  uint64
	WithdrawableEpoch          uint64
	Balances                   []Gwei               `ssz-max:"1024"`
	RandaoMixes                [][32]byte           `ssz-size:"64"`
	Slashings                  []Gwei               `ssz-size:"64"`
	Eth1DataVotes              []Eth1Data           `ssz-max:"128"`
	PendingAttestations        []PendingAttestation `ssz-max:"64"`
	SyncCommitteeRoots         []Root               `ssz-size:"64"`
}

type AttestationEnvelope struct {
	AggregationBits  []byte `ssz:"bitlist" ssz-max:"2048"`
	Data             AttestationData
	Signature        [96]byte
	CommitteeIndex   uint64
	AttestingIndices []ValidatorIndex `ssz-max:"2048"`
	CustodyBits      []byte           `ssz:"bitlist" ssz-max:"2048"`
}

type SignedBeaconBlockHeader struct {
	Message   BeaconBlockHeader
	Signature [96]byte
}

type ProposerSlashing struct {
	Header1 SignedBeaconBlockHeader
	Header2 SignedBeaconBlockHeader
}

type SignedAttestationData struct {
	Data      AttestationData
	Signature [96]byte
}

type AttesterSlashing struct {
	Att1 SignedAttestationData
	Att2 SignedAttestationData
}

type DepositData struct {
	Pubkey                [48]byte
	WithdrawalCredentials [32]byte
	Amount                Gwei
	Signature             [96]byte
}

type Deposit struct {
	Proof [4][32]byte
	Data  DepositData
}

type VoluntaryExit struct {
	Epoch          uint64
	ValidatorIndex ValidatorIndex
}

type SignedVoluntaryExit struct {
	Message   VoluntaryExit
	Signature [96]byte
}

type BlockBodyBench struct {
	RandaoReveal      [96]byte
	Eth1Data          Eth1Data
	Graffiti          [32]byte
	ProposerSlashings []ProposerSlashing    `ssz-max:"128"`
	AttesterSlashings []AttesterSlashing    `ssz-max:"128"`
	Attestations      []AttestationEnvelope `ssz-max:"128"`
	Deposits          []Deposit             `ssz-max:"128"`
	VoluntaryExits    []SignedVoluntaryExit `ssz-max:"128"`
}
