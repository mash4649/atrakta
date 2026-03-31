package main

import "github.com/mash4649/atrakta/v0/internal/onboarding"

func writeAcceptanceArtifacts(dir string, bundle onboarding.ProposalBundle) error {
	spec, rubric := onboarding.BuildAcceptanceArtifacts(bundle)
	if err := writeArtifact(dir, "acceptance-spec.generated.json", spec); err != nil {
		return err
	}
	if err := writeArtifact(dir, "acceptance-rubric.generated.json", rubric); err != nil {
		return err
	}
	return nil
}
