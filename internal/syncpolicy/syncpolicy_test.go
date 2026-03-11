package syncpolicy

import (
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
)

func TestProposeFromAGENTSHints(t *testing.T) {
	c := contract.Default("/tmp/repo")
	agents := "sync.prefer_interfaces: cursor,trae\n" +
		"sync.disable_interfaces: opencode\n"

	sp, proposed, err := ProposeFromAGENTS(c, agents)
	if err != nil {
		t.Fatalf("proposal failed: %v", err)
	}
	if !sp.Needed || !sp.RequiresApproval {
		t.Fatalf("expected needed proposal")
	}
	if proposed.Hints == nil {
		t.Fatalf("expected hints initialized")
	}
	if len(proposed.Hints.Prefer) != 2 {
		t.Fatalf("unexpected prefer: %#v", proposed.Hints.Prefer)
	}
	if len(proposed.Hints.DisableInterfaces) != 1 || proposed.Hints.DisableInterfaces[0] != "opencode" {
		t.Fatalf("unexpected disable: %#v", proposed.Hints.DisableInterfaces)
	}
	if len(sp.Allowed) == 0 {
		t.Fatalf("expected allowed field diffs")
	}
}

func TestProposeFromAGENTSExpandedAllowlistAndDeniedFields(t *testing.T) {
	c := contract.Default("/tmp/repo")
	agents := "sync.approval_required_for: boundary_expand,external_side_effect\n" +
		"sync.quick_checks: projection_integrity,managed_block_integrity\n" +
		"sync.heavy_checks: go_test_compile,slow_e2e\n" +
		"sync.prompt_min.required: true\n" +
		"sync.prompt_min.goal_label: Objective\n" +
		"sync.parity.output.plan_format: json\n" +
		"sync.parity.output.error_format: plain\n" +
		"sync.extensions.hooks.shell.on_cd: true\n" +
		"sync.extensions.hooks.ide.on_open: true\n" +
		"sync.extensions.plugins.demo: enabled\n"

	sp, proposed, err := ProposeFromAGENTS(c, agents)
	if err != nil {
		t.Fatalf("proposal failed: %v", err)
	}
	if !sp.Needed {
		t.Fatalf("expected proposal changes")
	}
	if len(sp.Allowed) < 8 {
		t.Fatalf("expected multiple allowlisted field diffs, got %#v", sp.Allowed)
	}
	if !hasField(sp.Allowed, "tools.approval_required_for") {
		t.Fatalf("missing tools.approval_required_for allowed diff: %#v", sp.Allowed)
	}
	if !hasField(sp.Allowed, "quality.quick_checks") {
		t.Fatalf("missing quality.quick_checks allowed diff")
	}
	if !hasField(sp.Allowed, "quality.heavy_checks") {
		t.Fatalf("missing quality.heavy_checks allowed diff")
	}
	if !hasField(sp.Allowed, "policies.prompt_min.required") || !hasField(sp.Allowed, "policies.prompt_min.goal_label") {
		t.Fatalf("missing prompt_min allowed diffs")
	}
	if !hasField(sp.Allowed, "parity.output_surface.plan_format") || !hasField(sp.Allowed, "parity.output_surface.error_format") {
		t.Fatalf("missing parity output surface allowed diffs")
	}
	if !hasField(sp.Allowed, "extensions.hooks.shell.on_cd") || !hasField(sp.Allowed, "extensions.hooks.ide.on_open") {
		t.Fatalf("missing hooks allowlisted diffs")
	}
	if !hasDenied(sp.Denied, "extensions.plugins.demo") {
		t.Fatalf("expected denied protected field diff, got %#v", sp.Denied)
	}
	if proposed.Extensions != nil && len(proposed.Extensions.Plugins) != 0 {
		t.Fatalf("protected plugin fields must not be reverse-synced")
	}
}

func TestProposeLevelParser(t *testing.T) {
	if ParseLevel("2") != Level2 {
		t.Fatal("expected level2")
	}
	if ParseLevel("level1") != Level1 {
		t.Fatal("expected level1")
	}
	if ParseLevel("") != Level0 {
		t.Fatal("expected level0")
	}
}

func hasField(list []model.SyncFieldDiff, field string) bool {
	for _, d := range list {
		if d.Field == field {
			return true
		}
	}
	return false
}

func hasDenied(list []model.SyncFieldDiff, field string) bool {
	for _, d := range list {
		if d.Field == field && d.Status == "denied" {
			return true
		}
	}
	return false
}
