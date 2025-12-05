package rl

import (
	"encoding/binary"
	"math/rand"
	"reflect"
	"strconv"

	"alma.local/ssz/concretizer"
)

// ApplyMutations modifies the serialized SSZ bytes based on the mutations list.
func ApplyMutations(sszBytes []byte, mutations []concretizer.Mutation, targetSchema interface{}) ([]byte, error) {
	if len(mutations) == 0 {
		return sszBytes, nil
	}

	mutatedBytes := make([]byte, len(sszBytes))
	copy(mutatedBytes, sszBytes)

	val := reflect.ValueOf(targetSchema)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	// 1. Map each field to its location in the Fixed Part
	type FieldInfo struct {
		FixedPartOffset int
		IsVariable      bool
		Name            string
	}

	fieldInfos := []FieldInfo{}
	currentFixedOffset := 0

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		isVar := false
		size := 0

		sszSizeTag := fieldType.Tag.Get("ssz-size")

		if field.Kind() == reflect.Slice {
			if sszSizeTag != "" {
				n, _ := strconv.Atoi(sszSizeTag)
				elemSize := guessFixedSizeByType(field.Type().Elem())
				if elemSize > 0 {
					size = n * elemSize
					isVar = false
				} else {
					size = n * 4
					isVar = false
				}
			} else {
				isVar = true
				size = 4
			}
		} else {
			size = guessFixedSizeByType(field.Type())
			if size == -1 {
				isVar = true
				size = 4
			}
		}

		fieldInfos = append(fieldInfos, FieldInfo{
			FixedPartOffset: currentFixedOffset,
			IsVariable:      isVar,
			Name:            fieldType.Name,
		})

		currentFixedOffset += size
	}

	// Apply Mutations
	for _, m := range mutations {
		if m.Type == concretizer.MutationGap && m.GapSize > 0 {
			// Find the first variable field to insert the gap before.
			// This is the specific trigger for the Container Gap bug.
			var firstVarField *FieldInfo
			for i := range fieldInfos {
				if fieldInfos[i].IsVariable {
					firstVarField = &fieldInfos[i]
					break
				}
			}

			if firstVarField != nil {
				ptrOffset := firstVarField.FixedPartOffset
				if ptrOffset+4 > len(mutatedBytes) {
					continue
				}
				currentHeapOffset := int(binary.LittleEndian.Uint32(mutatedBytes[ptrOffset:]))

				gap := make([]byte, m.GapSize)
				rand.Read(gap)

				if currentHeapOffset > len(mutatedBytes) {
					currentHeapOffset = len(mutatedBytes)
				}
				
				newBytes := make([]byte, 0, len(mutatedBytes)+m.GapSize)
				newBytes = append(newBytes, mutatedBytes[:currentHeapOffset]...)
				newBytes = append(newBytes, gap...)
				newBytes = append(newBytes, mutatedBytes[currentHeapOffset:]...)
				mutatedBytes = newBytes

				// Update ALL variable field offsets
				for _, f := range fieldInfos {
					if f.IsVariable {
						pOff := f.FixedPartOffset
						if pOff+4 <= len(mutatedBytes) {
							oldP := binary.LittleEndian.Uint32(mutatedBytes[pOff:])
							// All pointers are shifted by the gap size
							binary.LittleEndian.PutUint32(mutatedBytes[pOff:], oldP+uint32(m.GapSize))
						}
					}
				}
				// Only apply one gap mutation per execution for simplicity
				break
			}
		} else if m.Type == concretizer.MutationValue {
			// (Value mutation logic can be added here if needed for other bugs)
		}
	}

	return mutatedBytes, nil
}

// guessFixedSizeByType returns the size of the type in the Fixed Part.
func guessFixedSizeByType(typ reflect.Type) int {
	kind := typ.Kind()
	switch kind {
	case reflect.Bool, reflect.Uint8:
		return 1
	case reflect.Uint16:
		return 2
	case reflect.Uint32:
		return 4
	case reflect.Uint64:
		return 8
	case reflect.Array:
		elemSize := guessFixedSizeByType(typ.Elem())
		if elemSize > 0 {
			return elemSize * typ.Len()
		}
		return -1
	case reflect.Struct:
		sum := 0
		for i := 0; i < typ.NumField(); i++ {
			if typ.Field(i).PkgPath != "" {
				continue
			}
			s := guessFixedSizeByType(typ.Field(i).Type)
			if s == -1 {
				return -1
			}
			sum += s
		}
		return sum
	case reflect.Slice:
		return -1
	default:
		return -1
	}
}