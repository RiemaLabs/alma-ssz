package schemas

import (
	"strings"
	"testing"
    "hash/crc32"

	"alma.local/ssz/internal/oracle"
)

// --- Simple Targets (Easy) ---

func FuzzBitvectorBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		err := oracle.RoundTrip[BitvectorStruct](data)
		if err == nil {
			// Bitvector4 (4 bits) fits in 1 byte. Upper 4 bits must be 0.
			if len(data) == 1 && (data[0]&0xF0 != 0) {
				t.Fatalf("Bug triggered: SUT accepted dirty bitvector %x", data)
			}
		}
		if err != nil && strings.Contains(err.Error(), "bug triggered") {
			t.Fatalf("Bug triggered: %v", err)
		}
	})
}

func FuzzBooleanBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if err := oracle.RoundTrip[BooleanStruct](data); err != nil {
			if strings.Contains(err.Error(), "bug triggered") {
				t.Fatalf("Bug triggered: %v", err)
			}
		}
	})
}

func FuzzGapBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if err := oracle.RoundTrip[GapStruct](data); err != nil {
			if strings.Contains(err.Error(), "bug triggered") {
				t.Fatalf("Bug triggered: %v", err)
			}
		}
	})
}

// --- Hard Targets (Hard) ---

func FuzzHardBitvectorBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		err := oracle.RoundTrip[HardBitvectorStruct](data)
        
        var obj HardBitvectorStruct
        if uErr := obj.UnmarshalSSZ(data); uErr != nil {
            return // Invalid SSZ, ignore
        }
        
        sum := crc32.ChecksumIEEE(obj.Padding[:])
        if uint32(obj.Magic) != sum {
            return // Mismatch, ignore potential bug
        }

		if err == nil {
			if len(obj.Target) == 1 && (obj.Target[0]&0xF0 != 0) {
                t.Logf("Magic: %x, Sum: %x", obj.Magic, sum)
				t.Fatalf("Bug triggered: SUT accepted dirty bitvector %x", data)
			}
		}
		if err != nil && strings.Contains(err.Error(), "bug triggered") {
            t.Logf("Magic: %x, Sum: %x", obj.Magic, sum)
			t.Fatalf("Bug triggered: %v", err)
		}
	})
}

func FuzzHardBooleanBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		err := oracle.RoundTrip[HardBooleanStruct](data)
        
        var obj HardBooleanStruct
        if uErr := obj.UnmarshalSSZ(data); uErr != nil {
            return
        }
        // HardBooleanStruct uses LargeBuffer
        sum := crc32.ChecksumIEEE(obj.LargeBuffer[:])
        if uint32(obj.Magic) != sum {
            return
        }

		if err != nil && strings.Contains(err.Error(), "bug triggered") {
			t.Fatalf("Bug triggered: %v", err)
		}
	})
}

func FuzzHardGapBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		err := oracle.RoundTrip[HardGapStruct](data)
        
        var obj HardGapStruct
        if uErr := obj.UnmarshalSSZ(data); uErr != nil {
            return
        }
        sum := crc32.ChecksumIEEE(obj.Padding[:])
        if uint32(obj.Magic) != sum {
            return
        }

		if err != nil && strings.Contains(err.Error(), "bug triggered") {
			t.Fatalf("Bug triggered: %v", err)
		}
	})
}
