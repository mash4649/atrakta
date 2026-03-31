package resolvers_test

import (
	"strings"
	"testing"

	resolveauditrequirements "github.com/mash4649/atrakta/v0/resolvers/audit/resolve-audit-requirements"
	"github.com/mash4649/atrakta/v0/resolvers/common"
	resolveextensionorder "github.com/mash4649/atrakta/v0/resolvers/extension/resolve-extension-order"
	resolvefailuretier "github.com/mash4649/atrakta/v0/resolvers/failure/resolve-failure-tier"
	strictstatemachine "github.com/mash4649/atrakta/v0/resolvers/failure/strict-state-machine"
	resolveguidanceprecedence "github.com/mash4649/atrakta/v0/resolvers/guidance/resolve-guidance-precedence"
	classifylayer "github.com/mash4649/atrakta/v0/resolvers/layer/classify-layer"
	detectlegacydrift "github.com/mash4649/atrakta/v0/resolvers/legacy/detect-legacy-drift"
	resolvelegacystatus "github.com/mash4649/atrakta/v0/resolvers/legacy/resolve-legacy-status"
	checkmutationscope "github.com/mash4649/atrakta/v0/resolvers/mutation/check-mutation-scope"
	resolveoperationcapability "github.com/mash4649/atrakta/v0/resolvers/operations/resolve-operation-capability"
	checkprojectioneligibility "github.com/mash4649/atrakta/v0/resolvers/projection/check-projection-eligibility"
)

func FuzzResolverInputs(f *testing.F) {
	seeds := []string{
		"doctor|AGENTS.md|stale_review",
		"apply|.atrakta/generated/repo-map.generated.json|canonical_conflict",
		"preview|workflow|missing_approval",
		"simulate|hook|release_approved",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, seed string) {
		parts := splitSeed(seed)
		checkOutput := func(out common.ResolverOutput) {
			t.Helper()
			if out.Input == nil {
				t.Fatal("resolver output input is nil")
			}
			if out.Decision == nil {
				t.Fatal("resolver output decision is nil")
			}
			if strings.TrimSpace(out.Reason) == "" {
				t.Fatal("resolver output reason is empty")
			}
			if out.Evidence == nil {
				t.Fatal("resolver output evidence is nil")
			}
			switch out.NextAllowedAction {
			case "inspect", "preview", "simulate", "propose", "apply", "deny":
			default:
				t.Fatalf("invalid next_allowed_action=%q", out.NextAllowedAction)
			}
		}

		checkOutput(classifylayer.ClassifyLayer(classifylayer.Item{
			Kind:     pick(parts, 0, "request"),
			SchemaID: pick(parts, 1, "atrakta/schemas/core/request.schema.json"),
		}))

		checkOutput(resolveguidanceprecedence.ResolveGuidancePrecedence([]resolveguidanceprecedence.GuidanceItem{
			{ID: pick(parts, 0, "policy-1"), Type: pick(parts, 1, "policy")},
			{ID: pick(parts, 2, "workflow-1"), Type: pick(parts, 3, "workflow"), ClaimsDecisionOverride: len(seed)%2 == 0},
		}))

		checkOutput(checkprojectioneligibility.CheckProjectionEligibility(checkprojectioneligibility.Source{
			Type:               pick(parts, 0, "policy"),
			HasCanonicalAnchor: len(seed)%2 == 0,
		}))

		checkOutput(resolvefailuretier.ResolveFailureTier(pick(parts, 0, "approval_failure"), resolvefailuretier.Context{
			Scope:             pick(parts, 1, "task"),
			Triggers:          []string{pick(parts, 2, "missing_approval"), pick(parts, 3, "policy_ambiguity")},
			RequestedOverride: pick(parts, 4, ""),
			IsDiagnosticsOnly: len(seed)%3 == 0,
		}))

		checkOutput(strictstatemachine.Transition(strictstatemachine.StateInput{
			CurrentState:    pick(parts, 0, "normal"),
			Event:           pick(parts, 1, "trigger_guarded"),
			Scope:           pick(parts, 2, "task"),
			ReleaseApproved: len(seed)%2 == 0,
		}))

		checkOutput(checkmutationscope.CheckMutationScope(checkmutationscope.Target{
			Path:            pick(parts, 0, "AGENTS.md"),
			DeclaredScope:   pick(parts, 1, ""),
			AssetType:       pick(parts, 2, "policy"),
			Operation:       pick(parts, 3, "replace"),
			HasAmbiguity:    len(seed)%2 == 0,
			ManagedOnlyPath: len(seed)%3 == 0,
		}))

		checkOutput(resolvelegacystatus.ResolveLegacyStatus(resolvelegacystatus.Asset{
			AssetID:                   pick(parts, 0, "legacy-asset"),
			Ownership:                 pick(parts, 1, "known"),
			Freshness:                 pick(parts, 2, "acceptable"),
			CanonicalMapping:          pick(parts, 3, "exists"),
			Integrity:                 pick(parts, 4, "known"),
			CanonicalConflict:         len(seed)%2 == 0,
			StaleReview:               len(seed)%3 == 0,
			StaleTimestamp:            len(seed)%5 == 0,
			MissingMappedTarget:       len(seed)%7 == 0,
			DuplicateGuidanceRisk:     len(seed)%11 == 0,
			DeprecatedStillReferenced: len(seed)%13 == 0,
		}))

		checkOutput(detectlegacydrift.DetectLegacyDrift(detectlegacydrift.Input{
			Signals: []string{
				pick(parts, 0, "stale_review"),
				pick(parts, 1, "canonical_conflict"),
				pick(parts, 2, "unknown_signal"),
			},
		}))

		checkOutput(resolveoperationcapability.ResolveOperationCapability(resolveoperationcapability.Input{
			CommandOrAlias: pick(parts, 0, "doctor"),
			FailureTier:    pick(parts, 1, resolvefailuretier.TierWarnOnly),
		}))

		checkOutput(resolveextensionorder.ResolveExtensionOrder([]resolveextensionorder.Item{
			{
				ID:                             pick(parts, 0, "policy-1"),
				Kind:                           pick(parts, 1, "policy"),
				AttemptsCoreMutation:           len(seed)%2 == 0,
				DiagnosticsConstrainsExecution: len(seed)%3 == 0,
			},
			{
				ID:                   pick(parts, 2, "hook-1"),
				Kind:                 pick(parts, 3, "hook"),
				HookMutatesCanonical: len(seed)%5 == 0,
			},
		}))

		checkOutput(resolveauditrequirements.ResolveAuditRequirements(resolveauditrequirements.Input{
			Action:                  pick(parts, 0, "inspect"),
			RequestedIntegrityLevel: pick(parts, 1, "A2"),
			DestructiveCleanup:      len(seed)%2 == 0,
		}))
	})
}

func splitSeed(seed string) []string {
	parts := strings.FieldsFunc(seed, func(r rune) bool {
		switch r {
		case '|', ',', '/', ' ', '\n', '\t':
			return true
		default:
			return false
		}
	})
	if len(parts) == 0 {
		return []string{seed}
	}
	return parts
}

func pick(parts []string, idx int, fallback string) string {
	if idx < 0 || idx >= len(parts) {
		return fallback
	}
	v := strings.TrimSpace(parts[idx])
	if v == "" {
		return fallback
	}
	return v
}
