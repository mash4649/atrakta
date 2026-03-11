package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"atrakta/internal/util"
)

type ManagedRecord struct {
	Interface      string `json:"interface"`
	Kind           string `json:"kind"`
	Fingerprint    string `json:"fingerprint"`
	CreatedAt      string `json:"created_at"`
	TemplateID     string `json:"template_id"`
	Target         string `json:"target,omitempty"`
	LastVerifiedAt string `json:"last_verified_at,omitempty"`
}

type State struct {
	V            int                      `json:"v"`
	ContractHash string                   `json:"contract_hash"`
	ManagedPaths map[string]ManagedRecord `json:"managed_paths"`
	Autonomy     map[string]any           `json:"autonomy,omitempty"`
	Projection   *ProjectionState         `json:"projection,omitempty"`
	Integration  *IntegrationState        `json:"integration,omitempty"`
}

type ProjectionState struct {
	LastRenderedAt string `json:"last_rendered_at,omitempty"`
	SourceHash     string `json:"source_hash,omitempty"`
	RenderHash     string `json:"render_hash,omitempty"`
	Status         string `json:"status,omitempty"`
}

type IntegrationState struct {
	LastCheckedAt   string   `json:"last_checked_at,omitempty"`
	LastResult      string   `json:"last_result,omitempty"`
	BlockingReasons []string `json:"blocking_reasons,omitempty"`
}

type ApplyOpResult struct {
	TaskID      string
	Path        string
	Op          string
	Status      string
	Error       string
	Interface   string
	TemplateID  string
	Kind        string
	Target      string
	Fingerprint string
}

type ApplyResult struct {
	Ops []ApplyOpResult
}

func Empty(contractHash string) State {
	return State{V: 1, ContractHash: contractHash, ManagedPaths: map[string]ManagedRecord{}}
}

func LoadOrEmpty(repoRoot, contractHash string) (State, bool, error) {
	path := filepath.Join(repoRoot, ".atrakta", "state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Empty(contractHash), false, nil
		}
		return State{}, false, fmt.Errorf("read state: %w", err)
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return Empty(contractHash), true, fmt.Errorf("parse state: %w", err)
	}
	if s.V != 1 {
		return Empty(contractHash), true, fmt.Errorf("state.v must be 1")
	}
	if s.ManagedPaths == nil {
		s.ManagedPaths = map[string]ManagedRecord{}
	}
	if s.ContractHash == "" {
		s.ContractHash = contractHash
	}
	return s, true, nil
}

func Save(repoRoot string, s State) error {
	path := filepath.Join(repoRoot, ".atrakta", "state.json")
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "state.json.lock")
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	b = append(b, '\n')
	return util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		if err := util.AtomicWriteFile(path, b, 0o644); err != nil {
			return fmt.Errorf("write state: %w", err)
		}
		return nil
	})
}

func UpdateFromApply(in State, contractHash string, ap ApplyResult) State {
	out := in
	if out.V == 0 {
		out.V = 1
	}
	if out.ManagedPaths == nil {
		out.ManagedPaths = map[string]ManagedRecord{}
	}
	out.ContractHash = contractHash
	for _, r := range ap.Ops {
		switch r.Op {
		case "adopt", "link", "copy", "write":
			if r.Status == "ok" || r.Status == "skipped" {
				if r.Interface == "" || r.TemplateID == "" || r.Fingerprint == "" {
					continue
				}
				out.ManagedPaths[r.Path] = ManagedRecord{
					Interface:      r.Interface,
					Kind:           pickKind(r),
					Fingerprint:    r.Fingerprint,
					TemplateID:     r.TemplateID,
					Target:         r.Target,
					CreatedAt:      nowOrKeep(out.ManagedPaths[r.Path].CreatedAt),
					LastVerifiedAt: util.NowUTC(),
				}
			}
		case "delete", "unlink":
			if r.Status == "ok" || r.Status == "skipped" {
				delete(out.ManagedPaths, r.Path)
			}
		}
	}
	return out
}

func pickKind(r ApplyOpResult) string {
	if r.Kind != "" {
		return r.Kind
	}
	if r.Op == "link" {
		return "link"
	}
	if r.Op == "copy" || r.Op == "write" {
		return "copy"
	}
	return "copy"
}

func nowOrKeep(v string) string {
	if v != "" {
		return v
	}
	return util.NowUTC()
}
