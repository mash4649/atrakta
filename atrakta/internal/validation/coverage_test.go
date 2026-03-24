package validation

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestVerifyOperationsSchemaCoverage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if err := VerifyOperationsSchemaCoverage(root); err != nil {
		t.Fatalf("verify operations schema coverage: %v", err)
	}
}
