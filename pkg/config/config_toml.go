package config

import (
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
)

func loadConfigFromTOML(data []byte) (*Config, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("TOML parse error: %w", err)
	}
	// Inject current version — TOML configs are always v3
	raw["version"] = CurrentVersion

	// Round-trip through JSON to reuse the existing struct-level unmarshal
	jsonData, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("TOML→JSON marshal error: %w", err)
	}
	return loadConfig(jsonData)
}
