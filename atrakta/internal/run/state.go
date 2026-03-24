package run

import (
	"os"
	"path/filepath"
)

const (
	StateNone               = "none"
	StateCanonicalPresent   = "canonical_present"
	StateOnboardingComplete = "onboarding_complete"
	StatePartialState       = "partial_state"
	StateCorruptState       = "corrupt_state"
)

// DetectCanonicalState classifies canonical store readiness for atrakta run routing.
func DetectCanonicalState(projectRoot string) (string, error) {
	policyIndex := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry", "index.json")
	onboardingState := filepath.Join(projectRoot, ".atrakta", "state", "onboarding-state.json")
	canonicalDir := filepath.Join(projectRoot, ".atrakta", "canonical")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")

	hasPolicy, err := fileExists(policyIndex)
	if err != nil {
		return "", err
	}
	hasOnboardingState, err := fileExists(onboardingState)
	if err != nil {
		return "", err
	}
	hasCanonicalDir, err := dirExists(canonicalDir)
	if err != nil {
		return "", err
	}
	hasStateDir, err := dirExists(stateDir)
	if err != nil {
		return "", err
	}

	switch {
	case hasPolicy && hasOnboardingState:
		return StateOnboardingComplete, nil
	case hasPolicy:
		return StateCanonicalPresent, nil
	case hasOnboardingState:
		return StatePartialState, nil
	case hasCanonicalDir || hasStateDir:
		return StateCorruptState, nil
	default:
		return StateNone, nil
	}
}

func fileExists(path string) (bool, error) {
	st, err := os.Stat(path)
	if err == nil {
		return !st.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func dirExists(path string) (bool, error) {
	st, err := os.Stat(path)
	if err == nil {
		return st.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
