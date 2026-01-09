package rl

import (
	"fmt"

	"alma.local/ssz/domains"
)

// This file defines types and functions for managing the action space of the RL fuzzer.

// Action represents a selected bucket for a specific aspect of a field.
// This type is used internally by EncodingContext to represent discrete actions.
// This is *not* the RL agent's Action output, which is defined in rl/agent.go.
type EncodingContextAction struct { // Renamed from Action to avoid conflict
	FieldName string
	AspectID  domains.AspectID
	BucketID  domains.BucketID
	Tag       string
}

// EncodingContext holds all possible actions for an SSZ schema, derived from the analyzer's output.
// It maps between the structured domain representation and a flat index for the RL agent's policy.
type EncodingContext struct {
	Actions []EncodingContextAction // A flattened list of all possible (field, aspect, bucket) combinations
	// Lookup maps to quickly find action index from (field, aspect, bucket) and vice versa.
	actionToIndex map[string]int
	indexToAction []EncodingContextAction
}

// NewEncodingContext creates an EncodingContext from the analyzer's domain list.
func NewEncodingContext(domains []domains.Domain) *EncodingContext {
	actionIndex := 0
	ctx := &EncodingContext{
		actionToIndex: make(map[string]int),
		indexToAction: make([]EncodingContextAction, 0),
	}

	for _, d := range domains {
		for _, aspect := range d.Aspects {
			for _, bucket := range aspect.Buckets {
				action := EncodingContextAction{ // Use EncodingContextAction
					FieldName: d.FieldName,
					AspectID:  aspect.ID,
					BucketID:  bucket.ID,
					Tag:       bucket.Tag,
				}
				key := fmt.Sprintf("%s_%s_%s", action.FieldName, action.AspectID, action.BucketID)
				ctx.actionToIndex[key] = actionIndex
				ctx.indexToAction = append(ctx.indexToAction, action)
				actionIndex++
			}
		}
	}
	ctx.Actions = ctx.indexToAction // Initialize Actions slice with the flattened list
	return ctx
}

// ActionCount returns the total number of possible actions.
func (ec *EncodingContext) ActionCount() int {
	return len(ec.Actions)
}

// GetActionByIndex retrieves an EncodingContextAction by its flat index.
func (ec *EncodingContext) GetActionByIndex(index int) (EncodingContextAction, error) {
	if index < 0 || index >= len(ec.indexToAction) {
		return EncodingContextAction{}, fmt.Errorf("action index out of bounds: %d", index)
	}
	return ec.indexToAction[index], nil
}

// GetIndexByAction retrieves the flat index for a given EncodingContextAction.
func (ec *EncodingContext) GetIndexByAction(action EncodingContextAction) (int, error) {
	key := fmt.Sprintf("%s_%s_%s", action.FieldName, action.AspectID, action.BucketID)
	index, found := ec.actionToIndex[key]
	if !found {
		return -1, fmt.Errorf("action not found in encoding context: %v", action)
	}
	return index, nil
}
