package concretizer

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"

	"alma.local/ssz/domains"
	"alma.local/ssz/encoding"
)

type MutationType int

const (
	MutationValue MutationType = iota
	MutationOffset
	MutationGap // Insert bytes to create gap
)

type Mutation struct {
	Type        MutationType
	FieldName   string
	Value       []byte // Changed to slice for versatility
	OffsetDelta int    // For shifting offset
	GapSize     int    // For creating gaps
}

type Concretizer struct{}

func New() *Concretizer {
	return &Concretizer{}
}

// Concretize populates the struct `target` based on the `matrix` and domain definitions.
func (c *Concretizer) Concretize(target interface{}, matrix *encoding.EncodingMatrix, domainList []domains.Domain) ([]Mutation, error) {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("target must be a pointer to a struct")
	}
	val = val.Elem()

	// Create lookup map for domains
	domainMap := make(map[string]domains.Domain)
	for _, d := range domainList {
		domainMap[d.FieldName] = d
	}

	return c.concretizeStruct(val, matrix, domainMap)
}

func (c *Concretizer) concretizeStruct(structVal reflect.Value, matrix *encoding.EncodingMatrix, domainMap map[string]domains.Domain) ([]Mutation, error) {
	typ := structVal.Type()
	var mutations []Mutation

	for i := 0; i < structVal.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := structVal.Field(i)

		if field.PkgPath != "" { // Skip unexported fields
			continue
		}

		fieldDomain, domainFound := domainMap[field.Name]
		if !domainFound {
			// If field not found in domainsList (e.g., unexported, or analyzer skipped it),
			// default to recursive concretization for complex types or zero for primitives.
			switch fieldVal.Kind() {
			case reflect.Struct, reflect.Array, reflect.Slice:
				c.concretizeStructRecursive(fieldVal)
			default: // Primitive types without a domain get zero value
				if fieldVal.CanSet() {
					fieldVal.Set(reflect.Zero(fieldVal.Type()))
				}
			}
			continue
		}

		selectedAspects := matrix.Selections[field.Name]

		// Handle aspects for the field
		for _, aspect := range fieldDomain.Aspects {
			chosenBucketID, aspectChosen := selectedAspects[aspect.ID]
			var chosenBucket domains.Bucket
			
			if aspectChosen {
				// Find chosenBucket in aspect.Buckets
				for _, b := range aspect.Buckets {
					if b.ID == chosenBucketID {
						chosenBucket = b
						break
					}
				}
				if chosenBucket.ID == "" { // Chosen bucket not found in aspect's buckets
					return nil, fmt.Errorf("bucket ID '%s' not found for aspect '%s' of field '%s'", chosenBucketID, aspect.ID, field.Name)
				}
			} else {
				// Default behavior if aspect not explicitly chosen (e.g. for nested structs or unhandled aspects)
				if len(aspect.Buckets) > 0 {
					chosenBucket = aspect.Buckets[rand.Intn(len(aspect.Buckets))] // Random default for this aspect
				} else {
					chosenBucket = domains.Bucket{ID: "Default", Range: domains.Range{Min: 0, Max: 0}} // Fallback empty bucket
				}
			}
			
			fieldMutations, err := c.applyAspect(fieldVal, aspect.ID, chosenBucket, field, domainMap)
			if err != nil {
				return nil, fmt.Errorf("failed to concretize field %s, aspect %s: %v", field.Name, aspect.ID, err)
			}
			mutations = append(mutations, fieldMutations...)
		}
	}
	return mutations, nil
}

