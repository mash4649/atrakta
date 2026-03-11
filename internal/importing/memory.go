package importing

import (
	"fmt"
	"strings"

	"atrakta/internal/events"
	"atrakta/internal/util"
)

func ReviewMemoryPromotion(repoRoot, capabilityID, reviewStatus, operator string, promote bool) (MemoryReviewResult, error) {
	reg, _, err := LoadRegistry(repoRoot)
	if err != nil {
		return MemoryReviewResult{}, err
	}
	entry, idx := findEntry(&reg, capabilityID)
	if entry == nil {
		return MemoryReviewResult{}, fmt.Errorf("capability not found: %s", capabilityID)
	}
	if entry.Denied {
		return MemoryReviewResult{}, fmt.Errorf("denied capability cannot be promoted")
	}
	if entry.CurrentMemorySurface == "" {
		entry.CurrentMemorySurface = entry.DefaultMemorySurface
	}
	if entry.CurrentMemorySurface == "" {
		entry.CurrentMemorySurface = string(KindReferenceMemory)
	}

	review := strings.TrimSpace(strings.ToLower(reviewStatus))
	if review == "" {
		review = ReviewPending
	}
	if review != ReviewPending && review != ReviewApproved && review != ReviewRejected {
		return MemoryReviewResult{}, fmt.Errorf("invalid review status: %s", reviewStatus)
	}

	res := MemoryReviewResult{Promoted: false, Reason: "review pending"}
	entry.ReviewStatus = review
	if promote {
		if review == ReviewApproved && strings.TrimSpace(operator) != "" {
			entry.CurrentMemorySurface = "operational_memory"
			entry.ConversionStatus = ConversionReviewedApproved
			entry.ReviewStatus = ReviewApproved
			entry.Executable = false
			res.Promoted = true
			res.Reason = "promoted with review approval"
		} else {
			res.Promoted = false
			res.Reason = "promotion requires approved review and operator"
		}
	}
	entry.UpdatedAt = util.NowUTC()
	reg.Entries[idx] = *entry
	if err := SaveRegistry(repoRoot, reg); err != nil {
		return MemoryReviewResult{}, err
	}

	_, _ = events.Append(repoRoot, events.EventMemoryPromotionReviewed, "operator", map[string]any{
		"capability_id": capabilityID,
		"review_status": review,
		"operator":      strings.TrimSpace(operator),
		"promoted":      res.Promoted,
		"reason":        res.Reason,
	})
	if res.Promoted {
		_, _ = events.Append(repoRoot, events.EventCapabilityPromoted, "operator", map[string]any{
			"capability_id": capabilityID,
			"to_surface":    entry.CurrentMemorySurface,
			"review_status": review,
		})
	}
	return res, nil
}
