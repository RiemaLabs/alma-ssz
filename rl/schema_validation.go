package rl

import (
	"fmt"
	"math/rand"
	"time"

	"alma.local/ssz/oracle/pyssz"
)

// SchemaValidationKind identifies a schema-parameter validation category.
type SchemaValidationKind int

const (
	SchemaVectorLen SchemaValidationKind = iota
	SchemaBitvectorLen
	SchemaListMaxLen
)

// SchemaValidationCase describes a schema validation benchmark.
type SchemaValidationCase struct {
	Name string
	Kind SchemaValidationKind
}

// SchemaValidationOpts configures schema validation fuzzing.
type SchemaValidationOpts struct {
	CaseName       string
	BatchSize      int
	MaxSteps       int
	IsBaseline     bool
	NoRL           bool
	ExternalOracle string
	ExternalBug    string
	D_ctx          int
}

type lengthBucket struct {
	ID  string
	Min uint64
	Max uint64
	Tag string
}

var schemaLengthBuckets = []lengthBucket{
	{ID: "Zero", Min: 0, Max: 0, Tag: "zero"},
	{ID: "One", Min: 1, Max: 1, Tag: "min"},
	{ID: "Two", Min: 2, Max: 2, Tag: "small"},
	{ID: "Small", Min: 3, Max: 4, Tag: "small"},
	{ID: "Mid", Min: 5, Max: 16, Tag: "mid"},
	{ID: "Large", Min: 17, Max: 64, Tag: "large"},
	{ID: "XLarge", Min: 65, Max: 256, Tag: "large"},
}

func lookupSchemaValidationCase(name string) (SchemaValidationCase, error) {
	switch name {
	case "PSSZ-111":
		return SchemaValidationCase{Name: name, Kind: SchemaVectorLen}, nil
	case "PSSZ-112":
		return SchemaValidationCase{Name: name, Kind: SchemaBitvectorLen}, nil
	case "PSSZ-116":
		return SchemaValidationCase{Name: name, Kind: SchemaListMaxLen}, nil
	default:
		return SchemaValidationCase{}, fmt.Errorf("unknown schema validation case: %s", name)
	}
}

func expectedSchemaValid(kind SchemaValidationKind, length uint64) bool {
	switch kind {
	case SchemaVectorLen, SchemaBitvectorLen:
		return length > 0
	case SchemaListMaxLen:
		return true
	default:
		return true
	}
}

func buildSchemaLengthPrior() []float64 {
	prior := make([]float64, len(schemaLengthBuckets))
	for i, b := range schemaLengthBuckets {
		score := 0.0
		if b.Min == 0 && b.Max == 0 {
			score += 4.0
		}
		if b.Min == 1 && b.Max == 1 {
			score += 2.0
		}
		if b.Tag == "small" {
			score += 1.0
		}
		prior[i] = score
	}
	return prior
}

func sampleLength(bucket lengthBucket) uint64 {
	if bucket.Max <= bucket.Min {
		return bucket.Min
	}
	diff := bucket.Max - bucket.Min
	return bucket.Min + uint64(rand.Intn(int(diff+1)))
}

// RunSchemaValidationMetrics runs schema validation fuzzing and returns measurements.
func RunSchemaValidationMetrics(opts SchemaValidationOpts, budget time.Duration) (RunMetrics, error) {
	caseDef, err := lookupSchemaValidationCase(opts.CaseName)
	if err != nil {
		return RunMetrics{}, err
	}
	if opts.ExternalOracle != "pyssz" {
		return RunMetrics{}, fmt.Errorf("schema validation requires pyssz oracle")
	}

	if opts.BatchSize <= 0 {
		opts.BatchSize = 1
	}
	if opts.MaxSteps <= 0 {
		opts.MaxSteps = 1
	}

	oracle, err := pyssz.NewOracle(caseDef.Name, opts.ExternalBug)
	if err != nil {
		return RunMetrics{}, fmt.Errorf("failed to initialize py-ssz oracle: %w", err)
	}
	defer oracle.Close()

	obs := Observation{Vector: make([]float64, 1)}
	agent := NewPolicyAgent(len(schemaLengthBuckets), opts.IsBaseline, opts.NoRL, opts.D_ctx)
	agent.SetActionPrior(buildSchemaLengthPrior())

	seen := make(map[uint64]struct{})
	start := time.Now()
	steps := 0

	for steps < opts.MaxSteps {
		if time.Since(start) > budget {
			return RunMetrics{
				BugFound: false,
				Duration: time.Since(start),
				Coverage: float64(len(seen)),
				Steps:    steps,
			}, ErrBudgetExceeded
		}

		chosen := agent.Act(obs)
		if chosen.ID < 0 || chosen.ID >= len(schemaLengthBuckets) {
			chosen.ID = rand.Intn(len(schemaLengthBuckets))
		}
		bucket := schemaLengthBuckets[chosen.ID]

		batchNew := 0
		bugFound := false
		for i := 0; i < opts.BatchSize; i++ {
			length := sampleLength(bucket)
			if _, ok := seen[length]; !ok {
				seen[length] = struct{}{}
				batchNew++
			}
			result, err := oracle.SchemaCheck(caseDef.Name, length)
			if err != nil {
				return RunMetrics{}, fmt.Errorf("schema check failed: %w", err)
			}
			expected := expectedSchemaValid(caseDef.Kind, length)
			if result.OK != expected {
				bugFound = true
				break
			}
		}

		reward := float64(batchNew)
		nextObs := Observation{Vector: []float64{float64(len(seen))}}
		agent.Remember(obs, chosen, reward, nextObs, bugFound)
		agent.Learn()
		obs = nextObs
		steps++

		if bugFound {
			return RunMetrics{
				BugFound: true,
				BugStep:  steps,
				Duration: time.Since(start),
				Coverage: float64(len(seen)),
				Steps:    steps,
			}, nil
		}
	}

	return RunMetrics{
		BugFound: false,
		Duration: time.Since(start),
		Coverage: float64(len(seen)),
		Steps:    steps,
	}, ErrBudgetExceeded
}
