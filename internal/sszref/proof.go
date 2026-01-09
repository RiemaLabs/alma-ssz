package sszref

import (
	"bytes"
	"fmt"
	"math"
	"sort"
)

// VerifyMultiproof verifies a multi-proof against the given root.
func VerifyMultiproof(root [32]byte, proof [][]byte, leaves [][]byte, indices []int) (bool, error) {
	if len(indices) == 0 {
		return false, fmt.Errorf("sszref: indices length is zero")
	}
	if len(leaves) != len(indices) {
		return false, fmt.Errorf("sszref: number of leaves and indices mismatch")
	}

	reqIndices := getRequiredIndices(indices)
	if len(reqIndices) != len(proof) {
		return false, fmt.Errorf("sszref: number of proof hashes %d and required indices %d mismatch", len(proof), len(reqIndices))
	}

	userGenIndices := make([]int, len(indices)+len(reqIndices))
	pos := 0
	db := make(map[int][]byte)
	for i, leaf := range leaves {
		db[indices[i]] = normalize32(leaf)
		userGenIndices[pos] = indices[i]
		pos++
	}
	for i, h := range proof {
		db[reqIndices[i]] = normalize32(h)
		userGenIndices[pos] = reqIndices[i]
		pos++
	}

	sort.Sort(sort.Reverse(sort.IntSlice(userGenIndices)))
	capacity := int(math.Log2(float64(userGenIndices[0])))
	auxGenIndices := make([]int, 0, capacity)
	pos = 0
	posAux := 0

	for posAux < len(auxGenIndices) || pos < len(userGenIndices) {
		var index int
		if len(auxGenIndices) == 0 || (pos < len(userGenIndices) && auxGenIndices[posAux] < userGenIndices[pos]) {
			index = userGenIndices[pos]
			pos++
		} else {
			index = auxGenIndices[posAux]
			posAux++
		}

		if index == 1 {
			break
		}

		if _, hasParent := db[getParent(index)]; hasParent {
			continue
		}

		left, hasLeft := db[(index|1)^1]
		right, hasRight := db[index|1]
		if !hasRight || !hasLeft {
			return false, fmt.Errorf("sszref: proof is missing required nodes, either %d or %d", (index|1)^1, index|1)
		}

		parent := hashConcat(left, right)
		parentIndex := getParent(index)
		db[parentIndex] = parent[:]
		auxGenIndices = append(auxGenIndices, parentIndex)
	}

	res, ok := db[1]
	if !ok {
		return false, fmt.Errorf("sszref: root was not computed during proof verification")
	}
	return bytes.Equal(res, root[:]), nil
}

func normalize32(input []byte) []byte {
	if len(input) == 32 {
		return input
	}
	out := make([]byte, 32)
	copy(out, input)
	return out
}

func getPosAtLevel(index int, level int) bool {
	return (index & (1 << level)) > 0
}

func getPathLength(index int) int {
	return int(math.Log2(float64(index)))
}

func getSibling(index int) int {
	return index ^ 1
}

func getParent(index int) int {
	return index >> 1
}

func getRequiredIndices(leafIndices []int) []int {
	exists := struct{}{}
	required := make(map[int]struct{})
	computed := make(map[int]struct{})
	leaves := make(map[int]struct{})

	for _, leaf := range leafIndices {
		leaves[leaf] = exists
		cur := leaf
		for cur > 1 {
			sibling := getSibling(cur)
			parent := getParent(cur)
			required[sibling] = exists
			computed[parent] = exists
			cur = parent
		}
	}

	for leaf := range leaves {
		delete(required, leaf)
	}
	for comp := range computed {
		delete(required, comp)
	}

	res := make([]int, 0, len(required))
	for i := range required {
		res = append(res, i)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(res)))
	return res
}
