package sszref

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type fieldRef struct {
	value reflect.Value
	tag   reflect.StructTag
	name  string
}

// Marshal encodes a value into canonical SSZ bytes using reflection.
func Marshal(value interface{}) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("sszref: nil input")
	}
	return encodeValue(reflect.ValueOf(value), tagContext{})
}

func encodeValue(v reflect.Value, ctx tagContext) ([]byte, error) {
	if !v.IsValid() {
		return nil, fmt.Errorf("sszref: invalid value")
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v = reflect.New(v.Type().Elem()).Elem()
			break
		}
		v = v.Elem()
	}

	if isTimeType(v.Type()) {
		return encodeUint64(uint64(v.Interface().(time.Time).Unix())), nil
	}

	switch v.Kind() {
	case reflect.Bool:
		return encodeBool(v.Bool()), nil
	case reflect.Uint8:
		return []byte{byte(v.Uint())}, nil
	case reflect.Uint16:
		return encodeUint16(uint16(v.Uint())), nil
	case reflect.Uint32:
		return encodeUint32(uint32(v.Uint())), nil
	case reflect.Uint64:
		return encodeUint64(v.Uint()), nil
	case reflect.Array:
		return encodeArray(v, ctx)
	case reflect.Slice:
		return encodeSlice(v, ctx)
	case reflect.Struct:
		return encodeStruct(v)
	default:
		return nil, fmt.Errorf("sszref: unsupported kind %s", v.Kind())
	}
}

func encodeStruct(v reflect.Value) ([]byte, error) {
	fields, err := collectFields(v)
	if err != nil {
		return nil, err
	}

	fixedLen := 0
	for _, f := range fields {
		ctx := parseTagContext(f.tag)
		if size, ok := fixedSizeOfValue(f.value, ctx); ok {
			fixedLen += size
		} else {
			fixedLen += 4
		}
	}

	fixed := make([]byte, 0, fixedLen)
	variable := make([][]byte, 0)
	offset := fixedLen

	for _, f := range fields {
		ctx := parseTagContext(f.tag)
		if _, ok := fixedSizeOfValue(f.value, ctx); ok {
			enc, err := encodeValue(f.value, ctx)
			if err != nil {
				return nil, err
			}
			fixed = append(fixed, enc...)
		} else {
			fixed = append(fixed, encodeUint32(uint32(offset))...)
			enc, err := encodeValue(f.value, ctx)
			if err != nil {
				return nil, err
			}
			variable = append(variable, enc)
			offset += len(enc)
		}
	}

	out := make([]byte, 0, offset)
	out = append(out, fixed...)
	for _, part := range variable {
		out = append(out, part...)
	}
	return out, nil
}

func encodeArray(v reflect.Value, ctx tagContext) ([]byte, error) {
	elemType := v.Type().Elem()
	if elemType.Kind() == reflect.Uint8 {
		buf := make([]byte, v.Len())
		for i := 0; i < v.Len(); i++ {
			buf[i] = byte(v.Index(i).Uint())
		}
		return buf, nil
	}

	out := make([]byte, 0)
	elemCtx := ctx.shift()
	for i := 0; i < v.Len(); i++ {
		enc, err := encodeValue(v.Index(i), elemCtx)
		if err != nil {
			return nil, err
		}
		out = append(out, enc...)
	}
	return out, nil
}

func encodeSlice(v reflect.Value, ctx tagContext) ([]byte, error) {
	if ctx.isBitlist {
		return encodeBitlist(v, ctx)
	}

	elemCtx := ctx.shift()
	size, hasSize := ctx.size()
	max, hasMax := ctx.max()

	length := v.Len()
	if hasSize && length != size {
		return nil, fmt.Errorf("sszref: vector length mismatch %d != %d", length, size)
	}
	if !hasSize && hasMax && length > max {
		return nil, fmt.Errorf("sszref: list length %d exceeds max %d", length, max)
	}

	if elemFixedSize, ok := fixedSizeOfType(v.Type().Elem(), elemCtx); ok {
		out := make([]byte, 0, length*elemFixedSize)
		for i := 0; i < length; i++ {
			enc, err := encodeValue(v.Index(i), elemCtx)
			if err != nil {
				return nil, err
			}
			out = append(out, enc...)
		}
		return out, nil
	}

	// Variable-sized elements: offset table + data.
	offset := 4 * length
	fixed := make([]byte, 0, offset)
	variable := make([][]byte, 0, length)
	for i := 0; i < length; i++ {
		fixed = append(fixed, encodeUint32(uint32(offset))...)
		enc, err := encodeValue(v.Index(i), elemCtx)
		if err != nil {
			return nil, err
		}
		variable = append(variable, enc)
		offset += len(enc)
	}
	out := make([]byte, 0, offset)
	out = append(out, fixed...)
	for _, part := range variable {
		out = append(out, part...)
	}
	return out, nil
}

func encodeBitlist(v reflect.Value, ctx tagContext) ([]byte, error) {
	if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
		return nil, fmt.Errorf("sszref: bitlist must be []byte")
	}
	raw := make([]byte, v.Len())
	for i := 0; i < v.Len(); i++ {
		raw[i] = byte(v.Index(i).Uint())
	}
	maxBits, _ := ctx.max()
	if err := validateBitlist(raw, maxBits); err != nil {
		return nil, err
	}
	return raw, nil
}

func collectFields(v reflect.Value) ([]fieldRef, error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v = reflect.New(v.Type().Elem()).Elem()
		} else {
			v = v.Elem()
		}
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("sszref: expected struct, got %s", v.Kind())
	}

	var out []fieldRef
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" || strings.HasPrefix(field.Name, "_") {
			continue
		}
		fv := v.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			nested, err := collectFields(fv)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
			continue
		}
		out = append(out, fieldRef{
			value: fv,
			tag:   field.Tag,
			name:  field.Name,
		})
	}
	return out, nil
}

func encodeBool(val bool) []byte {
	if val {
		return []byte{1}
	}
	return []byte{0}
}

func encodeUint16(val uint16) []byte {
	out := make([]byte, 2)
	binary.LittleEndian.PutUint16(out, val)
	return out
}

func encodeUint32(val uint32) []byte {
	out := make([]byte, 4)
	binary.LittleEndian.PutUint32(out, val)
	return out
}

func encodeUint64(val uint64) []byte {
	out := make([]byte, 8)
	binary.LittleEndian.PutUint64(out, val)
	return out
}

func isTimeType(t reflect.Type) bool {
	return t.PkgPath() == "time" && t.Name() == "Time"
}
