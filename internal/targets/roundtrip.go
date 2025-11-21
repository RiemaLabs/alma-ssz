package targets

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// RoundTripTarget defines the struct + package for round-trip fuzzing.
type RoundTripTarget struct {
	Name       string `json:"name"`
	ImportPath string `json:"import"`
	Type       string `json:"type"`
}

// LoadRoundTripTargets parses the JSON config at path.
func LoadRoundTripTargets(path string) ([]RoundTripTarget, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read targets config: %w", err)
	}
	var targets []RoundTripTarget
	if err := json.Unmarshal(raw, &targets); err != nil {
		return nil, fmt.Errorf("parse targets config: %w", err)
	}
	for i, t := range targets {
		if t.Name == "" || t.ImportPath == "" || t.Type == "" {
			return nil, fmt.Errorf("target %d is missing fields: %+v", i, t)
		}
		targets[i].ImportPath = strings.TrimSpace(t.ImportPath)
	}
	return targets, nil
}
