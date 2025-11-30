package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"alma.local/ssz/internal/analyzer"
	"alma.local/ssz/tracer"

	// Import the instrumented library.
	ssz "github.com/ferranbt/fastssz"
)

type VarInfo struct {
	CID         uint64
	PackageName string
	FuncName    string
	BlockID     int
	VarName     string
	Location    string
}

type Metadata struct {
	Columns []string           
	Details map[string]VarInfo 
}

type Point struct {
	Iteration int
	Input     string
	Vector    []*int64 // Dense vector, nil where not executed
	Score     float64
}

var globalMetadata Metadata

func loadMetadata() {
	data, err := ioutil.ReadFile("corpus/metadata.json")
	if err != nil {
		log.Fatalf("Failed to read metadata: %v", err)
	}
	if err := json.Unmarshal(data, &globalMetadata); err != nil {
		log.Fatalf("Failed to unmarshal metadata: %v", err)
	}
	fmt.Printf("Loaded metadata with %d dimensions.\n", len(globalMetadata.Columns))
}

// We need a target to fuzz. `fastssz` has `ReadOffset`, `Decode`, etc.
// Let's assume we want to fuzz the `Decode` logic or some internal part.
// Since we instrumented everything, any call into `fastssz` that triggers logic is good.
// However, `fastssz` is a library that generates code.
// Ah, right. `fastssz` *library* has helper functions (ReadOffset, etc).
// The generated code uses these helpers.
//
// BUT, the instrumentor instrumented `workspace/fastssz`.
// Does `workspace/fastssz` contain the generated code? No, it contains the library `ReadOffset` etc.
//
// IF we want to test the *library functions* directly (like `ReadOffset`), we can just call them.
// IF we want to test `Unmarshal`, we need a struct that implements SSZ Marshaler/Unmarshaler.
//
// Let's create a dummy struct that uses `fastssz` helpers, or better, use `fastssz`'s own tests or a simple harness.
//
// For this experiment, let's just call `ssz.ReadOffset` and other helpers with random data?
// No, that's too unit-testy.
//
// Let's try to use `fastssz.HashTreeRoot` or something generic if available.
// Looking at `fastssz` exports:
// It seems `fastssz` provides `HashNode`, `Proof`, etc.
//
// WAIT: `fastssz` is primarily a code generator. The runtime library is small.
// The instrumentation I ran was on `workspace/fastssz` which is the runtime library.
// So I should call functions in that library.
//
// Let's look at `workspace/fastssz/tests` or `workspace/fastssz/encode_test.go` to see what's testable.
// `ReadOffset` is in `testutil.go`? No, `interface.go`?
//
// Let's try to verify if we can link first.

