package adapters

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExecuteRunLoopNeedsInputThenApprovalThenSuccess(t *testing.T) {
	attempt := 0
	invoker := func(args []string) (int, []byte, error) {
		attempt++
		switch attempt {
		case 1:
			return 2, mustJSON(t, map[string]any{
				"status":          "needs_input",
				"required_inputs": []string{"interface"},
			}), nil
		case 2:
			return 3, mustJSON(t, map[string]any{
				"status":         "needs_approval",
				"approval_scope": "managed_apply",
			}), nil
		default:
			return 0, mustJSON(t, map[string]any{"status": "ok"}), nil
		}
	}

	out, err := ExecuteRunLoop(ExecuteConfig{
		BaseArgs:    []string{"--json", "--apply"},
		MaxAttempts: 4,
		InvokeRun:   invoker,
		CollectInputs: func(required []string) (map[string]string, error) {
			return map[string]string{"interface": "cursor"}, nil
		},
		CollectApproval: func(scope string) (bool, error) {
			if scope != "managed_apply" {
				t.Fatalf("unexpected scope: %s", scope)
			}
			return true, nil
		},
	})
	if err != nil {
		t.Fatalf("ExecuteRunLoop: %v", err)
	}
	if out.FinalExitCode != 0 {
		t.Fatalf("final exit code=%d", out.FinalExitCode)
	}
	if len(out.Attempts) != 3 {
		t.Fatalf("attempt count=%d", len(out.Attempts))
	}
}

func TestExecuteRunLoopNeedsInputWithoutCollector(t *testing.T) {
	out, err := ExecuteRunLoop(ExecuteConfig{
		BaseArgs: []string{"--json"},
		InvokeRun: func(args []string) (int, []byte, error) {
			return 2, mustJSON(t, map[string]any{"status": "needs_input", "required_inputs": []string{"interface"}}), nil
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if out.FinalExitCode != 2 {
		t.Fatalf("final exit code=%d", out.FinalExitCode)
	}
}

func TestExecuteRunLoopApprovalDenied(t *testing.T) {
	out, err := ExecuteRunLoop(ExecuteConfig{
		BaseArgs: []string{"--json", "--apply"},
		InvokeRun: func(args []string) (int, []byte, error) {
			return 3, mustJSON(t, map[string]any{"status": "needs_approval", "approval_scope": "onboarding_accept"}), nil
		},
		CollectApproval: func(scope string) (bool, error) {
			return false, nil
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "approval denied by adapter") {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.FinalExitCode != 3 {
		t.Fatalf("final exit code=%d", out.FinalExitCode)
	}
}

func mustJSON(t *testing.T, payload map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
