package rl

import (
	"errors"
	"fmt"
	"time"

	"alma.local/ssz/fuzzer"
	ssz "github.com/ferranbt/fastssz"
)

// ErrBudgetExceeded indicates the run hit the wall-clock budget before finding a bug.
var ErrBudgetExceeded = errors.New("budget exceeded without bug")

// RunMetrics captures measurements for a single run.
type RunMetrics struct {
	BugFound bool
	BugStep  int
	Duration time.Duration
	Coverage float64
	Steps    int
}

// RunUntilBugMetrics runs a single (or multi-episode) campaign and returns measurements.
func RunUntilBugMetrics(targetSchema ssz.Unmarshaler, opts RLOpts, budget time.Duration) (RunMetrics, error) {
	env, err := NewFuzzingEnv(targetSchema, opts)
	if err != nil {
		return RunMetrics{}, fmt.Errorf("failed to create fuzzing environment: %w", err)
	}

	start := time.Now()
	if ipf, ok := env.Fuzzer.(*fuzzer.InProcessFuzzer); ok {
		if _, bug := ipf.PreflightChecks(); bug {
			return RunMetrics{
				BugFound: true,
				BugStep:  0,
				Duration: time.Since(start),
				Coverage: env.Fuzzer.TotalCoverage(),
				Steps:    env.StepsCount,
			}, nil
		}
	}

	obsSize := len(env.CurrentState.ToObservation().Vector)
	agent := NewPolicyAgent(env.EncodingCtx.ActionCount(), opts.IsBaseline, opts.NoRL, obsSize)
	bvSet := make(map[string]struct{}, len(env.BitvectorFields))
	for _, name := range env.BitvectorFields {
		bvSet[name] = struct{}{}
	}
	agent.SetActionPrior(BuildActionPrior(env.EncodingCtx.Actions, bvSet))
	episodes := opts.Episodes
	if episodes <= 0 {
		episodes = 1
	}

	for ep := 0; ep < episodes; ep++ {
		initialHistory := make([]float64, len(env.CurrentState.HistorySummary))
		oldState := env.Reset(initialHistory)
		done := false

		for !done {
			if time.Since(start) > budget {
				return RunMetrics{
					BugFound: false,
					Duration: time.Since(start),
					Coverage: env.Fuzzer.TotalCoverage(),
					Steps:    env.StepsCount,
				}, ErrBudgetExceeded
			}

			chosen := agent.Act(oldState.ToObservation())
			batchActions := make([]Action, opts.BatchSize)
			for i := 0; i < opts.BatchSize; i++ {
				batchActions[i] = chosen
			}

			newState, reward, stepDone, bugTriggerStep, stepErr := env.Step(batchActions)
			if stepErr != nil {
				return RunMetrics{}, fmt.Errorf("environment step failed: %w", stepErr)
			}

			if bugTriggerStep > 0 {
				return RunMetrics{
					BugFound: true,
					BugStep:  bugTriggerStep,
					Duration: time.Since(start),
					Coverage: env.Fuzzer.TotalCoverage(),
					Steps:    env.StepsCount,
				}, nil
			}

			agent.Remember(oldState.ToObservation(), batchActions[0], reward, newState.ToObservation(), stepDone)
			agent.Learn()

			oldState = newState
			done = stepDone
		}
	}

	return RunMetrics{
		BugFound: false,
		Duration: time.Since(start),
		Coverage: env.Fuzzer.TotalCoverage(),
		Steps:    env.StepsCount,
	}, ErrBudgetExceeded
}

// RunUntilBug runs a single (or multi-episode) campaign and returns the step when the first bug is triggered.
func RunUntilBug(targetSchema ssz.Unmarshaler, opts RLOpts, budget time.Duration) (int, error) {
	metrics, err := RunUntilBugMetrics(targetSchema, opts, budget)
	if err != nil {
		return 0, err
	}
	return metrics.BugStep, nil
}
