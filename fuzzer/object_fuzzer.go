package fuzzer

import (
	"alma.local/ssz/feedback"
	"alma.local/ssz/internal/analyzer"
)

// ObjectFuzzer can evaluate inputs while also seeing the original structured object.
type ObjectFuzzer interface {
	Fuzzer
	ExecuteWithObject(sszBytes []byte, obj interface{}) (feedback.RuntimeSignature, bool, bool, []analyzer.TraceEntry)
}
