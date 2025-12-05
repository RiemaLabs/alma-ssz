package fuzz

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"alma.local/ssz/internal/corpus"
	"alma.local/ssz/internal/oracle"
	"alma.local/ssz/internal/targets"
)

const seedLimit = 256

func runRoundTripFuzz[T any, PT oracle.RoundTripTarget[T]](f *testing.F, typeName string) {
	// Load all roundtrip targets
	allTargets, err := targets.LoadRoundTripTargets("../config/roundtrip_targets.json")
	if err != nil {
		absConfigPath, absErr := filepath.Abs("../config/roundtrip_targets.json")
		if absErr != nil {
			f.Fatalf("get absolute path error: %v", absErr)
		}
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			f.Fatalf("get cwd error: %v", cwdErr)
		}
		f.Fatalf("load roundtrip targets: read targets config: %v (CWD: %s, Attempted absolute path: %s)", err, cwd, absConfigPath)
	}

	var target targets.RoundTripTarget
	found := false
	for _, t := range allTargets {
		if t.Name == typeName {
			target = t
			found = true
			break
		}
	}
	if !found {
		f.Fatalf("roundtrip target %s not found in config", typeName)
	}

	loader := corpus.NewLoader(corpus.DefaultRoot, seedLimit)
	seeds, err := loader.Collect(target) // Pass the actual target
	if err != nil {
		f.Fatalf("load corpus for %s: %v", typeName, err)
	}
	if len(seeds) == 0 {
		f.Fatalf("no seeds discovered for %s under %s", typeName, corpus.DefaultRoot)
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if err := oracle.RoundTrip[T, PT](data); err != nil {
			if errors.Is(err, oracle.ErrInvalidInput) {
				return
			}
			t.Fatalf("roundtrip oracle failed: %v", err)
		}
	})
}
