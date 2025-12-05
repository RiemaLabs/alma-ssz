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
	flag.IntVar(&opts.BatchSize, "batch-size", 5, "Number of inputs per step")
	flag.BoolVar(&opts.IsBaseline, "baseline", false, "Run in baseline mode (no RL agent learning)")
	flag.IntVar(&opts.D_ctx, "d-ctx", 7, "Dimensionality of the observation context for the RL agent") // New flag
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
	// Add other schemas here as needed
	default:
		fmt.Printf("Unknown schema: %s\n", opts.SchemaName)
		os.Exit(1)
	}

	rl.RunRLProcess(targetSchema, opts)
}
