package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type conceptFixture struct {
	V                     int      `json:"v"`
	Intents               []string `json:"intents"`
	OperationalEntrypoint string   `json:"operational_entrypoint"`
}

func TestConceptCoverageMatrixIncludesFixtureIntents(t *testing.T) {
	root := repoRoot(t)
	fixturePath := filepath.Join(root, "fixtures", "core", "run-concept-coverage.fixture.json")
	docPath := filepath.Join(root, "docs", "plan", "concept-coverage-matrix.md")

	b, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture conceptFixture
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if fixture.OperationalEntrypoint != "run" {
		t.Fatalf("unexpected entrypoint: %q", fixture.OperationalEntrypoint)
	}
	if len(fixture.Intents) == 0 {
		t.Fatal("fixture intents must not be empty")
	}

	docBytes, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	doc := string(docBytes)
	for _, intent := range fixture.Intents {
		human := strings.ReplaceAll(intent, "_", " ")
		if !strings.Contains(strings.ToLower(doc), strings.ToLower(human)) {
			t.Fatalf("concept coverage doc missing intent mapping: %s", intent)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := wd
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatalf("repo root not found from %s", wd)
		}
		root = parent
	}
}