func (c *Concretizer) applyAspect(val reflect.Value, aspectID domains.AspectID, bucket domains.Bucket, fieldStruct reflect.StructField, domainMap map[string]domains.Domain) ([]Mutation, error) {
	var mutations []Mutation
	switch aspectID {
	case "Value":
		if val.Kind() == reflect.Bool {
			mut := c.setBool(val, bucket.Range, fieldStruct.Name)
			if mut != nil {
				mutations = append(mutations, *mut)
			}
		} else {
			c.setUint(val, bucket.Range)
		}
	case "ElementValue":
		// Special handling for dirty padding candidates
		if bucket.ID == "HighRange" {
			// For a dirty padding test, we set a clean value in the struct,
			// and return a mutation to make it dirty after marshalling.
			if err := c.setElementValue(val, domains.Range{Min: 1, Max: 1}); err != nil { // Set a clean 'true' like value
				return nil, err
			}
			// Sample a random dirty byte from the bucket's range
			dirtyByte := uint64(0)
			diff := bucket.Range.Max - bucket.Range.Min
			if diff == 0 {
				dirtyByte = bucket.Range.Min
			} else {
				dirtyByte = bucket.Range.Min + uint64(rand.Intn(int(diff+1)))
			}
			mutations = append(mutations, Mutation{
				Type:      MutationValue,
				FieldName: fieldStruct.Name,
				Value:     []byte{byte(dirtyByte)},
			})
		} else {
			if err := c.setElementValue(val, bucket.Range); err != nil {
				return nil, err
			}
		}
	case "Length":
		if err := c.setLength(val, bucket.Range, fieldStruct, domainMap); err != nil {
			return nil, err
		}
	case "Offset":
		// Handle offset mutations based on bucket range
		if bucket.Range.Min > 0 {
			gapSize := 0
			if bucket.Range.Min == bucket.Range.Max {
				gapSize = int(bucket.Range.Min)
			} else {
				gapSize = int(bucket.Range.Min) + rand.Intn(int(bucket.Range.Max-bucket.Range.Min+1))
			}
			
			mutations = append(mutations, Mutation{
				Type:      MutationGap,
				FieldName: fieldStruct.Name,
				GapSize:   gapSize,
			})
		}
	case "Default": // For structs/arrays of structs, default means recurse
		switch val.Kind() {
		case reflect.Struct:
			return nil, c.concretizeStructRecursive(val)
		case reflect.Array, reflect.Slice:
			// Recurse on elements. Length should have been set by Length aspect.
			for i := 0; i < val.Len(); i++ {
				if val.Index(i).Kind() == reflect.Struct {
					if err := c.concretizeStructRecursive(val.Index(i)); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return mutations, nil
}

// concretizeStructRecursive blindly populates a struct (used for nested fields not in the top-level matrix)
func (c *Concretizer) concretizeStructRecursive(val reflect.Value) error {
	// For MVP, just recursively fill with defaults/randoms using basic ranges
	// A more robust implementation would require passing a sub-matrix or more intelligent default aspect choices.
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		if f.CanSet() {
			switch f.Kind() {
			case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
				c.setUint(f, domains.Range{Min: 0, Max: 100}) // Random small value
			case reflect.Array:
				if f.Type().Elem().Kind() == reflect.Uint8 {
					c.setElementValue(f, domains.Range{Min: 0, Max: 255}) // Random bytes
				}
			case reflect.Struct:
				c.concretizeStructRecursive(f)
			case reflect.Slice:
				// Default slice length and then fill elements
				// Random length up to 4
				length := rand.Intn(4)
				slice := reflect.MakeSlice(f.Type(), length, length)
				f.Set(slice)
				if f.Type().Elem().Kind() == reflect.Uint8 {
					c.setElementValue(f, domains.Range{Min: 0, Max: 255})
				} else if f.Type().Elem().Kind() == reflect.Struct {
					for i := 0; i < length; i++ {
						c.concretizeStructRecursive(slice.Index(i))
					}
				}
			}
		}
	}
	return nil
}

func (c *Concretizer) setUint(val reflect.Value, r domains.Range) {
	var sample uint64
	if r.Min == r.Max {
		sample = r.Min
	} else {
		if r.Min == 0 && r.Max == math.MaxUint64 {
			sample = rand.Uint64()
		} else {
			diff := r.Max - r.Min
			if diff == math.MaxUint64 { // diff + 1 would overflow if diff is MaxUint64
				sample = r.Min + rand.Uint64() // Directly add a random 64-bit number
			} else {
				// Use rand.Int63n for smaller ranges, adjust for positive range.
				// If diff+1 exceeds MaxInt64, Int63n cannot be used.
				if diff < math.MaxInt64 { 
					sample = r.Min + uint64(rand.Int63n(int64(diff+1)))
				} else { 
					// For ranges between MaxInt64 and MaxUint64, use modulo from rand.Uint64()
					sample = r.Min + (rand.Uint64() % (diff + 1))
				}
			}
		}
	}
	val.SetUint(sample)
}

func (c *Concretizer) setBool(val reflect.Value, r domains.Range, fieldName string) *Mutation {
	// Standard Boolean Logic
	if r.Min == 0 && r.Max == 0 {
		val.SetBool(false)
		return nil
	} else if r.Min == 1 && r.Max == 1 {
		val.SetBool(true)
		return nil
	} 
	
	// Dirty Boolean Logic (Min > 1)
	if r.Min > 1 {
		// We set the boolean to true/false arbitrarily (usually false so 0x00 -> 0xDirty is a change)
		// but more importantly, we return a Mutation to override the byte.
		val.SetBool(false) // Placeholder
		
		// Sample a random dirty byte from the range
		dirtyByte := uint64(0)
		diff := r.Max - r.Min
		if diff == 0 {
			dirtyByte = r.Min
		} else {
			dirtyByte = r.Min + uint64(rand.Intn(int(diff+1)))
		}
		return &Mutation{Type: MutationValue, FieldName: fieldName, Value: []byte{byte(dirtyByte)}}
	}
	
	// Default fallback
	val.SetBool(false)
	return nil
}

// setElementValue handles setting the content for arrays/slices of bytes or individual bytes
func (c *Concretizer) setElementValue(val reflect.Value, r domains.Range) error {
	switch val.Kind() {
	case reflect.Uint8: // Individual byte
		c.setUint(val, r)
	case reflect.Array: // Array of bytes
		if val.Type().Elem().Kind() == reflect.Uint8 {
			for i := 0; i < val.Len(); i++ {
				c.setUint(val.Index(i), r)
			}
		} else {
			return fmt.Errorf("setElementValue not supported for array of type %s", val.Type().Elem().Kind())
		}
	case reflect.Slice: // Slice of bytes
		if val.Type().Elem().Kind() == reflect.Uint8 {
			for i := 0; i < val.Len(); i++ {
				c.setUint(val.Index(i), r)
			}
		} else {
			return fmt.Errorf("setElementValue not supported for slice of type %s", val.Type().Elem().Kind())
		}
	default:
		return fmt.Errorf("setElementValue not supported for kind %s", val.Kind())
	}
	return nil
}

func (c *Concretizer) setLength(val reflect.Value, r domains.Range, fieldStruct reflect.StructField, domainMap map[string]domains.Domain) error {
	if val.Kind() != reflect.Slice {
		return fmt.Errorf("setLength can only be applied to slices, got %s", val.Kind())
	}

	fixedSize := -1 // For Vectors, fixed size is read from ssz-size
	if tag := fieldStruct.Tag.Get("ssz-size"); tag != "" {
		if s, err := strconv.Atoi(tag); err == nil {
			fixedSize = s
		}
	}

	length := 0
	if fixedSize != -1 {
		length = fixedSize // Fixed length
	} else {
		// Sample length from the bucket's range for dynamic slices
		var sampleLen uint64
		if r.Min == r.Max {
			sampleLen = r.Min
		} else {
			// Similar sampling as setUint
			if r.Min == 0 && r.Max == math.MaxUint64 {
				sampleLen = rand.Uint64()
			} else {
				diff := r.Max - r.Min
				if diff == math.MaxUint64 {
					sampleLen = r.Min + rand.Uint64()
				} else {
					sampleLen = r.Min + uint64(rand.Int63n(int64(diff+1)))
				}
			}
		}
		length = int(sampleLen)
		
		// Cap length for MVP to avoid huge allocs
		if length > 1024 { // Cap to 1024 to prevent out-of-memory for very large random lengths
			length = 1024 
		}
	}

	// Make slice with chosen length
	slice := reflect.MakeSlice(val.Type(), length, length)
	val.Set(slice)

	// After setting length, populate elements based on their Aspect choices.
	// For MVP, if elements are structs, recursively concretize them.
	// If elements are primitive (e.g. []byte), ElementValue aspect will fill them.
	if val.Type().Elem().Kind() == reflect.Struct {
		for i := 0; i < length; i++ {
			if err := c.concretizeStructRecursive(slice.Index(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}