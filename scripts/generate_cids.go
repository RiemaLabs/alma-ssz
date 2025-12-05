package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("Generating global CID registry...")

	schemaDir := "schemas"
	cidMap := make(map[uint64]struct{})

	reNumeric := regexp.MustCompile(`tracer\.Record\(([0-9]+),`)
	reHashed := regexp.MustCompile(`tracer\.Hash\(\[]byte\(\"([^\"]+)\"\)\)`)

	err := filepath.Walk(schemaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "_encoding.go") {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
		sContent := string(content)

			matchesNum := reNumeric.FindAllStringSubmatch(sContent, -1)
			for _, match := range matchesNum {
				if len(match) > 1 {
					cid, err := strconv.ParseUint(match[1], 10, 64)
					if err == nil {
						cidMap[cid] = struct{}{}
					}
				}
			}

			matchesHash := reHashed.FindAllStringSubmatch(sContent, -1)
			for _, match := range matchesHash {
				if len(match) > 1 {
					fieldName := match[1]
					h := fnv.New64a()
					h.Write([]byte(fieldName))
					cidMap[h.Sum64()] = struct{}{}
				}
			}
		}
		return nil
	})

	if err != nil {
		panic(fmt.Sprintf("Error walking files: %v", err))
	}

	cids := make([]uint64, 0, len(cidMap))
	for cid := range cidMap {
		cids = append(cids, cid)
	}
	sort.Slice(cids, func(i, j int) bool { return cids[i] < cids[j] })

	outputFile := "config/cids.json"
	outputData, err := json.MarshalIndent(cids, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Error marshalling JSON: %v", err))
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		panic(fmt.Sprintf("Could not create config directory: %v", err))
	}

	err = ioutil.WriteFile(outputFile, outputData, 0644)
	if err != nil {
		panic(fmt.Sprintf("Error writing to file: %v", err))
	}

	fmt.Printf("Successfully found %d unique CIDs and wrote them to %s\n", len(cids), outputFile)
}
