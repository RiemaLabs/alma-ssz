package domains

// BucketID uniquely identifies a bucket within an aspect's domain.
type BucketID string

// Range defines the numeric bounds of a bucket (inclusive).
type Range struct {
	Min uint64
	Max uint64
}

// Bucket represents a specific abstract value or range.
type Bucket struct {
	ID          BucketID
	Description string
	Range       Range
	Tag         string // e.g., "boundary", "random", "length"
}

// AspectID identifies a particular property of a field (e.g., "Length", "ElementValue").
type AspectID string

// FieldAspect groups buckets that relate to a specific property of a field.
type FieldAspect struct {
	ID          AspectID
	Description string
	Buckets     []Bucket
}

// Domain represents all configurable aspects for a specific field.
type Domain struct {
	FieldName string
	Type      string // e.g., "uint64", "Bitvector[4]", "List[Root]"
	Aspects   []FieldAspect
}

// DomainProvider is an interface to get domains for a schema.
type DomainProvider interface {
	GetDomains(schemaName string) ([]Domain, error)
}
