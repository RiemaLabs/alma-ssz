package fuzzer

import (
	"fmt"
	"math/rand"
	"time"

	"alma.local/ssz/feedback"
	"alma.local/ssz/internal/analyzer"
	"github.com/ferranbt/fastssz/tracer"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// MockInstrumentedFuzzer is a mock implementation of the Fuzzer interface
// that calls into instrumented fastssz code and returns traces.
type MockInstrumentedFuzzer struct {
	currentCoverage float64
	lastNewCoverage float64
}

// NewMockInstrumentedFuzzer creates a new MockInstrumentedFuzzer.
func NewMockInstrumentedFuzzer() (*MockInstrumentedFuzzer, error) {
	// Simulate some initial coverage.
	return &MockInstrumentedFuzzer{
		currentCoverage: 0.1,
		lastNewCoverage: 0.0,
	}, nil
}

// Reset implements the Fuzzer interface.
func (mf *MockInstrumentedFuzzer) Reset() {
	mf.currentCoverage = 0.1 // Reset to initial state
	mf.lastNewCoverage = 0.0
}

// TotalCoverage implements the Fuzzer interface.
func (mf *MockInstrumentedFuzzer) TotalCoverage() float64 {
	return mf.currentCoverage
}

// NewCoverage implements the Fuzzer interface.
func (mf *MockInstrumentedFuzzer) NewCoverage() float64 {
	return mf.lastNewCoverage
}

// Execute implements the Fuzzer interface.
// It calls a simple instrumented function and captures its trace.
func (mf *MockInstrumentedFuzzer) Execute(sszBytes []byte) (
	signature feedback.RuntimeSignature,
	bugTriggered bool,
	newCoverageFound bool,
	trace []analyzer.TraceEntry,
) {
	// 1. Reset the global tracer buffer
	tracer.Reset()

	// 2. Execute a simple function in the instrumented target.
	// For this mock, we'll just use ssz.DemonstrateBranching from our CSVV test.
	// In a real scenario, this would be the actual fuzz target processing sszBytes.
	// flag := rand.Intn(2) == 0
	// ssz.DemonstrateBranching(flag)

	// 3. Capture the trace
	rawTrace := tracer.Snapshot()
	trace = make([]analyzer.TraceEntry, len(rawTrace))
	for j, r := range rawTrace {
		trace[j] = analyzer.TraceEntry{CID: r.CID, Value: r.Value}
	}

	// 4. Simulate signature, bug, and coverage for now
	signature = feedback.RuntimeSignature{}
	bugTriggered = false // No bug in DemonstrateBranching
	newCoverageFound = false

	// Simulate coverage gain based on trace length (very simplistic)
	if len(trace) > 0 {
		simulatedCoverageGain := rand.Float64() * 0.001 // Small random gain
		mf.currentCoverage += simulatedCoverageGain
		if simulatedCoverageGain > 0.0005 { // A threshold for "new coverage"
			newCoverageFound = true
			mf.lastNewCoverage = simulatedCoverageGain
		} else {
			mf.lastNewCoverage = 0.0
		}
	}

	// Randomly simulate a bug for testing the reward function
	if rand.Float64() < 0.01 { // 1% chance of finding a bug
		bugTriggered = true
		signature.BugFoundCount = 1
		fmt.Println("Mock bug triggered!")
	}

	return signature, bugTriggered, newCoverageFound, trace
}
