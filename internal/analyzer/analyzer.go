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
	Model       map[uint64]*Histogram
	totalEvents uint64
	mu          sync.RWMutex
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		Model:       make(map[uint64]*Histogram),
		totalEvents: 0,
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

// ScoreTrace computes the KL divergence of a single trace against the global model.
// It also updates the model if `update` is true.
func (a *Analyzer) ScoreTrace(trace []TraceEntry, update bool) float64 {
	return a.ScoreBatch([][]TraceEntry{trace}, update)
}

// ScoreBatch computes the KL divergence of a batch of traces against the global model.
// The distribution is defined over (CID, value) pairs observed in the batch.
// It also updates the model if `update` is true.
func (a *Analyzer) ScoreBatch(traces [][]TraceEntry, update bool) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	type traceKey struct {
		CID   uint64
		Value int64
	}

	batchCounts := make(map[traceKey]uint64)
	var totalBatch uint64
	for _, trace := range traces {
		for _, entry := range trace {
			key := traceKey{CID: entry.CID, Value: entry.Value}
			batchCounts[key]++
			totalBatch++
		}
	}

	if totalBatch == 0 {
		return 0.0
	}

	const alpha = 1.0
	denomHist := float64(a.totalEvents) + alpha*float64(len(batchCounts))
	kl := 0.0

	for key, cnt := range batchCounts {
		pBatch := float64(cnt) / float64(totalBatch)
		var histCount uint64
		if hist, ok := a.Model[key.CID]; ok {
			histCount = hist.Counts[key.Value]
		}
		pHist := (float64(histCount) + alpha) / denomHist
		kl += pBatch * math.Log(pBatch/pHist)
	}

	if update {
		for key, cnt := range batchCounts {
			hist, ok := a.Model[key.CID]
			if !ok {
				hist = NewHistogram()
				a.Model[key.CID] = hist
			}
			hist.Counts[key.Value] += cnt
			hist.Total += cnt
		}
		a.totalEvents += totalBatch
	}

	return kl
}

// TraceEntry duplicate from tracer to avoid cyclic imports if we were in same package
// But here we are in `internal/analyzer`.
type TraceEntry struct {
	CID   uint64
	Value int64
}
