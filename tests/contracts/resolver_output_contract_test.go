package contracts_test

import (
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

func TestAllResolversFollowOutputContract(t *testing.T) {
	outputs := []common.ResolverOutput{
		classifylayer.ClassifyLayer(classifylayer.Item{Kind: "request"}),
		resolveguidanceprecedence.ResolveGuidancePrecedence([]resolveguidanceprecedence.GuidanceItem{{ID: "p", Type: "policy"}}),
		checkprojectioneligibility.CheckProjectionEligibility(checkprojectioneligibility.Source{Type: "policy"}),
		resolvefailuretier.ResolveFailureTier("approval_failure", resolvefailuretier.Context{Scope: "task"}),
		strictstatemachine.Transition(strictstatemachine.StateInput{CurrentState: "normal", Event: "trigger_guarded", Scope: "task"}),
		checkmutationscope.CheckMutationScope(checkmutationscope.Target{Path: "AGENTS.md"}),
		resolvelegacystatus.ResolveLegacyStatus(resolvelegacystatus.Asset{AssetID: "a", Ownership: "known", Freshness: "acceptable", CanonicalMapping: "exists", Integrity: "known"}),
		detectlegacydrift.DetectLegacyDrift(detectlegacydrift.Input{Signals: []string{"stale_review"}}),
		resolveoperationcapability.ResolveOperationCapability(resolveoperationcapability.Input{CommandOrAlias: "doctor"}),
		resolveextensionorder.ResolveExtensionOrder([]resolveextensionorder.Item{{ID: "policy-1", Kind: "policy"}}),
		resolveauditrequirements.ResolveAuditRequirements(resolveauditrequirements.Input{Action: "inspect"}),
	}

	allowedNext := map[string]struct{}{
		"inspect":  {},
		"preview":  {},
		"simulate": {},
		"propose":  {},
		"apply":    {},
		"deny":     {},
	}

	for i, out := range outputs {
		if out.Input == nil {
			t.Fatalf("output[%d] input is nil", i)
		}
		if out.Decision == nil {
			t.Fatalf("output[%d] decision is nil", i)
		}
		if out.Reason == "" {
			t.Fatalf("output[%d] reason is empty", i)
		}
		if out.Evidence == nil {
			t.Fatalf("output[%d] evidence is nil", i)
		}
		if _, ok := allowedNext[out.NextAllowedAction]; !ok {
			t.Fatalf("output[%d] next_allowed_action invalid: %q", i, out.NextAllowedAction)
		}
	}
}