func main() {
	fmt.Println("CSVV Fuzzer Runner - Recording Points")
	loadMetadata()

	az := analyzer.NewAnalyzer()

	// Simple loop
	for i := 0; i < 1000; i++ {
		tracer.Reset()

		// Generate random input
		data := make([]byte, 32)
		// Randomize data
		// simple rand
		for k := range data {
			data[k] = byte(rand.Intn(256))
		}

		// Run target
		runTarget(data)

		// Collect
		// We need to convert tracer.TraceEntry to analyzer.TraceEntry
		rawTrace := tracer.Snapshot()
		trace := make([]analyzer.TraceEntry, len(rawTrace))
		for j, r := range rawTrace {
			trace[j] = analyzer.TraceEntry{CID: r.CID, Value: r.Value}
		}

		// Analyze
		score := az.ScoreTrace(trace, true)

		// Construct Vector
		// Map trace to quick lookup
		traceMap := make(map[uint64]int64)
		for _, t := range trace {
			traceMap[t.CID] = t.Value
		}

		vector := make([]*int64, len(globalMetadata.Columns))
		for idx, colStr := range globalMetadata.Columns {
			// We stored CIDs as strings in JSON keys, but we can parse or use string map if we had one.
			// globalMetadata.Details keys are strings.
			// But trace uses uint64.
			// We need to convert string CID back to uint64 or match.
			// Let's assume we parse colStr.
			var cid uint64
			fmt.Sscanf(colStr, "%d", &cid)
			
			if val, ok := traceMap[cid]; ok {
				v := val
				vector[idx] = &v
			} else {
				vector[idx] = nil
			}
		}

		// Record Point (Input + Trace)
		point := Point{
			Iteration: i,
			Input:     hex.EncodeToString(data),
			Vector:    vector,
			Score:     score,
		}
		if err := savePoint(point); err != nil {
			log.Printf("Failed to save point %d: %v", i, err)
		}

		if score > 10.0 {
			fmt.Printf("Iter %d: Interesting Trace! Score: %.2f\n", i, score)
			saveToCorpus(i, data)
		}
	}

	// Verification Step: Branching
	fmt.Println("\n--- Verifying Branching Dimensions ---")

	// Run Branch A
	tracer.Reset()
	ssz.DemonstrateBranching(true)
	rawTraceA := tracer.Snapshot()
	// Copy to avoid aliasing when buffer is reused
	traceA := make([]tracer.TraceEntry, len(rawTraceA))
	copy(traceA, rawTraceA)
	
	fmt.Printf("Branch A Trace Len: %d\n", len(traceA))
	if len(traceA) > 0 {
		fmt.Printf("  Branch A CID: %d, Val: %d\n", traceA[0].CID, traceA[0].Value)
	}

	// Run Branch B
	tracer.Reset()
	ssz.DemonstrateBranching(false)
	rawTraceB := tracer.Snapshot()
	traceB := make([]tracer.TraceEntry, len(rawTraceB))
	copy(traceB, rawTraceB)

	fmt.Printf("Branch B Trace Len: %d\n", len(traceB))
	if len(traceB) > 0 {
		fmt.Printf("  Branch B CID: %d, Val: %d\n", traceB[0].CID, traceB[0].Value)
	}

	if len(traceA) > 0 && len(traceB) > 0 {
		if traceA[0].CID != traceB[0].CID {
			fmt.Println("SUCCESS: Different CIDs for 'x' in if/else blocks! Context sensitivity works.")
		} else {
			fmt.Println("FAILURE: Same CID for 'x' in if/else blocks (Context sensitivity failed).")
		}
	}

	totalDims := az.GetTotalDimensions()
	fmt.Printf("\n--- Dimensionality Stats ---\n")
	fmt.Printf("Global Dimensionality (Total Unique Variables/Contexts Encountered): %d\n", totalDims)
	fmt.Printf("Note: A 'Point' in this space is a sparse vector of %d dimensions.\n", totalDims)
	if len(traceB) > 0 {
		fmt.Printf("Example Run (Branch B) touched %d dimensions (non-nil values).\n", len(traceB))
	}
}

func savePoint(p Point) error {
	dir := "corpus/points"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	filename := filepath.Join(dir, fmt.Sprintf("point_%d.json", p.Iteration))
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	
	enc := json.NewEncoder(f)
	// enc.SetIndent("", "  ") // Optional: pretty print
	return enc.Encode(p)
}

func saveToCorpus(iter int, data []byte) {
	dir := "corpus/csvv"
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("failed to create corpus dir: %v", err)
		return
	}
	name := fmt.Sprintf("iter_%d.ssz", iter)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("failed to write corpus file: %v", err)
	}
}

func runTarget(data []byte) {
	// 1. Test DecodeDynamicLength
	// It expects at least 4 bytes.
	// It interprets first 4 bytes as offset.
	// Then divides offset by 4.
	maxSize := uint64(100)
	_, _ = ssz.DecodeDynamicLength(data, maxSize)

	// 2. Test DivideInt2
	// func DivideInt2(a, b, max uint64)
	a := uint64(rand.Intn(1000))
	b := uint64(rand.Intn(10) + 1) // avoid div by zero if any
	m := uint64(rand.Intn(100))
	_, _ = ssz.DivideInt2(a, b, m)

	// 3. Test ValidateBitlist
	// func ValidateBitlist(buf []byte, bitLimit uint64) error
	limit := uint64(rand.Intn(256))
	_ = ssz.ValidateBitlist(data, limit)
}
