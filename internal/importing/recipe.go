package importing

import (
	"fmt"
	"sort"
	"strings"

	"atrakta/internal/events"
	"atrakta/internal/util"
)

func ConvertRecipeCandidate(repoRoot, capabilityID, reviewStatus, deterministicInputNote, inputContractRef string, allowlist []string) (CapabilityEntry, error) {
	reg, _, err := LoadRegistry(repoRoot)
	if err != nil {
		return CapabilityEntry{}, err
	}
	entry, idx := findEntry(&reg, capabilityID)
	if entry == nil {
		return CapabilityEntry{}, fmt.Errorf("capability not found: %s", capabilityID)
	}
	if entry.Denied {
		return CapabilityEntry{}, fmt.Errorf("capability denied and cannot be converted")
	}
	if entry.Kind != KindSkill && entry.Kind != KindRecipeCandidate {
		return CapabilityEntry{}, fmt.Errorf("only skill/recipe_candidate can be converted")
	}
	review := strings.TrimSpace(strings.ToLower(reviewStatus))
	if review == "" {
		review = ReviewPending
	}
	if review != ReviewPending && review != ReviewApproved && review != ReviewRejected {
		return CapabilityEntry{}, fmt.Errorf("invalid review status: %s", reviewStatus)
	}
	note := strings.TrimSpace(deterministicInputNote)
	if note == "" {
		return CapabilityEntry{}, fmt.Errorf("deterministic_input_note is required")
	}

	entry.Kind = KindRecipeCandidate
	entry.Recipe = &RecipeCandidate{
		TimeoutSec:             30,
		MaxSteps:               12,
		Allowlist:              dedupeSorted(allowlist),
		ApprovalRequired:       true,
		DeterministicInputNote: note,
		InputContractRef:       strings.TrimSpace(inputContractRef),
	}
	entry.ReviewStatus = review
	switch review {
	case ReviewApproved:
		entry.ConversionStatus = ConversionReviewedApproved
	case ReviewRejected:
		entry.ConversionStatus = ConversionReviewedRejected
	default:
		entry.ConversionStatus = ConversionReviewPending
	}
	entry.Executable = false
	entry.UpdatedAt = util.NowUTC()
	reg.Entries[idx] = *entry
	if err := SaveRegistry(repoRoot, reg); err != nil {
		return CapabilityEntry{}, err
	}

	_, _ = events.Append(repoRoot, events.EventRecipeConversionReviewed, "operator", map[string]any{
		"capability_id": capabilityID,
		"review_status": review,
		"allowlist":     entry.Recipe.Allowlist,
	})
	if review == ReviewApproved {
		_, _ = events.Append(repoRoot, events.EventRecipeCandidateCreated, "operator", map[string]any{
			"capability_id": capabilityID,
			"kind":          entry.Kind,
			"max_steps":     entry.Recipe.MaxSteps,
			"timeout_sec":   entry.Recipe.TimeoutSec,
		})
	}
	return *entry, nil
}

func RecipeAllows(entry CapabilityEntry, primitive string) bool {
	if entry.Recipe == nil {
		return false
	}
	primitive = strings.TrimSpace(strings.ToLower(primitive))
	if primitive == "" {
		return false
	}
	for _, allowed := range entry.Recipe.Allowlist {
		if strings.TrimSpace(strings.ToLower(allowed)) == primitive {
			return true
		}
	}
	return false
}

func dedupeSorted(in []string) []string {
	set := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			continue
		}
		if _, ok := set[s]; ok {
			continue
		}
		set[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return []string{}
	}
	sort.Strings(out)
	return out
}
