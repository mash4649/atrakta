package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	LevelA0 = "A0"
	LevelA1 = "A1"
	LevelA2 = "A2"
	LevelA3 = "A3"
)

// Event is an append-only audit event entry.
type Event struct {
	Seq         int             `json:"seq"`
	Action      string          `json:"action"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	PayloadHash string          `json:"payload_hash,omitempty"`
	PrevHash    string          `json:"prev_hash,omitempty"`
	Hash        string          `json:"hash,omitempty"`
	Level       string          `json:"level"`
}

// Checkpoint stores an integrity checkpoint for A3 validation.
type Checkpoint struct {
	LastSeq  int    `json:"last_seq"`
	LastHash string `json:"last_hash"`
	Level    string `json:"level"`
}

// AppendEvent appends one audit event to storeDir/events/install-events.jsonl.
func AppendEvent(storeDir, level, action string, payload any) (Event, error) {
	level = normalizeLevel(level)
	if level == "" {
		return Event{}, fmt.Errorf("unsupported audit level")
	}
	if strings.TrimSpace(action) == "" {
		return Event{}, fmt.Errorf("action required")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}

	eventsPath := filepath.Join(storeDir, "events", "install-events.jsonl")
	if err := os.MkdirAll(filepath.Dir(eventsPath), 0o755); err != nil {
		return Event{}, err
	}
	if err := os.MkdirAll(filepath.Join(storeDir, "checkpoints"), 0o755); err != nil {
		return Event{}, err
	}

	existing, err := readEvents(eventsPath)
	if err != nil {
		return Event{}, err
	}

	prevHash := ""
	seq := len(existing) + 1
	if len(existing) > 0 {
		prevHash = existing[len(existing)-1].Hash
	}

	ev := Event{
		Seq:     seq,
		Action:  action,
		Payload: payloadBytes,
		Level:   level,
	}
	if levelRank(level) >= levelRank(LevelA1) {
		ev.PayloadHash = hashHex(payloadBytes)
	}
	if levelRank(level) >= levelRank(LevelA2) {
		ev.PrevHash = prevHash
		ev.Hash = computeChainHash(ev.Seq, ev.Action, ev.PayloadHash, ev.PrevHash)
	}

	line, err := json.Marshal(ev)
	if err != nil {
		return Event{}, err
	}

	f, err := os.OpenFile(eventsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return Event{}, err
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return Event{}, err
	}

	if level == LevelA3 {
		cp := Checkpoint{LastSeq: ev.Seq, LastHash: ev.Hash, Level: level}
		cpBytes, err := json.MarshalIndent(cp, "", "  ")
		if err != nil {
			return Event{}, err
		}
		if err := os.WriteFile(filepath.Join(storeDir, "checkpoints", "head.json"), append(cpBytes, '\n'), 0o644); err != nil {
			return Event{}, err
		}
	}

	return ev, nil
}

// VerifyIntegrity validates append-only integrity guarantees for the requested level.
func VerifyIntegrity(storeDir, level string) error {
	level = normalizeLevel(level)
	if level == "" {
		return fmt.Errorf("unsupported audit level")
	}
	eventsPath := filepath.Join(storeDir, "events", "install-events.jsonl")
	events, err := readEvents(eventsPath)
	if err != nil {
		return err
	}

	prevHash := ""
	for i, ev := range events {
		if ev.Seq != i+1 {
			return fmt.Errorf("audit seq mismatch at index %d", i)
		}
		if strings.TrimSpace(ev.Action) == "" {
			return fmt.Errorf("audit action missing at seq %d", ev.Seq)
		}

		if levelRank(level) >= levelRank(LevelA1) {
			if strings.TrimSpace(ev.PayloadHash) == "" {
				return fmt.Errorf("payload_hash missing at seq %d", ev.Seq)
			}
			if got := hashHex(ev.Payload); got != ev.PayloadHash {
				return fmt.Errorf("payload_hash mismatch at seq %d", ev.Seq)
			}
		}
		if levelRank(level) >= levelRank(LevelA2) {
			if ev.PrevHash != prevHash {
				return fmt.Errorf("prev_hash mismatch at seq %d", ev.Seq)
			}
			expected := computeChainHash(ev.Seq, ev.Action, ev.PayloadHash, ev.PrevHash)
			if ev.Hash != expected {
				return fmt.Errorf("hash mismatch at seq %d", ev.Seq)
			}
			prevHash = ev.Hash
		}
	}

	if level == LevelA3 {
		cpPath := filepath.Join(storeDir, "checkpoints", "head.json")
		b, err := os.ReadFile(cpPath)
		if err != nil {
			return fmt.Errorf("read checkpoint: %w", err)
		}
		var cp Checkpoint
		if err := json.Unmarshal(b, &cp); err != nil {
			return fmt.Errorf("parse checkpoint: %w", err)
		}
		if len(events) == 0 {
			if cp.LastSeq != 0 || cp.LastHash != "" {
				return fmt.Errorf("checkpoint must be empty for empty log")
			}
			return nil
		}
		last := events[len(events)-1]
		if cp.LastSeq != last.Seq || cp.LastHash != last.Hash {
			return fmt.Errorf("checkpoint mismatch with log head")
		}
	}

	return nil
}

func readEvents(eventsPath string) ([]Event, error) {
	f, err := os.Open(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Event{}, nil
		}
		return nil, err
	}
	defer f.Close()

	out := make([]Event, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, fmt.Errorf("parse audit event: %w", err)
		}
		out = append(out, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func computeChainHash(seq int, action, payloadHash, prevHash string) string {
	raw := fmt.Sprintf("%d|%s|%s|%s", seq, action, payloadHash, prevHash)
	return hashHex([]byte(raw))
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func normalizeLevel(level string) string {
	level = strings.ToUpper(strings.TrimSpace(level))
	switch level {
	case LevelA0, LevelA1, LevelA2, LevelA3:
		return level
	default:
		return ""
	}
}

func levelRank(level string) int {
	switch level {
	case LevelA0:
		return 0
	case LevelA1:
		return 1
	case LevelA2:
		return 2
	case LevelA3:
		return 3
	default:
		return -1
	}
}
