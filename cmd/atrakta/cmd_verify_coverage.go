package main

import (
	"flag"
	"io"

	"github.com/mash4649/atrakta/v0/internal/fixtures"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func runVerifyCoverage(args []string) error {
	fs := flag.NewFlagSet("verify-coverage", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validation.VerifyOperationsSchemaCoverage(""); err != nil {
		return err
	}
	if err := fixtures.VerifyResolverFixtureCoverage(""); err != nil {
		return err
	}
	return nil
}

