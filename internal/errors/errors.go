package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
)

// StructuredError is the machine-readable error envelope used by JSON outputs.
type StructuredError struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	RecoverySteps []string `json:"recovery_steps,omitempty"`
}

func Usage(message string, steps ...string) StructuredError {
	return NewStructured("ERR_USAGE", message, steps...)
}

func ApprovalRequired(message string, steps ...string) StructuredError {
	return NewStructured("ERR_APPROVAL_REQUIRED", message, steps...)
}

func Blocked(message string, steps ...string) StructuredError {
	return NewStructured("ERR_BLOCKED", message, steps...)
}

func NotFound(message string, steps ...string) StructuredError {
	return NewStructured("ERR_NOT_FOUND", message, steps...)
}

func Runtime(message string, steps ...string) StructuredError {
	return NewStructured("ERR_RUNTIME", message, steps...)
}

// ExitError marks a command failure with an exit code and whether the command
// already emitted a structured response before returning the error.
type ExitError struct {
	Code           int
	Structured     StructuredError
	AlreadyPrinted bool
}

func (e *ExitError) Error() string {
	if strings.TrimSpace(e.Structured.Message) != "" {
		return e.Structured.Message
	}
	return "command failed"
}

func NewStructured(code, message string, steps ...string) StructuredError {
	out := StructuredError{
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
	}
	for _, step := range steps {
		step = strings.TrimSpace(step)
		if step != "" {
			out.RecoverySteps = append(out.RecoverySteps, step)
		}
	}
	return out
}

func NewExitError(code int, structured StructuredError, alreadyPrinted bool) error {
	return &ExitError{
		Code:           code,
		Structured:     structured,
		AlreadyPrinted: alreadyPrinted,
	}
}

func AsExitError(err error) (*ExitError, bool) {
	var exitErr *ExitError
	if stderrors.As(err, &exitErr) {
		return exitErr, true
	}
	return nil, false
}

// Classify returns a structured error for plain errors when no richer context exists.
func Classify(command string, err error) StructuredError {
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "requires existing atrakta state"):
		return Blocked(
			msg,
			"Run `atrakta start` to create the session state, then retry `atrakta resume`.",
		)
	case strings.Contains(lower, "blocked by handoff"):
		return Blocked(
			msg,
			"Inspect the saved handoff, resolve the deny condition, then retry the command.",
		)
	case strings.Contains(lower, "requires approval"):
		return ApprovalRequired(
			msg,
			fmt.Sprintf("Re-run `%s --approve` or use an interactive terminal.", command),
			"Review the proposed changes before approving writes.",
		)
	case strings.Contains(lower, "requires") || strings.Contains(lower, "unsupported") || strings.Contains(lower, "does not accept"):
		return Usage(
			msg,
			fmt.Sprintf("Run `atrakta %s --help` to inspect the accepted flags and subcommands.", command),
		)
	case strings.Contains(lower, "not found"):
		return NotFound(
			msg,
			"Create or restore the missing file, then retry the command.",
		)
	default:
		return Runtime(
			msg,
			"Inspect the command output and logs, then retry after fixing the reported issue.",
		)
	}
}
