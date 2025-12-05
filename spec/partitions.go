package spec

import (
	"fmt"
	"math"
	"alma.local/ssz/domains"
)

// GenerateUintBuckets creates a set of mutually exclusive buckets for unsigned integers.
func GenerateUintBuckets(bitSize int) []domains.Bucket {
	// ... (implementation is correct, keeping it folded for brevity)
	var buckets []domains.Bucket
	maxValForType := uint64(math.MaxUint64)
	if bitSize < 64 {
		maxValForType = (uint64(1) << bitSize) - 1
	}
	currentMin := uint64(0)
	buckets = append(buckets, domains.Bucket{ID: "Zero", Description: "Value is 0", Range: domains.Range{Min: 0, Max: 0}, Tag: "boundary"})
	currentMin = 1
	if currentMin <= maxValForType {
		buckets = append(buckets, domains.Bucket{ID: "One", Description: "Value is 1", Range: domains.Range{Min: 1, Max: 1}, Tag: "boundary"})
		currentMin = 2
	}
	for k := 1; k < bitSize; k++ {
		upperBound := (uint64(1) << (k + 1)) - 1
		if upperBound > maxValForType {
			upperBound = maxValForType
		}
		if currentMin <= upperBound {
			buckets = append(buckets, domains.Bucket{ID: domains.BucketID(fmt.Sprintf("%d..%d", currentMin, upperBound)), Description: fmt.Sprintf("Range %d to %d", currentMin, upperBound), Range: domains.Range{Min: currentMin, Max: upperBound}, Tag: "power_of_2_range"})
			currentMin = upperBound + 1
		}
		if currentMin > maxValForType {
			break
		}
	}
	if currentMin <= maxValForType {
		buckets = append(buckets, domains.Bucket{ID: domains.BucketID(fmt.Sprintf("%d..%d", currentMin, maxValForType)), Description: fmt.Sprintf("Remaining range %d to %d", currentMin, maxValForType), Range: domains.Range{Min: currentMin, Max: maxValForType}, Tag: "remaining_range"})
	}
	var finalBuckets []domains.Bucket
	for _, b := range buckets {
		if b.Range.Min <= b.Range.Max {
			finalBuckets = append(finalBuckets, b)
		}
	}
	return finalBuckets
}

func GenerateBytePartitions() []domains.Bucket {
	return GenerateUintBuckets(8)
}

var BoolBuckets = []domains.Bucket{
	{ID: "False", Description: "Boolean false (0x00)", Range: domains.Range{Min: 0, Max: 0}, Tag: "Clean"},
	{ID: "True", Description: "Boolean true (0x01)", Range: domains.Range{Min: 1, Max: 1}, Tag: "Clean"},
	{ID: "Dirty", Description: "Dirty boolean byte (0x02-0xFF)", Range: domains.Range{Min: 2, Max: 255}, Tag: "Dirty"},
}

var ByteContentBuckets = []domains.Bucket{
	{ID: "Zero", Description: "0x00", Range: domains.Range{Min: 0, Max: 0}, Tag: "content_byte"},
	{ID: "One", Description: "0x01", Range: domains.Range{Min: 1, Max: 1}, Tag: "content_byte"},
	{ID: "MidRange", Description: "Random byte in [2, 127]", Range: domains.Range{Min: 2, Max: 127}, Tag: "content_byte"},
	{ID: "HighRange", Description: "Random byte in [128, 255]", Range: domains.Range{Min: 128, Max: 255}, Tag: "content_byte"},
}

var SliceLengthBuckets = []domains.Bucket{
	{ID: "Empty", Description: "Length 0", Range: domains.Range{Min: 0, Max: 0}, Tag: "length"},
	{ID: "MinLen", Description: "Length 1", Range: domains.Range{Min: 1, Max: 1}, Tag: "length"},
	{ID: "SmallLen", Description: "Random length in [2, 16]", Range: domains.Range{Min: 2, Max: 16}, Tag: "length"},
	{ID: "MidLen", Description: "Random length in [17, 256]", Range: domains.Range{Min: 17, Max: 256}, Tag: "length"},
	{ID: "MaxLen", Description: "Max possible length", Range: domains.Range{Min: 257, Max: math.MaxUint64}, Tag: "length_max_placeholder"},
}

var OffsetBuckets = []domains.Bucket{
	{ID: "Correct", Description: "Canonical offset", Range: domains.Range{Min: 0, Max: 0}, Tag: "offset"},
	{ID: "SmallGap", Description: "Add 1-4 bytes gap", Range: domains.Range{Min: 1, Max: 4}, Tag: "offset"},
	{ID: "MediumGap", Description: "Add 5-64 bytes gap", Range: domains.Range{Min: 5, Max: 64}, Tag: "offset"},
	{ID: "LargeGap", Description: "Add 65-256 bytes gap", Range: domains.Range{Min: 65, Max: 256}, Tag: "offset"},
}

var ContainerDefaultBucket = []domains.Bucket{
	{ID: "Default", Description: "Recursive default", Range: domains.Range{Min: 0, Max: 0}, Tag: "default"},
}

func init() {
	// Dilute the search space with dummy buckets to make finding the bug harder
	for i := 0; i < 50; i++ {
		OffsetBuckets = append(OffsetBuckets, domains.Bucket{
			ID:          domains.BucketID(fmt.Sprintf("Dummy_Offset_%d", i)),
			Description: "Placeholder offset (no change)",
			Range:       domains.Range{Min: 0, Max: 0},
			Tag:         "offset_dummy",
		})
	}
	// Add dummy buckets for byte content as well to make dirty padding hard to find
	for i := 2; i < 255; i += 1 { // High dilution for byte values
		ByteContentBuckets = append(ByteContentBuckets, domains.Bucket{
			ID:          domains.BucketID(fmt.Sprintf("Dummy_Byte_%d", i)),
			Description: "Placeholder clean byte",
			Range:       domains.Range{Min: uint64(i), Max: uint64(i)},
			Tag:         "content_byte_dummy",
		})
	}
}