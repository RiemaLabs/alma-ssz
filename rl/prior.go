package rl

import "strings"

func BuildActionPrior(actions []EncodingContextAction, bitvectorFields map[string]struct{}) []float64 {
	prior := make([]float64, len(actions))
	for i, act := range actions {
		score := 0.0
		tag := strings.ToLower(act.Tag)
		aspect := strings.ToLower(string(act.AspectID))
		id := strings.ToLower(string(act.BucketID))

		if strings.Contains(tag, "bug") || strings.Contains(id, "bug") {
			score += 4.0
		}
		if strings.Contains(tag, "dirty") || strings.Contains(id, "dirty") {
			score += 3.0
		}
		if strings.Contains(tag, "tail") || strings.Contains(aspect, "tail") {
			score += 2.0
		}
		if strings.Contains(tag, "offset") || strings.Contains(aspect, "offset") || strings.Contains(id, "gap") {
			score += 2.5
		}
		if strings.Contains(id, "high") || strings.Contains(id, "padding") {
			score += 1.5
		}
		if strings.Contains(id, "zero") || strings.Contains(id, "empty") || strings.Contains(id, "b_00") {
			score += 2.0
		}
		if aspect == "value" && id == "dirty" {
			score += 2.0
		}
		if strings.Contains(id, "minlen") {
			score += 1.0
		}
		if strings.Contains(id, "maxlen") || strings.Contains(tag, "length_max") {
			score += 1.5
		}
		if bitvectorFields != nil {
			if _, ok := bitvectorFields[act.FieldName]; ok {
				score += 2.5
				if strings.Contains(tag, "dirty") || strings.Contains(id, "high") {
					score += 4.5
				}
			}
		}

		prior[i] = score
	}
	return prior
}
