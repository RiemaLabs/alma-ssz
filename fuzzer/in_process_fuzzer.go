package fuzzer

import (
	"math/rand"
	"reflect"
	"time"

	"alma.local/ssz/feedback"
	"alma.local/ssz/internal/analyzer"
	"github.com/ferranbt/fastssz"
	"github.com/ferranbt/fastssz/tracer"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// InProcessFuzzer is a high-performance fuzzer that runs in the same process space.
// It avoids the overhead of `go run` by directly calling the Unmarshal methods
// of the target schema and capturing the tracer output.
type InProcessFuzzer struct {
	globalSeenCIDs  map[uint64]struct{}
	currentCoverage float64
	lastNewCoverage float64
	targetPrototype reflect.Type
}

// NewInProcessFuzzer creates a new InProcessFuzzer.
func NewInProcessFuzzer(target interface{}) (*InProcessFuzzer, error) {
	t := reflect.TypeOf(target)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return &InProcessFuzzer{
		globalSeenCIDs:  make(map[uint64]struct{}),
		currentCoverage: 0.0,
		lastNewCoverage: 0.0,
		targetPrototype: t,
	}, nil
}

func (ipf *InProcessFuzzer) Reset() {
	ipf.globalSeenCIDs = make(map[uint64]struct{})
	ipf.currentCoverage = 0.0
	ipf.lastNewCoverage = 0.0
}

func (ipf *InProcessFuzzer) TotalCoverage() float64 {
	return ipf.currentCoverage
}

func (ipf *InProcessFuzzer) NewCoverage() float64 {
	return ipf.lastNewCoverage
}

// Execute performs the fuzzing step.
// 1. Resets tracer.
// 2. Calls UnmarshalSSZ on the target schema.
// 3. Captures the trace.
// 4. Determines outcome (success, error, panic/bug).
func (ipf *InProcessFuzzer) Execute(sszBytes []byte) (
	signature feedback.RuntimeSignature,
	bugTriggered bool,
	newCoverageFound bool,
	trace []analyzer.TraceEntry,
) {
	// 1. Reset Tracer
	tracer.Reset()

	// 2. Execute Target
	// Create a new instance of the target type
	targetVal := reflect.New(ipf.targetPrototype)
	target := targetVal.Interface()

	bugTriggered = false
	var err error

	// Assert that the target implements Unmarshaler and Marshaler
	unmarshaler, ok := target.(ssz.Unmarshaler)
	if !ok {
		return feedback.RuntimeSignature{NonBugErrorCount: 1}, false, false, nil
	}
	marshaler, ok := target.(ssz.Marshaler)
	if !ok {
		return feedback.RuntimeSignature{NonBugErrorCount: 1}, false, false, nil
	}

	// Use a deferred recovery to catch panics (bugs)
	func() {
		defer func() {
			if r := recover(); r != nil {
				bugTriggered = true
				// Log panic? fmt.Printf("Panic caught: %v\n", r)
			}
		}()

		// Call the unmarshaler
		err = unmarshaler.UnmarshalSSZ(sszBytes)
	}()

	// 3. Capture Trace
	rawTrace := tracer.Snapshot()

trace = make([]analyzer.TraceEntry, len(rawTrace))
	for j, r := range rawTrace {
		trace[j] = analyzer.TraceEntry{CID: r.CID, Value: r.Value}
	}

	// 4. Synthesize Feedback
	signature = feedback.RuntimeSignature{}

	if bugTriggered {
		signature.BugFoundCount = 1
	} else if err != nil {
		signature.NonBugErrorCount = 1
	} else {
		reencodedBytes, marshalErr := marshaler.MarshalSSZ()
		if marshalErr != nil {
			signature.NonBugErrorCount = 1
		} else {
			if string(sszBytes) != string(reencodedBytes) {
				bugTriggered = true
				signature.BugFoundCount = 1
				// fmt.Printf("BUG_FOUND: RoundTrip mismatch! Input len %d != Output len %d\n", len(sszBytes), len(reencodedBytes))
			} else {
				signature.RoundtripSuccessCount = 1
			}
		}
	}

	// Calculate Cumulative Coverage
	newlySeenCount := 0
	for _, t := range trace {
		if _, seen := ipf.globalSeenCIDs[t.CID]; !seen {
			ipf.globalSeenCIDs[t.CID] = struct{}{}
			newlySeenCount++
		}
	}

	if newlySeenCount > 0 {
		ipf.lastNewCoverage = float64(newlySeenCount)
		ipf.currentCoverage = float64(len(ipf.globalSeenCIDs))
		newCoverageFound = true
	} else {
		ipf.lastNewCoverage = 0.0
		newCoverageFound = false
	}

	return signature, bugTriggered, newCoverageFound, trace
}