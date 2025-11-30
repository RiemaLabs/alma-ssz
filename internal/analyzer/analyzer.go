package analyzer

import (
	"math"
	"sync"
)

// Histogram represents the distribution of values for a specific CID.
type Histogram struct {
	Counts map[int64]uint64
	Total  uint64
}

func NewHistogram() *Histogram {
	return &Histogram{
		Counts: make(map[int64]uint64),
		Total:  0,
	}
}

// Add updates the histogram with a new value.
func (h *Histogram) Add(val int64) {
	h.Counts[val]++
	h.Total++
}

// Probability calculates P(val) given the history.
// We use smoothing to avoid 0 probabilities.
func (h *Histogram) Probability(val int64) float64 {
	if h.Total == 0 {
		return 0.0
	}
	count := h.Counts[val]
	// Simple Laplace smoothing? Or just raw probability.
	// Let's use raw for now, but handle the "new value" case in the Divergence check.
	return float64(count) / float64(h.Total)
}

// Analyzer manages the global statistical model.
type Analyzer struct {
	Model map[uint64]*Histogram
	mu    sync.RWMutex
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		Model: make(map[uint64]*Histogram),
	}
}

// GetTotalDimensions returns the number of unique CIDs seen so far.
func (a *Analyzer) GetTotalDimensions() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.Model)
}

// GetDimensions returns all CIDs.
func (a *Analyzer) GetDimensions() []uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	keys := make([]uint64, 0, len(a.Model))
	for k := range a.Model {
		keys = append(keys, k)
	}
	return keys
}

// ScoreTrace computes the KL Divergence (or simple surprise score) of a trace
// against the global model.
// It returns a score (higher is more interesting).
// It also updates the model if `update` is true.
func (a *Analyzer) ScoreTrace(trace []TraceEntry, update bool) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	totalSurprise := 0.0

	for _, entry := range trace {
		hist, exists := a.Model[entry.CID]
		if !exists {
			// New Path! Highly interesting.
			totalSurprise += 100.0
			if update {
				h := NewHistogram()
				h.Add(entry.Value)
				a.Model[entry.CID] = h
			}
			continue
		}

		// Existing path, check value divergence.
		prob := hist.Probability(entry.Value)
		if prob < 0.01 {
			// Rare value!
			// Self-information: I(x) = -log2(P(x))
			// If prob is 0 (first time seeing this value), set prob to small epsilon
			if prob == 0 {
				prob = 0.0001
			}
			surprise := -math.Log2(prob)
			totalSurprise += surprise
		}

		if update {
			hist.Add(entry.Value)
		}
	}

	return totalSurprise
}

// TraceEntry duplicate from tracer to avoid cyclic imports if we were in same package
// But here we are in `internal/analyzer`.
type TraceEntry struct {
	CID   uint64
	Value int64
}
