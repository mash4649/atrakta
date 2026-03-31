package audit

import "testing"

func TestAppendAndVerify(t *testing.T) {
	root := t.TempDir()
	ev, err := AppendAndVerify(root, LevelA2, "run_execute", map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("append and verify: %v", err)
	}
	if ev.Seq != 1 {
		t.Fatalf("seq=%d", ev.Seq)
	}
	if ev.Hash == "" {
		t.Fatal("hash should not be empty for A2")
	}
}
