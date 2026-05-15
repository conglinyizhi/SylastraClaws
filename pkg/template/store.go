// Package template provides TOML-based prompt and message template loading.
// It loads a TOML file into a flat map and supports dot-path key access.
// This is intentionally schema-less — the TOML structure is defined entirely
// by the file content, not by Go structs, so users can freely evolve their
// templates without code changes.
package template

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Store holds the parsed TOML template data.
// Keys are dot-separated paths like "responses.file_only.messages".
type Store struct {
	data map[string]any
	path string
}

// LoadFile parses a TOML file into a Store.
// Returns an empty-but-functional Store if the file doesn't exist.
func LoadFile(path string) (*Store, error) {
	s := &Store{
		data: make(map[string]any),
		path: path,
	}

	if path == "" {
		return s, nil
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("template: bad path %q: %w", path, err)
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("template: read %q: %w", abs, err)
	}

	var parsed map[string]any
	if err := toml.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("template: parse %q: %w", abs, err)
	}

	flatten(parsed, "", s.data)
	return s, nil
}

// HasKey returns true if the dot path exists in the store.
func (s *Store) HasKey(dotPath string) bool {
	_, ok := s.data[dotPath]
	return ok
}

// GetString returns the string value at dotPath.
// If the value is a []any (TOML array), picks one at random.
// Returns "" if not found.
func (s *Store) GetString(dotPath string) string {
	val, ok := s.data[dotPath]
	if !ok {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case []any:
		if len(v) == 0 {
			return ""
		}
		// Random pick from array
		idx := rand.Intn(len(v))
		if s, ok := v[idx].(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v[idx])
	case []string:
		if len(v) == 0 {
			return ""
		}
		return v[rand.Intn(len(v))]
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetStringSlice returns all strings at dotPath.
// Returns nil if not found or not a list.
func (s *Store) GetStringSlice(dotPath string) []string {
	val, ok := s.data[dotPath]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return v
	default:
		return nil
	}
}

// flatten converts a nested map into dot-path keys.
// e.g. {"responses": {"file_only": {"messages": ["a", "b"]}}}
// becomes {"responses.file_only.messages": ["a", "b"]}
func flatten(input map[string]any, prefix string, out map[string]any) {
	for key, val := range input {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := val.(type) {
		case map[string]any:
			flatten(v, fullKey, out)
		default:
			out[fullKey] = val
		}
	}
}
