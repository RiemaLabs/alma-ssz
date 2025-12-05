package fuzzer

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"    // Needed for rand.Seed
	"math/rand" // For actual random numbers

	"alma.local/ssz/feedback" // Import feedback package
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RealBitvectorFuzzer implements the Fuzzer interface for the bitvector example.
// It interacts with the Go test framework and bug toggling scripts.
type RealBitvectorFuzzer struct {
	fuzzTestDir     string // Directory to place temporary Go test files
	tempTestCounter int    // Counter for unique temporary test file names
	currentCoverage float64 // Simulated, as real coverage is hard to get from this approach
	lastNewCoverage float64 // Simulated
}

// NewRealBitvectorFuzzer creates a new RealBitvectorFuzzer.
func NewRealBitvectorFuzzer() (*RealBitvectorFuzzer, error) {
	// Create a temporary directory for Go test files
	tempDir, err := ioutil.TempDir("", "bitvector_fuzz_tests")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir for fuzzer: %w", err)
	}
	return &RealBitvectorFuzzer{
		fuzzTestDir: tempDir,
		currentCoverage: 0.0,
		lastNewCoverage: 0.0,
	}, nil 
}

// Reset cleans up the temporary test directory and resets coverage metrics.
func (rbf *RealBitvectorFuzzer) Reset() {
	os.RemoveAll(rbf.fuzzTestDir) // Clean up old dir
	rbf.fuzzTestDir, _ = ioutil.TempDir("", "bitvector_fuzz_tests") // Create new one
	rbf.tempTestCounter = 0
	rbf.currentCoverage = 0.0
	rbf.lastNewCoverage = 0.0
}

// TotalCoverage returns the current simulated cumulative coverage.
func (rbf *RealBitvectorFuzzer) TotalCoverage() float64 {
	return rbf.currentCoverage
}

// NewCoverage returns the simulated new coverage found in the last execution.
func (rbf *RealBitvectorFuzzer) NewCoverage() float64 {
	return rbf.lastNewCoverage
}

// fuzzerTemplate is the Go test file template for the bitvector example.
// It calls oracle.RoundTrip for BeaconState, expecting specific error messages for bug detection.
const fuzzerTemplate = `
package main // Changed from main_test to main

import (
	"fmt"
	"strings"
	"os" // Added for os.Exit
	// "testing" // Removed as it's no longer a 'go test' file
	"alma.local/ssz/oracle""" // Correct import path
	"alma.local/ssz/schemas"""         // Correct import path
)

// runFuzzLogic is the core function that executes the SSZ input and checks for bugs.
func runFuzzLogic_{{.TestID}}(data []byte) (bool, string) {
	// Bug: Bitvector Dirty Padding. Target: schemas.BeaconState.
	// The oracle.RoundTrip checks for canonical roundtrip.
	// If the bug is active (via bug_toggle.sh), unmarshal accepts dirty data, marshal cleans it,
	// leading to a mismatch -> "bug triggered!" substring in error string from oracle.RoundTrip.
	err := oracle.RoundTrip[schemas.BeaconState](data) 
	if err != nil {
		if strings.Contains(err.Error(), "bug triggered!") {
			return true, fmt.Sprintf("BUG_FOUND: Bitvector Dirty Padding triggered! Error: %v\n", err)
		} else {
			// Other non-bug errors (e.g., invalid SSZ, malformed input) are simply logged.
			return false, fmt.Sprintf("NON_BUG_ERROR: %v\n", err)
		}
	}
	return false, "ROUNDTRIP_SUCCESS: Input processed without error.\n"
}

func main() {
	// Generated SSZ bytes injected here.
	data := []byte{ {{.SSZBytes}} } 

	bugTriggered, outputMsg := runFuzzLogic_{{.TestID}}(data)
	fmt.Print(outputMsg) // Always print output message

	if bugTriggered {
		os.Exit(1) // Exit with non-zero code if bug found
	}
	os.Exit(0) // Exit with zero code for success
}
`

type templateData struct {
	TestID   int
	SSZBytes string // Go byte slice literal string representation of SSZ bytes
}

