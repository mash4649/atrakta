package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/events"
	"atrakta/internal/util"
)

func TestCheckSchemaVersion(t *testing.T) {
	repo := t.TempDir()
	if _, err := events.Append(repo, "detect", "kernel", map[string]any{"signals": map[string]any{}, "target_set": []string{"cursor"}, "prune_allowed": false, "reason": "unknown"}); err != nil {
		t.Fatalf("append event failed: %v", err)
	}
	if err := Check(repo); err != nil {
		t.Fatalf("migrate check failed: %v", err)
	}
}

func TestCheckFailsForLegacySchemaV1(t *testing.T) {
	repo := t.TempDir()
	eventsPath := filepath.Join(repo, ".atrakta", "events.jsonl")
	if err := os.MkdirAll(filepath.Dir(eventsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	line1, hash1 := makeEventLineLegacy(nil, map[string]any{
		"v":              1,
		"schema_version": 1,
		"id":             "evt-1",
		"ts":             "2026-03-03T00:00:00Z",
		"type":           "detect",
		"actor":          "kernel",
		"signals":        map[string]any{},
		"target_set":     []any{"cursor"},
		"prune_allowed":  false,
		"reason":         "unknown",
	})
	line2, _ := makeEventLineLegacy(hash1, map[string]any{
		"v":              1,
		"schema_version": 1,
		"id":             "evt-2",
		"ts":             "2026-03-03T00:00:01Z",
		"type":           "step",
		"actor":          "worker",
		"task_id":        "feat-a",
		"outcome":        "DONE",
	})
	if err := os.WriteFile(eventsPath, []byte(line1+"\n"+line2+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Check(repo); err == nil {
		t.Fatalf("expected migrate check to fail for schema_version=1")
	}
}

func makeEventLineLegacy(prevHash any, payload map[string]any) (line string, hash string) {
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
