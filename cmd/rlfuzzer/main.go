package main

import (
	"flag"
	"fmt"
	"log"

	"alma.local/ssz/rl"
	"alma.local/ssz/schemas"
	ssz "github.com/ferranbt/fastssz"
)

var (
	episodes   = flag.Int("episodes", 10, "Number of training episodes")
	maxSteps   = flag.Int("max_steps", 100, "Maximum steps per episode")
	agentType  = flag.String("agent_type", "policy", "Type of RL agent (e.g., 'policy')")
	schemaName = flag.String("schema", "BeaconState", "Name of the SSZ schema to fuzz (e.g., 'BeaconState', 'BitvectorStruct')")
	batchSize  = flag.Int("batch_size", 10, "Number of inputs to process per step (batch size)")
)

func main() {
	flag.Parse()

	// Map schema name to actual Go type
	var targetSchema ssz.Unmarshaler
	switch *schemaName {
	case "BitvectorStruct":
		targetSchema = &schemas.BitvectorStruct{}
	case "BooleanStruct":
		targetSchema = &schemas.BooleanStruct{}
	case "GapStruct":
		targetSchema = &schemas.GapStruct{}
	case "AttestationData":
		targetSchema = &schemas.AttestationData{}
	case "PendingAttestation":
		targetSchema = &schemas.PendingAttestation{}
	case "BeaconBlockHeader":
		targetSchema = &schemas.BeaconBlockHeader{}
	case "BeaconState":
		targetSchema = &schemas.BeaconState{}
	default:
		log.Fatalf("Unknown schema name: %s", *schemaName)
	}
	
	opts := rl.RLOpts{
		Episodes:   *episodes,
		MaxSteps:   *maxSteps,
		AgentType:  *agentType,
		SchemaName: *schemaName,
		BatchSize:  *batchSize,
	}

	fmt.Printf("Starting RL Fuzzer for schema: %s\n", *schemaName)
	rl.RunRLProcess(targetSchema, opts)
}
