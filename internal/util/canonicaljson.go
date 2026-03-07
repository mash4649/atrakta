package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// MarshalCanonical marshals JSON with deterministic object key ordering.
func MarshalCanonical(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := writeCanonical(buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeCanonical(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			if err := writeCanonical(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	case []any:
		buf.WriteByte('[')
		for i, it := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, it); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	default:
		jb, err := json.Marshal(x)
		if err != nil {
			return fmt.Errorf("marshal canonical scalar: %w", err)
		}
		buf.Write(jb)
		return nil
	}
}
