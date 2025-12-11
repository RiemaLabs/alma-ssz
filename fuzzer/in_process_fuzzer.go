package fuzzer

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"alma.local/ssz/feedback"
	"alma.local/ssz/internal/analyzer"
	ssz "github.com/ferranbt/fastssz"
	"github.com/ferranbt/fastssz/tracer"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// detectDirtyPadding scans a struct for byte-array/slice bitvectors with non-zero high bits.
// Heuristics:
//   - Any [1]byte (Bitvector4) with high 4 bits set.
//   - Any byte array/slice length > 1 with any byte having high 2 bits set (0xC0).
func detectDirtyPadding(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return false
		}
		return detectDirtyPadding(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanInterface() {
				continue
			}
			if detectDirtyPadding(f) {
				return true
			}
		}
	case reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			if v.Len() == 1 {
				// Bitvector4-like
				if v.Index(0).Uint()&0xF0 != 0 {
					return true
				}
			} else {
				for i := 0; i < v.Len(); i++ {
					if v.Index(i).Uint()&0xC0 != 0 {
						return true
					}
				}
			}
		} else {
			for i := 0; i < v.Len(); i++ {
				if detectDirtyPadding(v.Index(i)) {
					return true
				}
			}
		}
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			for i := 0; i < v.Len(); i++ {
				if v.Index(i).Uint()&0xC0 != 0 {
					return true
				}
			}
		} else {
			for i := 0; i < v.Len(); i++ {
				if detectDirtyPadding(v.Index(i)) {
					return true
				}
			}
		}
	}
	return false
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
	signature = feedback.NewRuntimeSignature()

	if bugTriggered { // Already triggered by panic
		signature.BugFoundCount = 1
		signature.BugKinds["Panic"]++
	} else if err != nil { // Unmarshaling failed
		signature.NonBugErrorCount = 1
	} else { // Unmarshaling succeeded, check for roundtrip issues
		reencodedBytes, marshalErr := marshaler.MarshalSSZ()
		if marshalErr != nil {
			signature.NonBugErrorCount = 1
		} else {
			// Compute hash of the remarshaled bytes for comparison
			// Create a new instance for remarshaled data to compute its hash
			remarshaledTargetVal := reflect.New(ipf.targetPrototype)
			remarshaledTarget := remarshaledTargetVal.Interface().(ssz.Unmarshaler) // Must be unmarshaler to load reencodedBytes

			remarshalErr := remarshaledTarget.UnmarshalSSZ(reencodedBytes)
			if remarshalErr != nil {
				signature.NonBugErrorCount = 1
				return // Early exit if re-unmarshaling fails
			}

			reencodedHash, reencodedHashErr := remarshaledTarget.(ssz.HashRoot).HashTreeRoot()
			if reencodedHashErr != nil {
				signature.NonBugErrorCount = 1
				return // Early exit if reencoded HashTreeRoot computation fails
			}

			var originalHash [32]byte
			var hashErr error

			// If schema implements Canonicalizer, compare with canonical hash
			if canonicalizer, ok := target.(Canonicalizer); ok { // Using the new interface, "fuzzer." removed
				canonicalTarget, canonErr := canonicalizer.Canonicalize()
				if canonErr != nil {
					signature.NonBugErrorCount = 1
					return // Early exit if Canonicalize fails
				}
				originalHash, hashErr = canonicalTarget.(ssz.HashRoot).HashTreeRoot() // Direct call
				if hashErr != nil {
					signature.NonBugErrorCount = 1
					return // Early exit if Canonical hash computation fails
				}
			} else {
				// Otherwise, compute original hash directly from the initial unmarshaled target
				originalHash, hashErr = target.(ssz.HashRoot).HashTreeRoot() // Direct call
				if hashErr != nil {
					signature.NonBugErrorCount = 1
					return // Early exit if original HashTreeRoot fails
				}
			}

			// Compare hashes for semantic bugs (dirty padding)
			if !bytes.Equal(originalHash[:], reencodedHash[:]) {
				bugTriggered = true
				signature.BugFoundCount = 1
				if detectDirtyPadding(targetVal.Elem()) {
					signature.BugKinds["BitvectorDirtyPadding"]++
					fmt.Printf("BUG_FOUND: Bitvector Dirty Padding (Semantic Mismatch)! Original canonical hash %x, Re-encoded hash %x\n", originalHash, reencodedHash)
				} else {
					signature.BugKinds["SemanticMismatch"]++
					fmt.Printf("BUG_FOUND: Semantic Mismatch! Original canonical hash %x, Re-encoded hash %x\n", originalHash, reencodedHash)
				}
				return // Bug found, no further checks needed for this input
			}

			// Also check for byte-level roundtrip mismatch (might indicate other bugs)
			if !bytes.Equal(sszBytes, reencodedBytes) {
				bugTriggered = true
				signature.BugFoundCount = 1
				if detectDirtyPadding(targetVal.Elem()) {
					signature.BugKinds["BitvectorDirtyPadding"]++
					fmt.Printf("BUG_FOUND: Bitvector Dirty Padding (RoundTrip mismatch)! Input len %d != Output len %d\n", len(sszBytes), len(reencodedBytes))
				} else {
					signature.BugKinds["RoundTripMismatch"]++
					fmt.Printf("BUG_FOUND: Byte-level RoundTrip mismatch! Input len %d != Output len %d\n", len(sszBytes), len(reencodedBytes))
				}
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
