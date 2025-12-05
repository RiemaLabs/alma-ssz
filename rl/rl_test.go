package rl

import (
	"testing"

	"alma.local/ssz/domains"
	"alma.local/ssz/feedback"
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
	// 1. Setup Mock Context
	doms := []domains.Domain{
		{
			FieldName: "ValA",
			Type:      "uint64",
			Aspects: []domains.FieldAspect{
				{
					ID: "Value",
					Buckets: []domains.Bucket{
						{ID: "B1", Range: domains.Range{Min: 0, Max: 1}},
						{ID: "B2", Range: domains.Range{Min: 2, Max: 3}},
					},
				},
			},
		},
	}
	encodingCtx := NewEncodingContext(doms)

	// 2. Generate Embeddings (using OfflineProcessor logic)
	op := NewOfflineProcessor()
	gCtx, fDesc, bAct, err := op.GenerateEmbeddings(doms)
	if err != nil {
		t.Fatalf("Offline processing failed: %v", err)
	}

	// 3. Create Agent
	agent := NewPolicyAgent(gCtx, fDesc, bAct)

	// 4. Create Dummy State
	state := NewState(feedback.RuntimeSignature{}, []float64{}, 0.0, 0.0, make([]float64, d_ctx))

	// 5. Choose Action
	actions, err := agent.ChooseAction(state, encodingCtx, 5)
	if err != nil {
		t.Fatalf("ChooseAction failed: %v", err)
	}

	if len(actions) != 5 {
		t.Errorf("Expected 5 actions, got %d", len(actions))
	}
	for _, a := range actions {
		if a.FieldName != "ValA" {
			t.Errorf("Expected field ValA, got %s", a.FieldName)
		}
	}
}
