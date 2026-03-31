package resolvelegacystatus

import (
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

const (
	StatusReferenceOnly   = "reference_only"
	StatusPartiallyMapped = "partially_mapped"
	StatusCanonicalized   = "canonicalized"
)

// Asset is a legacy asset status input.
type Asset struct {
	AssetID                   string `json:"asset_id"`
	Ownership                 string `json:"ownership"`         // known | unknown
	Freshness                 string `json:"freshness"`         // acceptable | stale | unknown
	CanonicalMapping          string `json:"canonical_mapping"` // exists | missing
	CurrentStatus             string `json:"current_status,omitempty"`
	CanonicalConflict         bool   `json:"canonical_conflict,omitempty"`
	StaleReview               bool   `json:"stale_review,omitempty"`
	StaleTimestamp            bool   `json:"stale_timestamp,omitempty"`
	MissingMappedTarget       bool   `json:"missing_mapped_target,omitempty"`
	DuplicateGuidanceRisk     bool   `json:"duplicate_guidance_risk,omitempty"`
	DeprecatedStillReferenced bool   `json:"deprecated_still_referenced,omitempty"`
	Integrity                 string `json:"integrity,omitempty"` // known | unknown
}

// LegacyDecision contains status and drift routing.
type LegacyDecision struct {
	Status              string   `json:"status"`
	CanAutoPromote      bool     `json:"can_auto_promote"`
	DriftConditions     []string `json:"drift_conditions"`
	Routing             string   `json:"routing"`
	OwnershipKnown      bool     `json:"ownership_known"`
	FreshnessAcceptable bool     `json:"freshness_acceptable"`
	MappingExists       bool     `json:"mapping_exists"`
}

// ResolveLegacyStatus resolves promotion readiness and drift routing.
func ResolveLegacyStatus(asset Asset) common.ResolverOutput {
	ownershipKnown := normalize(asset.Ownership) == "known"
	freshnessAcceptable := normalize(asset.Freshness) == "acceptable"
	mappingExists := normalize(asset.CanonicalMapping) == "exists"
	integrityKnown := normalize(asset.Integrity) == "known" || normalize(asset.Integrity) == ""

	drift := collectDrift(asset)
	canAutoPromote := ownershipKnown && freshnessAcceptable && mappingExists && integrityKnown && len(drift) == 0

	status := StatusReferenceOnly
	switch {
	case canAutoPromote:
		status = StatusCanonicalized
	case mappingExists:
		status = StatusPartiallyMapped
	default:
		status = StatusReferenceOnly
	}

	routing := "warn"
	if hasStrictDrift(drift) {
		routing = "strict_escalate"
	} else if len(drift) > 0 {
		routing = "proposal"
	}

	reason := "legacy status resolved"
	next := "inspect"
	if status == StatusCanonicalized {
		next = "propose"
		reason = "canonicalized status ready"
	}
	if !canAutoPromote {
		reason = "metadata incomplete or drift detected; no auto promotion"
	}

	decision := LegacyDecision{
		Status:              status,
		CanAutoPromote:      canAutoPromote,
		DriftConditions:     drift,
		Routing:             routing,
		OwnershipKnown:      ownershipKnown,
		FreshnessAcceptable: freshnessAcceptable,
		MappingExists:       mappingExists,
	}

	evidence := []string{
		"ownership=" + normalize(asset.Ownership),
		"freshness=" + normalize(asset.Freshness),
		"mapping=" + normalize(asset.CanonicalMapping),
	}
	if !integrityKnown {
		evidence = append(evidence, "integrity=unknown")
	}
	for _, d := range drift {
		evidence = append(evidence, "drift="+d)
	}
	sort.Strings(evidence)

	return common.NewOutput(asset, decision, reason, evidence, next)
}

func collectDrift(asset Asset) []string {
	drift := []string{}
	if asset.CanonicalConflict {
		drift = append(drift, "canonical_conflict")
	}
	if asset.StaleTimestamp {
		drift = append(drift, "stale_timestamp")
	}
	if asset.StaleReview {
		drift = append(drift, "stale_review")
	}
	if asset.MissingMappedTarget {
		drift = append(drift, "missing_mapped_target")
	}
	if asset.DuplicateGuidanceRisk {
		drift = append(drift, "duplicate_guidance_risk")
	}
	if asset.DeprecatedStillReferenced {
		drift = append(drift, "deprecated_but_still_referenced")
	}
	if normalize(asset.Ownership) == "unknown" {
		drift = append(drift, "owner_unknown")
	}
	if normalize(asset.Freshness) == "unknown" {
		drift = append(drift, "freshness_unknown")
	}
	if normalize(asset.Integrity) == "unknown" {
		drift = append(drift, "integrity_unknown")
	}
	return drift
}

func hasStrictDrift(drift []string) bool {
	for _, d := range drift {
		if d == "canonical_conflict" || d == "missing_mapped_target" || d == "integrity_unknown" {
			return true
		}
	}
	return false
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
