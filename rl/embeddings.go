package rl

// This file defines types and constants related to embeddings and dimensionality.

// d_ctx is the dimensionality of the global context embedding.
const d_ctx = 100 

// d_base is the dimensionality of the base action embeddings.
const d_base = 50 

// GlobalContextEmbedding represents the embedding of the overall fuzzer context (v_ctx).
type GlobalContextEmbedding []float64

// FieldDescriptorEmbedding represents the embedding of a specific field (v_field).
type FieldDescriptorEmbedding []float64

// BaseEmbedding represents the embedding of a base action (v_base_a).
type BaseEmbedding []float64