package routing

import (
	"strings"

	"atrakta/internal/contract"
)

type Decision struct {
	TaskCategory string `json:"task_category"`
	Worker       string `json:"worker"`
	Quality      string `json:"quality"`
}

func Resolve(c contract.Contract, taskCategory string) Decision {
	category := normalizeCategory(taskCategory)
	rule := contract.RoutingRule{Worker: "general", Quality: "quick"}
	if c.Routing != nil {
		rule = c.Routing.Default
		if selected, ok := c.Routing.Categories[category]; ok {
			rule = selected
		}
	}
	worker := strings.TrimSpace(rule.Worker)
	if worker == "" {
		worker = "general"
	}
	return Decision{
		TaskCategory: category,
		Worker:       worker,
		Quality:      normalizeQuality(rule.Quality),
	}
}

func normalizeCategory(category string) string {
	category = strings.TrimSpace(strings.ToLower(category))
	if category == "" {
		return "sync"
	}
	return category
}

func normalizeQuality(quality string) string {
	switch strings.TrimSpace(strings.ToLower(quality)) {
	case "heavy":
		return "heavy"
	default:
		return "quick"
	}
}
