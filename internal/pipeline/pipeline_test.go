package pipeline

import "testing"

func TestExecuteOrderedAndModeClamp(t *testing.T) {
	out, err := ExecuteOrdered("inspect", DefaultInput("inspect"))
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if out.Mode != "inspect" {
		t.Fatalf("mode=%q", out.Mode)
	}
	if out.FinalAllowedAction != "inspect" && out.FinalAllowedAction != "deny" {
		t.Fatalf("inspect mode final action must be inspect/deny, got %q", out.FinalAllowedAction)
	}
	if len(out.Steps) == 0 {
		t.Fatalf("steps should not be empty")
	}
}
