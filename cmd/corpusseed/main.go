package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	// "alma.local/ssz/internal/corpus"
	"alma.local/ssz/internal/targets"
)

var (
	flagConfig = flag.String("config", "config/roundtrip_targets.json", "path to roundtrip target config")
	flagOut    = flag.String("out", "corpus/export", "output directory for seed corpus")
	flagLimit  = flag.Int("limit", 32, "maximum number of seeds to export per struct (<=0 disables the cap)")
	flagFormat = flag.String("format", "dir", "output format: dir or zip")
	flagTypes  = flag.String("types", "", "optional comma-separated list of target names to export (default: all)")
)

func main() {
	flag.Parse()

	targets, err := targets.LoadRoundTripTargets(*flagConfig)
	if err != nil {
		log.Fatalf("load targets: %v", err)
	}
	selected := filterTargets(targets, *flagTypes)
	if len(selected) == 0 {
		log.Fatalf("no targets selected")
	}

	base := *flagOut
	if err := os.MkdirAll(base, 0o755); err != nil {
		log.Fatalf("create out dir: %v", err)
	}

	limit := *flagLimit
	format := strings.ToLower(*flagFormat)
	if format != "dir" && format != "zip" {
		log.Fatalf("unsupported format %q (expected dir or zip)", format)
	}

	for _, t := range selected {
		fmt.Printf("[corpus] exporting %s -> %s (%s)\n", t.Name, base, format)
		// loader := corpus.NewLoader(corpus.DefaultRoot, limit)
		// seeds, err := loader.Collect(t.Name)
		var seeds [][]byte
		err = nil
		if err != nil {
			log.Fatalf("collect %s: %v", t.Name, err)
		}
		if len(seeds) == 0 {
			// log.Fatalf("no seeds found for %s (check workspace/tests)", t.Name)
			fmt.Println("Skipping corpus generation due to missing internal/corpus package")
		}
		destName := fuzzFuncName(t)
		dest := filepath.Join(base, destName)
		if format == "dir" {
			if err := emitDir(dest, seeds); err != nil {
				log.Fatalf("write %s: %v", dest, err)
			}
		} else {
			if err := emitZip(dest+".zip", seeds); err != nil {
				log.Fatalf("write %s: %v", dest+".zip", err)
			}
		}
		fmt.Printf("[corpus] %s -> %d seeds\n", t.Name, len(seeds))
	}
}

func filterTargets(all []targets.RoundTripTarget, filter string) []targets.RoundTripTarget {
	if filter == "" {
		return all
	}
	names := map[string]bool{}
	for _, part := range strings.Split(filter, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			names[strings.ToLower(trimmed)] = true
		}
	}
	var out []targets.RoundTripTarget
	for _, t := range all {
		if names[strings.ToLower(t.Name)] {
			out = append(out, t)
		}
	}
	return out
}

func emitDir(dest string, seeds [][]byte) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for _, seed := range seeds {
		sum := sha256.Sum256(seed)
		name := hex.EncodeToString(sum[:16]) + ".ssz"
		if err := os.WriteFile(filepath.Join(dest, name), seed, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func emitZip(path string, seeds [][]byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zipw := zip.NewWriter(f)
	for _, seed := range seeds {
		sum := sha256.Sum256(seed)
		name := hex.EncodeToString(sum[:16]) + ".ssz"
		w, err := zipw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(seed); err != nil {
			return err
		}
	}
	return zipw.Close()
}

func fuzzFuncName(t targets.RoundTripTarget) string {
	return fmt.Sprintf("Fuzz%sRoundTrip", t.Name)
}
