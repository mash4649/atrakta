package editsafety

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/util"
)

func ValidateCandidate(path string, sourceText string, cfg *contract.EditSafety) error {
	if cfg == nil {
		return nil
	}
	mode := strings.TrimSpace(strings.ToLower(cfg.Mode))
	if mode != "anchor+optional_ast" {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(path))
	policy := policyForExt(ext, cfg)
	switch policy {
	case "off", "":
		return nil
	case "ast":
		if ext == ".go" {
			return validateGoAST(path, sourceText)
		}
		return nil
	case "parse":
		if ext == ".json" {
			return validateJSON(path, sourceText)
		}
		return nil
	default:
		return nil
	}
}

func policyForExt(ext string, cfg *contract.EditSafety) string {
	if cfg != nil && cfg.Languages != nil {
		if v, ok := cfg.Languages[strings.TrimPrefix(ext, ".")]; ok {
			return strings.TrimSpace(strings.ToLower(v))
		}
	}
	switch ext {
	case ".go":
		return "ast"
	case ".json":
		return "parse"
	default:
		return "off"
	}
}

func validateGoAST(path string, sourceText string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, path, util.NormalizeContentLF(sourceText), parser.AllErrors)
	if err != nil {
		return fmt.Errorf("go ast validation failed for %s: %w", path, err)
	}
	return nil
}

func validateJSON(path string, sourceText string) error {
	if !json.Valid([]byte(sourceText)) {
		return fmt.Errorf("json parse validation failed for %s", path)
	}
	return nil
}
