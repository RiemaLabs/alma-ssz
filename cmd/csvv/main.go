package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"os"

	"alma.local/ssz/internal/analyzer"
	"github.com/ferranbt/fastssz/tracer"
	ssz "github.com/ferranbt/fastssz"
)

// Metadata structure
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

// Point structure for corpus
type Point struct {
	Iteration int
	Input     string
	Vector    []*int64 
	Score     float64
}

var globalMetadata Metadata

func loadMetadata() {
	data, err := ioutil.ReadFile("corpus/metadata.json")
	if err != nil {
		log.Printf("Warning: Failed to read metadata (first run?): %v", err)
		return
	}
	if err := json.Unmarshal(data, &globalMetadata); err != nil {
		log.Fatalf("Failed to unmarshal metadata: %v", err)
	}
	fmt.Printf("Loaded metadata with %d dimensions.\n", len(globalMetadata.Columns))
}

func main() {
	fmt.Println("CSVV Fuzzer Runner - Recording Points")
	loadMetadata()

	az := analyzer.NewAnalyzer()

	// Run verification logic
	verifyBranching(az)

	// Fuzz loop
	fmt.Println("\n--- Starting Fuzz Loop ---")
	for i := 0; i < 50; i++ {
		tracer.Reset()

		// Generate random input (placeholder)
		data := make([]byte, 32)
		for k := range data {
			data[k] = byte(rand.Intn(256))
		}

		// Run target
		// For now, we just call DemonstrateBranching with random bool
		flag := rand.Intn(2) == 0
		ssz.DemonstrateBranching(flag)

		// Collect
		rawTrace := tracer.Snapshot()
		trace := make([]analyzer.TraceEntry, len(rawTrace))
		for j, r := range rawTrace {
			trace[j] = analyzer.TraceEntry{CID: r.CID, Value: r.Value}
		}

		// Analyze
		score := az.ScoreTrace(trace, true)

		if score > 0.1 {
			fmt.Printf("Iter %d: Score: %.2f, Trace Len: %d\n", i, score, len(trace))
			// Save point logic here if needed
			savePoint(Point{Iteration: i, Score: score, Input: fmt.Sprintf("%x", data)})
		}
	}
	
	fmt.Printf("\nTotal Dimensions Explored: %d\n", az.GetTotalDimensions())
}

func verifyBranching(az *analyzer.Analyzer) {
	fmt.Println("\n--- Verifying Branching Dimensions ---")

	// Branch A
	tracer.Reset()
	ssz.DemonstrateBranching(true)
	rawTraceA := tracer.Snapshot()
	traceA := make([]analyzer.TraceEntry, len(rawTraceA))
	for i, r := range rawTraceA {
		traceA[i] = analyzer.TraceEntry{CID: r.CID, Value: r.Value}
	}

	fmt.Printf("Branch A Trace Len: %d\n", len(traceA))
	if len(traceA) > 0 {
		fmt.Printf("  Branch A CID: %d, Val: %d\n", traceA[0].CID, traceA[0].Value)
	}
	
	// Branch B
	tracer.Reset()
	ssz.DemonstrateBranching(false)
	rawTraceB := tracer.Snapshot()
	traceB := make([]analyzer.TraceEntry, len(rawTraceB))
	for i, r := range rawTraceB {
		traceB[i] = analyzer.TraceEntry{CID: r.CID, Value: r.Value}
	}

	fmt.Printf("Branch B Trace Len: %d\n", len(traceB))
	if len(traceB) > 0 {
		fmt.Printf("  Branch B CID: %d, Val: %d\n", traceB[0].CID, traceB[0].Value)
	}

	if len(traceA) > 0 && len(traceB) > 0 {
		if traceA[0].CID != traceB[0].CID {
			fmt.Println("SUCCESS: Different CIDs for 'x' in if/else blocks! Context sensitivity works.")
		} else {
			fmt.Println("FAILURE: Same CID for 'x' in if/else blocks.")
		}
	}
	
	// Score them to warm up analyzer
	az.ScoreTrace(traceA, true)
	az.ScoreTrace(traceB, true)
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
	return enc.Encode(p)
}
