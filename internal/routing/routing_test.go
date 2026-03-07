package routing

import (
	"testing"

	"atrakta/internal/contract"
)

func TestResolveUsesCategoryAndDefaultFallback(t *testing.T) {
	c := contract.Default(t.TempDir())

	syncDecision := Resolve(c, "sync")
	if syncDecision.Worker != "sync_safe" || syncDecision.Quality != "quick" {
		t.Fatalf("unexpected sync decision: %#v", syncDecision)
	}
	fallback := Resolve(c, "unknown_category")
	if fallback.Worker != "general" || fallback.Quality != "quick" {
		t.Fatalf("unexpected fallback decision: %#v", fallback)
	}
}

func TestResolveDefaultsWhenRoutingOmitted(t *testing.T) {
	c := contract.Default(t.TempDir())
	c.Routing = nil

	decision := Resolve(c, "")
	if decision.TaskCategory != "sync" {
		t.Fatalf("expected sync category fallback, got %s", decision.TaskCategory)
	}
	if decision.Worker != "general" || decision.Quality != "quick" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}
