package detectlegacydrift

import (
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

// Input contains drift candidates.
type Input struct {
	Signals []string `json:"signals"`
}

// DriftDecision contains drift detection result.
type DriftDecision struct {
	Detected []string `json:"detected"`
	Severity string   `json:"severity"`
}

var known = map[string]struct{}{
	"canonical_conflict":              {},
	"stale_timestamp":                 {},
	"stale_review":                    {},
	"missing_mapped_target":           {},
	"duplicate_guidance_risk":         {},
	"deprecated_but_still_referenced": {},
}

// DetectLegacyDrift returns recognized drift signals.
func DetectLegacyDrift(in Input) common.ResolverOutput {
	detected := []string{}
	severity := "none"
	for _, s := range in.Signals {
		n := normalize(s)
		if _, ok := known[n]; ok {
			detected = append(detected, n)
		}
	}

	if len(detected) > 0 {
		severity = "warn"
	}
	for _, d := range detected {
		if d == "canonical_conflict" || d == "missing_mapped_target" {
			severity = "strict"
			break
		}
	}

	next := "inspect"
	reason := "no drift detected"
	if len(detected) > 0 {
		next = "propose"
		reason = "drift detected"
	}
	if severity == "strict" {
		next = "deny"
		reason = "strict drift detected"
	}

	return common.NewOutput(in, DriftDecision{Detected: detected, Severity: severity}, reason, detected, next)
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
