package sszref

import (
	"fmt"
	"math/bits"
	"reflect"
)

func fixedSizeOfValue(v reflect.Value, ctx tagContext) (int, bool) {
	return fixedSizeOfType(v.Type(), ctx)
}

func fixedSizeOfType(t reflect.Type, ctx tagContext) (int, bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if isTimeType(t) {
		return 8, true
	}

	switch t.Kind() {
	case reflect.Bool, reflect.Uint8:
		return 1, true
	case reflect.Uint16:
		return 2, true
	case reflect.Uint32:
		return 4, true
	case reflect.Uint64:
		return 8, true
	case reflect.Array:
		elemCtx := ctx.shift()
		elemSize, ok := fixedSizeOfType(t.Elem(), elemCtx)
		if !ok {
			return 0, false
		}
		return elemSize * t.Len(), true
	case reflect.Slice:
		if ctx.isBitlist {
			return 0, false
		}
		size, hasSize := ctx.size()
		if !hasSize {
			return 0, false
		}
		elemCtx := ctx.shift()
		elemSize, ok := fixedSizeOfType(t.Elem(), elemCtx)
		if !ok {
			return 0, false
		}
		return elemSize * size, true
	case reflect.Struct:
		zero := reflect.New(t).Elem()
		fields, err := collectFields(zero)
		if err != nil {
			return 0, false
		}
		total := 0
		for _, f := range fields {
			size, ok := fixedSizeOfValue(f.value, parseTagContext(f.tag))
			if !ok {
				return 0, false
			}
			total += size
		}
		return total, true
	default:
		return 0, false
	}
}

func validateBitlist(buf []byte, maxBits int) error {
	if len(buf) == 0 {
		return fmt.Errorf("sszref: bitlist empty")
	}
	last := buf[len(buf)-1]
	if last == 0 {
		return fmt.Errorf("sszref: bitlist missing length bit")
	}
	if maxBits <= 0 {
		return nil
	}
	maxBytes := (maxBits >> 3) + 1
	if len(buf) > maxBytes {
		return fmt.Errorf("sszref: bitlist length %d exceeds max %d", len(buf), maxBytes)
	}
	msb := bits.Len8(last)
	if msb == 0 {
		return fmt.Errorf("sszref: bitlist missing length bit")
	}
	numBits := 8*(len(buf)-1) + msb - 1
	if numBits > maxBits {
		return fmt.Errorf("sszref: bitlist bits %d exceeds max %d", numBits, maxBits)
	}
	return nil
}
