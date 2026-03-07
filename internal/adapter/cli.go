package adapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"atrakta/internal/model"
)

type CLIAdapter struct {
	In          *os.File
	Out         *os.File
	Err         *os.File
	Interactive bool
}

func NewCLIAdapter() CLIAdapter {
	st, err := os.Stdin.Stat()
	interactive := err == nil && (st.Mode()&os.ModeCharDevice) != 0
	return CLIAdapter{In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Interactive: interactive}
}

func (a CLIAdapter) EmitStatus(message string) {
	fmt.Fprintln(a.Out, message)
}

func (a CLIAdapter) PresentDiff(summary string, details string) {
	if summary != "" {
		fmt.Fprintln(a.Out, summary)
	}
	if details != "" {
		fmt.Fprintln(a.Out, details)
	}
}

func (a CLIAdapter) RequestApproval(context any) ApprovalResponse {
	if !a.Interactive {
		return ApprovalResponse{Approved: false, Note: "non-interactive auto-deny"}
	}
	b, _ := json.MarshalIndent(context, "", "  ")
	fmt.Fprintln(a.Out, "Approval required:")
	fmt.Fprintln(a.Out, string(b))
	fmt.Fprint(a.Out, "Approve? [y/N]: ")
	reader := bufio.NewReader(a.In)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return ApprovalResponse{Approved: line == "y" || line == "yes", Note: "cli"}
}

func (a CLIAdapter) RequestInput(prompt string, schema map[string]any) InputResponse {
	if !a.Interactive {
		return InputResponse{Value: nil}
	}
	fmt.Fprint(a.Out, prompt+": ")
	reader := bufio.NewReader(a.In)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return InputResponse{Value: nil}
	}
	return InputResponse{Value: &line}
}

func (a CLIAdapter) PresentNextAction(next model.NextAction) {
	fmt.Fprintf(a.Out, "next_action: %s - %s\n", next.Kind, next.Hint)
	if next.Command != "" {
		fmt.Fprintf(a.Out, "suggested: %s\n", next.Command)
	}
}

func (a CLIAdapter) NotifyBlocked(reason string) {
	fmt.Fprintf(a.Err, "BLOCKED: %s\n", reason)
}
