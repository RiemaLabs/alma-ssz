package spec

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"alma.local/ssz/domains"
)

type GenericAnalyzer struct{}

func NewGenericAnalyzer() *GenericAnalyzer {
	return &GenericAnalyzer{}
}

// GetDomains analyzes a struct instance and returns domains for its fields using reflection.
// Each domain represents a field, and contains multiple aspects (e.g., length, element value).
func (a *GenericAnalyzer) GetDomains(instance interface{}) ([]domains.Domain, error) {
	val := reflect.ValueOf(instance)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	var results []domains.Domain
	lastExported := -1
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath == "" && !strings.HasPrefix(f.Name, "_") {
			lastExported = i
		}
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name
		fieldType := field.Type

		// Skip fields that are generic internal helpers if any (naive check)
		if strings.HasPrefix(fieldName, "_") {
			continue
		}

		domain := domains.Domain{
			FieldName: fieldName,
			Type:      fieldType.String(),
			Aspects:   []domains.FieldAspect{},
		}

		// Assign aspects and their buckets based on Kind
		switch fieldType.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			bitSize := fieldType.Bits()
			domain.Aspects = append(domain.Aspects, domains.FieldAspect{
				ID:          "Value",
				Description: fmt.Sprintf("Value of %s", fieldName),
				Buckets:     GenerateUintBuckets(bitSize),
			})
		case reflect.Bool:
			domain.Aspects = append(domain.Aspects, domains.FieldAspect{
				ID:          "Value",
				Description: fmt.Sprintf("Value of %s", fieldName),
				Buckets:     BoolBuckets,
			})
		case reflect.Array:
			// Fixed-size array (e.g., [32]byte, [4]byte)
			if fieldType.Elem().Kind() == reflect.Uint8 {
				// Treat all byte arrays (including [1]byte bitvectors) with generic byte buckets.
				domain.Aspects = append(domain.Aspects, domains.FieldAspect{
					ID:          "ElementValue",
					Description: fmt.Sprintf("Value of each element in %s", fieldName),
					Buckets:     ByteContentBuckets,
				})
			} else {
				// Array of other things (e.g., [4]Checkpoint) - recursion handled by Concretizer
				domain.Aspects = append(domain.Aspects, domains.FieldAspect{
					ID:          "Default",
					Description: fmt.Sprintf("Recursive default for elements of %s", fieldName),
					Buckets:     ContainerDefaultBucket,
				})
			}
		case reflect.Slice:
			// Dynamic slice (e.g., []byte, []Validator)
			// Length aspect
			sliceLengthBuckets := make([]domains.Bucket, len(SliceLengthBuckets))
			copy(sliceLengthBuckets, SliceLengthBuckets) // Copy to avoid modifying global

			// Resolve MaxLen and other length buckets based on ssz-max tag.
			// For bitlists, ssz-max is in bits, so convert to a byte length cap.
			resolvedMaxLen := uint64(math.MaxUint64) // Default to max possible
			if tag := field.Tag.Get("ssz-max"); tag != "" {
				if m, err := strconv.Atoi(tag); err == nil {
					resolvedMaxLen = uint64(m)
				}
			}
			if field.Tag.Get("ssz") == "bitlist" && resolvedMaxLen != math.MaxUint64 {
				resolvedMaxLen = (resolvedMaxLen + 7) / 8
			}

			// Adjust all slice length buckets based on resolvedMaxLen
			var validLengthBuckets []domains.Bucket
			for i := range sliceLengthBuckets {
				bucket := &sliceLengthBuckets[i] // Work with pointer to modify directly

				// Cap Max of all length buckets
				if bucket.Range.Max > resolvedMaxLen {
					bucket.Range.Max = resolvedMaxLen
				}
				// Ensure Min doesn't exceed new Max
				if bucket.Range.Min > bucket.Range.Max {
					bucket.Range.Min = bucket.Range.Max // Make it a single point or 0-0 range if Max is 0
				}

				// Adjust MaxLen description and range if it's the MaxLen bucket
				if bucket.ID == "MaxLen" {
					bucket.Description = fmt.Sprintf("Max length (%d)", resolvedMaxLen)
				}

				// Only add if the bucket still represents a valid, non-empty range
				if bucket.Range.Min <= bucket.Range.Max && bucket.Range.Max > 0 || (bucket.ID == "Empty" && bucket.Range.Min == 0 && bucket.Range.Max == 0) {
					validLengthBuckets = append(validLengthBuckets, *bucket)
				}
			}

			domain.Aspects = append(domain.Aspects, domains.FieldAspect{
				ID:          "Length",
				Description: fmt.Sprintf("Length of slice %s", fieldName),
				Buckets:     validLengthBuckets,
			})

			// Add Offset Aspect for variable length fields based on SSZ specification
			domain.Aspects = append(domain.Aspects, domains.FieldAspect{
				ID:          "Offset",
				Description: fmt.Sprintf("Offset contiguity manipulation for slice %s", fieldName),
				Buckets: []domains.Bucket{
					{
						ID:          "CanonicalOffset",
						Description: "Generate a valid, contiguous offset (Offset_n = Offset_n-1 + Length_n-1)",
						Range:       domains.Range{Min: 0, Max: 0}, // Signal to Concretizer to generate canonical offset
						Tag:         "canonical",
					},
					{
						ID:          "GapOffset",
						Description: "Generate an offset that creates a gap (violates contiguity: Offset_n != Offset_n-1 + Length_n-1)",
						Range:       domains.Range{Min: 4, Max: 100}, // Signal to Concretizer to generate a gap size between 4 and 100 bytes
						Tag:         "bug",
					},
				},
			})

			if fieldType.Elem().Kind() == reflect.Uint8 {
				// Element content aspect for byte slices
				domain.Aspects = append(domain.Aspects, domains.FieldAspect{
					ID:          "ElementValue",
					Description: fmt.Sprintf("Value of each element in %s", fieldName),
					Buckets:     ByteContentBuckets,
				})
			} else {
				// Slice of structs - recursion handled by Concretizer
				domain.Aspects = append(domain.Aspects, domains.FieldAspect{
					ID:          "Default",
					Description: fmt.Sprintf("Recursive default for elements of %s", fieldName),
					Buckets:     ContainerDefaultBucket,
				})
			}

			if field.Tag.Get("ssz") == "bitlist" {
				domain.Aspects = append(domain.Aspects, domains.FieldAspect{
					ID:          "BitlistSentinel",
					Description: fmt.Sprintf("Sentinel handling for bitlist %s", fieldName),
					Buckets: []domains.Bucket{
						{
							ID:          "Canonical",
							Description: "Keep canonical sentinel bit",
							Range:       domains.Range{Min: 0, Max: 0},
							Tag:         "canonical",
						},
						{
							ID:          "NullSentinel",
							Description: "Force missing sentinel bit (null last byte)",
							Range:       domains.Range{Min: 0, Max: 0},
							Tag:         "bug",
						},
					},
				})
			}
		case reflect.Struct:
			// Default recursion
			domain.Aspects = append(domain.Aspects, domains.FieldAspect{
				ID:          "Default",
				Description: fmt.Sprintf("Recursive default for struct %s", fieldName),
				Buckets:     ContainerDefaultBucket,
			})
		default:
			domain.Aspects = append(domain.Aspects, domains.FieldAspect{
				ID:          "Default",
				Description: fmt.Sprintf("Fallback default for %s", fieldName),
				Buckets:     ContainerDefaultBucket,
			})
		}

		if i == lastExported {
			domain.Aspects = append(domain.Aspects, domains.FieldAspect{
				ID:          "Tail",
				Description: fmt.Sprintf("Trailing bytes after %s", fieldName),
				Buckets:     TailBuckets,
			})
		}

		results = append(results, domain)
	}

	return results, nil
}
