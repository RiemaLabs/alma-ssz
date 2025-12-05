package encoding

import "alma.local/ssz/domains"

// SelectedAspects maps AspectID -> SelectedBucketID for a single field.
type SelectedAspects map[domains.AspectID]domains.BucketID

// EncodingMatrix maps FieldName -> SelectedAspects for the entire schema.
type EncodingMatrix struct {
	SchemaName string
	// Map FieldName -> SelectedAspects
	Selections map[string]SelectedAspects
}

// NewEncodingMatrix creates an empty matrix.
func NewEncodingMatrix(schemaName string) *EncodingMatrix {
	return &EncodingMatrix{
		SchemaName: schemaName,
		Selections: make(map[string]SelectedAspects),
	}
}

// Select sets the bucket for a specific aspect of a field.
func (m *EncodingMatrix) Select(field string, aspect domains.AspectID, bucket domains.BucketID) {
	if _, ok := m.Selections[field]; !ok {
		m.Selections[field] = make(SelectedAspects)
	}
	m.Selections[field][aspect] = bucket
}

// Get returns the selected bucket for a specific aspect of a field.
func (m *EncodingMatrix) Get(field string, aspect domains.AspectID) domains.BucketID {
	if sa, ok := m.Selections[field]; ok {
		return sa[aspect]
	}
	return ""
}