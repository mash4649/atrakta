package projection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/registry"
)

func TestOptionalTemplatesBounded(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := contract.Default(repo)
	c.Projections.MaxPerInterface = 1
	c.Projections.OptionalTemplates = map[string][]string{"cursor": {"contract-json", "atrakta-link"}}
	cb, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "contract.json"), cb, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := RequiredForTargets(repo, c, registry.Default(), []string{"cursor"}, contract.ContractHash(cb), "# A\n")
	if err == nil {
		t.Fatalf("expected max_per_interface bound failure")
	}
}

func TestOptionalTemplateContractJSON(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := contract.Default(repo)
	c.Projections.OptionalTemplates = map[string][]string{"cursor": {"contract-json"}}
	cb, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "contract.json"), cb, 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := RequiredForTargets(repo, c, registry.Default(), []string{"cursor"}, contract.ContractHash(cb), "# A\n")
	if err != nil {
		t.Fatalf("projection generation failed: %v", err)
	}
	found := false
	for _, it := range d {
		if it.TemplateID == "cursor:contract-json@1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected optional contract-json template")
	}
}
