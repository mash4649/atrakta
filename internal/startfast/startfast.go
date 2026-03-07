package startfast

import (
	"time"

	"atrakta/internal/runtimecache"
	"atrakta/internal/util"
)

const (
	cacheKey        = "start_fast_v2"
	cacheTTLSeconds = 24 * 30 * 3600
	strictInterval  = 10 * time.Minute
)

type Snapshot struct {
	ContractHash   string `json:"contract_hash,omitempty"`
	WorkspaceStamp string `json:"workspace_stamp,omitempty"`
	Interfaces     string `json:"interfaces,omitempty"`
	FeatureID      string `json:"feature_id,omitempty"`
	ConfigKey      string `json:"config_key,omitempty"`
	Outcome        string `json:"outcome,omitempty"`
	DetectReason   string `json:"detect_reason,omitempty"`
	LastStrictAt   string `json:"last_strict_at,omitempty"`
}

type Input struct {
	ContractHash   string
	WorkspaceStamp string
	Interfaces     string
	FeatureID      string
	ConfigKey      string
}

type Decision struct {
	Hit    bool
	Reason string
}

func Check(repoRoot string, in Input, now time.Time) (Decision, error) {
	if in.Interfaces == "" || in.WorkspaceStamp == "" || in.ContractHash == "" {
		return Decision{Hit: false, Reason: "missing_input"}, nil
	}
	st, err := runtimecache.Load(repoRoot)
	if err != nil {
		return Decision{}, err
	}
	e, ok := st.Entries[cacheKey]
	if !ok {
		return Decision{Hit: false, Reason: "cache_miss"}, nil
	}
	var snap Snapshot
	if err := runtimecache.UnmarshalPayload(e.Payload, &snap); err != nil {
		return Decision{Hit: false, Reason: "cache_unmarshal_failed"}, nil
	}
	if snap.Outcome != "done" {
		return Decision{Hit: false, Reason: "outcome_not_done"}, nil
	}
	if snap.ContractHash != in.ContractHash {
		return Decision{Hit: false, Reason: "contract_changed"}, nil
	}
	if snap.WorkspaceStamp != in.WorkspaceStamp {
		return Decision{Hit: false, Reason: "workspace_changed"}, nil
	}
	if snap.Interfaces != in.Interfaces {
		return Decision{Hit: false, Reason: "interfaces_changed"}, nil
	}
	if snap.FeatureID != in.FeatureID {
		return Decision{Hit: false, Reason: "feature_changed"}, nil
	}
	if snap.ConfigKey != in.ConfigKey {
		return Decision{Hit: false, Reason: "config_changed"}, nil
	}
	if snap.LastStrictAt == "" {
		return Decision{Hit: false, Reason: "strict_anchor_missing"}, nil
	}
	lastStrictAt, err := time.Parse(time.RFC3339, snap.LastStrictAt)
	if err != nil {
		return Decision{Hit: false, Reason: "strict_anchor_invalid"}, nil
	}
	if now.UTC().Sub(lastStrictAt.UTC()) >= strictInterval {
		return Decision{Hit: false, Reason: "strict_interval_elapsed"}, nil
	}
	return Decision{Hit: true, Reason: "snapshot_match"}, nil
}

func SaveSuccess(repoRoot string, in Input, detectReason string, now time.Time) error {
	snap := Snapshot{
		ContractHash:   in.ContractHash,
		WorkspaceStamp: in.WorkspaceStamp,
		Interfaces:     in.Interfaces,
		FeatureID:      in.FeatureID,
		ConfigKey:      in.ConfigKey,
		Outcome:        "done",
		DetectReason:   detectReason,
		LastStrictAt:   now.UTC().Format(time.RFC3339),
	}
	return runtimecache.Update(repoRoot, func(st *runtimecache.State) error {
		st.Entries[cacheKey] = runtimecache.Entry{
			UpdatedAt:  util.NowUTC(),
			TTLSeconds: cacheTTLSeconds,
			Payload:    runtimecache.MarshalPayload(snap),
		}
		return nil
	})
}
