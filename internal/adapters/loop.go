package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// RunInvoker invokes `atrakta run` and returns exit code and stdout payload.
type RunInvoker func(args []string) (int, []byte, error)

// InputCollector provides missing values required by run responses.
type InputCollector func(required []string) (map[string]string, error)

// ApprovalCollector decides whether the requested approval scope is granted.
type ApprovalCollector func(scope string) (bool, error)

// ExecuteConfig controls adapter loop execution.
type ExecuteConfig struct {
	BaseArgs           []string
	MaxAttempts        int
	InvokeRun          RunInvoker
	CollectInputs      InputCollector
	CollectApproval    ApprovalCollector
	NonInteractiveMode bool
}

// AttemptResult stores one run invocation outcome.
type AttemptResult struct {
	ExitCode int                 `json:"exit_code"`
	Payload  map[string]any      `json:"payload,omitempty"`
	Args     []string            `json:"args"`
	Error    string              `json:"error,omitempty"`
	Added    map[string][]string `json:"added,omitempty"`
}

// ExecuteResult stores the full adapter loop trace.
type ExecuteResult struct {
	Attempts      []AttemptResult `json:"attempts"`
	FinalExitCode int             `json:"final_exit_code"`
}

// ExecuteRunLoop executes atrakta run and handles NEEDS_INPUT/NEEDS_APPROVAL retries.
func ExecuteRunLoop(cfg ExecuteConfig) (ExecuteResult, error) {
	if cfg.InvokeRun == nil {
		return ExecuteResult{}, errors.New("invoke run is required")
	}
	max := cfg.MaxAttempts
	if max <= 0 {
		max = 3
	}

	currentArgs := append([]string{}, cfg.BaseArgs...)
	out := ExecuteResult{Attempts: make([]AttemptResult, 0, max)}

	for attempt := 0; attempt < max; attempt++ {
		exitCode, raw, err := cfg.InvokeRun(currentArgs)
		result := AttemptResult{
			ExitCode: exitCode,
			Args:     append([]string{}, currentArgs...),
		}
		if err != nil {
			result.Error = err.Error()
			out.Attempts = append(out.Attempts, result)
			out.FinalExitCode = exitCode
			return out, err
		}

		payload, parseErr := decodePayload(raw)
		if parseErr == nil {
			result.Payload = payload
		} else if strings.TrimSpace(string(raw)) != "" {
			result.Error = parseErr.Error()
		}
		out.Attempts = append(out.Attempts, result)

		switch exitCode {
		case 0:
			out.FinalExitCode = 0
			return out, nil
		case 2:
			nextArgs, added, retryErr := resolveNeedsInput(currentArgs, payload, cfg.CollectInputs)
			if retryErr != nil {
				out.FinalExitCode = exitCode
				return out, retryErr
			}
			out.Attempts[len(out.Attempts)-1].Added = added
			currentArgs = nextArgs
		case 3:
			nextArgs, added, retryErr := resolveNeedsApproval(currentArgs, payload, cfg.CollectApproval, cfg.NonInteractiveMode)
			if retryErr != nil {
				out.FinalExitCode = exitCode
				return out, retryErr
			}
			out.Attempts[len(out.Attempts)-1].Added = added
			currentArgs = nextArgs
		default:
			out.FinalExitCode = exitCode
			return out, fmt.Errorf("run failed with exit code %d", exitCode)
		}
	}

	out.FinalExitCode = 1
	return out, fmt.Errorf("adapter retry limit reached (%d)", max)
}

func decodePayload(raw []byte) (map[string]any, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func resolveNeedsInput(currentArgs []string, payload map[string]any, collect InputCollector) ([]string, map[string][]string, error) {
	if collect == nil {
		return nil, nil, errors.New("input collector required for NEEDS_INPUT")
	}
	required := getRequiredInputs(payload)
	collected, err := collect(required)
	if err != nil {
		return nil, nil, err
	}

	nextArgs := append([]string{}, currentArgs...)
	added := map[string][]string{}
	if iface, ok := collected["interface"]; ok && strings.TrimSpace(iface) != "" && !hasFlag(nextArgs, "--interface") {
		nextArgs = append(nextArgs, "--interface", iface)
		added["flags"] = append(added["flags"], "--interface")
	}
	if len(nextArgs) == len(currentArgs) {
		return nil, nil, errors.New("input collected but no applicable retry argument was produced")
	}
	return nextArgs, added, nil
}

func resolveNeedsApproval(currentArgs []string, payload map[string]any, collect ApprovalCollector, nonInteractive bool) ([]string, map[string][]string, error) {
	if hasFlag(currentArgs, "--approve") {
		return nil, nil, errors.New("approval required but --approve already present")
	}
	if nonInteractive {
		return nil, nil, errors.New("approval required in non-interactive mode")
	}
	if collect == nil {
		return nil, nil, errors.New("approval collector required for NEEDS_APPROVAL")
	}

	scope := ""
	if v, ok := payload["approval_scope"].(string); ok {
		scope = v
	}
	approved, err := collect(scope)
	if err != nil {
		return nil, nil, err
	}
	if !approved {
		return nil, nil, errors.New("approval denied by adapter")
	}

	nextArgs := append([]string{}, currentArgs...)
	nextArgs = append(nextArgs, "--approve")
	return nextArgs, map[string][]string{"flags": []string{"--approve"}}, nil
}

func getRequiredInputs(payload map[string]any) []string {
	items, ok := payload["required_inputs"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func hasFlag(args []string, name string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == name {
			return true
		}
	}
	return false
}
