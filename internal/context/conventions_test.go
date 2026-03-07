package context

import (
	"strings"
	"testing"
)

func TestBuildConventionSnippetWithinBudget(t *testing.T) {
	raw := "# Rules\nmust keep tests green\n\n# Examples\nfoo\nbar\n"
	out, used := buildConventionSnippet("CONVENTIONS.md", raw, 10)
	if out == "" {
		t.Fatalf("expected non-empty snippet")
	}
	if used <= 0 || used > 10 {
		t.Fatalf("unexpected token usage: %d", used)
	}
	if !strings.Contains(out, "conventions:index") {
		t.Fatalf("expected index marker")
	}
}

func TestBuildConventionSnippetPrioritizesCriticalSection(t *testing.T) {
	raw := "# Nice To Have\nstyle notes\n\n# Security Rules\nmust validate all inputs\n\n# Misc\nnotes\n"
	out, _ := buildConventionSnippet("CONVENTIONS.md", raw, 20)
	if !strings.Contains(strings.ToLower(out), "security rules") {
		t.Fatalf("expected critical section in snippet")
	}
}
