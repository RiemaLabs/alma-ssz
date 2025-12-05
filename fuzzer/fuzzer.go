package fuzzer

import (
	"alma.local/ssz/feedback" // New import for RuntimeSignature
	"alma.local/ssz/internal/analyzer" // New import for TraceEntry
)

// Fuzzer defines the interface for a component that executes fuzzing inputs
// and provides feedback relevant to an RL agent (bug triggers, coverage).
type Fuzzer interface {
	// Execute takes SSZ bytes, runs them against the target, and returns
	// a compact RuntimeSignature, whether a bug was triggered, if new coverage was found,
	// and the raw trace entries.
	Execute(sszBytes []byte) (feedback.RuntimeSignature, bool, bool, []analyzer.TraceEntry)
	
	// Reset initializes the fuzzer's internal state (e.g., coverage counters).
	Reset()

	// TotalCoverage returns the cumulative coverage achieved by the fuzzer.
	TotalCoverage() float64

	// NewCoverage returns the coverage newly found in the last execution.
	NewCoverage() float64
}

