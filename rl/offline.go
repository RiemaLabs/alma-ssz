package rl

import (
	"fmt"
	"hash/fnv"
	"math"

	"alma.local/ssz/domains"
)

// OfflineProcessor is responsible for the "Offline Stage" of the pipeline,
// primarily generating and managing embeddings for fields, buckets, and global context.
type OfflineProcessor struct {
	// Potentially hold configurations for embedding models, SMT solvers etc.
}

// NewOfflineProcessor creates a new OfflineProcessor.
func NewOfflineProcessor() *OfflineProcessor {
	return &OfflineProcessor{}
}

// hashTextToVector implements Feature Hashing (the "Hashing Trick").
// It maps a string description to a fixed-size vector.
// This preserves semantic similarity for identical strings and is deterministic.
func hashTextToVector(text string, dim int) []float64 {
	vec := make([]float64, dim)
	h := fnv.New32a()
	
	for i := 0; i < dim; i++ {
		h.Reset()
		h.Write([]byte(text))
		h.Write([]byte(fmt.Sprintf("salt_%d", i)))
		val := h.Sum32()
		// Normalize to [-1, 1]
		floatVal := (float64(val) / float64(math.MaxUint32)) * 2.0 - 1.0
		vec[i] = floatVal
	}
	return vec
}

// GenerateEmbeddings simulates the "Offline Stage" (Pipeline 1.3 & 2) by generating
// mock embeddings based on the provided domains.
// In a real system, this would involve:
// - Parsing impl/spec/schema code.
// - Calling LLM/embedding models.
// - Running SMT solvers to refine buckets.
func (op *OfflineProcessor) GenerateEmbeddings(domainsList []domains.Domain) (
	GlobalContextEmbedding,
	map[string]FieldDescriptorEmbedding,
	map[EncodingContextAction]BaseEmbedding, // Changed key type
	error,
) {
	fmt.Println("OfflineProcessor: Generating deterministic feature-hashed embeddings...")

	// Mock GlobalContextEmbedding (v_ctx)
	// In reality, this would be hashed text of the spec/schema definition.
	globalText := "GlobalContext:SSZ_BeaconState_Spec_v1.0"
	globalCtxEmb := hashTextToVector(globalText, d_ctx)

	fieldDescEmbs := make(map[string]FieldDescriptorEmbedding)
	processedFields := make(map[string]struct{})
	baseActionEmbs := make(map[EncodingContextAction]BaseEmbedding) // Changed key type

	encodingCtx := NewEncodingContext(domainsList) // Use from action_space
	if encodingCtx.ActionCount() == 0 {
		return nil, nil, nil, fmt.Errorf("no actions found in domains to generate embeddings")
	}

	for _, encodingCtxAction := range encodingCtx.Actions { // Iterate over EncodingContextAction
		// Generate FieldDescriptorEmbedding
		if _, ok := processedFields[encodingCtxAction.FieldName]; !ok {
			// Hash the field name and type (if we had it handy in Action, but FieldName is unique enough for now)
			fieldText := fmt.Sprintf("Field:%s", encodingCtxAction.FieldName)
			fieldDescEmbs[encodingCtxAction.FieldName] = hashTextToVector(fieldText, d_ctx)
			processedFields[encodingCtxAction.FieldName] = struct{}{}
		}

		// Generate BaseEmbedding for the (field, aspect, bucket) action
		// Hash the description of the bucket
		bucketText := fmt.Sprintf("Bucket:%s|Aspect:%s|ID:%s", encodingCtxAction.FieldName, encodingCtxAction.AspectID, encodingCtxAction.BucketID)
		baseActionEmbs[encodingCtxAction] = hashTextToVector(bucketText, d_base) // Use EncodingContextAction as key
	}

	fmt.Printf("OfflineProcessor: Generated embeddings for %d fields and %d actions.\n", len(fieldDescEmbs), len(baseActionEmbs))
	return globalCtxEmb, fieldDescEmbs, baseActionEmbs, nil
}