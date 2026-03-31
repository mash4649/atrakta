package resolvesurfaceportability

import (
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

const (
	TargetAgentsMD    = "agents_md"
	TargetIDERules    = "ide_rules"
	TargetRepoDocs    = "repo_docs"
	TargetSkillBundle = "skill_bundle"

	SourceCanonicalPolicy = "canonical_policy"
	SourceWorkflowBinding = "workflow_binding"
	SourceSkillAsset      = "skill_asset"
	SourceRepoDocs        = "repo_docs"
	SourceAgentsMD        = "agents_md"
	SourceIDERules        = "ide_rules"

	PortabilitySupported   = "supported"
	PortabilityDegraded    = "degraded"
	PortabilityUnsupported = "unsupported"

	DegradePolicyProposalOnly = "proposal_only"

	PortabilityModeRequired    = "semantic_required"
	PortabilityModeBestEffort  = "semantic_best_effort"
	PortabilityModeUnsupported = "unsupported"
)

// BindingCapabilities describes what a binding can read, project, and approve.
type BindingCapabilities struct {
	InterfaceID           string   `json:"id"`
	Kind                  string   `json:"kind,omitempty"`
	Surfaces              []string `json:"surfaces,omitempty"`
	ProjectionTargets     []string `json:"projection_targets,omitempty"`
	IngestSources         []string `json:"ingest_sources,omitempty"`
	ApprovalChannel       string   `json:"approval_channel,omitempty"`
	PortabilityMode       string   `json:"portability_mode,omitempty"`
	CanMutateCoreContract bool     `json:"can_mutate_core_contract,omitempty"`
}

// Input is the portability resolver contract.
type Input struct {
	InterfaceID         string              `json:"interface_id"`
	RequestedTargets    []string            `json:"requested_targets"`
	AvailableSources    []string            `json:"available_sources"`
	BindingCapabilities BindingCapabilities `json:"binding_capabilities"`
	DegradePolicy       string              `json:"degrade_policy"`
}

// ProjectionPlanItem describes how one requested target is realized.
type ProjectionPlanItem struct {
	RequestedTarget string `json:"requested_target"`
	EffectiveTarget string `json:"effective_target,omitempty"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
}

// PortabilityDecision is the normalized surface portability result.
type PortabilityDecision struct {
	SupportedTargets   []string             `json:"supported_targets"`
	DegradedTargets    []string             `json:"degraded_targets"`
	UnsupportedTargets []string             `json:"unsupported_targets"`
	IngestPlan         []string             `json:"ingest_plan"`
	ProjectionPlan     []ProjectionPlanItem `json:"projection_plan"`
	PortabilityStatus  string               `json:"portability_status"`
	DegradePolicy      string               `json:"degrade_policy"`
}

// ResolveSurfacePortability resolves surface portability for an interface.
func ResolveSurfacePortability(input Input) common.ResolverOutput {
	in := input.normalize()
	decision := PortabilityDecision{
		SupportedTargets:   []string{},
		DegradedTargets:    []string{},
		UnsupportedTargets: []string{},
		IngestPlan:         buildIngestPlan(in.BindingCapabilities.IngestSources, in.AvailableSources),
		ProjectionPlan:     []ProjectionPlanItem{},
		PortabilityStatus:  PortabilitySupported,
		DegradePolicy:      in.DegradePolicy,
	}

	next := "propose"
	reason := "surface portability resolved"
	evidence := []string{
		"degrade_policy=" + in.DegradePolicy,
		"interface_id=" + in.InterfaceID,
		"portability_mode=" + in.BindingCapabilities.PortabilityMode,
	}

	if len(in.RequestedTargets) == 0 {
		evidence = append(evidence, "requested_targets=0")
		sort.Strings(evidence)
		return common.NewOutput(in, decision, "no portability targets requested", evidence, next)
	}

	if in.BindingCapabilities.PortabilityMode == PortabilityModeUnsupported {
		for _, target := range in.RequestedTargets {
			decision.UnsupportedTargets = append(decision.UnsupportedTargets, target)
			decision.ProjectionPlan = append(decision.ProjectionPlan, ProjectionPlanItem{
				RequestedTarget: target,
				Status:          PortabilityUnsupported,
				Reason:          "binding portability mode unsupported",
			})
		}
		decision.PortabilityStatus = PortabilityUnsupported
		reason = "binding portability mode unsupported"
		next = "inspect"
		evidence = append(evidence, "unsupported_targets_present")
		sort.Strings(evidence)
		return common.NewOutput(in, decision, reason, evidence, next)
	}

	hasIngest := len(decision.IngestPlan) > 0
	supportedTargets := asSet(in.BindingCapabilities.ProjectionTargets)

	for _, target := range in.RequestedTargets {
		switch {
		case hasIngest && hasString(supportedTargets, target):
			decision.SupportedTargets = append(decision.SupportedTargets, target)
			decision.ProjectionPlan = append(decision.ProjectionPlan, ProjectionPlanItem{
				RequestedTarget: target,
				EffectiveTarget: target,
				Status:          PortabilitySupported,
				Reason:          "binding supports requested target",
			})
		default:
			fallback := fallbackTarget(target)
			if hasIngest && fallback != "" && hasString(supportedTargets, fallback) {
				decision.DegradedTargets = append(decision.DegradedTargets, target)
				decision.ProjectionPlan = append(decision.ProjectionPlan, ProjectionPlanItem{
					RequestedTarget: target,
					EffectiveTarget: fallback,
					Status:          PortabilityDegraded,
					Reason:          "requested target degraded to supported fallback",
				})
				continue
			}
			reasonText := "binding does not support requested target"
			if !hasIngest {
				reasonText = "binding has no ingestable advisory sources"
			}
			decision.UnsupportedTargets = append(decision.UnsupportedTargets, target)
			decision.ProjectionPlan = append(decision.ProjectionPlan, ProjectionPlanItem{
				RequestedTarget: target,
				Status:          PortabilityUnsupported,
				Reason:          reasonText,
			})
		}
	}

	switch {
	case len(decision.UnsupportedTargets) > 0:
		decision.PortabilityStatus = PortabilityUnsupported
		reason = "unsupported portability targets detected"
		next = "inspect"
		evidence = append(evidence, "unsupported_targets_present")
	case len(decision.DegradedTargets) > 0:
		decision.PortabilityStatus = PortabilityDegraded
		reason = "portability degraded to supported fallback targets"
		evidence = append(evidence, "degraded_targets_present")
	default:
		reason = "requested targets supported without degradation"
	}

	sort.Strings(evidence)
	return common.NewOutput(in, decision, reason, evidence, next)
}

// Normalize returns a normalized binding capability descriptor.
func (b BindingCapabilities) Normalize() BindingCapabilities {
	b.InterfaceID = normalize(b.InterfaceID)
	b.Kind = normalize(b.Kind)
	b.Surfaces = normalizeUnique(b.Surfaces)
	b.ProjectionTargets = normalizeUnique(b.ProjectionTargets)
	b.IngestSources = normalizeUnique(b.IngestSources)
	b.ApprovalChannel = normalize(b.ApprovalChannel)
	b.PortabilityMode = normalize(b.PortabilityMode)
	if b.ApprovalChannel == "" {
		b.ApprovalChannel = "unsupported"
	}
	if b.PortabilityMode == "" {
		b.PortabilityMode = PortabilityModeUnsupported
	}
	return b
}

func (in Input) normalize() Input {
	in.InterfaceID = normalize(in.InterfaceID)
	in.RequestedTargets = normalizeUnique(in.RequestedTargets)
	in.AvailableSources = normalizeUnique(in.AvailableSources)
	in.BindingCapabilities = in.BindingCapabilities.Normalize()
	if in.BindingCapabilities.InterfaceID == "" {
		in.BindingCapabilities.InterfaceID = in.InterfaceID
	}
	in.DegradePolicy = normalize(in.DegradePolicy)
	if in.DegradePolicy == "" {
		in.DegradePolicy = DegradePolicyProposalOnly
	}
	return in
}

func buildIngestPlan(bindingSources, availableSources []string) []string {
	available := asSet(availableSources)
	plan := make([]string, 0, len(bindingSources))
	for _, src := range bindingSources {
		if hasString(available, src) {
			plan = append(plan, src)
		}
	}
	return normalizeUnique(plan)
}

func fallbackTarget(target string) string {
	switch normalize(target) {
	case TargetAgentsMD:
		return TargetIDERules
	case TargetIDERules:
		return TargetAgentsMD
	case TargetSkillBundle:
		return TargetRepoDocs
	default:
		return ""
	}
}

func asSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[normalize(value)] = struct{}{}
	}
	return out
}

func hasString(values map[string]struct{}, value string) bool {
	_, ok := values[normalize(value)]
	return ok
}

func normalizeUnique(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		n := normalize(value)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
