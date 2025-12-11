package feedback

// RuntimeSignature is a compact representation of the client's internal behavior.
// It synthesizes key events from the raw fuzzer output.
type RuntimeSignature struct {
	RoundtripSuccessCount int // Number of inputs that passed without error
	NonBugErrorCount      int // Number of inputs that failed with non-bug errors (e.g., malformed input)
	BugFoundCount         int // Number of inputs that triggered the specific bug
	// BugKinds counts how many times each bug category was observed (e.g., "SemanticMismatch", "RoundTripMismatch", "Panic").
	BugKinds map[string]int
	// Future: Could include hashes of coverage maps, specific branch hit counts,
	// or other distilled metrics for a richer signature for KL divergence.
}

// NewRuntimeSignature initializes a RuntimeSignature with a non-nil BugKinds map.
func NewRuntimeSignature() RuntimeSignature {
	return RuntimeSignature{
		BugKinds: make(map[string]int),
	}
}