// Execute performs one fuzzing execution step with the given SSZ bytes.
func (rbf *RealBitvectorFuzzer) Execute(sszBytes []byte) (signature feedback.RuntimeSignature, bugTriggered bool, newCoverageFound bool) { 
	rbf.tempTestCounter++
	
	// Initialize named return parameters
	bugTriggered = false
	newCoverageFound = false

	// Create a unique temporary Go file name.
	testFileName := filepath.Join(rbf.fuzzTestDir, fmt.Sprintf("temp_fuzz_test_%d.go", rbf.tempTestCounter))

	// Convert sszBytes to a Go byte slice literal string (e.g., "0x01, 0x02, 0x03").
	parts := make([]string, len(sszBytes))
	for i, b := range sszBytes {
		parts[i] = fmt.Sprintf("0x%02x", b)
	}
	sszBytesStr := strings.Join(parts, ", ")

	// Parse the template.
	tmpl, err := template.New("fuzzerTest").Parse(fuzzerTemplate)
	if err != nil {
		return feedback.RuntimeSignature{}, false, false // Return empty signature on error
	}

	// Prepare data for the template.
	data := templateData{
		TestID:   rbf.tempTestCounter,
		SSZBytes: sszBytesStr,
	}

	// Write the Go test file.
	file, err := os.Create(testFileName)
	if err != nil {
		return feedback.RuntimeSignature{}, false, false // Return empty signature on error
	}
	if err := tmpl.Execute(file, data); err != nil {
		file.Close()
		return feedback.RuntimeSignature{}, false, false // Return empty signature on error
	}
	file.Close() // Close the file before executing go run.

	// Defer cleanup of the temporary test file.
	// defer os.Remove(testFileName) // Temporarily commented out for debugging

	// --- Execute the test file ---
	// 1. Activate the bitvector bug.
	rbf.toggleBug("activate", "bitvector")
	
	// 2. Build the generated Go test file into an executable.
	execBinary := filepath.Join(rbf.fuzzTestDir, fmt.Sprintf("temp_fuzz_exec_%d", rbf.tempTestCounter))
	buildCmd := exec.Command("go", "build", "-o", execBinary, testFileName)
	buildCmd.Dir = "."
	buildOutput, buildErr := buildCmd.CombinedOutput()
	if buildErr != nil {
		rbf.toggleBug("deactivate", "bitvector") // Deactivate if build failed
		fmt.Fprintf(os.Stderr, "Go build failed for %s: %v\nOutput:\n%s\n", testFileName, buildErr, buildOutput)
		return feedback.RuntimeSignature{}, false, false // Return empty signature on error
	}
	defer os.Remove(execBinary) // Clean up the executable

	// 3. Run the compiled test executable.
	runCmd := exec.Command(execBinary)
	runCmd.Dir = "."
	output, cmdErr := runCmd.CombinedOutput() // Capture both stdout and stderr.
	
	// 4. Deactivate the bitvector bug.
	rbf.toggleBug("deactivate", "bitvector")

	outputStr := string(output)
	
	// Synthesize RuntimeSignature from output
	signature = rbf.generateSignature(outputStr) // Assign to named return parameter

	bugTriggered = signature.BugFoundCount > 0

	// Simulate coverage gain. In a real fuzzer, this would come from instrumentation.
	rbf.lastNewCoverage = 0.0
	if !bugTriggered && cmdErr == nil { // If it ran successfully (exit code 0) and no bug was explicitly found.
		// If it's a successful roundtrip, simulate coverage gain.
		if signature.RoundtripSuccessCount > 0 {
			simulatedCoverageGain := 0.01 + (rand.Float64() * 0.05) 
			rbf.currentCoverage += simulatedCoverageGain
			rbf.lastNewCoverage = simulatedCoverageGain
			newCoverageFound = simulatedCoverageGain > 0.01 // Report new coverage if it's above a minimal threshold.
		}
	}

	return signature, bugTriggered, newCoverageFound // Return values explicitly
}

// generateSignature synthesizes a compact RuntimeSignature from raw fuzzer output.
func (rbf *RealBitvectorFuzzer) generateSignature(output string) feedback.RuntimeSignature {
	sig := feedback.RuntimeSignature{}
	if strings.Contains(output, "ROUNDTRIP_SUCCESS") {
		sig.RoundtripSuccessCount++
	}
	if strings.Contains(output, "NON_BUG_ERROR") {
		sig.NonBugErrorCount++
	}
	if strings.Contains(output, "BUG_FOUND") {
		sig.BugFoundCount++
	}
	return sig
}

// toggleBug executes the bug_toggle.sh script to activate or deactivate a bug.
func (rbf *RealBitvectorFuzzer) toggleBug(action, bugName string) {
	scriptPath := "./scripts/bug_toggle.sh" // Path relative to module root.
	cmd := exec.Command("bash", scriptPath, action, bugName)
	cmd.Dir = "." // Command needs to be run from the module root.
	// We usually redirect stdout/stderr to avoid clutter.
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error toggling bug '%s' %s: %v\n", bugName, action, err)
	}
}
