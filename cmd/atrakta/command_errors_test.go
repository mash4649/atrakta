package main

import (
	"encoding/json"
	"errors"
	"testing"

	atraktaerrors "github.com/mash4649/atrakta/v0/internal/errors"
)

func TestReportCommandErrorEmitsStructuredJSONEnvelope(t *testing.T) {
	raw := captureStdout(t, func() {
		code := reportCommandError(
			"wrap",
			[]string{"--json"},
			exitRuntimeError,
			atraktaerrors.NewExitError(
				exitRuntimeError,
				atraktaerrors.Usage(
					"wrap requires install, uninstall, or run",
					"Run `atrakta wrap --help` to inspect the supported subcommands.",
				),
				false,
			),
		)
		if code != exitRuntimeError {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal command error output: %v", err)
	}
	if out["status"] != "error" {
		t.Fatalf("status=%v", out["status"])
	}
	errPayload, ok := out["error"].(map[string]any)
	if !ok {
		t.Fatalf("error payload missing: %#v", out["error"])
	}
	if errPayload["code"] != "ERR_USAGE" {
		t.Fatalf("error code=%v", errPayload["code"])
	}
	steps, ok := errPayload["recovery_steps"].([]any)
	if !ok || len(steps) == 0 {
		t.Fatalf("recovery_steps missing: %#v", errPayload["recovery_steps"])
	}
}

func TestReportCommandErrorClassifiesPlainErrors(t *testing.T) {
	raw := captureStdout(t, func() {
		code := reportCommandError(
			"resume",
			[]string{"--json"},
			exitRuntimeError,
			errors.New("resume requires existing atrakta state; run `atrakta start` first"),
		)
		if code != exitRuntimeError {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal command error output: %v", err)
	}
	errPayload, ok := out["error"].(map[string]any)
	if !ok {
		t.Fatalf("error payload missing: %#v", out["error"])
	}
	if errPayload["code"] != "ERR_BLOCKED" {
		t.Fatalf("error code=%v", errPayload["code"])
	}
}
