package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mash4649/atrakta/v0/internal/validation"
)

func ContractPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atrakta", "contract.json")
}

// LoadMachineContract reads and validates .atrakta/contract.json.
func LoadMachineContract(projectRoot string) (map[string]any, error) {
	path := ContractPath(projectRoot)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("machine contract missing: %s", filepath.ToSlash(path))
		}
		return nil, fmt.Errorf("read machine contract: %w", err)
	}
	if err := validation.ValidateMachineContractRaw(raw); err != nil {
		return nil, fmt.Errorf("invalid machine contract: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode machine contract: %w", err)
	}
	return out, nil
}
