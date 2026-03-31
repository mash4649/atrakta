package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const runEventSchemaVersion = 1

// RunEvent is an append-only runtime/session event entry.
type RunEvent struct {
	SchemaVersion int            `json:"schema_version"`
	Seq           int            `json:"seq"`
	Timestamp     string         `json:"timestamp"`
	EventType     string         `json:"event_type"`
	Integrity     string         `json:"integrity_level"`
	Payload       map[string]any `json:"payload"`
	PayloadHash   string         `json:"payload_hash,omitempty"`
	PrevHash      string         `json:"prev_hash,omitempty"`
	Hash          string         `json:"hash,omitempty"`
	EventID       string         `json:"event_id,omitempty"`
	Actor         string         `json:"actor,omitempty"`
	RunID         string         `json:"run_id,omitempty"`
	Interface     string         `json:"interface,omitempty"`
	FeatureID     string         `json:"feature_id,omitempty"`
}

// RunEventOptions carries optional metadata fields for run events.
type RunEventOptions struct {
	EventID   string
	Actor     string
	RunID     string
	Interface string
	FeatureID string
}

// RunEventsPath returns the runtime events stream path under the audit store.
func RunEventsPath(storeDir string) string {
	return filepath.Join(storeDir, "events", "run-events.jsonl")
}

// AppendRunEvent appends one runtime event to storeDir/events/run-events.jsonl.
func AppendRunEvent(storeDir, level, eventType string, payload any, opts RunEventOptions) (RunEvent, error) {
	level = normalizeLevel(level)
	if level == "" {
		return RunEvent{}, fmt.Errorf("unsupported audit level")
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return RunEvent{}, fmt.Errorf("event_type required")
	}

	payloadObj, payloadBytes, err := canonicalPayloadObject(payload)
	if err != nil {
		return RunEvent{}, err
	}

	eventsPath := RunEventsPath(storeDir)
	if err := os.MkdirAll(filepath.Dir(eventsPath), 0o755); err != nil {
		return RunEvent{}, err
	}
	if err := os.MkdirAll(filepath.Join(storeDir, "checkpoints"), 0o755); err != nil {
		return RunEvent{}, err
	}

	existing, err := readRunEvents(eventsPath)
	if err != nil {
		return RunEvent{}, err
	}

	prevHash := ""
	seq := len(existing) + 1
	if len(existing) > 0 {
		prevHash = existing[len(existing)-1].Hash
	}

	ev := RunEvent{
		SchemaVersion: runEventSchemaVersion,
		Seq:           seq,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		EventType:     eventType,
		Integrity:     level,
		Payload:       payloadObj,
		EventID:       strings.TrimSpace(opts.EventID),
		Actor:         strings.TrimSpace(opts.Actor),
		RunID:         strings.TrimSpace(opts.RunID),
		Interface:     strings.TrimSpace(opts.Interface),
		FeatureID:     strings.TrimSpace(opts.FeatureID),
	}
	if levelRank(level) >= levelRank(LevelA1) {
		ev.PayloadHash = hashHex(payloadBytes)
	}
	if levelRank(level) >= levelRank(LevelA2) {
		ev.PrevHash = prevHash
		ev.Hash = computeChainHash(ev.Seq, ev.EventType, ev.PayloadHash, ev.PrevHash)
	}

	line, err := json.Marshal(ev)
	if err != nil {
		return RunEvent{}, err
	}

	f, err := os.OpenFile(eventsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return RunEvent{}, err
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return RunEvent{}, err
	}

	if level == LevelA3 {
		cp := Checkpoint{LastSeq: ev.Seq, LastHash: ev.Hash, Level: level}
		cpBytes, err := json.MarshalIndent(cp, "", "  ")
		if err != nil {
			return RunEvent{}, err
		}
		if err := os.WriteFile(filepath.Join(storeDir, "checkpoints", "run-head.json"), append(cpBytes, '\n'), 0o644); err != nil {
			return RunEvent{}, err
		}
	}

	return ev, nil
}

