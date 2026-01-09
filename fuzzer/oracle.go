package fuzzer

// ExternalDecodeResult captures decoded canonical bytes and hash root.
type ExternalDecodeResult struct {
	Canonical []byte
	Root      [32]byte
}

// ExternalOracle provides cross-language decode/encode/hash support.
type ExternalOracle interface {
	Decode(schema string, data []byte) (ExternalDecodeResult, error)
	Close() error
}
