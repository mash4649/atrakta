package validation

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

type compiledSchema struct {
	Types                []string
	Enum                 []any
	Required             map[string]struct{}
	Properties           map[string]*compiledSchema
	Items                *compiledSchema
	AdditionalProperties *bool
	MinLength            *int
	MinItems             *int
	Minimum              *float64
}

type rawSchema struct {
	Type                 any                  `json:"type"`
	Enum                 []any                `json:"enum"`
	Required             []string             `json:"required"`
	Properties           map[string]rawSchema `json:"properties"`
	Items                *rawSchema           `json:"items"`
	AdditionalProperties any                  `json:"additionalProperties"`
	MinLength            *int                 `json:"minLength"`
	MinItems             *int                 `json:"minItems"`
	Minimum              *float64             `json:"minimum"`
}

var (
	schemaMu    sync.Mutex
	schemaCache = map[string]*compiledSchema{}
)

func validateJSONAgainstSchema(raw []byte, schemaPath string) error {
	s, err := loadCompiledSchema(schemaPath)
	if err != nil {
		return err
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("invalid json payload: %w", err)
	}

	errs := make([]string, 0)
	validateValue(v, s, "$", &errs)
	if len(errs) > 0 {
		return fmt.Errorf("schema validation failed (%s): %s", schemaPath, strings.Join(errs, "; "))
	}
	return nil
}

func loadCompiledSchema(schemaPath string) (*compiledSchema, error) {
	schemaMu.Lock()
	if s, ok := schemaCache[schemaPath]; ok {
		schemaMu.Unlock()
		return s, nil
	}
	schemaMu.Unlock()

	root, err := resolveProjectRoot()
	if err != nil {
		return nil, err
	}
	abs := filepath.Join(root, schemaPath)
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", abs, err)
	}

	var rs rawSchema
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, fmt.Errorf("parse schema %s: %w", abs, err)
	}
	cs := compileSchema(rs)

	schemaMu.Lock()
	schemaCache[schemaPath] = cs
	schemaMu.Unlock()
	return cs, nil
}

func resolveProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if st, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("project root with go.mod not found")
}

func compileSchema(rs rawSchema) *compiledSchema {
	cs := &compiledSchema{
		Types:                parseTypes(rs.Type),
		Enum:                 rs.Enum,
		Required:             map[string]struct{}{},
		Properties:           map[string]*compiledSchema{},
		Items:                nil,
		AdditionalProperties: nil,
		MinLength:            rs.MinLength,
		MinItems:             rs.MinItems,
		Minimum:              rs.Minimum,
	}

	for _, f := range rs.Required {
		cs.Required[f] = struct{}{}
	}
	for k, p := range rs.Properties {
		cs.Properties[k] = compileSchema(p)
	}
	if rs.Items != nil {
		cs.Items = compileSchema(*rs.Items)
	}
	if b, ok := rs.AdditionalProperties.(bool); ok {
		cs.AdditionalProperties = &b
	}
	return cs
}

func parseTypes(v any) []string {
	out := []string{}
	switch t := v.(type) {
	case string:
		out = append(out, t)
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func validateValue(v any, s *compiledSchema, path string, errs *[]string) {
	if s == nil {
		return
	}

	if len(s.Types) > 0 && !matchesAnyType(v, s.Types) {
		*errs = append(*errs, fmt.Sprintf("%s: expected type %v", path, s.Types))
		return
	}

	if len(s.Enum) > 0 && !matchesEnum(v, s.Enum) {
		*errs = append(*errs, fmt.Sprintf("%s: value not in enum %v", path, s.Enum))
	}

	switch vv := v.(type) {
	case map[string]any:
		for req := range s.Required {
			if _, ok := vv[req]; !ok {
				*errs = append(*errs, fmt.Sprintf("%s.%s: required", path, req))
			}
		}

		if s.AdditionalProperties != nil && !*s.AdditionalProperties {
			for k := range vv {
				if _, ok := s.Properties[k]; !ok {
					*errs = append(*errs, fmt.Sprintf("%s.%s: additional property not allowed", path, k))
				}
			}
		}

		for k, child := range s.Properties {
			if val, ok := vv[k]; ok {
				validateValue(val, child, path+"."+k, errs)
			}
		}
	case []any:
		if s.MinItems != nil && len(vv) < *s.MinItems {
			*errs = append(*errs, fmt.Sprintf("%s: minItems=%d", path, *s.MinItems))
		}
		if s.Items != nil {
			for i, item := range vv {
				validateValue(item, s.Items, fmt.Sprintf("%s[%d]", path, i), errs)
			}
		}
	case string:
		if s.MinLength != nil && len(vv) < *s.MinLength {
			*errs = append(*errs, fmt.Sprintf("%s: minLength=%d", path, *s.MinLength))
		}
	case float64:
		if s.Minimum != nil && vv < *s.Minimum {
			*errs = append(*errs, fmt.Sprintf("%s: minimum=%v", path, *s.Minimum))
		}
	}
}

func matchesAnyType(v any, types []string) bool {
	for _, t := range types {
		if matchesType(v, t) {
			return true
		}
	}
	return false
}

func matchesType(v any, t string) bool {
	switch t {
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "integer":
		n, ok := v.(float64)
		return ok && math.Trunc(n) == n
	case "number":
		_, ok := v.(float64)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "null":
		return v == nil
	default:
		return false
	}
}

func matchesEnum(v any, enumVals []any) bool {
	for _, ev := range enumVals {
		if reflect.DeepEqual(v, ev) {
			return true
		}
	}
	return false
}
