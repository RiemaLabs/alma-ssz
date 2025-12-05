package schemas

import (
	"strings"
	"testing"
    "hash/crc32"

	"alma.local/ssz/oracle" // Corrected import path
)

// --- Simple Targets (Easy) ---

func FuzzBitvectorBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		// BeaconState contains a Bitvector[4] (JustificationBits).
		// Vulnerability: Dirty padding in JustificationBits is accepted.
		// If bug is active: Unmarshal succeeds. Marshal cleans it. RoundTrip detects mismatch.
		err := oracle.RoundTrip[BeaconState](data)
		if err != nil {
			msg := err.Error()
			// If it's a hash mismatch, it means Unmarshal accepted dirty data but Marshal produced clean data.
			// This confirms the bug (lax Unmarshal).
			if strings.Contains(msg, "bug triggered!") { // oracle.RoundTrip directly contains "bug triggered!"
				t.Fatalf("Bug triggered: %v", err)
			}
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

func FuzzUnionBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		var obj UnionStruct
		// We manually check UnmarshalSSZ because RoundTrip might fail on encoding mismatch
		err := obj.UnmarshalSSZ(data)
		if err == nil {
			// If unmarshal succeeds, check if it accepted dirty data.
			// DebugUnion selector 0 (None) takes 1 byte.
			if len(data) > 1 && data[0] == 0 {
				t.Fatalf("Bug triggered: SUT accepted dirty union %x", data)
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
		if err != nil && strings.Contains(err.Error(), "bug triggered!") { // Changed to "bug triggered!"
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

		if err != nil && strings.Contains(err.Error(), "bug triggered!") { // Changed to "bug triggered!"
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

		if err != nil && strings.Contains(err.Error(), "bug triggered!") { // Changed to "bug triggered!"
			t.Fatalf("Bug triggered: %v", err)
		}
	})
}

func FuzzHardUnionBug(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		var obj UnionStruct // Use UnionStruct for now as HardUnionStruct is same
		err := obj.UnmarshalSSZ(data)
		if err == nil {
			if len(data) > 1 && data[0] == 0 {
				t.Fatalf("Bug triggered: SUT accepted dirty union %x", data)
			}
		}
	})
}