package rl

import (
	"fmt"
	"math"
	"reflect"

	ssz "github.com/ferranbt/fastssz"
)

// RLOpts defines options for configuring the RL training process.
type RLOpts struct {
	Episodes            int
	MaxSteps            int
	AgentType           string // e.g., "random", "policy" (for our new agent)
	SchemaName          string // Name of the schema to fuzz, e.g., "BeaconState"
	BatchSize           int    // Number of inputs per step
	IsBaseline          bool   // New: Flag to indicate baseline mode
	NoRL                bool   // Disable policy learning while keeping SGIA buckets
	D_ctx               int    // New: Dimensionality of the observation context for the RL agent
	RequireBitvectorBug bool   // Only stop when Bitvector dirty padding bug is hit
	ExternalOracle      string // External oracle identifier (e.g., "pyssz")
	ExternalBug         string // External bug toggle id
	DisableTail         bool   // Disable tail mutations
	DisableGap          bool   // Disable offset-gap mutations
	EnableBitlistNull   bool   // Enable null-sentinel bitlist mutation
}

// RLAgent defines the interface for an agent that interacts with the fuzzing environment.
type RLAgent interface {
	Act(obs Observation) Action
	Remember(obs Observation, action Action, reward float64, nextObs Observation, done bool)
	Learn()
	ClearMemory()
}

// RunRLProcess sets up and runs the RL-based fuzzer.
func RunRLProcess(targetSchema ssz.Unmarshaler, opts RLOpts) {
	// 1. Setup Environment
	env, err := NewFuzzingEnv(targetSchema, opts)
	if err != nil {
		fmt.Printf("Error creating fuzzing environment: %v\n", err)
		return
	}

	// 2. Setup Agent
	var agent RLAgent
	switch opts.AgentType {
	case "random":
		// RandomAgent implementation is currently removed/replaced by PolicyAgent in this iteration.
		// For now, let's just use PolicyAgent as default.
		fallthrough
	case "policy": // Use our new PolicyAgent
		obsSize := len(env.CurrentState.ToObservation().Vector)
		agent = NewPolicyAgent(env.EncodingCtx.ActionCount(), opts.IsBaseline, opts.NoRL, obsSize)
		if policy, ok := agent.(*PolicyAgent); ok {
			bvSet := make(map[string]struct{}, len(env.BitvectorFields))
			for _, name := range env.BitvectorFields {
				bvSet[name] = struct{}{}
			}
			policy.SetActionPrior(BuildActionPrior(env.EncodingCtx.Actions, bvSet))
		}
	default:
		fmt.Printf("Unknown agent type: %s\n", opts.AgentType)
		return
	}

	// 3. Create Orchestrator and Train
	orchestrator := NewRLOrchestrator(agent, env, opts)
	orchestrator.Train()
}

// RLOrchestrator manages the interaction between the RLAgent and the FuzzingEnv.
type RLOrchestrator struct {
	Agent RLAgent
	Env   *FuzzingEnv
	Opts  RLOpts
}

// NewRLOrchestrator creates a new RLOrchestrator.
func NewRLOrchestrator(agent RLAgent, env *FuzzingEnv, opts RLOpts) *RLOrchestrator {
	return &RLOrchestrator{
		Agent: agent,
		Env:   env,
		Opts:  opts,
	}
}

