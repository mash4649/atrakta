package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	atraktaerrors "github.com/mash4649/atrakta/v0/internal/errors"
)

func runWithErrorHandling(command string, args []string, fn func([]string) (int, error)) {
	code, err := fn(args)
	if err != nil {
		code = reportCommandError(command, args, code, err)
		os.Exit(code)
	}
	if code != exitOK {
		os.Exit(code)
	}
}

func reportCommandError(command string, args []string, code int, err error) int {
	if exitErr, ok := atraktaerrors.AsExitError(err); ok {
		if !exitErr.AlreadyPrinted {
			emitCommandError(command, args, exitErr.Structured)
		}
		if exitErr.Code != 0 {
			return exitErr.Code
		}
	}
	if code == exitOK {
		code = exitRuntimeError
	}
	emitCommandError(command, args, atraktaerrors.Classify(command, err))
	return code
}

func emitCommandError(command string, args []string, structured atraktaerrors.StructuredError) {
	if commandRequestsJSON(args) {
		payload := map[string]any{
			"status": "error",
			"error":  structured,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
		return
	}
	fmt.Fprintf(os.Stderr, "error_code: %s\n", structured.Code)
	fmt.Fprintf(os.Stderr, "error_message: %s\n", structured.Message)
	if len(structured.RecoverySteps) > 0 {
		fmt.Fprintf(os.Stderr, "recovery_steps: %s\n", strings.Join(structured.RecoverySteps, " | "))
	}
}

func commandRequestsJSON(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}
