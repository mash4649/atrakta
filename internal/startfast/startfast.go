package startfast

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
)

const schemaVersionV1 = "start-fast.v1"

type Key struct {
	Key                 string `json:"key"`
	ContractHash        string `json:"contract_hash"`
	CanonicalPolicyHash string `json:"canonical_policy_hash"`
	WorkspaceStamp      string `json:"workspace_stamp"`
	InterfaceID         string `json:"interface_id"`
	ApplyRequested      bool   `json:"apply_requested"`
}

type Snapshot struct {
	SchemaVersion       string `json:"schema_version"`
	Key                 string `json:"key"`
	ContractHash        string `json:"contract_hash"`
	CanonicalPolicyHash string `json:"canonical_policy_hash"`
	WorkspaceStamp      string `json:"workspace_stamp"`
	InterfaceID         string `json:"interface_id"`
	ApplyRequested      bool   `json:"apply_requested"`
}

func ComputeKey(projectRoot, interfaceID string, applyRequested bool) (Key, error) {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return Key{}, err
	}
	contractHash, err := hashFile(runpkg.ContractPath(root))
	if err != nil {
		return Key{}, err
	}
	canonicalIndexPath := filepath.Join(root, ".atrakta", "canonical", "policies", "registry", "index.json")
	policyHash, err := hashFile(canonicalIndexPath)
	if err != nil {
		return Key{}, err
	}
	assets, err := onboarding.DetectAssets(root)
	if err != nil {
		return Key{}, err
	}
	workspaceStamp := hashString(strings.Join(assets, "\n"))
	payload := keyPayload(contractHash, interfaceID, policyHash, workspaceStamp, applyRequested)
	return Key{
		Key:                 hashString(payload),
		ContractHash:        contractHash,
		CanonicalPolicyHash: policyHash,
		WorkspaceStamp:      workspaceStamp,
		InterfaceID:         strings.TrimSpace(interfaceID),
		ApplyRequested:      applyRequested,
	}, nil
}

func legacyKey(contractHash, interfaceID, policyHash, workspaceStamp string) string {
	return hashString(strings.Join([]string{
		schemaVersionV1,
		contractHash,
		strings.TrimSpace(interfaceID),
		policyHash,
		workspaceStamp,
	}, "|"))
}

func keyPayload(contractHash, interfaceID, policyHash, workspaceStamp string, applyRequested bool) string {
	return strings.Join([]string{
		schemaVersionV1,
		contractHash,
		strings.TrimSpace(interfaceID),
		policyHash,
		workspaceStamp,
		fmt.Sprintf("%t", applyRequested),
	}, "|")
}

func SnapshotPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atrakta", "runtime", "start-fast.v1.json")
}

func LoadSnapshot(projectRoot string) (Snapshot, error) {
	path := SnapshotPath(projectRoot)
	raw, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return Snapshot{}, fmt.Errorf("decode start-fast snapshot: %w", err)
	}
	return snap, nil
}

func SaveSnapshot(projectRoot string, snap Snapshot) error {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(snap.SchemaVersion) == "" {
		snap.SchemaVersion = schemaVersionV1
	}
	path := SnapshotPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func IsMatch(snap Snapshot, key Key) bool {
	if strings.TrimSpace(snap.Key) == "" {
		return false
	}
	if strings.TrimSpace(snap.Key) == strings.TrimSpace(key.Key) {
		return true
	}
	if !snap.ApplyRequested {
		return strings.TrimSpace(snap.Key) == legacyKey(key.ContractHash, key.InterfaceID, key.CanonicalPolicyHash, key.WorkspaceStamp)
	}
	return false
}

func hashFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashString(string(raw)), nil
}

func hashString(in string) string {
	sum := sha256.Sum256([]byte(in))
	return hex.EncodeToString(sum[:])
}