// AppendRunEventAndVerify appends one runtime event and verifies integrity at the same level.
func AppendRunEventAndVerify(storeDir, level, eventType string, payload any, opts RunEventOptions) (RunEvent, error) {
	ev, err := AppendRunEvent(storeDir, level, eventType, payload, opts)
	if err != nil {
		return RunEvent{}, err
	}
	if err := VerifyRunEventsIntegrity(storeDir, level); err != nil {
		return RunEvent{}, fmt.Errorf("run-events verify after append: %w", err)
	}
	return ev, nil
}

// VerifyRunEventsIntegrity validates append-only integrity guarantees for run-events.
func VerifyRunEventsIntegrity(storeDir, level string) error {
	level = normalizeLevel(level)
	if level == "" {
		return fmt.Errorf("unsupported audit level")
	}
	eventsPath := RunEventsPath(storeDir)
	events, err := readRunEvents(eventsPath)
	if err != nil {
		return err
	}

	prevHash := ""
	for i, ev := range events {
		if ev.SchemaVersion != runEventSchemaVersion {
			return fmt.Errorf("run-events schema_version mismatch at index %d", i)
		}
		if ev.Seq != i+1 {
			return fmt.Errorf("run-events seq mismatch at index %d", i)
		}
		if strings.TrimSpace(ev.EventType) == "" {
			return fmt.Errorf("run-events event_type missing at seq %d", ev.Seq)
		}
		if _, err := time.Parse(time.RFC3339, ev.Timestamp); err != nil {
			return fmt.Errorf("run-events timestamp invalid at seq %d", ev.Seq)
		}
		if ev.Payload == nil {
			return fmt.Errorf("run-events payload missing at seq %d", ev.Seq)
		}
		payloadBytes, err := json.Marshal(ev.Payload)
		if err != nil {
			return fmt.Errorf("run-events payload marshal at seq %d: %w", ev.Seq, err)
		}

		if levelRank(level) >= levelRank(LevelA1) {
			if strings.TrimSpace(ev.PayloadHash) == "" {
				return fmt.Errorf("run-events payload_hash missing at seq %d", ev.Seq)
			}
			if got := hashHex(payloadBytes); got != ev.PayloadHash {
				return fmt.Errorf("run-events payload_hash mismatch at seq %d", ev.Seq)
			}
		}
		if levelRank(level) >= levelRank(LevelA2) {
			if ev.PrevHash != prevHash {
				return fmt.Errorf("run-events prev_hash mismatch at seq %d", ev.Seq)
			}
			expected := computeChainHash(ev.Seq, ev.EventType, ev.PayloadHash, ev.PrevHash)
			if ev.Hash != expected {
				return fmt.Errorf("run-events hash mismatch at seq %d", ev.Seq)
			}
			prevHash = ev.Hash
		}
	}

	if level == LevelA3 {
		cpPath := filepath.Join(storeDir, "checkpoints", "run-head.json")
		b, err := os.ReadFile(cpPath)
		if err != nil {
			return fmt.Errorf("read run-events checkpoint: %w", err)
		}
		var cp Checkpoint
		if err := json.Unmarshal(b, &cp); err != nil {
			return fmt.Errorf("parse run-events checkpoint: %w", err)
		}
		if len(events) == 0 {
			if cp.LastSeq != 0 || cp.LastHash != "" {
				return fmt.Errorf("run-events checkpoint must be empty for empty log")
			}
			return nil
		}
		last := events[len(events)-1]
		if cp.LastSeq != last.Seq || cp.LastHash != last.Hash {
			return fmt.Errorf("run-events checkpoint mismatch with log head")
		}
	}

	return nil
}

func canonicalPayloadObject(payload any) (map[string]any, []byte, error) {
	if payload == nil {
		empty := map[string]any{}
		b, err := json.Marshal(empty)
		return empty, b, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, nil, fmt.Errorf("payload must be an object")
	}
	if obj == nil {
		obj = map[string]any{}
	}

	canonical, err := json.Marshal(obj)
	if err != nil {
		return nil, nil, err
	}
	return obj, canonical, nil
}

func readRunEvents(eventsPath string) ([]RunEvent, error) {
	f, err := os.Open(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunEvent{}, nil
		}
		return nil, err
	}
	defer f.Close()

	out := make([]RunEvent, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev RunEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, fmt.Errorf("parse run event: %w", err)
		}
		out = append(out, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
