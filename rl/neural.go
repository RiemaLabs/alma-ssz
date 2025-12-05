package rl

import (
	"math"
	"math/rand"
)

// Simple neural network helpers

// Vector is a slice of float64
type Vector []float64

// Matrix is a slice of Vectors
type Matrix []Vector

// NewMatrix creates a matrix of size rows x cols initialized with random small weights
func NewMatrix(rows, cols int) Matrix {
	m := make(Matrix, rows)
	for i := 0; i < rows; i++ {
		m[i] = make(Vector, cols)
		for j := 0; j < cols; j++ {
			// Xavier initialization-ish
			m[i][j] = rand.NormFloat64() * math.Sqrt(2.0/float64(cols))
		}
	}
	return m
}

// MatMul multiplies matrix m by vector v
func MatMul(m Matrix, v Vector) Vector {
	rows := len(m)
	cols := len(m[0])
	if len(v) != cols {
		panic("dimension mismatch in MatMul")
	}
	res := make(Vector, rows)
	for i := 0; i < rows; i++ {
		sum := 0.0
		for j := 0; j < cols; j++ {
			sum += m[i][j] * v[j]
		}
		res[i] = sum
	}
	return res
}

// AddVec adds two vectors
func AddVec(v1, v2 Vector) Vector {
	if len(v1) != len(v2) {
		panic("dimension mismatch in AddVec")
	}
	res := make(Vector, len(v1))
	for i := range v1 {
		res[i] = v1[i] + v2[i]
	}
	return res
}

// ReLU activation
func ReLU(v Vector) Vector {
	res := make(Vector, len(v))
	for i, val := range v {
		if val > 0 {
			res[i] = val
		} else {
			res[i] = 0
		}
	}
	return res
}

// SimpleMLP is a 2-layer neural network (Hidden -> Output)
type SimpleMLP struct {
	W1 Matrix // Input -> Hidden
	B1 Vector // Hidden Bias
	W2 Matrix // Hidden -> Output (Score)
	B2 Vector // Output Bias
	
	// Cache for backprop
	inputCache  Vector
	hiddenCache Vector
}

// NewSimpleMLP creates a new MLP
func NewSimpleMLP(inputSize, hiddenSize, outputSize int) *SimpleMLP {
	return &SimpleMLP{
		W1: NewMatrix(hiddenSize, inputSize),
		B1: make(Vector, hiddenSize),
		W2: NewMatrix(outputSize, hiddenSize),
		B2: make(Vector, outputSize),
	}
}

// Forward pass: Input -> Score (Output is size 1 for scoring a single (ctx, action) pair)
func (nn *SimpleMLP) Forward(input Vector) float64 {
	nn.inputCache = input
	
	// Layer 1
	hidden := AddVec(MatMul(nn.W1, input), nn.B1)
	nn.hiddenCache = ReLU(hidden)
	
	// Layer 2 (Output)
	output := AddVec(MatMul(nn.W2, nn.hiddenCache), nn.B2)
	
	// We expect outputSize to be 1 for a "score"
	return output[0]
}

// Train performs a gradient update step.
// Since we are scoring (ctx, action) pairs individually to get logits for a Softmax over actions,
// we actually need to know the gradient of the Loss w.r.t. this specific Score output.
// In REINFORCE, for a chosen action 'a' with probability 'p', the gradient of Log(p) is:
// d(ln p)/d(score_i) = 1 - p  (if i == a)
// d(ln p)/d(score_i) = -p     (if i != a)
// 
// So we pass 'gradOutput' which is d(Reward * ln p)/d(output).
func (nn *SimpleMLP) Train(gradOutput float64, learningRate float64) {
	// Backprop Layer 2
	// dL/dOutput = gradOutput
	// Output = W2 * Hidden + B2
	// dL/dW2 = dL/dOutput * Hidden
	// dL/dB2 = dL/dOutput
	// dL/dHidden = W2.T * dL/dOutput
	
	dOutput := gradOutput
	
	// Gradients for W2, B2
	dW2 := make(Matrix, len(nn.W2))
	for i := range dW2 {
		dW2[i] = make(Vector, len(nn.W2[0]))
	}
	dB2 := make(Vector, len(nn.B2))
	
	// Since Output is size 1 (scalar score), W2 is 1xHidden
	for j := range nn.W2[0] {
		dW2[0][j] = dOutput * nn.hiddenCache[j]
	}
	dB2[0] = dOutput
	
	// Gradients for Hidden
	dHidden := make(Vector, len(nn.hiddenCache))
	for j := range nn.hiddenCache {
		dHidden[j] = nn.W2[0][j] * dOutput
	}
	
	// Backprop ReLU
	for j, val := range nn.hiddenCache {
		if val <= 0 {
			dHidden[j] = 0
		}
	}
	
	// Backprop Layer 1
	// Hidden = W1 * Input + B1
	dW1 := make(Matrix, len(nn.W1))
	for i := range dW1 {
		dW1[i] = make(Vector, len(nn.W1[0]))
	}
	dB1 := make(Vector, len(nn.B1))
	
	for i := range nn.W1 {
		for j := range nn.W1[0] {
			dW1[i][j] = dHidden[i] * nn.inputCache[j]
		}
		dB1[i] = dHidden[i]
	}
	
	// Update Weights (Gradient Ascent on Reward -> W + alpha * grad)
	// Note: usually it's Gradient Descent on Loss (-Reward). 
	// Here we assume gradOutput is direction of improvement.
	
	for i := range nn.W2 {
		for j := range nn.W2[0] {
			nn.W2[i][j] += learningRate * dW2[i][j]
		}
	}
	for i := range nn.B2 {
		nn.B2[i] += learningRate * dB2[i]
	}
	for i := range nn.W1 {
		for j := range nn.W1[0] {
			nn.W1[i][j] += learningRate * dW1[i][j]
		}
	}
	for i := range nn.B1 {
		nn.B1[i] += learningRate * dB1[i]
	}
}
