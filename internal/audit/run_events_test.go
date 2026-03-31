package audit

import (
	"os"
	"testing"
)

func TestAppendRunEventAndVerifyA2(t *testing.T) {
	root := t.TempDir()

	ev, err := AppendRunEventAndVerify(root, LevelA2, "start.begin", map[string]any{
		"status":       "ok",
		"interface_id": "generic-cli",
	}, RunEventOptions{Actor: "kernel", Interface: "generic-cli"})
	if err != nil {
		t.Fatalf("append run event: %v", err)
	}
	if ev.SchemaVersion != runEventSchemaVersion {
		t.Fatalf("schema_version=%d", ev.SchemaVersion)
	}
	if ev.Seq != 1 {
		t.Fatalf("seq=%d", ev.Seq)
	}
	if ev.EventType != "start.begin" {
		t.Fatalf("event_type=%q", ev.EventType)
	}
	if ev.PayloadHash == "" || ev.Hash == "" {
		t.Fatal("expected payload_hash/hash for A2")
	}
	if ev.Interface != "generic-cli" {
		t.Fatalf("interface=%q", ev.Interface)
	}
}

func TestVerifyRunEventsIntegrityDetectsTamper(t *testing.T) {
	root := t.TempDir()

	if _, err := AppendRunEventAndVerify(root, LevelA2, "plan.created", map[string]any{
		"planned_count": 3,
	}, RunEventOptions{}); err != nil {
		t.Fatalf("append run event: %v", err)
	}

	path := RunEventsPath(root)
	if err := os.WriteFile(path, []byte("{\"not\":\"valid\"}\n"), 0o644); err != nil {
		t.Fatalf("tamper file: %v", err)
	}

	if err := VerifyRunEventsIntegrity(root, LevelA2); err == nil {
		t.Fatal("expected integrity error")
	}
}
