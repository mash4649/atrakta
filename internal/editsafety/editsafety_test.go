package editsafety

import (
	"testing"

	"atrakta/internal/contract"
)

func TestValidateCandidateGoAST(t *testing.T) {
	cfg := &contract.EditSafety{Mode: "anchor+optional_ast"}
	if err := ValidateCandidate("main.go", "package main\nfunc main(){}\n", cfg); err != nil {
		t.Fatalf("unexpected go validation error: %v", err)
	}
	if err := ValidateCandidate("main.go", "package main\nfunc main(\n", cfg); err == nil {
		t.Fatalf("expected invalid go source to fail")
	}
}

func TestValidateCandidateJSONParse(t *testing.T) {
	cfg := &contract.EditSafety{Mode: "anchor+optional_ast"}
	if err := ValidateCandidate("a.json", "{\"a\":1}", cfg); err != nil {
		t.Fatalf("unexpected json validation error: %v", err)
	}
	if err := ValidateCandidate("a.json", "{\"a\":", cfg); err == nil {
		t.Fatalf("expected invalid json to fail")
	}
}

func TestValidateCandidatePolicyOverride(t *testing.T) {
	cfg := &contract.EditSafety{Mode: "anchor+optional_ast", Languages: map[string]string{"go": "off"}}
	if err := ValidateCandidate("main.go", "package main\nfunc main(\n", cfg); err != nil {
		t.Fatalf("expected override=off to skip validation, got %v", err)
	}
}
