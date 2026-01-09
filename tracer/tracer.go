package tracer

import (
	"hash/fnv"
	"reflect"
	"sync/atomic"
)

// TraceEntry represents a single data point in the execution trace.
type TraceEntry struct {
	CID   uint64
	Value int64
}

// RingBuffer is a simple circular buffer for storing traces.
// We use a power of 2 size for bitwise masking.
const BufferSize = 1024 * 1024

var (
	Buffer [BufferSize]TraceEntry
	Index  uint64
)

// Record captures a single execution point.
// cid: Context ID (hash of location+variable)
// val: The value observed
//
//go:noinline
func Record(cid uint64, val int64) {
	idx := atomic.AddUint64(&Index, 1)
	// Use simple wrapping. Note: idx starts at 1.
	Buffer[(idx-1)%BufferSize] = TraceEntry{CID: cid, Value: val}
}

// Reset clears the trace index.
func Reset() {
	atomic.StoreUint64(&Index, 0)
}

// Snapshot returns the valid part of the buffer.
func Snapshot() []TraceEntry {
	currentIdx := atomic.LoadUint64(&Index)
	if currentIdx == 0 {
		return nil
	}
	if currentIdx > BufferSize {
		return Buffer[:]
	}
	return Buffer[:currentIdx]
}

// ToScalar converts various types to an int64 representation for the tracer.
// This is a helper to avoid complex type checking in the instrumentor.
// It is optimized for speed.
func ToScalar(v any) int64 {
	if v == nil {
		return 0
	}
	// Fast path for common types
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case uint64:
		return int64(val) // bitwise cast essentially
	case int32:
		return int64(val)
	case uint32:
		return int64(val)
	case int16:
		return int64(val)
	case uint16:
		return int64(val)
	case int8:
		return int64(val)
	case uint8:
		return int64(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		return hashSampledString(val)
	case []byte:
		return hashSampledBytes(val)
	}

	// Fallback to reflection for more complex types (arrays, slices, maps)
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
		return int64(rv.Len())
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return 0
		}
		// Dereference pointer? No, might cycle. Just return 1 (exists).
		return 1
	case reflect.Struct:
		// Hash the struct? Too slow. Just return 1.
		return 1
	}

	return 0
}

// Helper to hash strings/bytes to int64 if needed, but currently unused
func hash64(data []byte) int64 {
	h := fnv.New64a()
	h.Write(data)
	return int64(h.Sum64())
}

func hashSampledBytes(b []byte) int64 {
	if len(b) == 0 {
		return 0
	}
	const (
		fnvOffset64 = 1469598103934665603
		fnvPrime64  = 1099511628211
		maxSample   = 8
	)
	h := uint64(fnvOffset64)
	limit := len(b)
	if limit > maxSample {
		limit = maxSample
	}
	for i := 0; i < limit; i++ {
		h ^= uint64(b[i])
		h *= fnvPrime64
	}
	if len(b) > maxSample {
		for i := len(b) - maxSample; i < len(b); i++ {
			h ^= uint64(b[i])
			h *= fnvPrime64
		}
	}
	h ^= uint64(len(b))
	return int64(h)
}

func hashSampledString(s string) int64 {
	if len(s) == 0 {
		return 0
	}
	const (
		fnvOffset64 = 1469598103934665603
		fnvPrime64  = 1099511628211
		maxSample   = 8
	)
	h := uint64(fnvOffset64)
	limit := len(s)
	if limit > maxSample {
		limit = maxSample
	}
	for i := 0; i < limit; i++ {
		h ^= uint64(s[i])
		h *= fnvPrime64
	}
	if len(s) > maxSample {
		for i := len(s) - maxSample; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= fnvPrime64
		}
	}
	h ^= uint64(len(s))
	return int64(h)
}
