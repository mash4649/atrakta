package platform

import (
	"testing"

	"atrakta/internal/contract"
)

func TestA10PathTraversalBlocked(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	if err := ValidateMutationPath(repo, "../outside.txt", c.Boundary); err == nil {
		t.Fatalf("expected traversal to be blocked")
	}
}
