package main

import (
	"flag"
	"fmt"
	"time"

	"alma.local/ssz/rl"
	"alma.local/ssz/schemas"
	ssz "github.com/ferranbt/fastssz"
)

func main() {
	var (
		schemaName = flag.String("schema", "BitvectorStruct", "Schema to fuzz")
		mode       = flag.String("mode", "baseline", "baseline | norl | rl")
		budget     = flag.Duration("budget", 30*time.Minute, "Wall-clock budget")
		maxSteps   = flag.Int("max-steps", 50000, "Max steps per episode")
		batchSize  = flag.Int("batch-size", 50, "Batch size")
		requireBV  = flag.Bool("require-bitvector-bug", false, "Only stop when Bitvector dirty padding bug is hit")
	)
	flag.Parse()

	var targetSchema ssz.Unmarshaler
	switch *schemaName {
	case "BitvectorStruct":
		targetSchema = &schemas.BitvectorStruct{}
	case "HardBitvectorStruct":
		targetSchema = &schemas.HardBitvectorStruct{}
	case "BooleanStruct":
		targetSchema = &schemas.BooleanStruct{}
	case "HardBooleanStruct":
		targetSchema = &schemas.HardBooleanStruct{}
	case "GapStruct":
		targetSchema = &schemas.GapStruct{}
	case "HardGapStruct":
		targetSchema = &schemas.HardGapStruct{}
	case "UnionStruct":
		targetSchema = &schemas.UnionStruct{}
	case "HardUnionStruct":
		targetSchema = &schemas.HardUnionStruct{}
	case "SuffixStateDiff":
		targetSchema = &schemas.SuffixStateDiff{}
	case "BeaconState":
		targetSchema = &schemas.BeaconState{}
	case "Validator":
		targetSchema = &schemas.Validator{}
	case "PendingAttestation":
		targetSchema = &schemas.PendingAttestation{}
	case "AggregationBitsContainer":
		targetSchema = &schemas.AggregationBitsContainer{}
	default:
		panic("unknown schema")
	}

	opts := rl.RLOpts{
		Episodes:   1,
		MaxSteps:   *maxSteps,
		AgentType:  "policy",
		SchemaName: *schemaName,
		BatchSize:  *batchSize,
		D_ctx:      7,
		RequireBitvectorBug: *requireBV,
	}

	switch *mode {
	case "baseline":
		opts.IsBaseline = true
	case "norl":
		opts.NoRL = true
	case "rl":
		// full RL
	default:
		panic("unknown mode")
	}

	start := time.Now()
	bugStep, err := rl.RunUntilBug(targetSchema, opts, *budget)
	dur := time.Since(start)
	if err != nil {
		fmt.Printf("MODE=%s SCHEMA=%s RESULT=timeout DURATION=%s\n", *mode, *schemaName, dur)
		return
	}
	fmt.Printf("MODE=%s SCHEMA=%s RESULT=bug STEP=%d DURATION=%s\n", *mode, *schemaName, bugStep, dur)
}
