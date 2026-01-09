package fuzzer

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"alma.local/ssz/feedback"
	"alma.local/ssz/internal/analyzer"
	"alma.local/ssz/internal/sszref"
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
	externalOracle  ExternalOracle
	externalSchema  string
}

type sizeSSZer interface {
	SizeSSZ() int
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

// SetExternalOracle attaches an external oracle for cross-language checks.
func (ipf *InProcessFuzzer) SetExternalOracle(oracle ExternalOracle, schema string) {
	ipf.externalOracle = oracle
	ipf.externalSchema = schema
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
		reencodedBytes, marshalErr, marshalPanic := safeMarshal(marshaler)
		if marshalPanic {
			bugTriggered = true
			signature.BugFoundCount = 1
			signature.BugKinds["MarshalPanic"]++
		} else if marshalErr != nil {
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

			reencodedHash, reencodedHashErr, reencodedPanic := safeHashRoot(remarshaledTarget.(ssz.HashRoot))
			if reencodedPanic {
				bugTriggered = true
				signature.BugFoundCount = 1
				signature.BugKinds["HashTreeRootPanic"]++
				return // Bug found, skip further checks
			}
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
				var hashPanic bool
				originalHash, hashErr, hashPanic = safeHashRoot(canonicalTarget.(ssz.HashRoot))
				if hashPanic {
					bugTriggered = true
					signature.BugFoundCount = 1
					signature.BugKinds["HashTreeRootPanic"]++
					return
				}
				if hashErr != nil {
					signature.NonBugErrorCount = 1
					return // Early exit if Canonical hash computation fails
				}
			} else {
				// Otherwise, compute original hash directly from the initial unmarshaled target
				var hashPanic bool
				originalHash, hashErr, hashPanic = safeHashRoot(target.(ssz.HashRoot))
				if hashPanic {
					bugTriggered = true
					signature.BugFoundCount = 1
					signature.BugKinds["HashTreeRootPanic"]++
					return
				}
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

// ExecuteWithObject runs the standard fuzzer plus reference-oracle checks using the original object.
func (ipf *InProcessFuzzer) ExecuteWithObject(sszBytes []byte, obj interface{}) (
	signature feedback.RuntimeSignature,
	bugTriggered bool,
	newCoverageFound bool,
	trace []analyzer.TraceEntry,
) {
	signature, bugTriggered, newCoverageFound, trace = ipf.Execute(sszBytes)
	if obj == nil {
		return signature, bugTriggered, newCoverageFound, trace
	}

	refBug := ipf.applyReferenceChecks(obj, &signature)
	if refBug {
		bugTriggered = true
	}

	if ipf.externalOracle != nil {
		refBytes, refErr := sszref.Marshal(obj)
		refRoot, refRootErr := sszref.HashTreeRoot(obj)
		if refErr == nil {
			extResult, extErr := ipf.externalOracle.Decode(ipf.externalSchema, sszBytes)
			if extErr != nil {
				if bytes.Equal(sszBytes, refBytes) {
					signature.BugFoundCount++
					signature.BugKinds["ExternalDecodeError"]++
					bugTriggered = true
				}
			} else {
				if !bytes.Equal(extResult.Canonical, sszBytes) {
					signature.BugFoundCount++
					signature.BugKinds["ExternalRoundTripMismatch"]++
					bugTriggered = true
				}
				if refRootErr == nil && bytes.Equal(sszBytes, refBytes) && !bytes.Equal(extResult.Root[:], refRoot[:]) {
					signature.BugFoundCount++
					signature.BugKinds["ExternalHTRMismatch"]++
					bugTriggered = true
				}
			}
		}
	}
	return signature, bugTriggered, newCoverageFound, trace
}

// PreflightChecks runs lightweight, deterministic oracle checks before fuzzing.
func (ipf *InProcessFuzzer) PreflightChecks() (feedback.RuntimeSignature, bool) {
	signature := feedback.NewRuntimeSignature()
	zeroObj := reflect.New(ipf.targetPrototype).Interface()
	if ipf.applyReferenceChecks(zeroObj, &signature) {
		return signature, true
	}

	if nilObj, ok := buildNilPointerSliceObject(ipf.targetPrototype); ok {
		if ipf.applyReferenceChecks(nilObj, &signature) {
			return signature, true
		}
	}

	if minObj, ok := buildMinLenObject(ipf.targetPrototype); ok {
		if ipf.applyReferenceChecks(minObj, &signature) {
			return signature, true
		}
	}

	if violObj, ok := buildMaxViolationObject(ipf.targetPrototype); ok {
		if checkMaxViolation(violObj, &signature) {
			return signature, true
		}
	}

	return signature, false
}

func (ipf *InProcessFuzzer) applyReferenceChecks(obj interface{}, signature *feedback.RuntimeSignature) bool {
	var bugFound bool

	if sizer, ok := obj.(sizeSSZer); ok {
		if _, panicked := safeSizeSSZ(sizer); panicked {
			signature.BugFoundCount++
			signature.BugKinds["SizeSSZPanic"]++
			bugFound = true
		}
	}

	refBytes, refErr := sszref.Marshal(obj)
	marshaler, hasMarshal := obj.(ssz.Marshaler)
	if hasMarshal {
		fastBytes, fastErr, fastPanic := safeMarshal(marshaler)
		if fastPanic {
			signature.BugFoundCount++
			signature.BugKinds["MarshalPanic"]++
			bugFound = true
		} else if refErr == nil && fastErr != nil {
			signature.BugFoundCount++
			signature.BugKinds["ReferenceMarshalError"]++
			bugFound = true
		} else if refErr == nil && fastErr == nil && !bytes.Equal(refBytes, fastBytes) {
			signature.BugFoundCount++
			signature.BugKinds["ReferenceMarshalMismatch"]++
			bugFound = true
		} else if refErr != nil && fastErr == nil {
			signature.NonBugErrorCount++
		}
	}

	refRoot, refRootErr := sszref.HashTreeRoot(obj)
	if hasRoot, ok := obj.(ssz.HashRoot); ok {
		fastRoot, fastErr, fastPanic := safeHashRoot(hasRoot)
		if fastPanic {
			signature.BugFoundCount++
			signature.BugKinds["HashTreeRootPanic"]++
			bugFound = true
		} else if refRootErr == nil && fastErr != nil {
			signature.BugFoundCount++
			signature.BugKinds["ReferenceHTRError"]++
			bugFound = true
		} else if refRootErr == nil && fastErr == nil && !bytes.Equal(refRoot[:], fastRoot[:]) {
			signature.BugFoundCount++
			signature.BugKinds["ReferenceHTRMismatch"]++
			bugFound = true
		} else if refRootErr != nil && fastErr == nil {
			signature.NonBugErrorCount++
		}
	}

	if refRootErr != nil {
		return bugFound
	}

	if proofTarget, ok := obj.(ssz.HashRootProof); ok {
		var (
			tree *ssz.Node
			err  error
		)
		func() {
			defer func() {
				if r := recover(); r != nil {
					signature.BugFoundCount++
					signature.BugKinds["ProofPanic"]++
					bugFound = true
				}
			}()
			tree, err = ssz.ProofTree(proofTarget)
		}()
		if err != nil {
			signature.BugFoundCount++
			signature.BugKinds["ProofTreeError"]++
			bugFound = true
			return bugFound
		}
		if tree != nil {
			treeRoot := tree.Hash()
			if !bytes.Equal(treeRoot, refRoot[:]) {
				signature.BugFoundCount++
				signature.BugKinds["ProofTreeMismatch"]++
				bugFound = true
			}
			indices := selectProofIndices(tree, 2)
			if len(indices) > 0 {
				mp, mpErr := tree.ProveMulti(indices)
				if mpErr != nil {
					signature.BugFoundCount++
					signature.BugKinds["MultiproofError"]++
					bugFound = true
				} else {
					refOK, refErr := sszref.VerifyMultiproof(refRoot, mp.Hashes, mp.Leaves, mp.Indices)
					if refErr != nil {
						signature.NonBugErrorCount++
					} else if !refOK {
						signature.BugFoundCount++
						signature.BugKinds["MultiproofInvalid"]++
						bugFound = true
					} else {
						fastOK, fastErr := ssz.VerifyMultiproof(refRoot[:], mp.Hashes, mp.Leaves, mp.Indices)
						if fastErr != nil || fastOK != refOK {
							signature.BugFoundCount++
							signature.BugKinds["MultiproofVerifyMismatch"]++
							bugFound = true
						}
					}
				}
			}
		}
	}

	return bugFound
}

func selectProofIndices(tree *ssz.Node, max int) []int {
	if tree == nil || max <= 0 {
		return nil
	}
	indices := make([]int, 0, max)
	for i := 2; i < 64 && len(indices) < max; i++ {
		if _, err := tree.Get(i); err == nil {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		indices = append(indices, 1)
	}
	return indices
}

func safeMarshal(m ssz.Marshaler) (out []byte, err error, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	out, err = m.MarshalSSZ()
	return out, err, panicked
}

func safeHashRoot(h ssz.HashRoot) (out [32]byte, err error, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	out, err = h.HashTreeRoot()
	return out, err, panicked
}

func safeSizeSSZ(s sizeSSZer) (size int, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	size = s.SizeSSZ()
	return size, panicked
}

func buildMaxViolationObject(proto reflect.Type) (interface{}, bool) {
	t := proto
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	obj := reflect.New(t)
	if !setMaxViolation(obj.Elem()) {
		return nil, false
	}
	return obj.Interface(), true
}

func buildNilPointerSliceObject(proto reflect.Type) (interface{}, bool) {
	t := proto
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	obj := reflect.New(t)
	if !setNilPointerSlice(obj.Elem()) {
		return nil, false
	}
	return obj.Interface(), true
}

func buildMinLenObject(proto reflect.Type) (interface{}, bool) {
	t := proto
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	obj := reflect.New(t)
	if !setMinLen(obj.Elem()) {
		return nil, false
	}
	return obj.Interface(), true
}

func setMaxViolation(v reflect.Value) bool {
	if v.Kind() != reflect.Struct {
		return false
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		if fv.Kind() == reflect.Slice {
			maxes := parseMaxTag(field.Tag.Get("ssz-max"))
			if len(maxes) == 0 {
				continue
			}
			const maxLenCap = 2048
			if fv.Type().Elem().Kind() == reflect.Slice && len(maxes) > 1 {
				innerMax := maxes[1]
				if innerMax > 0 && innerMax <= maxLenCap {
					outer := reflect.MakeSlice(fv.Type(), 1, 1)
					inner := reflect.MakeSlice(fv.Type().Elem(), innerMax+1, innerMax+1)
					outer.Index(0).Set(inner)
					fv.Set(outer)
					return true
				}
			}
			max := maxes[0]
			if max > 0 && max <= maxLenCap {
				slice := reflect.MakeSlice(fv.Type(), max+1, max+1)
				fv.Set(slice)
				return true
			}
		}
		if fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.Struct {
			elem := reflect.New(fv.Type().Elem())
			if setMaxViolation(elem.Elem()) {
				fv.Set(elem)
				return true
			}
		}
		if fv.Kind() == reflect.Struct {
			if setMaxViolation(fv) {
				return true
			}
		}
	}
	return false
}

func setNilPointerSlice(v reflect.Value) bool {
	if v.Kind() != reflect.Struct {
		return false
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		if fv.Kind() == reflect.Slice && fv.Type().Elem().Kind() == reflect.Ptr {
			slice := reflect.MakeSlice(fv.Type(), 1, 1)
			fv.Set(slice)
			return true
		}
		if fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.Struct {
			elem := reflect.New(fv.Type().Elem())
			if setNilPointerSlice(elem.Elem()) {
				fv.Set(elem)
				return true
			}
		}
		if fv.Kind() == reflect.Struct {
			if setNilPointerSlice(fv) {
				return true
			}
		}
	}
	return false
}

func setMinLen(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			fv := v.Field(i)
			if !fv.CanSet() {
				continue
			}
			if setMinLen(fv) {
				return true
			}
		}
	case reflect.Ptr:
		if v.IsNil() {
			elem := reflect.New(v.Type().Elem())
			v.Set(elem)
		}
		return setMinLen(v.Elem())
	case reflect.Slice:
		if v.Len() == 0 {
			slice := reflect.MakeSlice(v.Type(), 1, 1)
			v.Set(slice)
		}
		if v.Len() > 0 {
			return setMinLen(v.Index(0))
		}
	case reflect.Array:
		if v.Len() > 0 {
			return setMinLen(v.Index(0))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
		return true
	case reflect.Bool:
		v.SetBool(true)
		return true
	}
	return false
}

func parseMaxTag(tag string) []int {
	if tag == "" {
		return nil
	}
	parts := strings.Split(tag, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		val, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		out = append(out, val)
	}
	return out
}

func checkMaxViolation(obj interface{}, signature *feedback.RuntimeSignature) bool {
	marshaler, hasMarshal := obj.(ssz.Marshaler)
	if !hasMarshal {
		return false
	}

	refBytes, refErr := sszref.Marshal(obj)
	fastBytes, fastErr, fastPanic := safeMarshal(marshaler)
	if fastPanic {
		signature.BugFoundCount++
		signature.BugKinds["MarshalPanic"]++
		return true
	}
	if refErr != nil && fastErr == nil {
		signature.BugFoundCount++
		signature.BugKinds["MaxLenBypass"]++
		return true
	}
	if refErr == nil && fastErr != nil {
		signature.BugFoundCount++
		signature.BugKinds["MaxLenReject"]++
		return true
	}
	if refErr == nil && fastErr == nil && !bytes.Equal(refBytes, fastBytes) {
		signature.BugFoundCount++
		signature.BugKinds["ReferenceMarshalMismatch"]++
		return true
	}
	return false
}
