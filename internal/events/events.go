package events

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"atrakta/internal/runtimecache"
	"atrakta/internal/util"
)

type Event struct {
	Raw map[string]any
}

const SchemaVersion = 2
const (
	groupCommitWindow  = 40 * time.Millisecond
	runtimeCacheVerify = "events_verify"
)

var (
	commitMu sync.Mutex
	dirtyMap = map[string]bool{}
	lastSync = map[string]time.Time{}
)

func path(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "events.jsonl")
}

func lockPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", ".locks", "events.jsonl.lock")
}

type verifyPayload struct {
	Size      int64  `json:"size"`
	ModUnixNS int64  `json:"mod_unix_ns"`
	LastHash  string `json:"last_hash,omitempty"`
}

func Append(repoRoot, eventType, actor string, payload map[string]any) (string, error) {
	ids, err := AppendBatch(repoRoot, []AppendInput{{
		Type:    eventType,
		Actor:   actor,
		Payload: payload,
	}})
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}

type AppendInput struct {
	Type    string
	Actor   string
	Payload map[string]any
	Urgent  bool
}

func AppendBatch(repoRoot string, inputs []AppendInput) ([]string, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	ep := path(repoRoot)
	if err := os.MkdirAll(filepath.Dir(ep), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir events dir: %w", err)
	}
	lp := lockPath(repoRoot)
	wroteIDs := make([]string, 0, len(inputs))
	lastWrittenHash := ""
	syncNow := shouldSyncNow(repoRoot, inputs)
	err := util.WithFileLock(lp, util.DefaultFileLockOptions(), func() error {
		prevHash, err := lastHashFast(ep)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(ep, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("open events: %w", err)
		}
		defer f.Close()
		bw := bufio.NewWriterSize(f, 64*1024)
		for _, in := range inputs {
			m := make(map[string]any, len(in.Payload)+8)
			m["v"] = 1
			m["schema_version"] = SchemaVersion
			m["id"] = util.NewEventID()
			m["ts"] = util.NowUTC()
			m["type"] = in.Type
			m["actor"] = in.Actor
			m["prev_hash"] = prevHash
			for k, v := range in.Payload {
				m[k] = v
			}
			norm, err := normalizeMap(m)
			if err != nil {
				return err
			}
			h, err := hashWithoutHash(norm)
			if err != nil {
				return err
			}
			norm["hash"] = h
			b, err := util.MarshalCanonical(norm)
			if err != nil {
				return fmt.Errorf("marshal event: %w", err)
			}
			if _, err := bw.Write(append(b, '\n')); err != nil {
				return fmt.Errorf("append event: %w", err)
			}
			if id, _ := norm["id"].(string); id != "" {
				wroteIDs = append(wroteIDs, id)
			}
			prevHash = h
			lastWrittenHash = h
		}
		if err := bw.Flush(); err != nil {
			return fmt.Errorf("flush events: %w", err)
		}
		if syncNow {
			if err := f.Sync(); err != nil {
				return fmt.Errorf("sync events: %w", err)
			}
			markSynced(repoRoot)
		} else {
			markDirty(repoRoot)
		}
		if syncNow && lastWrittenHash != "" {
			if fi, err := f.Stat(); err == nil {
				_ = updateVerifyCache(repoRoot, verifyPayload{
					Size:      fi.Size(),
					ModUnixNS: fi.ModTime().UnixNano(),
					LastHash:  lastWrittenHash,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return wroteIDs, nil
}

func AppendUrgent(repoRoot, eventType, actor string, payload map[string]any) (string, error) {
	ids, err := AppendBatch(repoRoot, []AppendInput{{
		Type:    eventType,
		Actor:   actor,
		Payload: payload,
		Urgent:  true,
	}})
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}

func Flush(repoRoot string) error {
	if !isDirty(repoRoot) {
		return nil
	}
	ep := path(repoRoot)
	f, err := os.OpenFile(ep, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			clearDirty(repoRoot)
			return nil
		}
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return err
	}
	markSynced(repoRoot)
	last, _ := lastHashFast(ep)
	lastHash, _ := last.(string)
	if fi, err := f.Stat(); err == nil {
		_ = updateVerifyCache(repoRoot, verifyPayload{
			Size:      fi.Size(),
			ModUnixNS: fi.ModTime().UnixNano(),
			LastHash:  lastHash,
		})
	}
	return nil
}

func LastHash(repoRoot string) (any, error) {
	ev, err := ReadAll(repoRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(ev) == 0 {
		return nil, nil
	}
	return ev[len(ev)-1].Raw["hash"], nil
}

func VerifyChain(repoRoot string) error {
	return verifyChainAll(repoRoot)
}

func VerifyChainCached(repoRoot string) error {
	ep := path(repoRoot)
	fi, err := os.Stat(ep)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_ = runtimecache.Update(repoRoot, func(st *runtimecache.State) error {
				delete(st.Entries, runtimeCacheVerify)
				return nil
			})
			return nil
		}
		return err
	}
	if vp, ok := getVerifyCache(repoRoot); ok && vp.Size == fi.Size() && vp.ModUnixNS == fi.ModTime().UnixNano() {
		last, lerr := lastHashFast(ep)
		if lerr == nil {
			if s, _ := last.(string); s == vp.LastHash {
				return nil
			}
		}
	}
	if err := verifyChainAll(repoRoot); err != nil {
		return err
	}
	last, _ := lastHashFast(ep)
	lastHash, _ := last.(string)
	_ = updateVerifyCache(repoRoot, verifyPayload{
		Size:      fi.Size(),
		ModUnixNS: fi.ModTime().UnixNano(),
		LastHash:  lastHash,
	})
	return nil
}

func verifyChainAll(repoRoot string) error {
	ep := path(repoRoot)
	f, err := os.Open(ep)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var prev any = nil
	i := 0
	for s.Scan() {
		line := bytes.TrimSpace(s.Bytes())
		if len(line) == 0 {
			continue
		}
		var r map[string]any
		if err := json.Unmarshal(line, &r); err != nil {
			return fmt.Errorf("parse event line: %w", err)
		}
		got, err := verifyEvent(i, r, prev)
		if err != nil {
			return err
		}
		prev = got
		i++
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("scan events: %w", err)
	}
	return nil
}

func verifyEvent(i int, r map[string]any, prev any) (string, error) {
	if _, ok := r["v"]; !ok {
		return "", fmt.Errorf("event[%d] missing v", i)
	}
	if _, ok := r["id"]; !ok {
		return "", fmt.Errorf("event[%d] missing id", i)
	}
	if _, ok := r["type"]; !ok {
		return "", fmt.Errorf("event[%d] missing type", i)
	}
	if _, ok := r["actor"]; !ok {
		return "", fmt.Errorf("event[%d] missing actor", i)
	}
	if _, ok := r["ts"]; !ok {
		return "", fmt.Errorf("event[%d] missing ts", i)
	}
	sv, ok := r["schema_version"]
	if !ok {
		return "", fmt.Errorf("event[%d] missing schema_version", i)
	}
	switch n := sv.(type) {
	case float64:
		if int(n) != SchemaVersion {
			return "", fmt.Errorf("event[%d] schema_version unsupported", i)
		}
	default:
		return "", fmt.Errorf("event[%d] schema_version invalid", i)
	}
	if r["prev_hash"] != prev {
		return "", fmt.Errorf("event[%d] prev_hash mismatch", i)
	}
	expected, err := hashWithoutHash(r)
	if err != nil {
		return "", err
	}
	got, _ := r["hash"].(string)
	if got != expected {
		return "", fmt.Errorf("event[%d] hash mismatch", i)
	}
	return got, nil
}

func ReadAll(repoRoot string) ([]Event, error) {
	ep := path(repoRoot)
	f, err := os.Open(ep)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := []Event{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, fmt.Errorf("parse event line: %w", err)
		}
		out = append(out, Event{Raw: m})
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("scan events: %w", err)
	}
	return out, nil
}

func hashWithoutHash(m map[string]any) (string, error) {
	cp := make(map[string]any, len(m))
	for k, v := range m {
		if k == "hash" {
			continue
		}
		cp[k] = v
	}
	norm, err := normalizeMap(cp)
	if err != nil {
		return "", err
	}
	b, err := util.MarshalCanonical(norm)
	if err != nil {
		return "", err
	}
	return util.SHA256Tagged(b), nil
}

func normalizeMap(m map[string]any) (map[string]any, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("normalize map marshal: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("normalize map unmarshal: %w", err)
	}
	return out, nil
}

func lastHashFast(eventsPath string) (any, error) {
	lastLine, err := readLastNonEmptyLine(eventsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(lastLine) == 0 {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(lastLine, &m); err != nil {
		return nil, fmt.Errorf("parse last event line: %w", err)
	}
	return m["hash"], nil
}

func readLastNonEmptyLine(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size <= 0 {
		return nil, nil
	}
	const chunk = int64(4096)
	pos := size
	buf := make([]byte, 0, minInt64(size, chunk))
	for pos > 0 {
		readLen := chunk
		if pos < readLen {
			readLen = pos
		}
		start := pos - readLen
		part := make([]byte, readLen)
		if _, err := f.ReadAt(part, start); err != nil {
			return nil, err
		}
		buf = append(part, buf...)
		if line := lastNonEmptyFromBuffer(buf); len(line) > 0 {
			copied := make([]byte, len(line))
			copy(copied, line)
			return copied, nil
		}
		pos = start
	}
	line := bytes.TrimSpace(buf)
	if len(line) == 0 {
		return nil, nil
	}
	copied := make([]byte, len(line))
	copy(copied, line)
	return copied, nil
}

func lastNonEmptyFromBuffer(buf []byte) []byte {
	trimmed := bytes.TrimRight(buf, "\r\n\t ")
	if len(trimmed) == 0 {
		return nil
	}
	lines := bytes.Split(trimmed, []byte{'\n'})
	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) > 0 {
			return line
		}
	}
	return nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func shouldSyncNow(repoRoot string, inputs []AppendInput) bool {
	for _, in := range inputs {
		if in.Urgent || isUrgentByPayload(in) {
			return true
		}
	}
	if len(inputs) <= 1 {
		return true
	}
	commitMu.Lock()
	last := lastSync[repoRoot]
	commitMu.Unlock()
	if last.IsZero() {
		return true
	}
	return time.Since(last) >= groupCommitWindow
}

func isUrgentByPayload(in AppendInput) bool {
	if in.Type == "step" {
		if out, _ := in.Payload["outcome"].(string); strings.EqualFold(strings.TrimSpace(out), "BLOCKED") || strings.EqualFold(strings.TrimSpace(out), "FAIL") {
			return true
		}
	}
	if in.Type == "apply" {
		if result, _ := in.Payload["result"].(string); strings.EqualFold(strings.TrimSpace(result), "fail") {
			return true
		}
	}
	return false
}

func markDirty(repoRoot string) {
	commitMu.Lock()
	dirtyMap[repoRoot] = true
	commitMu.Unlock()
}

func clearDirty(repoRoot string) {
	commitMu.Lock()
	delete(dirtyMap, repoRoot)
	commitMu.Unlock()
}

func markSynced(repoRoot string) {
	commitMu.Lock()
	delete(dirtyMap, repoRoot)
	lastSync[repoRoot] = time.Now().UTC()
	commitMu.Unlock()
}

func isDirty(repoRoot string) bool {
	commitMu.Lock()
	defer commitMu.Unlock()
	return dirtyMap[repoRoot]
}

func updateVerifyCache(repoRoot string, p verifyPayload) error {
	return runtimecache.Update(repoRoot, func(st *runtimecache.State) error {
		st.Entries[runtimeCacheVerify] = runtimecache.Entry{
			UpdatedAt: util.NowUTC(),
			Stamp:     fmt.Sprintf("%d:%d", p.Size, p.ModUnixNS),
			Payload:   runtimecache.MarshalPayload(p),
		}
		return nil
	})
}

func getVerifyCache(repoRoot string) (verifyPayload, bool) {
	st, err := runtimecache.Load(repoRoot)
	if err != nil {
		return verifyPayload{}, false
	}
	e, ok := st.Entries[runtimeCacheVerify]
	if !ok {
		return verifyPayload{}, false
	}
	var p verifyPayload
	if err := runtimecache.UnmarshalPayload(e.Payload, &p); err != nil {
		return verifyPayload{}, false
	}
	return p, true
}
