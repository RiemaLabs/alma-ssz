package fuzz

import (
	"errors"
	"testing"

	"alma-ssz/internal/corpus"
	"alma-ssz/internal/oracle"
)

const seedLimit = 256

func runRoundTripFuzz[T any, PT oracle.RoundTripTarget[T]](f *testing.F, typeName string) {
	loader := corpus.NewLoader(corpus.DefaultRoot, seedLimit)
	seeds, err := loader.Collect(typeName)
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
