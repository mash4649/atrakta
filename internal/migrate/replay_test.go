package migrate

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"atrakta/internal/doctor"
	"atrakta/internal/events"
	"atrakta/internal/util"
)

func TestH7ReplayDeterminismWithSchemaVersion2AdditiveFields(t *testing.T) {
	repo := t.TempDir()
	eventsPath := filepath.Join(repo, ".atrakta", "events.jsonl")
	if err := os.MkdirAll(filepath.Dir(eventsPath), 0o755); err != nil {
		t.Fatal(err)
	}

	line1, hash1 := makeEventLine(nil, map[string]any{
		"v":              1,
		"schema_version": 2,
		"id":             "evt-1",
		"ts":             "2026-03-02T00:00:00Z",
		"type":           "detect",
		"actor":          "kernel",
		"signals":        map[string]any{},
		"target_set":     []any{"cursor"},
		"prune_allowed":  false,
		"reason":         "unknown",
		"additive_note":  "allowed field",
	})
	line2, _ := makeEventLine(hash1, map[string]any{
		"v":              1,
		"schema_version": 2,
		"id":             "evt-2",
		"ts":             "2026-03-02T00:00:01Z",
		"type":           "apply",
		"actor":          "worker",
		"plan_id":        "plan-1",
		"feature_id":     "feat-1",
		"result":         "success",
		"ops": []any{map[string]any{
			"path":        ".cursor/AGENTS.md",
			"op":          "link",
			"status":      "ok",
			"error":       "",
			"interface":   "cursor",
			"template_id": "cursor:agents-md@1",
			"fingerprint": "sha256:test",
			"kind":        "link",
			"target":      "AGENTS.md",
			"extra":       "ignored",
		}},
	})
	if err := os.WriteFile(eventsPath, []byte(line1+"\n"+line2+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := events.VerifyChain(repo); err != nil {
		t.Fatalf("chain verify failed: %v", err)
	}
	if err := Check(repo); err != nil {
		t.Fatalf("migration check failed: %v", err)
	}

	s1, err := doctor.RebuildStateFromEvents(repo)
	if err != nil {
		t.Fatalf("rebuild #1 failed: %v", err)
	}
	s2, err := doctor.RebuildStateFromEvents(repo)
	if err != nil {
		t.Fatalf("rebuild #2 failed: %v", err)
	}
	if !reflect.DeepEqual(s1.ManagedPaths, s2.ManagedPaths) {
		t.Fatalf("rebuild is not deterministic")
	}
	rec, ok := s1.ManagedPaths[".cursor/AGENTS.md"]
	if !ok {
		t.Fatalf("expected managed path from apply event")
	}
	if rec.Fingerprint != "sha256:test" || rec.TemplateID != "cursor:agents-md@1" {
		t.Fatalf("unexpected rebuilt record: %#v", rec)
	}
}

func makeEventLine(prevHash any, payload map[string]any) (line string, hash string) {
	m := map[string]any{}
	for k, v := range payload {
		m[k] = v
	}
	m["prev_hash"] = prevHash

	canonNoHash := map[string]any{}
	for k, v := range m {
		canonNoHash[k] = v
	}
	b, _ := util.MarshalCanonical(canonNoHash)
	hash = util.SHA256Tagged(b)
	m["hash"] = hash
	out, _ := util.MarshalCanonical(m)
	return string(out), hash
}
