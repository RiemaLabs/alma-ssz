package rl

import "math/rand"

// PolicyAgent represents a lightweight bandit-style agent that learns action values.
type PolicyAgent struct {
	actionSize int
	epsilon    float64 // Exploration rate
	minEpsilon float64
	decay      float64
	ucbC       float64
	valueAlpha float64
	IsBaseline bool // New: Flag to indicate if agent is in baseline mode
	NoRL       bool // New: Disable learning while keeping structured actions

	actionCounts []int
	actionValues []float64
	totalActions int
	actionPrior  []float64
}

// NewPolicyAgent creates a new PolicyAgent.
// obsSize is reserved for future contextual features; the current agent is action-value based.
func NewPolicyAgent(actionSize int, isBaseline bool, noRL bool, obsSize int) *PolicyAgent {
	return &PolicyAgent{
		actionSize:   actionSize,
		epsilon:      0.15,
		minEpsilon:   0.02,
		decay:        0.99,
		ucbC:         0.6,
		valueAlpha:   0.2,
		IsBaseline:   isBaseline,
		NoRL:         noRL,
		actionCounts: make([]int, actionSize),
		actionValues: make([]float64, actionSize),
	}
}

// SetActionPrior provides a fixed prior for action scoring.
func (agent *PolicyAgent) SetActionPrior(prior []float64) {
	if len(prior) == 0 {
		return
	}
	agent.actionPrior = make([]float64, len(prior))
	copy(agent.actionPrior, prior)
}

// Act selects an action based on the current observation.
func (agent *PolicyAgent) Act(obs Observation) Action {
	if agent.IsBaseline || agent.NoRL {
		// Baseline and NoRL both sample actions uniformly.
		return Action{ID: rand.Intn(agent.actionSize)}
	}
	// Epsilon-greedy exploration for learning agent
	if rand.Float64() < agent.epsilon {
		// Explore: sample using prior if available.
		if idx, ok := agent.sampleByPrior(); ok {
			return Action{ID: idx}
		}
		return Action{ID: rand.Intn(agent.actionSize)}
	}

	// Exploit: weighted sampling over learned values + prior.
	weights := make([]float64, agent.actionSize)
	sum := 0.0
	for i := 0; i < agent.actionSize; i++ {
		prior := 0.0
		if i < len(agent.actionPrior) {
			prior = agent.actionPrior[i]
		}
		score := agent.actionValues[i] + prior
		if score < 0 {
			score = 0
		}
		weights[i] = score
		sum += score
	}
	if sum == 0 {
		return Action{ID: rand.Intn(agent.actionSize)}
	}
	r := rand.Float64() * sum
	for i, w := range weights {
		r -= w
		if r <= 0 {
			return Action{ID: i}
		}
	}
	return Action{ID: agent.actionSize - 1}
}

func (agent *PolicyAgent) sampleByPrior() (int, bool) {
	if len(agent.actionPrior) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, w := range agent.actionPrior {
		if w > 0 {
			sum += w
		}
	}
	if sum == 0 {
		return 0, false
	}
	r := rand.Float64() * sum
	for i, w := range agent.actionPrior {
		if w <= 0 {
			continue
		}
		r -= w
		if r <= 0 {
			return i, true
		}
	}
	return len(agent.actionPrior) - 1, true
}

// Remember stores an experience in the agent's memory.
func (agent *PolicyAgent) Remember(obs Observation, action Action, reward float64, nextObs Observation, done bool) {
	agent.totalActions++
	idx := action.ID
	if idx < 0 || idx >= agent.actionSize {
		return
	}
	agent.actionCounts[idx]++
	if agent.actionCounts[idx] == 1 {
		agent.actionValues[idx] = reward
		return
	}
	alpha := agent.valueAlpha
	agent.actionValues[idx] = (1.0-alpha)*agent.actionValues[idx] + alpha*reward
}

// ClearMemory clears the agent's memory.
func (agent *PolicyAgent) ClearMemory() {
	for i := range agent.actionCounts {
		agent.actionCounts[i] = 0
		agent.actionValues[i] = 0
	}
	agent.totalActions = 0
	agent.epsilon = 0.25
}

// Learn updates the agent's policy based on experiences in memory.
// This uses epsilon decay for exploration control.
func (agent *PolicyAgent) Learn() {
	if agent.IsBaseline || agent.NoRL {
		return // No learning in baseline mode
	}

	if agent.epsilon > agent.minEpsilon {
		agent.epsilon *= agent.decay
		if agent.epsilon < agent.minEpsilon {
			agent.epsilon = agent.minEpsilon
		}
	}
}
