package fixtures

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestVerifyResolverFixtureCoverage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if err := VerifyResolverFixtureCoverage(root); err != nil {
		t.Fatalf("verify resolver fixture coverage: %v", err)
	}
}
