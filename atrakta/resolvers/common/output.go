package common

// ResolverOutput is the standard inspect/preview/simulate response envelope.
type ResolverOutput struct {
	Input             any      `json:"input"`
	Decision          any      `json:"decision"`
	Reason            string   `json:"reason"`
	Evidence          []string `json:"evidence"`
	NextAllowedAction string   `json:"next_allowed_action"`
}

// NewOutput returns a normalized resolver output.
func NewOutput(input any, decision any, reason string, evidence []string, next string) ResolverOutput {
	if evidence == nil {
		evidence = []string{}
	}
	return ResolverOutput{
		Input:             input,
		Decision:          decision,
		Reason:            reason,
		Evidence:          evidence,
		NextAllowedAction: next,
	}
}
