package main

import (
	"flag"
	"fmt"
	"os"

	"alma.local/ssz/rl"
	"alma.local/ssz/schemas"
	ssz "github.com/ferranbt/fastssz"
)

func main() {
	opts := rl.RLOpts{}
	flag.IntVar(&opts.Episodes, "episodes", 10, "Number of training episodes")
	flag.IntVar(&opts.MaxSteps, "max-steps", 100, "Maximum steps per episode")
	flag.StringVar(&opts.AgentType, "agent", "policy", "Agent type (e.g., 'random', 'policy')")
	flag.StringVar(&opts.SchemaName, "schema", "AttestationData", "Name of the schema to fuzz (e.g., 'AttestationData', 'BeaconState')")
	flag.IntVar(&opts.BatchSize, "batch-size", 10, "Number of inputs per step")
	flag.BoolVar(&opts.IsBaseline, "baseline", false, "Run in baseline mode (no RL agent learning)")
	flag.BoolVar(&opts.NoRL, "no-rl", false, "Disable learning while keeping SGIA buckets (uniform random bucket selection)")
	flag.IntVar(&opts.D_ctx, "d-ctx", 7, "Dimensionality of the observation context for the RL agent") // New flag
	flag.BoolVar(&opts.RequireBitvectorBug, "require-bitvector-bug", false, "Only treat Bitvector dirty padding as bug trigger")
	flag.Parse()

	var targetSchema ssz.Unmarshaler
	switch opts.SchemaName {
	case "AttestationData":
		targetSchema = &schemas.AttestationData{}
	case "BeaconState":
		targetSchema = &schemas.BeaconState{}
	case "PendingAttestation":
		targetSchema = &schemas.PendingAttestation{}
	case "BitvectorStruct":
		targetSchema = &schemas.BitvectorStruct{}
	case "BooleanStruct":
		targetSchema = &schemas.BooleanStruct{}
	case "GapStruct":
		targetSchema = &schemas.GapStruct{}
	case "HardBitvectorStruct":
		targetSchema = &schemas.HardBitvectorStruct{}
	case "HardBooleanStruct":
		targetSchema = &schemas.HardBooleanStruct{}
	case "HardGapStruct":
		targetSchema = &schemas.HardGapStruct{}
	case "UnionStruct":
		targetSchema = &schemas.UnionStruct{}
	case "HardUnionStruct":
		targetSchema = &schemas.HardUnionStruct{}
	// Add other schemas here as needed
	default:
		fmt.Printf("Unknown schema: %s\n", opts.SchemaName)
		os.Exit(1)
	}

	rl.RunRLProcess(targetSchema, opts)
}
