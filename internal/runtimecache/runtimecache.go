package runtimecache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"atrakta/internal/util"
)

const version = 2

type Entry struct {
	UpdatedAt  string          `json:"updated_at,omitempty"`
	Stamp      string          `json:"stamp,omitempty"`
	ConfigHash string          `json:"config_hash,omitempty"`
	TTLSeconds int             `json:"ttl_seconds,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

type State struct {
	V         int              `json:"v"`
	UpdatedAt string           `json:"updated_at,omitempty"`
	Entries   map[string]Entry `json:"entries,omitempty"`
}

func Load(repoRoot string) (State, error) {
	p := path(repoRoot)
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{V: version, Entries: map[string]Entry{}}, nil
		}
		return State{}, fmt.Errorf("read runtime cache: %w", err)
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return State{}, fmt.Errorf("parse runtime cache: %w", err)
	}
	if st.V != version {
		return State{V: version, Entries: map[string]Entry{}}, nil
	}
	if st.Entries == nil {
		st.Entries = map[string]Entry{}
	}
	return st, nil
}

func Save(repoRoot string, st State) error {
	p := path(repoRoot)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("mkdir runtime cache: %w", err)
	}
	if st.V == 0 {
		st.V = version
	}
	if st.Entries == nil {
		st.Entries = map[string]Entry{}
	}
	st.UpdatedAt = util.NowUTC()
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime cache: %w", err)
	}
	b = append(b, '\n')
	return util.WithFileLock(lockPath(repoRoot), util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(p, b, 0o644)
	})
}

func Update(repoRoot string, fn func(st *State) error) error {
	return util.WithFileLock(lockPath(repoRoot), util.DefaultFileLockOptions(), func() error {
		st, err := loadUnlocked(path(repoRoot))
		if err != nil {
			return err
		}
		if err := fn(&st); err != nil {
			return err
		}
		st.UpdatedAt = util.NowUTC()
		b, err := json.MarshalIndent(st, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal runtime cache: %w", err)
		}
		b = append(b, '\n')
		return util.AtomicWriteFile(path(repoRoot), b, 0o644)
	})
}

func IsFresh(e Entry, expectedStamp, expectedCfgHash string, now time.Time) bool {
	if expectedStamp != "" && e.Stamp != expectedStamp {
		return false
	}
	if expectedCfgHash != "" && e.ConfigHash != expectedCfgHash {
		return false
	}
	if e.TTLSeconds <= 0 {
		return false
	}
	if e.UpdatedAt == "" {
		return false
	}
	ts, err := time.Parse(time.RFC3339, e.UpdatedAt)
	if err != nil {
		return false
	}
	return now.Sub(ts) < time.Duration(e.TTLSeconds)*time.Second
}

func MarshalPayload(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

func UnmarshalPayload(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty payload")
	}
	return json.Unmarshal(raw, out)
}

func path(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "runtime", "meta.v2.json")
}

func lockPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", ".locks", "runtime.meta.v2.lock")
}

func loadUnlocked(p string) (State, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{V: version, Entries: map[string]Entry{}}, nil
		}
		return State{}, fmt.Errorf("read runtime cache: %w", err)
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return State{}, fmt.Errorf("parse runtime cache: %w", err)
	}
	if st.V != version {
		return State{V: version, Entries: map[string]Entry{}}, nil
	}
	if st.Entries == nil {
		st.Entries = map[string]Entry{}
	}
	return st, nil
}
