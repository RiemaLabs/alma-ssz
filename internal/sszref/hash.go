package sszref

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/bits"
	"reflect"
	"time"
)

// HashTreeRoot computes the SSZ hash tree root of a value using reflection.
func HashTreeRoot(value interface{}) ([32]byte, error) {
	if value == nil {
		return [32]byte{}, fmt.Errorf("sszref: nil input")
	}
	return hashValue(reflect.ValueOf(value), tagContext{})
}

func hashValue(v reflect.Value, ctx tagContext) ([32]byte, error) {
	if !v.IsValid() {
		return [32]byte{}, fmt.Errorf("sszref: invalid value")
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v = reflect.New(v.Type().Elem()).Elem()
			break
		}
		v = v.Elem()
	}

	if isTimeType(v.Type()) {
		return hashUint64(uint64(v.Interface().(time.Time).Unix())), nil
	}

	switch v.Kind() {
	case reflect.Bool:
		return hashBool(v.Bool()), nil
	case reflect.Uint8:
		return hashUint64(v.Uint()), nil
	case reflect.Uint16:
		return hashUint64(v.Uint()), nil
	case reflect.Uint32:
		return hashUint64(v.Uint()), nil
	case reflect.Uint64:
		return hashUint64(v.Uint()), nil
	case reflect.Array:
		return hashArray(v, ctx)
	case reflect.Slice:
		return hashSlice(v, ctx)
	case reflect.Struct:
		return hashStruct(v)
	default:
		return [32]byte{}, fmt.Errorf("sszref: unsupported kind %s", v.Kind())
	}
}

func hashStruct(v reflect.Value) ([32]byte, error) {
	fields, err := collectFields(v)
	if err != nil {
		return [32]byte{}, err
	}

	roots := make([][32]byte, 0, len(fields))
	for _, f := range fields {
		ctx := parseTagContext(f.tag)
		root, err := hashValue(f.value, ctx)
		if err != nil {
			return [32]byte{}, err
		}
		roots = append(roots, root)
	}
	return merkleizeRoots(roots, uint64(len(roots)))
}

func hashArray(v reflect.Value, ctx tagContext) ([32]byte, error) {
	elemType := v.Type().Elem()
	elemCtx := ctx.shift()
	length := v.Len()

	if elemType.Kind() == reflect.Uint8 {
		bytes := make([]byte, length)
		for i := 0; i < length; i++ {
			bytes[i] = byte(v.Index(i).Uint())
		}
		return hashBytesVector(bytes)
	}

	if elemSize, ok := fixedSizeOfType(elemType, elemCtx); ok {
		data := make([]byte, 0, elemSize*length)
		for i := 0; i < length; i++ {
			enc, err := encodeValue(v.Index(i), elemCtx)
			if err != nil {
				return [32]byte{}, err
			}
			data = append(data, enc...)
		}
		return hashPackedBasicVector(data, elemSize, length)
	}

	roots := make([][32]byte, 0, length)
	for i := 0; i < length; i++ {
		root, err := hashValue(v.Index(i), elemCtx)
		if err != nil {
			return [32]byte{}, err
		}
		roots = append(roots, root)
	}
	return merkleizeRoots(roots, uint64(length))
}

func hashSlice(v reflect.Value, ctx tagContext) ([32]byte, error) {
	if ctx.isBitlist {
		return hashBitlist(v, ctx)
	}

	elemCtx := ctx.shift()
	length := v.Len()
	size, hasSize := ctx.size()
	max, hasMax := ctx.max()

	if hasSize && length != size {
		return [32]byte{}, fmt.Errorf("sszref: vector length mismatch %d != %d", length, size)
	}
	if !hasSize && hasMax && length > max {
		return [32]byte{}, fmt.Errorf("sszref: list length %d exceeds max %d", length, max)
	}

	if elemSize, ok := fixedSizeOfType(v.Type().Elem(), elemCtx); ok {
		data := make([]byte, 0, elemSize*length)
		for i := 0; i < length; i++ {
			enc, err := encodeValue(v.Index(i), elemCtx)
			if err != nil {
				return [32]byte{}, err
			}
			data = append(data, enc...)
		}
		if hasSize {
			return hashPackedBasicVector(data, elemSize, size)
		}
		limit := length
		if hasMax {
			limit = max
		}
		root, err := hashPackedBasicList(data, elemSize, limit)
		if err != nil {
			return [32]byte{}, err
		}
		return mixInLength(root, uint64(length)), nil
	}

	roots := make([][32]byte, 0, length)
	for i := 0; i < length; i++ {
		root, err := hashValue(v.Index(i), elemCtx)
		if err != nil {
			return [32]byte{}, err
		}
		roots = append(roots, root)
	}

	if hasSize {
		return merkleizeRoots(roots, uint64(size))
	}
	limit := length
	if hasMax {
		limit = max
	}
	root, err := merkleizeRoots(roots, uint64(limit))
	if err != nil {
		return [32]byte{}, err
	}
	return mixInLength(root, uint64(length)), nil
}

func hashBytesVector(data []byte) ([32]byte, error) {
	chunks := chunkify(data)
	limit := uint64(len(chunks))
	if limit == 0 {
		limit = 1
	}
	return merkleizeChunks(chunks, limit)
}

