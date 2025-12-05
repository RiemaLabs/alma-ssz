package tracer

import (
	"testing"
)

func TestRecordAndSnapshot(t *testing.T) {
	// Reset the tracer before the test
	Reset()

	// Record some dummy data
	Record(1, 100)
	Record(2, 200)
	Record(3, 300)

	// Get a snapshot of the recorded data
	snapshot := Snapshot()

	// Check if the snapshot has the expected length
	if len(snapshot) != 3 {
		t.Errorf("Expected snapshot length of 3, but got %d", len(snapshot))
	}

	// Check if the recorded data is correct
	expected := []TraceEntry{
		{CID: 1, Value: 100},
		{CID: 2, Value: 200},
		{CID: 3, Value: 300},
	}

	for i, entry := range snapshot {
		if entry.CID != expected[i].CID || entry.Value != expected[i].Value {
			t.Errorf("Snapshot entry %d is incorrect. Expected %+v, but got %+v", i, expected[i], entry)
		}
	}
}

func TestReset(t *testing.T) {
	// Reset the tracer
	Reset()

	// Record some data
	Record(1, 100)

	// Reset again
	Reset()

	// Get a snapshot
	snapshot := Snapshot()

	// The snapshot should be empty
	if len(snapshot) != 0 {
		t.Errorf("Expected empty snapshot after reset, but got length %d", len(snapshot))
	}
}

func TestRingBufferWrapping(t *testing.T) {
	// Reset the tracer
	Reset()

	// Record more entries than the buffer size to test wrapping
	for i := 0; i < BufferSize+10; i++ {
		Record(uint64(i), int64(i*10))
	}

	// Get a snapshot
	snapshot := Snapshot()

	// The snapshot should have the size of the buffer
	if len(snapshot) != BufferSize {
		t.Errorf("Expected snapshot length of %d, but got %d", BufferSize, len(snapshot))
	}
}
