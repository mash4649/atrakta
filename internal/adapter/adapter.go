package adapter

import "atrakta/internal/model"

type ApprovalResponse struct {
	Approved bool
	Note     string
}

type InputResponse struct {
	Value *string
}

type Adapter interface {
	EmitStatus(message string)
	PresentDiff(summary string, details string)
	RequestApproval(context any) ApprovalResponse
	RequestInput(prompt string, schema map[string]any) InputResponse
	PresentNextAction(next model.NextAction)
	NotifyBlocked(reason string)
}