func hashPackedBasicVector(data []byte, elemSize int, length int) ([32]byte, error) {
	if elemSize <= 0 {
		return [32]byte{}, fmt.Errorf("sszref: invalid element size")
	}
	chunks := chunkify(data)
	limit := calculateLimit(uint64(length), uint64(elemSize))
	return merkleizeChunks(chunks, limit)
}

func hashPackedBasicList(data []byte, elemSize int, limit int) ([32]byte, error) {
	if elemSize <= 0 {
		return [32]byte{}, fmt.Errorf("sszref: invalid element size")
	}
	chunks := chunkify(data)
	chunkLimit := calculateLimit(uint64(limit), uint64(elemSize))
	return merkleizeChunks(chunks, chunkLimit)
}

func hashBitlist(v reflect.Value, ctx tagContext) ([32]byte, error) {
	if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
		return [32]byte{}, fmt.Errorf("sszref: bitlist must be []byte")
	}
	raw := make([]byte, v.Len())
	for i := 0; i < v.Len(); i++ {
		raw[i] = byte(v.Index(i).Uint())
	}
	maxBits, _ := ctx.max()
	if err := validateBitlist(raw, maxBits); err != nil {
		return [32]byte{}, err
	}

	content, sizeBits, err := parseBitlist(raw)
	if err != nil {
		return [32]byte{}, err
	}

	chunks := chunkify(content)
	limit := bitlistChunkLimit(maxBits)
	root, err := merkleizeChunks(chunks, limit)
	if err != nil {
		return [32]byte{}, err
	}
	return mixInLength(root, sizeBits), nil
}

func hashBool(val bool) [32]byte {
	var out [32]byte
	if val {
		out[0] = 1
	}
	return out
}

func hashUint64(val uint64) [32]byte {
	var out [32]byte
	binary.LittleEndian.PutUint64(out[:8], val)
	return out
}

func mixInLength(root [32]byte, length uint64) [32]byte {
	var lenBytes [32]byte
	binary.LittleEndian.PutUint64(lenBytes[:8], length)
	return hashConcat(root[:], lenBytes[:])
}

func hashConcat(left, right []byte) [32]byte {
	sum := sha256.Sum256(append(left, right...))
	return sum
}

func chunkify(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}
	n := (len(data) + 31) / 32
	chunks := make([][]byte, n)
	for i := 0; i < n; i++ {
		chunk := make([]byte, 32)
		start := i * 32
		end := start + 32
		if end > len(data) {
			end = len(data)
		}
		copy(chunk, data[start:end])
		chunks[i] = chunk
	}
	return chunks
}

func merkleizeChunks(chunks [][]byte, limit uint64) ([32]byte, error) {
	if limit == 0 {
		limit = 1
	}
	if uint64(len(chunks)) > limit {
		return [32]byte{}, fmt.Errorf("sszref: chunk count %d exceeds limit %d", len(chunks), limit)
	}
	leafCount := nextPowerOfTwo(limit)
	leaves := make([][32]byte, leafCount)
	for i := 0; i < len(chunks); i++ {
		copy(leaves[i][:], chunks[i])
	}
	for leafCount > 1 {
		next := make([][32]byte, leafCount/2)
		for i := 0; i < int(leafCount); i += 2 {
			next[i/2] = hashConcat(leaves[i][:], leaves[i+1][:])
		}
		leaves = next
		leafCount = uint64(len(leaves))
	}
	return leaves[0], nil
}

func merkleizeRoots(roots [][32]byte, limit uint64) ([32]byte, error) {
	if limit == 0 {
		limit = 1
	}
	if uint64(len(roots)) > limit {
		return [32]byte{}, fmt.Errorf("sszref: root count %d exceeds limit %d", len(roots), limit)
	}
	chunks := make([][]byte, len(roots))
	for i := range roots {
		chunk := make([]byte, 32)
		copy(chunk, roots[i][:])
		chunks[i] = chunk
	}
	return merkleizeChunks(chunks, limit)
}

func nextPowerOfTwo(n uint64) uint64 {
	if n <= 1 {
		return 1
	}
	p := uint64(1)
	for p < n {
		p <<= 1
	}
	return p
}

func calculateLimit(maxItems, elemSize uint64) uint64 {
	limit := (maxItems*elemSize + 31) / 32
	if limit != 0 {
		return limit
	}
	if maxItems == 0 {
		return 1
	}
	return maxItems
}

func bitlistChunkLimit(maxBits int) uint64 {
	if maxBits <= 0 {
		return 1
	}
	return uint64((maxBits + 255) / 256)
}

func parseBitlist(buf []byte) ([]byte, uint64, error) {
	if len(buf) == 0 {
		return nil, 0, fmt.Errorf("sszref: bitlist empty")
	}
	last := buf[len(buf)-1]
	if last == 0 {
		return nil, 0, fmt.Errorf("sszref: bitlist missing length bit")
	}
	msb := uint8(bits.Len8(last)) - 1
	size := uint64(8*(len(buf)-1) + int(msb))

	out := append([]byte(nil), buf...)
	out[len(out)-1] &^= uint8(1 << msb)

	newLen := len(out)
	for i := len(out) - 1; i >= 0; i-- {
		if out[i] != 0x00 {
			newLen = i + 1
			break
		}
		newLen = i
	}
	return out[:newLen], size, nil
}
