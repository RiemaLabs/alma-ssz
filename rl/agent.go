package rl

import (
	"math"
	"math/rand"
)

// PolicyAgent represents an RL agent that learns a policy.
type PolicyAgent struct {
	actionSize int
	policyNet  *NeuralNetwork
	optimizer  *Optimizer
	memory     []*Experience
	gamma      float64 // Discount factor
	epsilon    float64 // Exploration rate
	IsBaseline bool    // New: Flag to indicate if agent is in baseline mode
}

// Experience stores an agent's experience.
type Experience struct {
	Observation Observation
	Action      Action
	Reward      float64
	Next        Observation
	Done        bool
}

// NewPolicyAgent creates a new PolicyAgent.
// obsSize controls the input dimensionality; if zero, the agent defaults to a bias-only policy.
func NewPolicyAgent(actionSize int, isBaseline bool, obsSize int) *PolicyAgent {
	if obsSize <= 0 {
		obsSize = 1 // Avoid zero-sized networks so the agent can still learn a bias
	}
	policyNet := NewNeuralNetwork(obsSize, actionSize) // Input size is the length of the observation vector
	optimizer := NewOptimizer(policyNet, 0.001)        // Learning rate
	return &PolicyAgent{
		actionSize: actionSize,
		policyNet:  policyNet,
		optimizer:  optimizer,
		memory:     make([]*Experience, 0),
		gamma:      0.5,  // Discount factor
		epsilon:    0.9,  // Exploration rate for RL learning
		IsBaseline: isBaseline,
	}
}

// Act selects an action based on the current observation.
func (agent *PolicyAgent) Act(obs Observation) Action {
	if agent.IsBaseline {
		// In baseline mode, always explore (random actions)
		return Action{ID: rand.Intn(agent.actionSize)}
	}
	// Epsilon-greedy exploration for learning agent
	if rand.Float64() < agent.epsilon {
		// Explore: pick a random action ID
		return Action{ID: rand.Intn(agent.actionSize)}
	}

	// Exploit: use the policy network
	probs := agent.policyNet.Forward(obs.Vector)
	actionID := Softmax(probs) // Softmax sampling gives action ID

	return Action{ID: actionID}
}

// Softmax applies the softmax function to a slice of floats and samples an index.
func Softmax(scores []float64) int {
	expScores := make([]float64, len(scores))
	var sumExp float64
	for i, s := range scores {
		expScores[i] = math.Exp(s)
		sumExp += expScores[i]
	}

	probabilities := make([]float64, len(scores))
	for i, es := range expScores {
		probabilities[i] = es / sumExp
	}

	// Sample an action based on probabilities
	r := rand.Float64()
	var cumulativeProb float64
	for i, p := range probabilities {
		cumulativeProb += p
		if r <= cumulativeProb {
			return i
		}
	}
	return len(scores) - 1 // Fallback
}

// Remember stores an experience in the agent's memory.
func (agent *PolicyAgent) Remember(obs Observation, action Action, reward float64, nextObs Observation, done bool) {
	agent.memory = append(agent.memory, &Experience{
		Observation: obs,
		Action:      action,
		Reward:      reward,
		Next:        nextObs,
		Done:        done,
	})
}

// ClearMemory clears the agent's memory.
func (agent *PolicyAgent) ClearMemory() {
	agent.memory = make([]*Experience, 0)
}

// Learn updates the agent's policy based on experiences in memory.
// This is a simplified REINFORCE-like update.
func (agent *PolicyAgent) Learn() {
	if agent.IsBaseline {
		return // No learning in baseline mode
	}

	if len(agent.memory) == 0 {
		return
	}

	// Calculate discounted rewards (returns)
	returns := make([]float64, len(agent.memory))
	var g float64
	for i := len(agent.memory) - 1; i >= 0; i-- {
		exp := agent.memory[i]
		g = exp.Reward + agent.gamma*g // Simple accumulation, not full TD
		returns[i] = g
	}

	// Normalize returns (optional, but often helps stability)
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	std := 0.0
	for _, r := range returns {
		std += math.Pow(r-mean, 2)
	}
	std = math.Sqrt(std/float64(len(returns))) + 1e-8 // Add epsilon for stability

	for i, exp := range agent.memory {
		// Calculate advantage (return - baseline)
		advantage := (returns[i] - mean) / std

		// Update policy network
		agent.optimizer.Train(exp.Observation.Vector, exp.Action.ID, advantage)
	}

	// Decrease epsilon over time for less exploration
	if agent.epsilon > 0.01 {
		agent.epsilon *= 0.999 // Decay rate
	}
}

// Simplified NeuralNetwork (for policy approximation)
type NeuralNetwork struct {
	inputSize  int
	outputSize int
	weights    [][]float64 // Single layer for simplicity
	biases     []float64
}

// NewNeuralNetwork creates a simple feed-forward network.
func NewNeuralNetwork(input, output int) *NeuralNetwork {
	weights := make([][]float64, input)
	for i := range weights {
		weights[i] = make([]float64, output)
		for j := range weights[i] {
			weights[i][j] = rand.NormFloat64() * 0.1 // Small random weights
		}
	}
	biases := make([]float64, output)
	return &NeuralNetwork{input, output, weights, biases}
}

// Forward computes the network output (logits for actions).
func (nn *NeuralNetwork) Forward(input []float64) []float64 {
	output := make([]float64, nn.outputSize)
	for j := 0; j < nn.outputSize; j++ {
		var sum float64
		for i := 0; i < nn.inputSize; i++ {
			sum += input[i] * nn.weights[i][j]
		}
		output[j] = sum + nn.biases[j]
	}
	return output
}

// Optimizer (simplified gradient descent for REINFORCE)
type Optimizer struct {
	net        *NeuralNetwork
	learningRate float64
}

// NewOptimizer creates an optimizer for the policy network.
func NewOptimizer(net *NeuralNetwork, lr float64) *Optimizer {
	return &Optimizer{net, lr}
}

// Train updates network weights based on advantage.
// This is a very simplified update rule, proportional to the advantage and action probability.
func (opt *Optimizer) Train(observation []float64, actionIdx int, advantage float64) {
	// For simplicity, directly adjust weights towards the chosen action
	// This is a heuristic, not a formal gradient calculation for Softmax
	for i := 0; i < opt.net.inputSize; i++ {
		opt.net.weights[i][actionIdx] += opt.learningRate * observation[i] * advantage
	}
	opt.net.biases[actionIdx] += opt.learningRate * advantage
}
