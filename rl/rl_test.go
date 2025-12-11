package rl

import (
	"testing"

	"alma.local/ssz/domains"
)

// MockSchema for testing
type MockSchema struct {
	ValA uint64
	ValB uint64
}

func TestEncodingContext(t *testing.T) {
	// 1. Setup Mock Domains
	doms := []domains.Domain{
		{
			FieldName: "ValA",
			Type:      "uint64",
			Aspects: []domains.FieldAspect{
				{
					ID: "Value",
					Buckets: []domains.Bucket{
						{ID: "Small", Range: domains.Range{Min: 0, Max: 10}},
						{ID: "Large", Range: domains.Range{Min: 11, Max: 100}},
					},
				},
			},
		},
	}

	// 2. Test Context Creation
	ctx := NewEncodingContext(doms)
	if ctx.ActionCount() != 2 {
		t.Errorf("Expected 2 actions, got %d", ctx.ActionCount())
	}

	// 3. Test Indexing
	act, err := ctx.GetActionByIndex(0)
	if err != nil {
		t.Fatalf("Failed to get action 0: %v", err)
	}
	if act.BucketID != "Small" && act.BucketID != "Large" {
		t.Errorf("Unexpected bucket ID: %s", act.BucketID)
	}

	idx, err := ctx.GetIndexByAction(act)
	if err != nil {
		t.Fatalf("Failed to get index for action: %v", err)
	}
	if idx != 0 {
		t.Errorf("Expected index 0, got %d", idx)
	}
}

func TestPolicyAgent_ChooseAction(t *testing.T) {
	t.Skip("PolicyAgent API was simplified; skipping legacy ChooseAction test")
}
