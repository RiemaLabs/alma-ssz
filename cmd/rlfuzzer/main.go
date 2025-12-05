package main

import (
	"flag"
	"fmt"
	"log"

	"alma.local/ssz/rl"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1" // Example schema, add others as needed
)

var (
	episodes   = flag.Int("episodes", 10, "Number of training episodes")
	maxSteps   = flag.Int("max_steps", 100, "Maximum steps per episode")
	agentType  = flag.String("agent_type", "policy", "Type of RL agent (e.g., 'policy')")
	schemaName = flag.String("schema", "Attestation", "Name of the SSZ schema to fuzz (e.g., 'Attestation', 'BeaconBlockBody')")
	batchSize  = flag.Int("batch_size", 10, "Number of inputs to process per step (batch size)")
)

func main() {
	flag.Parse()

	// Map schema name to actual Go type
	var targetSchema fastssz.Unmarshaler
	switch *schemaName {
	case "Attestation":
		targetSchema = &v1alpha1.Attestation{}
	case "BeaconBlockBody":
		targetSchema = &v1alpha1.BeaconBlockBody{}
	case "SignedBeaconBlock":
		targetSchema = &v1alpha1.SignedBeaconBlock{}
	case "IndexedAttestation":
		targetSchema = &v1alpha1.IndexedAttestation{}
	case "BeaconState":
		targetSchema = &v1alpha1.BeaconState{}
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
