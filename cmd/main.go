package main

import (
	"fmt"
	"math/rand"
	"time"

	"alma.local/ssz/schemas" // The SSZ schema definitions
	"alma.local/ssz/rl"      // The new RL package
)

// This file serves as the main entry point for the RL-based fuzzing system.

func main() {
	fmt.Println("Starting the Fuzzing Agent...")

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Configure the RL process
	opts := rl.RLOpts{
		Episodes:   500,
		MaxSteps:   200,
		AgentType:  "policy",
		SchemaName: "BeaconState",
		BatchSize:  50,
	}

	// Define the target SSZ schema struct instance
	// This is passed to the environment for analysis and concretization.
	targetSchema := &schemas.BeaconState{}

	// Run the RL-based fuzzing process
	rl.RunRLProcess(targetSchema, opts)

	fmt.Println("\nFuzzing process finished.")
}
