package rl

import (
	"alma.local/ssz/concretizer" // Import concretizer for MutationType
)

// Action represents a mutation operation selected by the agent.
type Action struct {
	MutationType concretizer.MutationType
	// Parameters for the mutation, e.g., offset, size, value
	Param1 int
	Param2 int
}
