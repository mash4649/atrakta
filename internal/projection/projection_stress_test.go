package projection

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/registry"
)

func TestProjectionScalingLinearBound(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "contract.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	const n = 200
	c := contract.Default(repo)
	c.Interfaces.Supported = make([]string, 0, n)
	c.Interfaces.CoreSet = []string{"if0"}
	c.Projections.MaxPerInterface = 1
	c.Projections.OptionalTemplates = map[string][]string{}

	reg := registry.Registry{Entries: map[string]registry.Entry{}}
	targets := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("if%d", i)
		c.Interfaces.Supported = append(c.Interfaces.Supported, id)
		c.Projections.OptionalTemplates[id] = []string{"atrakta-link"}
		reg.Entries[id] = registry.Entry{InterfaceID: id, Surface: "editor", Anchor: "." + id + "/", ProjectionDir: "." + id + "/"}
		targets = append(targets, id)
	}

	d, err := RequiredForTargets(repo, c, reg, targets, "sha256:contract", "# A\n")
	if err != nil {
		t.Fatalf("projection generation failed: %v", err)
	}

	maxExpected := n * 2 // required agents-md + one optional template
	if len(d) > maxExpected {
		t.Fatalf("projection count exceeded linear bound: got=%d expected<=%d", len(d), maxExpected)
	}
	if len(d) != maxExpected {
		t.Fatalf("unexpected projection count: got=%d expected=%d", len(d), maxExpected)
	}
}