// Train (Conceptual) runs a simulated RL training loop.
func (rlo *RLOrchestrator) Train() {
	fmt.Printf("\n--- Starting RL Training for %d Episodes ---", rlo.Opts.Episodes)
	fmt.Printf("\nTarget Schema: %s, Max Steps per Episode: %d, Batch Size: %d\n",
		reflect.TypeOf(rlo.Env.TargetSchema).Elem().Name(), rlo.Env.MaxSteps, rlo.Env.BatchSize)

	var (
		baselineBugTriggerTime = math.MaxInt // Stores the step count when the first bug is found in baseline mode
		rlBugTriggerTime       = math.MaxInt // Stores the step count when the first bug is found in RL mode
		bugKindsTotals         = make(map[string]int)
	)

	for i := 1; i <= rlo.Opts.Episodes; i++ {
		// Pass an initial empty history summary for the first state
		initialHistory := make([]float64, len(rlo.Env.CurrentState.HistorySummary))
		oldState := rlo.Env.Reset(initialHistory) // Reset environment for a new episode
		done := false
		episodeReward := 0.0
		steps := 0

		fmt.Printf("\nEpisode %d:\n", i)
		for !done {
			// 1. Agent chooses a single action (bucket); the whole step uses this bucket across all samples.
			chosen := rlo.Agent.Act(oldState.ToObservation())
			batchActions := make([]Action, rlo.Opts.BatchSize)
			for b := 0; b < rlo.Opts.BatchSize; b++ {
				batchActions[b] = chosen
			}

			// 2. Environment executes the batch of actions
			newState, reward, done, bugTriggerStep, err := rlo.Env.Step(batchActions) // Capture bugTriggerStep
			if err != nil {
				fmt.Printf("Error during environment step: %v\n", err)
				break
			}

			// Check and record bug trigger time
			if bugTriggerStep > 0 {
				if rlo.Opts.IsBaseline {
					if bugTriggerStep < baselineBugTriggerTime {
						baselineBugTriggerTime = bugTriggerStep
					}
				} else {
					if bugTriggerStep < rlBugTriggerTime {
						rlBugTriggerTime = bugTriggerStep
					}
				}
			}

			// Accumulate episode reward and increment step count
			episodeReward += reward
			steps++

			// 3. Agent learns from the aggregated batch experience
			// For simplicity, remember the *overall* oldState, a *representative* action from the batch (e.g., the first),
			// the aggregated reward, and the newState.
			// A more robust RL implementation might remember each (obs, action, reward, next_obs, done) for each item in the batch.
			// However, for REINFORCE-like updates, we often update once per episode or per batch aggregate.
			rlo.Agent.Remember(oldState.ToObservation(), batchActions[0], reward, newState.ToObservation(), done)
			rlo.Agent.Learn()

			oldState = newState

			// Accumulate bug kinds for reporting
			for kind, cnt := range newState.Signature.BugKinds {
				bugKindsTotals[kind] += cnt
			}

			// For demonstration, print some progress for the batch every 50 steps
			if steps%50 == 0 || done {
				fmt.Printf("  Step %d - Batch Reward: %.2f - Bug Count: %d (kinds: %d) - Errors: %d - Total Ctx: %.0f - Avg KL Score: %.4f\n",
					steps, reward, newState.Signature.BugFoundCount, len(newState.Signature.BugKinds), newState.Signature.NonBugErrorCount, newState.TotalCoverage, newState.NewCoverage)
			}

			if done {
				fmt.Printf("Episode %d finished. Total Reward: %.2f in %d steps. Bug Found: %t\n",
					i, episodeReward, steps, bugFoundFromState(newState))
				break
			}
		}
	}
	fmt.Println("\n--- RL Training Complete ---")

	// Print comparison metrics
	fmt.Println("\n--- Bug Trigger Time Comparison ---")
	if rlo.Opts.IsBaseline {
		if baselineBugTriggerTime != math.MaxInt {
			fmt.Printf("Baseline First Bug Trigger Time: %d steps\n", baselineBugTriggerTime)
		} else {
			fmt.Println("Baseline: Bug not found within max steps.")
		}
	} else {
		if rlBugTriggerTime != math.MaxInt {
			fmt.Printf("RL First Bug Trigger Time: %d steps\n", rlBugTriggerTime)
		} else {
			fmt.Println("RL: Bug not found within max steps.")
		}
	}

	if len(bugKindsTotals) > 0 {
		fmt.Println("Bug Kinds Triggered (counts):")
		for kind, cnt := range bugKindsTotals {
			fmt.Printf("  - %s: %d\n", kind, cnt)
		}
	}
}

// For demonstration, a simple way to check if a bug was found.
// In a real system, this would be based on more robust oracle signals.
func bugFoundFromState(s *State) bool {
	return s.Signature.BugFoundCount > 0
}
